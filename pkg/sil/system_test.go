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

import "testing"

// MSTIME (6.71) takes the time from the host and clears the other two
// fields.
func TestMSTIME(t *testing.T) {
	s, h := ioMachine()
	h.time = 1234
	s.set(sd1, 55, 56, 57)
	s.PC = 3

	s.MSTIME(sd1)
	if got, want := s.full(sd1), [3]int{1234, 0, 0}; got != want {
		t.Errorf("DESCR is %v, want %v", got, want)
	}
	if s.PC != 3 {
		t.Errorf("PC is %d; MSTIME does not branch", s.PC)
	}

	// Note 4: a host with no clock gives zero, which is the documented
	// answer rather than a deviation.
	h.time = 0
	s.MSTIME(sd1)
	if got, want := s.full(sd1), [3]int{0, 0, 0}; got != want {
		t.Errorf("DESCR is %v, want %v", got, want)
	}
}

// DATE (6.22) puts the host's representation of the date into its own
// buffer and specifies it.
func TestDATE(t *testing.T) {
	s, h := ioMachine()
	h.date = []byte("04/01/81")
	s.fullSpec(sp1, 0, 0, 0, 0, 0)

	s.DATE(sp1)
	if got := string(s.Text(sp1)); got != "04/01/81" {
		t.Errorf("DATE gives %q, want %q", got, "04/01/81")
	}
	addr, flag, value, offset, length := s.Specifier(sp1)
	if flag != 0 || value != 0 || offset != 0 || length != 8 {
		t.Errorf("SPEC is %d,%d,%d,%d,%d", addr, flag, value, offset, length)
	}
	if addr < specCore {
		t.Errorf("DATE specifies %d, inside the assembled image", addr)
	}
}

// Note 4: with no calendar, DATE sets the length of SPEC to zero and
// does nothing else -- so the other four fields survive, as they do in
// LOCSP's null string.
func TestDATEWithNoCalendar(t *testing.T) {
	s, _ := ioMachine()
	s.fullSpec(sp1, sc1, 3, 9, 2, 5)

	s.DATE(sp1)
	if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 2, 0}; got != want {
		t.Errorf("SPEC is %v, want %v", got, want)
	}
}

// DATE has its own buffer, as 6.22 note 2 asks, so an outstanding
// INTSPC or REALST specifier is not disturbed by one.
func TestDATEHasItsOwnBuffer(t *testing.T) {
	s, h := ioMachine()
	h.date = []byte("81.092")
	s.set(sd1, 42, 0, 0)
	s.INTSPC(sp1, sd1)
	s.DATE(sp2)

	if got := string(s.Text(sp1)); got != "42" {
		t.Errorf("the INTSPC specifier now reads %q, want %q", got, "42")
	}
	if got := string(s.Text(sp2)); got != "81.092" {
		t.Errorf("the DATE specifier reads %q, want %q", got, "81.092")
	}
}

// The external-function group takes the alternatives 7.1 lists for it,
// each to the label its own section names.
func TestExternalFunctions(t *testing.T) {
	t.Run("LOAD goes to UNDF", func(t *testing.T) {
		s, _ := ioMachine()
		s.Symbols[Undefined] = specOK
		if err := s.LOAD(sd1, sp1, sp2, specFail, specFail); err != nil {
			t.Fatalf("LOAD: %v", err)
		}
		if s.PC != specOK {
			t.Errorf("PC is %d, want %d, the address of UNDF", s.PC, specOK)
		}
	})

	t.Run("LINK goes to INTR10", func(t *testing.T) {
		s, _ := ioMachine()
		s.Symbols[Interrupt] = specOK
		if err := s.LINK(sd1, sd2, sd1, sd2, specFail, specFail); err != nil {
			t.Fatalf("LINK: %v", err)
		}
		if s.PC != specOK {
			t.Errorf("PC is %d, want %d, the address of INTR10", s.PC, specOK)
		}
	})

	// A program that defines neither gets a fault, since there is
	// nowhere to go.
	t.Run("with nowhere to go", func(t *testing.T) {
		s, _ := ioMachine()
		if err := s.LOAD(sd1, sp1, sp2, specFail, specFail); err == nil {
			t.Error("LOAD: no error with UNDF undefined")
		}
		if err := s.LINK(sd1, sd2, sd1, sd2, specFail, specFail); err == nil {
			t.Error("LINK: no error with INTR10 undefined")
		}
	})

	// 6.126 note 2 requires the no-operation rather than permitting
	// it: the source-language UNLOAD undefines ordinary functions too,
	// and this macro is called on the way through.
	t.Run("UNLOAD does nothing", func(t *testing.T) {
		s, _ := ioMachine()
		s.fullSpec(sp1, sc1, 3, 9, 2, 5)
		s.PC = 3
		s.UNLOAD(sp1)
		if s.PC != 3 {
			t.Errorf("PC is %d; UNLOAD does not branch", s.PC)
		}
		if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 2, 5}; got != want {
			t.Errorf("SPEC is %v, want %v", got, want)
		}
	})
}
