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

// The four indicator values PARMS chooses (4.2), and a character set
// small enough that two whole tables fit in a test machine. ALPHSZ is
// 256 in the real PARMS; nothing here depends on which it is, which is
// the point of reading it out of the assembly.
const (
	contin = 0
	stopT  = 1
	stopsh = 2
	errorT = 3

	alphsz = 128 // one table, at a character code per entry
	table1 = 0   // the two tables, at the bottom of core
	table2 = alphsz
	styp   = 2*alphsz + 1 // STYPE
	tsp1   = 2*alphsz + 4 // three specifiers
	tsp2   = 2*alphsz + 6
	tsp3   = 2*alphsz + 8
	ttext  = 2*alphsz + 16 // characters
	tset   = 2*alphsz + 32
	tcore  = 2*alphsz + 64

	tError  = 40 // the three branch points of STREAM
	tRunout = 50
	tSloc   = 60
)

func tableMachine() *VM {
	s := machine()
	s.Core = make([]Cell, tcore)
	s.Symbols["CONTIN"] = contin
	s.Symbols["STOP"] = stopT
	s.Symbols["STOPSH"] = stopsh
	s.Symbols["ERROR"] = errorT
	s.Symbols["ALPHSZ"] = alphsz
	s.Symbols["STYPE"] = styp
	// 5572: STYPE is a function descriptor, and STREAM writes only its
	// address field.
	s.set(styp, 0, 8, 0)
	return s
}

// entry reads a syntax table entry: the next table, the indicator, the
// put field (5.1).
func (s *VM) entry(table int, c byte) [3]int {
	at := table + int(c)*s.Descr
	return [3]int{s.Core[at].A, s.Core[at].F, s.Core[at].V}
}

// CLERTB (6.19) sets every indicator, and for CONTIN also points every
// entry back at the table.
func TestCLERTB(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  int
		want [3]int
	}{
		{"ERROR", errorT, [3]int{99, errorT, 7}},
		{"STOP", stopT, [3]int{99, stopT, 7}},
		{"STOPSH", stopsh, [3]int{99, stopsh, 7}},
		// For CONTIN the address field becomes the table and the
		// indicator is cleared; the put field is untouched either way.
		{"CONTIN", contin, [3]int{table1, 0, 7}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := tableMachine()
			for c := 0; c < alphsz; c++ {
				s.set(table1+c, 99, 55, 7)
			}
			if err := s.CLERTB(table1, tt.key); err != nil {
				t.Fatalf("CLERTB: %v", err)
			}
			for _, c := range []byte{0, 'A', alphsz - 1} {
				if got := s.entry(table1, c); got != tt.want {
					t.Errorf("the entry for %d is %v, want %v", c, got, tt.want)
				}
			}
		})
	}
}

// PLUGTB (6.76) does the same thing to the entries the specified
// string names, and leaves the rest alone.
func TestPLUGTB(t *testing.T) {
	s := tableMachine()
	if err := s.CLERTB(table1, errorT); err != nil {
		t.Fatal(err)
	}
	s.chars(tset, "AB")
	s.spec(tsp1, tset, 0, 2)

	if err := s.PLUGTB(table1, stopT, tsp1); err != nil {
		t.Fatalf("PLUGTB: %v", err)
	}
	for _, c := range []byte{'A', 'B'} {
		if got, want := s.entry(table1, c)[1], stopT; got != want {
			t.Errorf("the indicator for %q is %d, want %d", c, got, want)
		}
	}
	if got, want := s.entry(table1, 'C')[1], errorT; got != want {
		t.Errorf("the indicator for %q is %d, want %d; PLUGTB plugs only what it is given", 'C', got, want)
	}

	// CONTIN through PLUGTB points the entry at the table, as it does
	// through CLERTB.
	if err := s.PLUGTB(table1, contin, tsp1); err != nil {
		t.Fatalf("PLUGTB: %v", err)
	}
	if got, want := s.entry(table1, 'A'), [3]int{table1, 0, 0}; got != want {
		t.Errorf("the entry for %q is %v, want %v", 'A', got, want)
	}
}

// The four values PARMS chooses and the two other program symbols are
// read out of the assembly, so an assembly without one cannot run
// these.
func TestSyntaxOperationsNeedTheirSymbols(t *testing.T) {
	for _, name := range []string{"CONTIN", "STOP", "STOPSH", "ERROR", "ALPHSZ"} {
		for op, run := range map[string]func(*VM) error{
			"CLERTB": func(s *VM) error { return s.CLERTB(table1, contin) },
			"PLUGTB": func(s *VM) error { return s.PLUGTB(table1, contin, tsp1) },
			"STREAM": func(s *VM) error { return s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc) },
		} {
			s := tableMachine()
			delete(s.Symbols, name)
			if err := run(s); err == nil {
				t.Errorf("%s: no error with %s undefined", op, name)
			}
		}
	}

	s := tableMachine()
	delete(s.Symbols, "STYPE")
	if err := s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc); err == nil {
		t.Error("STREAM: no error with STYPE undefined")
	}
}

// The four SNOBOL4 primitives the syntax tables exist for, built the
// way the source builds them at lines 2521 to 2581: clear SNABTB to
// one indicator, plug the character set with another, and stream.
//
// This is M7's exit criterion, and it is here as well as in
// pkg/sil/asm/testdata/stream.sil because the table has to be examined
// from both ends -- what STREAM accepted, and what was left.
func TestStreamReproducesANYBREAKNOTANYAndSPAN(t *testing.T) {
	for _, tt := range []struct {
		name        string
		clear, plug int
		subject     string
		wantPC      int
		accepted    string
		left        string
	}{
		// ANY(S): the first character, if it is in S.
		{"ANY matches", errorT, stopT, "BXY", tSloc, "B", "XY"},
		{"ANY fails", errorT, stopT, "XYZ", tError, "", ""},
		// BREAK(S): everything up to the first character in S, which
		// is not accepted.
		{"BREAK stops short", contin, stopsh, "XYAB", tSloc, "XY", "AB"},
		{"BREAK runs out", contin, stopsh, "XYZ", tRunout, "XYZ", ""},
		{"BREAK on the first character", contin, stopsh, "AXY", tSloc, "", "AXY"},
		// NOTANY(S): the first character, if it is not in S.
		{"NOTANY matches", stopT, errorT, "XYZ", tSloc, "X", "YZ"},
		{"NOTANY fails", stopT, errorT, "ABC", tError, "", ""},
		// SPAN(S): the run of characters in S.
		{"SPAN stops short", stopsh, contin, "ABXY", tSloc, "AB", "XY"},
		{"SPAN runs out", stopsh, contin, "ABC", tRunout, "ABC", ""},
		// 6.116 note 2: the null string is RUNOUT.
		{"the null string", errorT, stopT, "", tRunout, "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := tableMachine()
			if err := s.CLERTB(table1, tt.clear); err != nil {
				t.Fatal(err)
			}
			s.chars(tset, "ABC")
			s.spec(tsp3, tset, 0, 3)
			if err := s.PLUGTB(table1, tt.plug, tsp3); err != nil {
				t.Fatal(err)
			}

			s.chars(ttext, tt.subject)
			s.spec(tsp2, ttext, 0, len(tt.subject))
			s.spec(tsp1, 0, 0, 0)

			if err := s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc); err != nil {
				t.Fatalf("STREAM: %v", err)
			}
			if s.PC != tt.wantPC {
				t.Fatalf("PC is %d, want %d", s.PC, tt.wantPC)
			}
			if tt.wantPC == tError {
				// The whole string comes back in SPEC1 and SPEC2 is
				// not altered.
				if got := string(s.Text(tsp1)); got != tt.subject {
					t.Errorf("SPEC1 specifies %q, want %q", got, tt.subject)
				}
				if got := string(s.Text(tsp2)); got != tt.subject {
					t.Errorf("SPEC2 specifies %q, want the whole subject", got)
				}
				if got := s.Core[styp].A; got != 0 {
					t.Errorf("STYPE is %d, want 0 on the error arm", got)
				}
				return
			}
			if got := string(s.Text(tsp1)); got != tt.accepted {
				t.Errorf("accepted %q, want %q", got, tt.accepted)
			}
			if got := string(s.Text(tsp2)); got != tt.left {
				t.Errorf("left %q, want %q", got, tt.left)
			}
		})
	}
}

// STYPE takes the last nonzero put field seen, and keeps its flag: the
// source declares it DESCR 0,FNC,0 and the figure names only the
// address field.
func TestSTREAMCarriesThePutField(t *testing.T) {
	s := tableMachine()
	if err := s.CLERTB(table1, contin); err != nil {
		t.Fatal(err)
	}
	// 'A' puts 11, 'B' puts nothing, 'C' puts 33 and stops. The last
	// nonzero one wins.
	s.Core[table1+'A'].V = 11
	s.Core[table1+'C'].V = 33
	s.Core[table1+'C'].F = stopT
	s.chars(ttext, "ABC")
	s.spec(tsp2, ttext, 0, 3)

	if err := s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc); err != nil {
		t.Fatal(err)
	}
	if got, want := s.full(styp), [3]int{33, 8, 0}; got != want {
		t.Errorf("STYPE is %v, want %v", got, want)
	}

	// With no PUT after the first, the first one is still the last
	// nonzero one.
	s.Core[table1+'C'].V = 0
	s.spec(tsp2, ttext, 0, 3)
	if err := s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc); err != nil {
		t.Fatal(err)
	}
	if got := s.Core[styp].A; got != 11 {
		t.Errorf("STYPE is %d, want 11", got)
	}
}

// GOTO(TABLE) and CONTIN are one mechanism: the entry's address field
// is the table that reads the next character. This is the half of
// STREAM that SNABTB alone cannot reach, since CLERTB and PLUGTB only
// ever point an entry back at its own table.
func TestSTREAMFollowsGOTO(t *testing.T) {
	s := tableMachine()
	if err := s.CLERTB(table1, errorT); err != nil {
		t.Fatal(err)
	}
	if err := s.CLERTB(table2, errorT); err != nil {
		t.Fatal(err)
	}
	// In the first table a digit goes to the second; in the second a
	// letter stops. So "1A" is accepted and "11" is an error.
	s.Core[table1+'1'] = Cell{Kind: Data, A: table2, F: contin}
	s.Core[table2+'A'] = Cell{Kind: Data, A: table2, F: stopT, V: 7}

	for _, tt := range []struct {
		subject string
		wantPC  int
		want    string
	}{
		{"1A", tSloc, "1A"},
		{"11", tError, ""},
	} {
		s.chars(ttext, tt.subject)
		s.spec(tsp2, ttext, 0, len(tt.subject))
		if err := s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc); err != nil {
			t.Fatal(err)
		}
		if s.PC != tt.wantPC {
			t.Errorf("%q went to %d, want %d", tt.subject, s.PC, tt.wantPC)
		}
		if tt.wantPC == tSloc {
			if got := string(s.Text(tsp1)); got != tt.want {
				t.Errorf("%q accepted %q, want %q", tt.subject, got, tt.want)
			}
			if got := s.Core[styp].A; got != 7 {
				t.Errorf("STYPE is %d, want 7", got)
			}
		}
	}
}

// 6.116 note 1: a table may stop on the last character.
func TestSTREAMStopsOnTheLastCharacter(t *testing.T) {
	s := tableMachine()
	if err := s.CLERTB(table1, contin); err != nil {
		t.Fatal(err)
	}
	s.Core[table1+'C'].F = stopT
	s.chars(ttext, "ABC")
	s.spec(tsp2, ttext, 0, 3)

	if err := s.STREAM(tsp1, tsp2, table1, tError, tRunout, tSloc); err != nil {
		t.Fatal(err)
	}
	if s.PC != tSloc {
		t.Errorf("PC is %d, want %d", s.PC, tSloc)
	}
	if got := string(s.Text(tsp1)); got != "ABC" {
		t.Errorf("accepted %q, want %q", got, "ABC")
	}
	if _, _, _, _, l := s.Specifier(tsp2); l != 0 {
		t.Errorf("%d characters are left, want none", l)
	}
}

// The table operations reach addresses the program computed.
func TestSyntaxOperationsCheckTheirAddresses(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"CLERTB": func(s *VM) error { return s.CLERTB(tcore+9, contin) },
		"PLUGTB": func(s *VM) error { return s.PLUGTB(tcore+9, stopT, tsp1) },
		"STREAM": func(s *VM) error { return s.STREAM(tsp1, tsp2, tcore+9, tError, tRunout, tSloc) },
	} {
		s := tableMachine()
		s.chars(tset, "A")
		s.spec(tsp1, tset, 0, 1)
		s.spec(tsp2, tset, 0, 1)
		if err := run(s); err == nil {
			t.Errorf("%s: no error reaching outside core", name)
		}
	}
}
