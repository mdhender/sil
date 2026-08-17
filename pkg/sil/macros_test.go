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

package sil

import (
	"errors"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/op"
)

// Instruction tests, built from controlled machine state rather than
// from assembled source. The programs that exercise the same
// operations end to end are in pkg/sil/asm.

// Addresses used throughout, chosen so that a descriptor, a specifier
// and the stack never overlap.
const (
	d1    = 1  // three scratch descriptors
	d2    = 2  //
	d3    = 3  //
	spec1 = 4  // a specifier, two cells
	spec2 = 6  // a second specifier
	str1  = 8  // characters
	str2  = 16 // more characters
	stack = 40 // the system stack
	core  = 64
)

// machine returns a machine with the parameters PARMS chooses, core
// full of data cells, and a stack.
func machine() *VM {
	s := &VM{
		Core: make([]Cell, core),
		Symbols: map[string]int{
			// The subset of PARMS these operations read.
			"STACK": stack, "SIZLIM": 1 << 24, "STTL": 16,
		},
		Descr:     1,
		Spec:      2,
		CPA:       1,
		MaxCycles: 100,
	}
	return s
}

// set writes a descriptor.
func (s *VM) set(a, addr, flag, value int) { s.Core[a] = Cell{Kind: Data, A: addr, F: flag, V: value} }

// instr puts an instruction at an address and points the machine at it.
func (s *VM) instr(at int, k op.Kind, ops ...int) {
	s.Core[at] = Cell{Kind: Instr, Op: k, Ops: ops}
	s.PC = at
}

// ACOMP: the three arms, and that it always assigns the counter.
func TestACOMP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		a1, a2 int
		want   int
	}{
		{"greater", 7, 3, 40},
		{"equal", 3, 3, 50},
		{"less", 1, 3, 60},
		{"negative addresses compare as signed integers", -9, -3, 60},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, tt.a1, 0, 0)
			s.set(d2, tt.a2, 0, 0)
			s.PC = 99
			s.ACOMP(d1, d2, 40, 50, 60)
			if s.PC != tt.want {
				t.Errorf("PC is %d, want %d", s.PC, tt.want)
			}
			// Nothing but the counter moves.
			if s.Core[d1].A != tt.a1 || s.Core[d2].A != tt.a2 {
				t.Errorf("ACOMP altered its operands")
			}
		})
	}
}

// ACOMPC compares against a constant rather than a second descriptor.
func TestACOMPC(t *testing.T) {
	for _, tt := range []struct {
		a, n int
		want int
	}{{7, 3, 40}, {3, 3, 50}, {1, 3, 60}} {
		s := machine()
		s.set(d1, tt.a, 0, 0)
		s.ACOMPC(d1, tt.n, 40, 50, 60)
		if s.PC != tt.want {
			t.Errorf("ACOMPC %d against %d put PC at %d, want %d", tt.a, tt.n, s.PC, tt.want)
		}
	}
}

func TestBRANCH(t *testing.T) {
	s := machine()
	s.PC = 3
	s.BRANCH(42)
	if s.PC != 42 {
		t.Errorf("PC is %d, want 42", s.PC)
	}
}

// ENDEX reads its operand into Status and stops. 6.29: "If I is
// nonzero, a post-mortem dump of user core should be given."
func TestENDEX(t *testing.T) {
	for _, want := range []int{0, 3} {
		s := machine()
		s.set(d1, want, 0, 0)
		s.ENDEX(d1)
		if !s.Halted {
			t.Errorf("ENDEX did not halt")
		}
		if s.Status != want {
			t.Errorf("Status is %d, want %d", s.Status, want)
		}
		// A halted machine does not step.
		s.PC = 0
		if err := s.Step(); err != nil {
			t.Errorf("stepping a halted machine: %v", err)
		}
		if s.Cycles != 0 {
			t.Errorf("a halted machine executed %d instructions", s.Cycles)
		}
	}
}

// ISTACK sets CSTACK to the program symbol STACK and OSTACK to zero
// (6.50), and says so when the program does not define STACK.
func TestISTACK(t *testing.T) {
	s := machine()
	s.CStack, s.OStack = 99, 99
	if err := s.ISTACK(); err != nil {
		t.Fatal(err)
	}
	if s.CStack != stack || s.OStack != 0 {
		t.Errorf("CSTACK is %d and OSTACK is %d, want %d and 0", s.CStack, s.OStack, stack)
	}

	s = machine()
	delete(s.Symbols, "STACK")
	if err := s.ISTACK(); err == nil {
		t.Error("no error with STACK undefined")
	} else if !strings.Contains(err.Error(), "STACK is not defined") {
		t.Errorf("reported %v", err)
	}
}

// MOVD copies all three descriptor fields and nothing else.
func TestMOVD(t *testing.T) {
	s := machine()
	s.set(d1, 1, 2, 3)
	s.set(d2, 40, 50, 60)
	s.MOVD(d1, d2)
	if got := s.Core[d1]; got.A != 40 || got.F != 50 || got.V != 60 {
		t.Errorf("DESCR1 is %s, want 40,50,60", got)
	}
	if got := s.Core[d2]; got.A != 40 || got.F != 50 || got.V != 60 {
		t.Errorf("MOVD altered its source: %s", got)
	}
}

func TestSETAC(t *testing.T) {
	s := machine()
	s.set(d1, 1, 2, 3)
	s.SETAC(d1, 40)
	if got := s.Core[d1]; got.A != 40 || got.F != 2 || got.V != 3 {
		t.Errorf("DESCR is %s, want 40,2,3: SETAC sets only the address field", got)
	}
}

// SUM adds two address fields, keeps the flag and value of the second
// operand, and takes SLOC. A sum outside SIZLIM takes FLOC.
func TestSUM(t *testing.T) {
	s := machine()
	s.set(d2, 40, 7, 9)
	s.set(d3, 2, 0, 0)
	s.SUM(d1, d2, d3, 50, 60)
	if got := s.Core[d1]; got.A != 42 || got.F != 7 || got.V != 9 {
		t.Errorf("DESCR1 is %s, want 42,7,9", got)
	}
	if s.PC != 60 {
		t.Errorf("PC is %d, want SLOC 60", s.PC)
	}

	s = machine()
	s.set(d2, s.Symbols["SIZLIM"], 0, 0)
	s.set(d3, 1, 0, 0)
	s.SUM(d1, d2, d3, 50, 60)
	if s.PC != 50 {
		t.Errorf("PC is %d, want FLOC 50 for a sum outside SIZLIM", s.PC)
	}
	if s.Core[d1].A != 0 {
		t.Errorf("DESCR1 is %s, want it left alone when the sum overflows", s.Core[d1])
	}
}

// POP takes descriptors off the top of the stack in order and moves
// CSTACK down by N descriptors (6.77).
func TestPOP(t *testing.T) {
	s := machine()
	s.CStack = stack + 2
	s.set(stack, 10, 0, 0)
	s.set(stack+1, 20, 0, 0)
	s.set(stack+2, 30, 0, 0)

	if err := s.POP([]int{d1, d2}); err != nil {
		t.Fatal(err)
	}
	if s.Core[d1].A != 30 || s.Core[d2].A != 20 {
		t.Errorf("popped %d and %d, want 30 and 20", s.Core[d1].A, s.Core[d2].A)
	}
	if want := stack; s.CStack != want {
		t.Errorf("CSTACK is %d, want %d", s.CStack, want)
	}
}

// 6.77 note 1: stack underflow transfers to INTR10. A program that
// does not define it gets a fault, because there is nowhere to go.
func TestPOPUnderflow(t *testing.T) {
	s := machine()
	s.CStack = stack
	if err := s.POP([]int{d1, d2}); err == nil {
		t.Fatal("no error popping two descriptors off an empty stack")
	} else if !strings.Contains(err.Error(), "underflow") {
		t.Errorf("reported %v", err)
	}

	s = machine()
	s.CStack = stack
	s.Symbols[Interrupt] = 44
	if err := s.POP([]int{d1, d2}); err != nil {
		t.Fatalf("with %s defined: %v", Interrupt, err)
	}
	if s.PC != 44 {
		t.Errorf("PC is %d, want %s at 44", s.PC, Interrupt)
	}
}

// The call model, built by hand rather than assembled: RCALL lays down
// the frame S4D58 6.87 draws, and RRTURN puts everything back and
// lands on LOC+N.
func TestRCALLAndRRTURN(t *testing.T) {
	const (
		call = 30 // the RCALL
		loc  = 31 // the return point
		exit = 32 // the one exit
		next = 33 // the operation after the whole call
		proc = 36 // the procedure
	)

	s := machine()
	s.CStack, s.OStack = stack, 0
	s.set(d1, 11, 1, 111) // the arguments
	s.set(d2, 22, 2, 222)
	s.Core[loc] = Cell{Kind: Return, A: d3} // where the value comes back
	s.instr(call, op.RCALL, d3, proc, d1, d2)
	s.PC = loc // Step has advanced past the RCALL

	if err := s.RCALL(d3, proc, []int{d1, d2}); err != nil {
		t.Fatal(err)
	}

	// 6.87's figure, with A the value CSTACK had on entry.
	a := stack
	if got := s.Core[a+1]; got.A != 0 || got.F != 0 || got.V != 0 {
		t.Errorf("A+D is %s, want the old OSTACK with zero flags (note 8)", got)
	}
	if got := s.Core[a+2]; got.A != loc || got.F != 0 || got.V != 0 {
		t.Errorf("A+2D is %s, want LOC %d with zero flags", got, loc)
	}
	// Arguments go on in reverse, so the first POP gets the first one.
	if got := s.Core[a+3]; got.A != 22 || got.F != 2 || got.V != 222 {
		t.Errorf("A+3D is %s, want the last argument", got)
	}
	if got := s.Core[a+4]; got.A != 11 || got.F != 1 || got.V != 111 {
		t.Errorf("A+D*(2+N) is %s, want the first argument", got)
	}
	if s.CStack != a+4 || s.OStack != a {
		t.Errorf("CSTACK is %d and OSTACK is %d, want %d and %d", s.CStack, s.OStack, a+4, a)
	}
	if s.PC != proc {
		t.Errorf("PC is %d, want the procedure at %d", s.PC, proc)
	}

	// The procedure pops its arguments in the order they were written.
	if err := s.POP([]int{d1, d2}); err != nil {
		t.Fatal(err)
	}
	if s.Core[d1].A != 11 || s.Core[d2].A != 22 {
		t.Errorf("popped %d and %d, want 11 and 22", s.Core[d1].A, s.Core[d2].A)
	}

	// And returns. N = 1 is the first exit; the machine never knows M.
	s.set(d3, 99, 0, 0)
	if err := s.RRTURN(d3, 1); err != nil {
		t.Fatal(err)
	}
	if s.PC != exit {
		t.Errorf("PC is %d, want LOC+1 = %d", s.PC, exit)
	}
	if s.CStack != a || s.OStack != 0 {
		t.Errorf("CSTACK is %d and OSTACK is %d, want %d and 0", s.CStack, s.OStack, a)
	}
	if got := s.Core[d3].A; got != 99 {
		t.Errorf("the returned descriptor is %d, want 99", got)
	}
	if next <= exit {
		t.Fatal("the addresses in this test are wrong")
	}
}

// 6.87 note 3 and 6.95 note 2: an omitted DESCR means no value comes
// back, which reaches the machine as a zero in the return point.
func TestRRTURNWithNoValue(t *testing.T) {
	const loc, proc = 31, 40
	s := machine()
	s.CStack, s.OStack = stack, 0
	s.Core[loc] = Cell{Kind: Return, A: 0}
	s.PC = loc
	if err := s.RCALL(0, proc, nil); err != nil {
		t.Fatal(err)
	}
	s.set(d3, 99, 0, 0)
	s.set(0, 0, 0, 0)
	if err := s.RRTURN(d3, 1); err != nil {
		t.Fatal(err)
	}
	if s.Core[0].A != 0 {
		t.Errorf("RRTURN stored a value with no descriptor to store it in")
	}
}

// A frame whose return point is not one traps at the return rather
// than letting the machine execute data.
func TestRRTURNChecksTheReturnPoint(t *testing.T) {
	s := machine()
	s.OStack = stack
	s.set(stack+1, 0, 0, 0)
	s.set(stack+2, 7, 0, 0) // 7 is a data cell, not a return point

	err := s.RRTURN(d1, 1)
	if err == nil {
		t.Fatal("no error returning to a cell that is not a return point")
	}
	var f *Fault
	if !errors.As(err, &f) {
		t.Fatalf("reported %T, want a *Fault", err)
	}
	if !strings.Contains(f.Msg, "not a return point") {
		t.Errorf("reported %v", err)
	}
}

// STPRNT walks the print block of 6.114, finds the format in a string
// structure, and hands both to the host.
func TestSTPRNT(t *testing.T) {
	const (
		block = 12 // the print block: unit at block, format at block+1
		title = 16 // the format string structure, characters at title+BCDFLD
		unit  = 6
	)
	s := machine()
	h := &recorder{}
	s.Host = h

	s.set(d2, block-1, 0, 0) // 6.114: DESCR2 addresses A, and A+D is the unit
	s.set(block, unit, 0, 0)
	s.set(block+1, title, 0, 0)
	s.set(title, 0, 0, 3) // the title's value field is the length
	for i, c := range "(A)" {
		s.Core[title+bcdField+i] = Cell{Kind: Data, Ch: byte(c)}
	}

	// The string to print, through a specifier with an offset.
	s.Core[spec1] = Cell{Kind: Data, A: str1}
	s.Core[spec1+1] = Cell{Kind: Data, A: 1, V: 2}
	for i, c := range "xHIy" {
		s.Core[str1+i] = Cell{Kind: Data, Ch: byte(c)}
	}

	h.condition = 5
	if err := s.STPRNT(d1, d2, spec1); err != nil {
		t.Fatal(err)
	}
	if h.unit != unit {
		t.Errorf("printed on unit %d, want %d", h.unit, unit)
	}
	if string(h.format) != "(A)" {
		t.Errorf("format is %q, want %q", h.format, "(A)")
	}
	if string(h.text) != "HI" {
		t.Errorf("printed %q, want %q", h.text, "HI")
	}
	if got := s.Core[d1].A; got != 5 {
		t.Errorf("DESCR1 is %d, want the condition the host signalled", got)
	}
}

// A print block that addresses outside core is a fault, not a panic.
func TestSTPRNTChecksItsBlock(t *testing.T) {
	s := machine()
	s.Host = &recorder{}
	s.set(d2, core+10, 0, 0)
	if err := s.STPRNT(d1, d2, spec1); err == nil {
		t.Error("no error with a print block outside core")
	}
}

// Step advances the counter, counts the instruction, refuses to
// execute a cell that is not one, and names an operation it has no
// implementation for.
func TestStep(t *testing.T) {
	t.Run("advances and counts", func(t *testing.T) {
		s := machine()
		s.set(d2, 1, 0, 0)
		s.instr(10, op.SETAC, d1, 7)
		if err := s.Step(); err != nil {
			t.Fatal(err)
		}
		if s.PC != 11 {
			t.Errorf("PC is %d, want 11", s.PC)
		}
		if s.Cycles != 1 {
			t.Errorf("Cycles is %d, want 1", s.Cycles)
		}
		if s.Core[d1].A != 7 {
			t.Errorf("SETAC did not run")
		}
	})

	t.Run("refuses to execute data", func(t *testing.T) {
		s := machine()
		s.PC = d1
		err := s.Step()
		if err == nil || !strings.Contains(err.Error(), "reached as an instruction") {
			t.Errorf("reported %v", err)
		}
	})

	t.Run("names an unimplemented operation", func(t *testing.T) {
		s := machine()
		// LOAD is the last operation this machine will ever get:
		// PLAN.md's risk register has it at 0.000% of execution time
		// with one call site, and 7.1 makes it optional. Any operation
		// the dispatch does not name will do here.
		s.instr(10, op.LOAD, d1, spec1, spec2, gt, lt)
		err := s.Step()
		if err == nil || !strings.Contains(err.Error(), "LOAD is not implemented") {
			t.Errorf("reported %v", err)
		}
	})

	t.Run("stops a runaway program", func(t *testing.T) {
		s := machine()
		s.MaxCycles = 3
		s.instr(10, op.BRANCH, 10)
		err := s.Run()
		if err == nil || !strings.Contains(err.Error(), "without halting") {
			t.Errorf("reported %v", err)
		}
		if s.Cycles != 3 {
			t.Errorf("ran %d instructions, want 3", s.Cycles)
		}
	})

	t.Run("catches a counter outside core", func(t *testing.T) {
		s := machine()
		s.PC = core + 1
		if err := s.Step(); err == nil {
			t.Error("no error with the counter outside core")
		}
	})
}

// recorder is a Host that keeps what it was asked to print.
type recorder struct {
	unit      int
	format    []byte
	text      []byte
	condition int
}

func (r *recorder) Print(unit int, format, s []byte) (int, error) {
	r.unit, r.format, r.text = unit, format, s
	return r.condition, nil
}
