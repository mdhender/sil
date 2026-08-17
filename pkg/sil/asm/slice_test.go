/*
 * SIL - SNOBOL Interpretation Language
 * Copyright (c) 2021, Michael D Henderson
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice, this
 *    list of conditions and the following disclaimer.
 *
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
 * CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
 * OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package asm_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil"
	"github.com/mdhender/sil/pkg/sil/asm"
	"github.com/mdhender/sil/pkg/sil/op"
)

// The vertical slice: a SIL program goes from source through the
// assembler into the machine and prints, and both arms of a recursive
// return are taken.
//
// The program adds two numbers in a procedure. The procedure returns
// by exit 1 when the sum is within a limit and by exit 2 when it is
// not. Exit 1 is written as a null in the RCALL, so it is the
// fall-through return: the assembler resolves it to the operation
// after the whole RCALL, and RRTURN's PC = LOC+N lands there with no
// special case. Exit 2 is an alternate return to a named label.
//
// Changing one number in the source takes the other arm, which is what
// makes the pair a test of the dispatch rather than of one path
// through it.
func TestVerticalSlice(t *testing.T) {
	for _, tt := range []struct {
		name string
		acl  string // the value assembled into ACL
		want string
		sum  int
	}{
		{
			name: "the fall-through return",
			acl:  "40",
			want: "SUM IS WITHIN THE LIMIT\n",
			sum:  42,
		},
		{
			name: "the alternate return",
			acl:  "400",
			want: "SUM IS OVER THE LIMIT\n",
			sum:  402,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, trace bytes.Buffer
			vm := assemble(t, slice(t, tt.acl), asm.Options{
				Host:  sil.WriterHost{W: &out},
				Trace: &trace,
			})
			vm.MaxCycles = 1000

			if err := vm.Run(); err != nil {
				t.Fatalf("%v\n%s", err, trace.String())
			}
			if got := out.String(); got != tt.want {
				t.Errorf("printed %q, want %q\n%s", got, tt.want, trace.String())
			}

			// The value RRTURN returned reached the descriptor the
			// RCALL named, which is the other half of the call model.
			if got := vm.Core[vm.Symbols["SUMCL"]].A; got != tt.sum {
				t.Errorf("SUMCL is %d, want %d", got, tt.sum)
			}
			// ENDEX read its operand and stopped.
			if !vm.Halted || vm.Status != 0 {
				t.Errorf("halted %v with status %d, want true and 0", vm.Halted, vm.Status)
			}
			// The stack came back to where ISTACK left it.
			if got, want := vm.CStack, vm.Symbols["STACK"]; got != want {
				t.Errorf("CSTACK is %d, want %d", got, want)
			}
			if vm.OStack != 0 {
				t.Errorf("OSTACK is %d, want 0", vm.OStack)
			}
			t.Logf("%d instructions, %d units of core", vm.Cycles, len(vm.Core))
		})
	}
}

// What the assembler laid down for the call, checked against S4D58
// 6.87's "Return Code at LOC" figure: the RCALL, then the return point
// holding the descriptor the value comes back in, then one cell per
// exit with the omitted one resolved to the operation after them all.
func TestRCALLAssemblesItsBranchVector(t *testing.T) {
	vm := assemble(t, slice(t, "40"), asm.Options{})

	at := -1
	for a, c := range vm.Core {
		if c.Kind == sil.Instr && c.Op == op.RCALL {
			at = a
			break
		}
	}
	if at < 0 {
		t.Fatal("no RCALL in core")
	}

	call := vm.Core[at]
	if got, want := call.Ops, []int{
		vm.Symbols["SUMCL"], vm.Symbols["PLUS"], vm.Symbols["ACL"], vm.Symbols["BCL"],
	}; !equal(got, want) {
		t.Errorf("RCALL operands are %v, want %v", got, want)
	}

	// M is 2, so the whole thing is four cells and the operation after
	// it is at at+4.
	next := at + 4
	if c := vm.Core[at+1]; c.Kind != sil.Return || c.A != vm.Symbols["SUMCL"] {
		t.Errorf("%d is %s, want the return point holding SUMCL", at+1, c)
	}
	// The exits are BRANCH instructions, which is how 6.87 prints
	// them and what makes RRTURN's PC = LOC+N the whole dispatch.
	for _, tt := range []struct {
		at   int
		to   int
		what string
	}{
		{at + 2, next, "exit 1, omitted, so the operation after the RCALL"},
		{at + 3, vm.Symbols["BIG"], "exit 2"},
	} {
		c := vm.Core[tt.at]
		if c.Kind != sil.Instr || c.Op != op.BRANCH || c.Ops[0] != tt.to {
			t.Errorf("%d is %s, want BRANCH %d (%s)", tt.at, c, tt.to, tt.what)
		}
	}
	if vm.Core[next].Kind != sil.Instr || vm.Core[next].Op != op.STPRNT {
		t.Errorf("%d is %s, want the STPRNT the fall-through return lands on", next, vm.Core[next])
	}
}

// An omitted branch point on an ordinary operation is the address of
// the operation after it (5.2), so the machine never sees a null. SUM
// in the slice is written with neither FLOC nor SLOC.
func TestOmittedBranchPointsResolveToFallThrough(t *testing.T) {
	vm := assemble(t, slice(t, "40"), asm.Options{})

	for a, c := range vm.Core {
		if c.Kind != sil.Instr || c.Op != op.SUM {
			continue
		}
		floc, sloc := c.Ops[3], c.Ops[4]
		if floc != a+1 || sloc != a+1 {
			t.Errorf("SUM at %d has FLOC %d and SLOC %d, want %d for both", a, floc, sloc, a+1)
		}
		return
	}
	t.Fatal("no SUM in core")
}

// STRING assembles a specifier and then the characters it points at
// (6.117 note 1). This is the first thing that can catch a specifier
// assembled the wrong width -- see the note in the layout corpus test.
func TestSTRINGAssemblesASpecifierAndItsCharacters(t *testing.T) {
	vm := assemble(t, slice(t, "40"), asm.Options{})

	at := vm.Symbols["SMLSP"]
	addr, _, _, offset, length := vm.Specifier(at)
	if want := at + vm.Spec; addr != want {
		t.Errorf("SMLSP addresses %d, want %d, one specifier past itself", addr, want)
	}
	if offset != 0 {
		t.Errorf("SMLSP has offset %d, want 0", offset)
	}
	const text = "SUM IS WITHIN THE LIMIT"
	if length != len(text) {
		t.Errorf("SMLSP has length %d, want %d", length, len(text))
	}
	if got := string(vm.Text(at)); got != text {
		t.Errorf("SMLSP specifies %q, want %q", got, text)
	}
}

// Core doubles as its own listing: every cell cites the line that
// assembled it.
func TestEveryCellCitesItsSource(t *testing.T) {
	vm := assemble(t, slice(t, "40"), asm.Options{})

	var listing bytes.Buffer
	if err := asm.Listing(&listing, vm); err != nil {
		t.Fatal(err)
	}
	for a, c := range vm.Core {
		if c.Src.File == "" {
			t.Fatalf("%d was never assembled: %s", a, c)
		}
	}
	if n := strings.Count(listing.String(), "\n"); n != len(vm.Core) {
		t.Errorf("the listing has %d lines for %d cells", n, len(vm.Core))
	}
}

// slice returns the vertical slice with the first number replaced, so
// that the two runs differ by one operand and nothing else.
func slice(t *testing.T, acl string) []byte {
	t.Helper()
	src, err := os.ReadFile("testdata/slice.sil")
	if err != nil {
		t.Fatal(err)
	}
	const original = "ACL    DESCR   40,0,0"
	replaced := strings.Replace(string(src), original,
		"ACL    DESCR   "+acl+",0,0", 1)
	if acl != "40" && replaced == string(src) {
		t.Fatalf("testdata/slice.sil no longer contains %q", original)
	}
	return []byte(replaced)
}

func assemble(t *testing.T, src []byte, opts asm.Options) *sil.VM {
	t.Helper()
	vm, ds := asm.Assemble("slice.sil", src, opts)
	if err := ds.Err(); err != nil {
		t.Fatalf("the assembler reported diagnostics:\n%v", err)
	}
	return vm
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The descriptor and address-field batches of S4D58 7.5, as a SIL
// program that checks itself.
//
// Every check is a round trip or an identity the document states --
// what PUSH puts on the stack POP takes off, what PUTDC writes GETDC
// reads, a string structure is one descriptor larger than the storage
// its string needs -- compared with ACOMP. The program keeps the
// number of the check it is about to make in WHICH and ends with it,
// so a failure names itself instead of just being a wrong number.
func TestDescriptorOperations(t *testing.T) { selfChecking(t, "testdata/descriptors.sil") }

// The comparison, flag and value-field batches of S4D58 7.5, the same
// way: each check branches to FAIL on the arm the document says it
// must not take. A comparison has nothing to inspect afterwards --
// none of them alters anything -- so the branch is the only thing
// there is to check, and a SIL program is a better place to check it
// than a Go test that supplies its own idea of where the arms go.
func TestComparisonOperations(t *testing.T) { selfChecking(t, "testdata/compare.sil") }

// selfChecking runs a SIL program that ends with the number of the
// check that failed, or zero.
func selfChecking(t *testing.T, path string) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var trace bytes.Buffer
	vm := assemble(t, src, asm.Options{Trace: &trace})
	vm.MaxCycles = 1000

	if err := vm.Run(); err != nil {
		t.Fatalf("%v\n%s", err, trace.String())
	}
	if vm.Status != 0 {
		t.Errorf("check %d failed\n%s", vm.Status, trace.String())
	}
	t.Logf("%s: %d instructions, %d units of core", path, vm.Cycles, len(vm.Core))
}

// The integer arithmetic batch of S4D58 7.5, checked the same way.
// Each failure arm is checked by arranging for it to be taken: a
// division by zero, zero to the zeroth power, and a sum outside
// SIZLIM, which must also leave its result descriptor alone.
func TestIntegerArithmeticOperations(t *testing.T) { selfChecking(t, "testdata/integers.sil") }

// The specifier batch of S4D58 7.5, less STREAM. The checks are round
// trips and identities compared with LEXCMP -- what SETSP copies is
// lexically equal to what it copied from, what PUTSPC writes GETSPC
// reads, ADDLG undoes SETLC -- and both arms of SUBSP and GETBAL are
// taken, including the two the document gives as failures: a right
// parenthesis and a window with no balanced string in it.
func TestSpecifierOperations(t *testing.T) { selfChecking(t, "testdata/specifiers.sil") }

// The tree and pattern-node batches. Every alteration the sections
// draw is read back with GETDC and compared, and the fields they leave
// blank are given a value beforehand, so that a field the document
// does not name is checked to be unchanged rather than assumed to be.
// This is also where FATHER, LSON, RSIB and CODE are supplied by the
// program, which is where 6.4 note 2 says they come from.
func TestNodeOperations(t *testing.T) { selfChecking(t, "testdata/nodes.sil") }
