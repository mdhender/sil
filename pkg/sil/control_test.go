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
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/op"
)

// PSTACK (6.79) posts the position the stack was in before whatever
// is on top of it was put there.
func TestPSTACK(t *testing.T) {
	s := machine()
	s.CStack = stack + 4
	s.set(d1, 55, 56, 57)
	s.PSTACK(d1)
	if got, want := s.descr(d1), [3]int{stack + 3, 0, 0}; got != want {
		t.Errorf("DESCR is %v, want %v", got, want)
	}
}

// SPUSH and SPOP (6.113, 6.111) are PUSH and POP with a specifier's
// width in place of a descriptor's, and they are inverses.
func TestSPUSHAndSPOP(t *testing.T) {
	s := specMachine()
	s.Symbols["STACK"], s.Symbols["STSIZE"] = specStack, 16
	s.CStack = specStack

	s.fullSpec(sp1, sc1, 1, 2, 3, 4)
	s.fullSpec(sp2, sc2, 5, 6, 7, 8)
	if err := s.SPUSH([]int{sp1, sp2}); err != nil {
		t.Fatalf("SPUSH: %v", err)
	}
	if got, want := s.CStack, specStack+2*s.Spec; got != want {
		t.Errorf("CSTACK is %d, want %d", got, want)
	}
	// The first specifier goes just above the old top, the second
	// above that.
	if got, want := s.readSpec(specStack+s.Descr), [5]int{sc1, 1, 2, 3, 4}; got != want {
		t.Errorf("the first pushed specifier is %v, want %v", got, want)
	}
	if got, want := s.readSpec(specStack+s.Descr+s.Spec), [5]int{sc2, 5, 6, 7, 8}; got != want {
		t.Errorf("the second is %v, want %v", got, want)
	}

	// SPOP takes the top one first, which is the second one pushed.
	s.fullSpec(sp1, 0, 0, 0, 0, 0)
	s.fullSpec(sp2, 0, 0, 0, 0, 0)
	if err := s.SPOP([]int{sp1, sp2}); err != nil {
		t.Fatalf("SPOP: %v", err)
	}
	if got, want := s.CStack, specStack; got != want {
		t.Errorf("CSTACK is %d, want %d", got, want)
	}
	if got, want := s.readSpec(sp1), [5]int{sc2, 5, 6, 7, 8}; got != want {
		t.Errorf("SPEC1 is %v, want %v", got, want)
	}
	if got, want := s.readSpec(sp2), [5]int{sc1, 1, 2, 3, 4}; got != want {
		t.Errorf("SPEC2 is %v, want %v", got, want)
	}
}

// 6.113 note 1: overflow goes to OVER, as PUSH's does.
func TestSPUSHOverflow(t *testing.T) {
	s := specMachine()
	s.Symbols["STACK"], s.Symbols["STSIZE"] = specStack, 2
	s.Symbols[Overflow] = specOK
	s.CStack = specStack

	if err := s.SPUSH([]int{sp1, sp2}); err != nil {
		t.Fatalf("SPUSH: %v", err)
	}
	if s.PC != specOK {
		t.Errorf("PC is %d, want %d, the address of OVER", s.PC, specOK)
	}

	// With no OVER to go to there is nowhere to branch, so it faults.
	delete(s.Symbols, Overflow)
	if err := s.SPUSH([]int{sp1, sp2}); err == nil {
		t.Error("no error with OVER undefined")
	}
}

// 6.111 note 1: underflow goes to INTR10, as POP's does, because it is
// a bug in the implementation of the macro language rather than a
// program that ran out of room.
func TestSPOPUnderflow(t *testing.T) {
	s := specMachine()
	s.Symbols["STACK"] = specStack
	s.Symbols[Interrupt] = specOK
	s.CStack = specStack

	if err := s.SPOP([]int{sp1}); err != nil {
		t.Fatalf("SPOP: %v", err)
	}
	if s.PC != specOK {
		t.Errorf("PC is %d, want %d, the address of INTR10", s.PC, specOK)
	}

	delete(s.Symbols, "STACK")
	if err := s.SPOP([]int{sp1}); err == nil {
		t.Error("no error with STACK undefined")
	}
}

// BRANIC (6.16) reads the location out of core through the address
// field of a descriptor.
func TestBRANIC(t *testing.T) {
	s := machine()
	s.set(d1, str1, 0, 0)
	s.set(str1, 33, 0, 0)
	s.set(str1+2, 44, 0, 0)

	if err := s.BRANIC(d1, 0); err != nil {
		t.Fatalf("BRANIC: %v", err)
	}
	if s.PC != 33 {
		t.Errorf("PC is %d, want 33", s.PC)
	}
	// Note 1 says the source always writes zero, but the operand is
	// an operand, so a nonzero one is added rather than refused.
	if err := s.BRANIC(d1, 2); err != nil {
		t.Fatalf("BRANIC: %v", err)
	}
	if s.PC != 44 {
		t.Errorf("PC is %d, want 44", s.PC)
	}

	s.set(d1, core+9, 0, 0)
	if err := s.BRANIC(d1, 0); err == nil {
		t.Error("no error reaching outside core")
	}
}

// SELBRA (6.98) branches to the I'th cell after itself, which the
// assembler has made a BRANCH. I = N+1 is the operation after the
// whole vector, by note 2, and needs no case of its own.
func TestSELBRA(t *testing.T) {
	const at = 10
	for _, tt := range []struct {
		i    int
		want int
	}{
		{1, at + 1},
		{2, at + 2},
		{3, at + 3},
		{4, at + 4}, // N+1: the operation following
	} {
		s := machine()
		s.set(d1, tt.i, 0, 0)
		s.instr(at, op.SELBRA, d1, 3)
		if err := s.Step(); err != nil {
			t.Fatalf("SELBRA with I = %d: %v", tt.i, err)
		}
		if s.PC != tt.want {
			t.Errorf("SELBRA with I = %d went to %d, want %d", tt.i, s.PC, tt.want)
		}
	}
}

// Note 3 asks for a check that I is in range, which is why the
// assembler puts N in the instruction.
func TestSELBRAChecksItsIndex(t *testing.T) {
	for _, i := range []int{0, -1, 5} {
		s := machine()
		s.set(d1, i, 0, 0)
		s.instr(10, op.SELBRA, d1, 3)
		err := s.Step()
		if err == nil || !strings.Contains(err.Error(), "outside the range") {
			t.Errorf("SELBRA with I = %d reported %v", i, err)
		}
	}
}
