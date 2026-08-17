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

package op_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/op"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

// S4D58 6 lists the macros alphabetically and numbers the sections as
// it goes, so an operation's position in the alphabet *is* the section
// that documents it. Sorting the table's mnemonics and reading off the
// index therefore checks three things at once: that Doc cites the
// right section, that no entry is missing, and that the table is in
// the order the enum declares.
func TestDocCitesTheSectionTheAlphabetImplies(t *testing.T) {
	var mnemonics []string
	for _, k := range op.Kinds() {
		mnemonics = append(mnemonics, op.Get(k).Mnemonic)
	}
	if got := len(mnemonics); got != 131 {
		t.Fatalf("the table holds %d operations, want 131", got)
	}
	if !slices.IsSorted(mnemonics) {
		t.Fatal("the table is not in alphabetical order, so the section numbers cannot be checked against it")
	}

	for i, name := range mnemonics {
		want := fmt.Sprintf("S4D58 6.%d", i+1)
		if got := op.Get(op.Lookup(name)).Doc; got != want {
			t.Errorf("%s cites %q, want %q", name, got, want)
		}
	}
}

// Kind is the index of the entry. Anything that reads the table by
// Kind, or reports one, depends on it.
func TestKindIsTheTableIndex(t *testing.T) {
	if got := op.Get(op.Invalid).Mnemonic; got != "" {
		t.Errorf("Invalid has the mnemonic %q, want none", got)
	}
	if got := op.Invalid.String(); got != "Kind(0)" {
		t.Errorf("Invalid prints as %q", got)
	}
	if got := op.Count; got != 131 {
		t.Errorf("Count is %d, want 131", got)
	}
	for _, k := range op.Kinds() {
		e := op.Get(k)
		if e.Mnemonic == "" {
			t.Errorf("Kind(%d) has no entry", uint8(k))
			continue
		}
		if got := op.Lookup(e.Mnemonic); got != k {
			t.Errorf("Lookup(%q) is Kind(%d), want Kind(%d)", e.Mnemonic, uint8(got), uint8(k))
		}
		if got := k.String(); got != e.Mnemonic {
			t.Errorf("Kind(%d) prints as %q, want %q", uint8(k), got, e.Mnemonic)
		}
	}
	if got := op.Lookup("NOTANOP"); got != op.Invalid {
		t.Errorf("Lookup of an unknown mnemonic is Kind(%d), want Invalid", uint8(got))
	}
}

// The twelve directives of S4D58 7.5 are the five assembly control
// macros, the six that assemble data, and PROC, which 6.78 note 2 says
// may be implemented as LHERE. They are the only operations that are
// not executed, and the only ones whose size is not one address unit.
func TestDirectives(t *testing.T) {
	want := []string{
		"ARRAY", "BUFFER", "COPY", "DESCR", "END", "EQU",
		"FORMAT", "LHERE", "PROC", "SPEC", "STRING", "TITLE",
	}

	var got []string
	for _, k := range op.Kinds() {
		e := op.Get(k)
		if e.Directive {
			got = append(got, e.Mnemonic)
		}
		switch {
		case e.Directive && e.Size == op.SizeUnit:
			t.Errorf("%s is a directive but assembles one address unit", e.Mnemonic)
		case !e.Directive && e.Size != op.SizeUnit &&
			e.Mnemonic != "RCALL" && e.Mnemonic != "SELBRA":
			t.Errorf("%s is not a directive but has size %v", e.Mnemonic, e.Size)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("the directives are %v, want %v", got, want)
	}

	// 7.5 puts the twelve in two of its groups, plus PROC among the
	// stack macros.
	for _, k := range op.Kinds() {
		e := op.Get(k)
		if !e.Directive {
			continue
		}
		switch e.Cat {
		case op.CatAssembly, op.CatData:
		case op.CatStack:
			if e.Mnemonic != "PROC" {
				t.Errorf("%s is a directive classified under %s", e.Mnemonic, e.Cat)
			}
		default:
			t.Errorf("%s is a directive classified under %s", e.Mnemonic, e.Cat)
		}
	}
}

// Vector names the operand that becomes the branch vector, and only
// for the two operations that have one.
func TestVector(t *testing.T) {
	for _, k := range op.Kinds() {
		e := op.Get(k)
		i, ok := e.Vector()
		switch e.Mnemonic {
		case "RCALL":
			if !ok || e.Operands[i].Name != "LOCm" {
				t.Errorf("RCALL: Vector is operand %d, %v", i, ok)
			}
		case "SELBRA":
			if !ok || e.Operands[i].Name != "LOCn" {
				t.Errorf("SELBRA: Vector is operand %d, %v", i, ok)
			}
		default:
			if ok {
				t.Errorf("%s has a branch vector at operand %d", e.Mnemonic, i)
			}
		}
	}
}

// The four operations that always transfer control. RCALL and SELBRA
// are deliberately not among them: 6.87 note 6 and 6.98 note 2 let
// both pass control to the operation that follows.
func TestTerminates(t *testing.T) {
	want := []string{"BRANCH", "BRANIC", "ENDEX", "RRTURN"}

	var got []string
	for _, k := range op.Kinds() {
		if e := op.Get(k); e.Terminates {
			got = append(got, e.Mnemonic)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("the terminating operations are %v, want %v", got, want)
	}
	for _, name := range want {
		if op.Get(op.Lookup(name)).Branches() {
			continue
		}
		if name == "BRANIC" || name == "ENDEX" || name == "RRTURN" {
			continue // they transfer without naming where in the operand field
		}
		t.Errorf("%s always transfers but has no branch point", name)
	}
}

// Structural invariants that hold for every entry, so that a hand
// edit to the table cannot leave it in a shape the checker or the
// location counter would misread.
func TestEveryEntryIsWellFormed(t *testing.T) {
	for _, k := range op.Kinds() {
		e := op.Get(k)
		t.Run(e.Mnemonic, func(t *testing.T) {
			if n := e.MinArgs(); n > e.MaxArgs() {
				t.Errorf("MinArgs is %d with %d operands", n, e.MaxArgs())
			} else if n > 0 && e.Operands[n-1].Optional {
				t.Errorf("MinArgs is %d but operand %d is optional", n, n)
			}
			seen := map[string]bool{}
			for i, o := range e.Operands {
				if o.Name == "" {
					t.Errorf("operand %d has no name", i+1)
				}
				if seen[o.Name] {
					t.Errorf("operand %d repeats the name %s", i+1, o.Name)
				}
				seen[o.Name] = true
				if o.Slot == op.SlotNone {
					t.Errorf("%s has no slot", o.Name)
				}
				if (o.Elem != op.SlotNone) != (o.Slot == op.SlotList) {
					t.Errorf("%s: Elem is %v with slot %v", o.Name, o.Elem, o.Slot)
				}
				if o.Slot == op.SlotBranch && !o.Optional {
					// 5.2 makes every branch point omissible. BRANCH is
					// the exception: 6.15 gives it nowhere else to go.
					if e.Mnemonic != "BRANCH" {
						t.Errorf("%s is a branch point but is not optional", o.Name)
					}
				}
			}
			if e.Doc == "" || !strings.HasPrefix(e.Doc, "S4D58 6.") {
				t.Errorf("Doc is %q", e.Doc)
			}
		})
	}
}

// Sizes other than one address unit belong to the six data-assembling
// directives and to the two operations that assemble a branch vector,
// and each takes the operand its size is computed from.
func TestSizesMatchTheirOperands(t *testing.T) {
	for _, tt := range []struct {
		mnemonic string
		size     op.Size
		from     op.Slot
	}{
		{"ARRAY", op.SizeArray, op.SlotConst},
		{"BUFFER", op.SizeBuffer, op.SlotConst},
		{"DESCR", op.SizeDescr, op.SlotNone},
		{"SPEC", op.SizeSpec, op.SlotNone},
		{"STRING", op.SizeString, op.SlotLiteral},
		{"FORMAT", op.SizeChars, op.SlotLiteral},
		{"RCALL", op.SizeCall, op.SlotNone},
		{"SELBRA", op.SizeVector, op.SlotNone},
	} {
		e := op.Get(op.Lookup(tt.mnemonic))
		if e.Size != tt.size {
			t.Errorf("%s has size %v, want %v", tt.mnemonic, e.Size, tt.size)
		}
		if tt.from == op.SlotNone {
			continue
		}
		if len(e.Operands) != 1 || e.Operands[0].Slot != tt.from {
			t.Errorf("%s: size %v needs a single %v operand, found %d operands", tt.mnemonic, tt.size, tt.from, len(e.Operands))
		}
	}
}

// The shape checker.
func TestCheck(t *testing.T) {
	for _, tt := range []struct {
		name string
		line string
		want string // "" when the statement is legal
	}{
		{name: "a full operand list", line: stmt("", "ACOMP", "A,B,C,D,E")},
		{name: "omitted trailing branch points", line: stmt("", "ACOMP", "A,B,C")},
		{name: "omitted interior branch points", line: stmt("", "ACOMP", "A,B,,,E")},
		{name: "no operands at all", line: stmt("P", "PROC", ",")},
		{name: "a list", line: stmt("", "PUSH", "(A,B,C)")},
		{name: "a list of one written without parentheses", line: stmt("", "POP", "A")},
		{name: "an empty exit in a list of branch points", line: stmt("", "RCALL", "A,P,(B),(,C)")},
		{name: "a character literal", line: stmt("S", "STRING", "'HELLO'")},
		{name: "a descriptor with every field omitted", line: stmt("D", "DESCR", ",,")},

		{
			name: "an unknown operation",
			line: stmt("", "MOVEIT", "A,B"),
			want: "unknown operation MOVEIT",
		},
		{
			name: "too many operands",
			line: stmt("", "MOVD", "A,B,C"),
			want: "MOVD takes 2 operands, found 3",
		},
		{
			name: "an operand dropped off the end",
			line: stmt("", "MOVD", "A"),
			want: "MOVD: DESCR2 is required",
		},
		{
			name: "a required operand written as a null",
			line: stmt("", "MOVD", ",B"),
			want: "MOVD: DESCR1 is required",
		},
		{
			name: "a literal where an expression belongs",
			line: stmt("", "MOVD", "A,'B'"),
			want: "MOVD: DESCR2 must be an expression, found literal",
		},
		{
			name: "an expression where a literal belongs",
			line: stmt("S", "STRING", "HELLO"),
			want: "STRING: C1...CL must be a character literal, found expression",
		},
		{
			name: "an empty element in a list of descriptors",
			line: stmt("", "PUSH", "(A,,C)"),
			want: "PUSH: DESCRn has an empty element",
		},
		{
			name: "a list where a single operand belongs",
			line: stmt("", "MOVD", "(A,B),C"),
			want: "MOVD: DESCR1 must be an expression, found list",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ds := op.Check(parse(t, tt.line))
			err := ds.Err()
			switch {
			case tt.want == "" && err != nil:
				t.Errorf("%q reported\n%v", tt.line, err)
			case tt.want != "" && err == nil:
				t.Errorf("%q reported nothing, want a diagnostic containing %q", tt.line, tt.want)
			case tt.want != "" && !strings.Contains(err.Error(), tt.want):
				t.Errorf("%q reported\n%v\nwant a diagnostic containing %q", tt.line, err, tt.want)
			}
		})
	}
}

// Check reports every bad statement rather than the first.
func TestCheckAccumulates(t *testing.T) {
	ds := op.Check(parse(t,
		stmt("", "MOVD", "A"),
		stmt("", "MOVD", "A,B"),
		stmt("", "SUM", "A"),
	))
	if len(ds) != 3 {
		t.Errorf("reported %d diagnostics, want 3:\n%v", len(ds), ds.Err())
	}
}

// stmt renders one SIL statement into its columns (S4D58 7.6).
func stmt(label, o, operand string) string {
	return strings.TrimRight(fmt.Sprintf("%-6s %-6s  %s", label, o, operand), " ")
}

func parse(t *testing.T, lines ...string) []parser.Statement {
	t.Helper()
	src := []byte(strings.Join(lines, "\n") + "\n")
	scanned, ds := scanner.Scan("test.sil", src)
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner reported diagnostics:\n%v", err)
	}
	var pd diag.List
	stmts, pd := parser.Parse(scanned)
	if err := pd.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}
	return stmts
}
