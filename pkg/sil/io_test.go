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
	"bytes"
	"slices"
	"strings"
	"testing"
)

func ioMachine() (*VM, *recorder) {
	s := specMachine()
	h := &recorder{}
	s.Host = h
	return s, h
}

// OUTPUT (6.75) takes the unit out of an address field, the format out
// of core, and the values out of the address fields of its list.
func TestOUTPUT(t *testing.T) {
	s, h := ioMachine()
	s.chars(sc1, "(1H ,I5)")
	s.set(sd1, 6, 0, 0)
	s.set(sp2, 11, 0, 0)
	s.set(sp3, 22, 0, 0)

	if err := s.OUTPUT(sd1, sc1, 8, []int{sp2, sp3}); err != nil {
		t.Fatalf("OUTPUT: %v", err)
	}
	if h.unit != 6 {
		t.Errorf("unit is %d, want 6", h.unit)
	}
	if got := string(h.format); got != "(1H ,I5)" {
		t.Errorf("format is %q, want %q", got, "(1H ,I5)")
	}
	if want := []int{11, 22}; !slices.Equal(h.values, want) {
		t.Errorf("values are %v, want %v", h.values, want)
	}
}

// An empty list is what an omitted one means, which is how fifteen of
// the sites in the SNOBOL4 source write it.
func TestOUTPUTWithNoValues(t *testing.T) {
	s, h := ioMachine()
	s.chars(sc1, "(1H1)")
	s.set(sd1, 6, 0, 0)
	if err := s.OUTPUT(sd1, sc1, 5, nil); err != nil {
		t.Fatalf("OUTPUT: %v", err)
	}
	if len(h.values) != 0 {
		t.Errorf("values are %v, want none", h.values)
	}
}

func TestOUTPUTChecksItsFormat(t *testing.T) {
	s, _ := ioMachine()
	s.set(sd1, 6, 0, 0)
	if err := s.OUTPUT(sd1, specCore+9, 4, nil); err == nil {
		t.Error("no error with a format outside core")
	}
}

// STREAD (6.115) reads L characters into the specified string and
// takes SLOC. 6.115 note 1 sends a short record to the FORTRAN IV
// convention, which is blanks.
func TestSTREAD(t *testing.T) {
	for _, tt := range []struct {
		name   string
		record string
		length int
		want   string
	}{
		{"exactly", "ABCD", 4, "ABCD"},
		{"short, so padded", "AB", 4, "AB  "},
		{"long, so truncated", "ABCDEF", 4, "ABCD"},
		{"nothing", "", 4, "    "},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, h := ioMachine()
			h.record = []byte(tt.record)
			s.chars(sc1, "xxxxxxxx")
			s.fullSpec(sp1, sc1, 0, 0, 0, tt.length)
			s.set(sd1, 5, 0, 0)

			if err := s.STREAD(sp1, sd1, specFail, specFail, specOK); err != nil {
				t.Fatalf("STREAD: %v", err)
			}
			if s.PC != specOK {
				t.Errorf("PC is %d, want %d", s.PC, specOK)
			}
			if got := string(s.Text(sp1)); got != tt.want {
				t.Errorf("read %q, want %q", got, tt.want)
			}
			// The host is told how much was wanted, so that it can
			// read additional records if it would rather.
			if h.asked != tt.length {
				t.Errorf("the host was asked for %d characters, want %d", h.asked, tt.length)
			}
			if h.unit != 5 {
				t.Errorf("unit is %d, want 5", h.unit)
			}
		})
	}
}

// End of file and a reading error are branches of their own, not
// faults, and neither writes anything.
func TestSTREADBranches(t *testing.T) {
	for _, tt := range []struct {
		name        string
		eof, failed bool
		wantPC      int
	}{
		{"end of file", true, false, 40},
		{"a reading error", false, true, 50},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, h := ioMachine()
			h.atEOF, h.failed = tt.eof, tt.failed
			s.chars(sc1, "xxxx")
			s.fullSpec(sp1, sc1, 0, 0, 0, 4)
			s.set(sd1, 5, 0, 0)

			if err := s.STREAD(sp1, sd1, 40, 50, specOK); err != nil {
				t.Fatalf("STREAD: %v", err)
			}
			if s.PC != tt.wantPC {
				t.Errorf("PC is %d, want %d", s.PC, tt.wantPC)
			}
			if got := string(s.Text(sp1)); got != "xxxx" {
				t.Errorf("the string is %q; nothing is read on this arm", got)
			}
		})
	}
}

func TestSTREADChecksItsAddresses(t *testing.T) {
	s, _ := ioMachine()
	s.fullSpec(sp1, specCore+9, 0, 0, 0, 4)
	if err := s.STREAD(sp1, sd1, specFail, specFail, specOK); err == nil {
		t.Error("no error reaching outside core")
	}
}

// The three positioning operations (6.14, 6.30, 6.92) pass a unit
// number through and alter nothing.
func TestFilePositioning(t *testing.T) {
	s, h := ioMachine()
	s.set(sd1, 7, 0, 0)
	s.PC = 3

	for _, run := range []func() error{
		func() error { return s.BKSPCE(sd1) },
		func() error { return s.ENFILE(sd1) },
		func() error { return s.REWIND(sd1) },
	} {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"BACKSPACE", "ENDFILE", "REWIND"}; !slices.Equal(h.moved, want) {
		t.Errorf("the host was asked for %v, want %v", h.moved, want)
	}
	if h.unit != 7 {
		t.Errorf("unit is %d, want 7", h.unit)
	}
	if s.PC != 3 {
		t.Errorf("PC is %d; none of them branches", s.PC)
	}
}

// None of the three has a failure branch, so a host that cannot do it
// stops the machine.
func TestFilePositioningFaults(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"BKSPCE": func(s *VM) error { return s.BKSPCE(sd1) },
		"ENFILE": func(s *VM) error { return s.ENFILE(sd1) },
		"REWIND": func(s *VM) error { return s.REWIND(sd1) },
	} {
		s, h := ioMachine()
		h.broken = true
		s.set(sd1, 7, 0, 0)
		if err := run(s); err == nil {
			t.Errorf("%s: no error from a host that cannot position", name)
		}
	}
}

// WriterHost is what the tests and the runner use, so what it does
// with a unit it cannot position is part of the contract: nothing,
// rather than an error the machine would turn into a fault.
func TestWriterHost(t *testing.T) {
	var out bytes.Buffer
	h := &WriterHost{W: &out, R: strings.NewReader("first\nsecond\n")}

	if _, err := h.Print(6, []byte("(A)"), []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := h.Output(6, []byte("(I5)"), []int{1, 2}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "hello\n(I5) 1 2\n"; got != want {
		t.Errorf("wrote %q, want %q", got, want)
	}

	for _, want := range []string{"first", "seco"} {
		s, eof, err := h.Read(5, len(want))
		if err != nil || eof {
			t.Fatalf("read %q: eof %v, err %v", want, eof, err)
		}
		if string(s) != want {
			t.Errorf("read %q, want %q", s, want)
		}
	}
	if _, eof, err := h.Read(5, 8); err != nil || !eof {
		t.Errorf("at the end: eof %v, err %v, want true and nil", eof, err)
	}

	// With no reader, every unit is at end of file at once.
	empty := &WriterHost{W: &out}
	if _, eof, err := empty.Read(5, 8); err != nil || !eof {
		t.Errorf("with no reader: eof %v, err %v, want true and nil", eof, err)
	}

	// 6.71 note 4 and 6.22 note 4: no clock and no calendar are
	// documented answers, and they are what keep a run reproducible.
	if h.Time() != 0 {
		t.Errorf("Time is %d, want 0", h.Time())
	}
	if h.Date() != nil {
		t.Errorf("Date is %q, want nothing", h.Date())
	}
	for _, err := range []error{h.Backspace(5), h.EndFile(5), h.Rewind(5)} {
		if err != nil {
			t.Errorf("positioning: %v", err)
		}
	}
}
