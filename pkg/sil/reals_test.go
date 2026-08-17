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
	"math"
	"testing"
)

// The two data type codes 5.3 says the source program defines, at the
// values it gives them on lines 293 and 298.
const (
	intCode  = 6
	realCode = 7
)

func realMachine() *VM {
	s := specMachine()
	s.Symbols["I"] = intCode
	s.Symbols["R"] = realCode
	return s
}

// setReal writes a descriptor holding a real number.
func (s *VM) setReal(a int, r float64, flag, value int) {
	s.Core[a] = Cell{Kind: Data, A: realBits(r), F: flag, V: value}
}

func (s *VM) realAt(a int) float64 { return realIn(s.Core[a]) }

// A real number survives every operation that moves a descriptor
// without knowing what is in it, which is the whole reason it lives in
// the address field (3.1.1).
func TestARealTravelsWithItsDescriptor(t *testing.T) {
	s := realMachine()
	s.setReal(sd1, -12.5, 3, realCode)
	s.MOVD(sd2, sd1)
	if got := s.realAt(sd2); got != -12.5 {
		t.Errorf("MOVD moved %v, want %v", got, -12.5)
	}
	if got, want := s.full(sd2), s.full(sd1); got != want {
		t.Errorf("the descriptor is %v, want %v", got, want)
	}
}

// The five arithmetic operations keep the flag and value fields of
// DESCR2, write into the address field of DESCR1, and branch.
func TestRealArithmetic(t *testing.T) {
	const huge = math.MaxFloat64
	for _, tt := range []struct {
		name   string
		run    func(*VM)
		r2, r3 float64
		want   float64
		wantPC int
	}{
		{"ADREAL", func(s *VM) { s.ADREAL(sd1, sp1, sp2, specFail, specOK) }, 1.5, 2.25, 3.75, specOK},
		{"ADREAL out of range", func(s *VM) { s.ADREAL(sd1, sp1, sp2, specFail, specOK) }, huge, huge, 0, specFail},
		{"SBREAL", func(s *VM) { s.SBREAL(sd1, sp1, sp2, specFail, specOK) }, 1.5, 2.25, -0.75, specOK},
		{"MPREAL", func(s *VM) { s.MPREAL(sd1, sp1, sp2, specFail, specOK) }, 1.5, 2.0, 3.0, specOK},
		{"MPREAL out of range", func(s *VM) { s.MPREAL(sd1, sp1, sp2, specFail, specOK) }, huge, 2.0, 0, specFail},
		{"DVREAL", func(s *VM) { s.DVREAL(sd1, sp1, sp2, specFail, specOK) }, 3.0, 2.0, 1.5, specOK},
		// 6.27: R3 = 0 is the failing arm, not an infinity.
		{"DVREAL by zero", func(s *VM) { s.DVREAL(sd1, sp1, sp2, specFail, specOK) }, 3.0, 0, 0, specFail},
		{"EXREAL", func(s *VM) { s.EXREAL(sd1, sp1, sp2, specFail, specOK) }, 2.0, 10.0, 1024.0, specOK},
		{"EXREAL of a fraction", func(s *VM) { s.EXREAL(sd1, sp1, sp2, specFail, specOK) }, 9.0, 0.5, 3.0, specOK},
		// 6.33: "if the result is not a real number".
		{"EXREAL with no real result", func(s *VM) { s.EXREAL(sd1, sp1, sp2, specFail, specOK) }, -9.0, 0.5, 0, specFail},
		{"EXREAL out of range", func(s *VM) { s.EXREAL(sd1, sp1, sp2, specFail, specOK) }, huge, 2.0, 0, specFail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := realMachine()
			s.setReal(sp1, tt.r2, 5, realCode)
			s.setReal(sp2, tt.r3, 9, 99)
			s.set(sd1, 111, 112, 113)

			tt.run(s)
			if s.PC != tt.wantPC {
				t.Fatalf("PC is %d, want %d", s.PC, tt.wantPC)
			}
			if tt.wantPC == specFail {
				// Nothing is written on the failing arm.
				if got, want := s.full(sd1), [3]int{111, 112, 113}; got != want {
					t.Errorf("DESCR1 is %v, want %v", got, want)
				}
				return
			}
			if got := s.realAt(sd1); got != tt.want {
				t.Errorf("DESCR1 holds %v, want %v", got, tt.want)
			}
			// The flag and value fields come from DESCR2.
			if f, v := s.Core[sd1].F, s.Core[sd1].V; f != 5 || v != realCode {
				t.Errorf("DESCR1 has flag %d and value %d, want 5 and %d", f, v, realCode)
			}
		})
	}
}

// MNREAL (6.63) has no failing arm and keeps the other two fields.
func TestMNREAL(t *testing.T) {
	s := realMachine()
	s.setReal(sp1, 12.5, 5, realCode)
	s.PC = 0
	s.MNREAL(sd1, sp1)
	if got := s.realAt(sd1); got != -12.5 {
		t.Errorf("DESCR1 holds %v, want %v", got, -12.5)
	}
	if f, v := s.Core[sd1].F, s.Core[sd1].V; f != 5 || v != realCode {
		t.Errorf("DESCR1 has flag %d and value %d, want 5 and %d", f, v, realCode)
	}
	if s.PC != 0 {
		t.Errorf("PC is %d; MNREAL does not branch", s.PC)
	}
	s.MNREAL(sd1, sd1)
	if got := s.realAt(sd1); got != 12.5 {
		t.Errorf("negated twice gives %v, want %v", got, 12.5)
	}
}

// RCOMP (6.88) is the three-way comparison for reals. 6.88's own
// sentence sends R1 = R2 to GTLOC, which is a slip; EQLOC is in the
// operand list.
func TestRCOMP(t *testing.T) {
	for _, tt := range []struct {
		r1, r2 float64
		want   int
	}{
		{2.5, 1.5, gt},
		{1.5, 1.5, eq},
		{1.5, 2.5, lt},
		{-1.5, -2.5, gt},
		{0, -0.0, eq},
	} {
		s := realMachine()
		s.setReal(sd1, tt.r1, 0, 0)
		s.setReal(sd2, tt.r2, 0, 0)
		s.RCOMP(sd1, sd2, gt, eq, lt)
		if s.PC != tt.want {
			t.Errorf("RCOMP(%v,%v) went to %d, want %d", tt.r1, tt.r2, s.PC, tt.want)
		}
	}
}

// INTRL and RLINT (6.48, 6.93) convert between the two numeric types
// and stamp the value field with the type code the source defines.
func TestINTRLAndRLINT(t *testing.T) {
	s := realMachine()
	s.set(sd1, -7, 9, 99)
	if err := s.INTRL(sd2, sd1); err != nil {
		t.Fatalf("INTRL: %v", err)
	}
	if got := s.realAt(sd2); got != -7 {
		t.Errorf("INTRL gives %v, want %v", got, -7.0)
	}
	if f, v := s.Core[sd2].F, s.Core[sd2].V; f != 0 || v != realCode {
		t.Errorf("INTRL leaves flag %d and value %d, want 0 and %d", f, v, realCode)
	}

	// Note 2: the fractional part is discarded, toward zero.
	for _, tt := range []struct {
		r    float64
		want int
	}{{7.9, 7}, {-7.9, -7}, {0.4, 0}} {
		s.setReal(sd1, tt.r, 0, realCode)
		if err := s.RLINT(sd2, sd1, specFail, specOK); err != nil {
			t.Fatalf("RLINT: %v", err)
		}
		if s.PC != specOK {
			t.Errorf("RLINT(%v) went to %d, want %d", tt.r, s.PC, specOK)
		}
		if got := s.Core[sd2].A; got != tt.want {
			t.Errorf("RLINT(%v) gives %d, want %d", tt.r, got, tt.want)
		}
		if f, v := s.Core[sd2].F, s.Core[sd2].V; f != 0 || v != intCode {
			t.Errorf("RLINT leaves flag %d and value %d, want 0 and %d", f, v, intCode)
		}
	}

	// "If the magnitude of R exceeds the magnitude of the largest
	// integer, transfer is to FLOC." SIZLIM is that magnitude.
	for _, r := range []float64{1 << 30, -(1 << 30), math.MaxFloat64} {
		s.setReal(sd1, r, 0, realCode)
		s.set(sd2, 55, 56, 57)
		if err := s.RLINT(sd2, sd1, specFail, specOK); err != nil {
			t.Fatalf("RLINT: %v", err)
		}
		if s.PC != specFail {
			t.Errorf("RLINT(%v) went to %d, want %d", r, s.PC, specFail)
		}
		if got, want := s.full(sd2), [3]int{55, 56, 57}; got != want {
			t.Errorf("RLINT(%v) altered DESCR1 to %v, want %v", r, got, want)
		}
	}
}

// The conversions need the type codes the source program defines
// (6.48 note 1, 6.93 note 3, 6.112 note 1).
func TestConversionsNeedTheTypeCodes(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"INTRL":  func(s *VM) error { return s.INTRL(sd1, sd2) },
		"RLINT":  func(s *VM) error { return s.RLINT(sd1, sd2, specFail, specOK) },
		"SPREAL": func(s *VM) error { return s.SPREAL(sd1, sp1, specFail, specOK) },
	} {
		s := realMachine()
		delete(s.Symbols, "I")
		delete(s.Symbols, "R")
		if err := run(s); err == nil {
			t.Errorf("%s: no error with the type codes undefined", name)
		}
	}
}

// REALST (6.89) writes the SNOBOL4 representation: a decimal point, a
// digit before it, a minus sign for a negative, and no exponent.
func TestREALST(t *testing.T) {
	for _, tt := range []struct {
		r    float64
		want string
	}{
		{0, "0.0"},
		{1, "1.0"},
		{-1, "-1.0"},
		{12.5, "12.5"},
		{-0.25, "-0.25"},
		{1e21, "1000000000000000000000.0"},
	} {
		s := realMachine()
		s.setReal(sd1, tt.r, 0, realCode)
		s.REALST(sp1, sd1)
		if got := string(s.Text(sp1)); got != tt.want {
			t.Errorf("REALST(%v) gives %q, want %q", tt.r, got, tt.want)
		}
		addr, flag, value, offset, length := s.Specifier(sp1)
		if flag != 0 || value != 0 || offset != 0 || length != len(tt.want) {
			t.Errorf("REALST(%v) gives flag %d, value %d, offset %d, length %d",
				tt.r, flag, value, offset, length)
		}
		if addr < specCore {
			t.Errorf("REALST(%v) specifies %d, inside the assembled image", tt.r, addr)
		}
	}
}

// 6.49 note 2 and 6.89 note 3 each promise only that a buffer survives
// until the next use of its own operation, so INTSPC and REALST cannot
// share one.
func TestINTSPCAndREALSTHaveSeparateBuffers(t *testing.T) {
	s := realMachine()
	s.set(sd1, 42, 0, 0)
	s.INTSPC(sp1, sd1)
	s.setReal(sd2, 1.5, 0, realCode)
	s.REALST(sp2, sd2)

	if got := string(s.Text(sp1)); got != "42" {
		t.Errorf("the INTSPC specifier now reads %q, want %q", got, "42")
	}
	if got := string(s.Text(sp2)); got != "1.5" {
		t.Errorf("the REALST specifier reads %q, want %q", got, "1.5")
	}
}

// SPREAL (6.112) converts a specified string, and takes FLOC for
// anything that is not a SNOBOL4 real.
func TestSPREAL(t *testing.T) {
	for _, tt := range []struct {
		text string
		want float64
		ok   bool
	}{
		{"1.5", 1.5, true},
		{"-1.5", -1.5, true},
		{"+1.5", 1.5, true},
		{"0001.5", 1.5, true}, // note 2: leading zeros
		{"12", 12, true},      // the point is not required; see the doc comment
		{".5", 0.5, true},
		{"1.", 1, true},
		{"", 0, true}, // note 3
		{"1.5.5", 0, false},
		{"1E5", 0, false}, // not a SNOBOL4 real
		{"Inf", 0, false},
		{"NaN", 0, false},
		{" 1.5", 0, false},
		{"-", 0, false},
		{"abc", 0, false},
		{"1e999999", 0, false}, // out of the range available for reals
	} {
		s := realMachine()
		s.chars(sc1, tt.text)
		s.fullSpec(sp1, sc1, 0, 0, 0, len(tt.text))
		s.set(sd1, 55, 56, 57)

		if err := s.SPREAL(sd1, sp1, specFail, specOK); err != nil {
			t.Fatalf("SPREAL(%q): %v", tt.text, err)
		}
		if !tt.ok {
			if s.PC != specFail {
				t.Errorf("SPREAL(%q) went to %d, want %d", tt.text, s.PC, specFail)
			}
			if got, want := s.full(sd1), [3]int{55, 56, 57}; got != want {
				t.Errorf("SPREAL(%q) altered DESCR to %v", tt.text, got)
			}
			continue
		}
		if s.PC != specOK {
			t.Errorf("SPREAL(%q) went to %d, want %d", tt.text, s.PC, specOK)
		}
		if got := s.realAt(sd1); got != tt.want {
			t.Errorf("SPREAL(%q) gives %v, want %v", tt.text, got, tt.want)
		}
		if f, v := s.Core[sd1].F, s.Core[sd1].V; f != 0 || v != realCode {
			t.Errorf("SPREAL(%q) leaves flag %d and value %d, want 0 and %d",
				tt.text, f, v, realCode)
		}
	}
}

// REALST and SPREAL are inverses over the numbers SNOBOL4 can write.
func TestRealTextRoundTrips(t *testing.T) {
	for _, r := range []float64{0, 1, -1, 0.5, -0.125, 12345.6789, 1e-5} {
		s := realMachine()
		s.setReal(sd1, r, 0, realCode)
		s.REALST(sp1, sd1)
		if err := s.SPREAL(sd2, sp1, specFail, specOK); err != nil {
			t.Fatalf("SPREAL: %v", err)
		}
		if s.PC != specOK {
			t.Fatalf("SPREAL rejected %q, which REALST wrote for %v", string(s.Text(sp1)), r)
		}
		if got := s.realAt(sd2); got != r {
			t.Errorf("%v came back as %v", r, got)
		}
	}
}

// SPREAL reads through a specifier the program computed.
func TestSPREALChecksItsAddresses(t *testing.T) {
	s := realMachine()
	s.fullSpec(sp1, specCore+9, 0, 0, 0, 4)
	if err := s.SPREAL(sd1, sp1, specFail, specOK); err == nil {
		t.Error("no error reaching outside core")
	}
}
