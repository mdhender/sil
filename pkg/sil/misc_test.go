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

// A pattern for LINKOR and LVALUE to walk. Each node is four
// descriptors, its alternative field is the third, and the value the
// chain is searched for is in the fourth. The links are offsets from
// the base of the pattern; see the note at the head of misc.go.
//
//	base+0   the head node
//	base+2   its alternative field: the offset of the next node
//	base+3   its minimum length
func pattern(s *VM, base int, nodes [][2]int) {
	for i, n := range nodes {
		at := base + i*4
		s.set(at+2, n[0], 0, 0)
		s.set(at+3, n[1], 0, 0)
	}
}

// LINKOR (6.56) walks to the end of the chain and writes I there.
func TestLINKOR(t *testing.T) {
	s := nodeMachine()
	// Two nodes: the head links to the one four descriptors on, which
	// ends the chain.
	pattern(s, ndA, [][2]int{{4, 0}, {0, 0}})
	s.set(np1, ndA, 0, 0)
	s.set(np2, 8, 0, 0)

	if err := s.LINKOR(np1, np2); err != nil {
		t.Fatalf("LINKOR: %v", err)
	}
	if got := s.Core[ndA+4+2].A; got != 8 {
		t.Errorf("the end of the chain is %d, want 8", got)
	}
	if got := s.Core[ndA+2].A; got != 4 {
		t.Errorf("the head's link is %d, want 4; only the end changes", got)
	}
}

// A chain of one is the head itself, which is 6.61 note 2's "N1 may be
// zero" seen from LINKOR's side.
func TestLINKORWithOneNode(t *testing.T) {
	s := nodeMachine()
	pattern(s, ndA, [][2]int{{0, 0}})
	s.set(np1, ndA, 0, 0)
	s.set(np2, 12, 0, 0)

	if err := s.LINKOR(np1, np2); err != nil {
		t.Fatalf("LINKOR: %v", err)
	}
	if got := s.Core[ndA+2].A; got != 12 {
		t.Errorf("the head's link is %d, want 12", got)
	}
}

// LVALUE (6.61) is the least of the fourth field over the whole chain.
func TestLVALUE(t *testing.T) {
	for _, tt := range []struct {
		name  string
		nodes [][2]int
		want  int
	}{
		{"one node", [][2]int{{0, 7}}, 7},
		{"the head is least", [][2]int{{4, 2}, {8, 5}, {0, 9}}, 2},
		{"the middle is least", [][2]int{{4, 9}, {8, 2}, {0, 5}}, 2},
		{"the end is least", [][2]int{{4, 9}, {8, 5}, {0, 2}}, 2},
		{"zero counts", [][2]int{{4, 9}, {0, 0}}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := nodeMachine()
			pattern(s, ndA, tt.nodes)
			s.set(np2, ndA, 0, 0)
			s.set(np1, 55, 56, 57)
			if err := s.LVALUE(np1, np2); err != nil {
				t.Fatalf("LVALUE: %v", err)
			}
			if got, want := s.full(np1), [3]int{tt.want, 0, 0}; got != want {
				t.Errorf("DESCR1 is %v, want %v", got, want)
			}
		})
	}
}

// A chain that points at itself would otherwise run forever.
func TestChainWalksTerminate(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"LINKOR": func(s *VM) error { return s.LINKOR(np1, np2) },
		"LVALUE": func(s *VM) error { return s.LVALUE(np1, np2) },
	} {
		s := nodeMachine()
		s.set(np1, ndA, 0, 0)
		s.set(np2, ndA, 0, 0)
		// The head links to the node four descriptors on, and that one
		// links back to itself: never zero, so never an end.
		s.set(ndA+2, 4, 0, 0)
		s.set(ndA+4+2, 4, 0, 0)
		if err := run(s); err == nil {
			t.Errorf("%s: no error on a chain that does not end", name)
		}
	}
}

// LOCAPT and LOCAPV (6.58, 6.59) search the odd and the even
// descriptors of an attribute list, and each leaves DESCR1 addressing
// the start of the pair.
func TestLocateAttributePair(t *testing.T) {
	// A block of two type-value pairs: the title, then T1,V1,T2,V2.
	const block = ndA
	setup := func(s *VM) {
		s.set(block, 0, 9, 4*s.Descr) // the title: 2K*D with K = 2
		s.set(block+1, 11, 0, 0)      // type 1
		s.set(block+2, 21, 0, 0)      // value 1
		s.set(block+3, 12, 0, 0)      // type 2
		s.set(block+4, 22, 0, 0)      // value 2
		s.set(np2, block, 5, 6)
	}

	for _, tt := range []struct {
		name   string
		run    func(*VM) error
		want   [3]int
		wantPC int
	}{
		// Note 1: one descriptor less than the descriptor located, so
		// the second type is at block+3 and DESCR1 addresses block+2.
		{"LOCAPT finds the first type", func(s *VM) error { return s.LOCAPT(np1, np2, np3, specFail, specOK) },
			[3]int{block, 5, 6}, specOK},
		// Note 1 for LOCAPV: two descriptors less, so the first value
		// is at block+2 and DESCR1 addresses block.
		{"LOCAPV finds the first value", func(s *VM) error { return s.LOCAPV(np1, np2, np3, specFail, specOK) },
			[3]int{block, 5, 6}, specOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := nodeMachine()
			setup(s)
			// The one each finds first: type 1 for LOCAPT, value 1 for
			// LOCAPV.
			if tt.name == "LOCAPT finds the first type" {
				s.set(np3, 11, 0, 0)
			} else {
				s.set(np3, 21, 0, 0)
			}
			if err := tt.run(s); err != nil {
				t.Fatalf("%v", err)
			}
			if s.PC != tt.wantPC {
				t.Errorf("PC is %d, want %d", s.PC, tt.wantPC)
			}
			if got := s.full(np1); got != tt.want {
				t.Errorf("DESCR1 is %v, want %v", got, tt.want)
			}
		})
	}

	// The second pair, and the flag and value fields of DESCR1 coming
	// from DESCR2.
	t.Run("LOCAPT finds the second type", func(t *testing.T) {
		s := nodeMachine()
		setup(s)
		s.set(np3, 12, 0, 0)
		if err := s.LOCAPT(np1, np2, np3, specFail, specOK); err != nil {
			t.Fatal(err)
		}
		if got, want := s.full(np1), [3]int{block + 2, 5, 6}; got != want {
			t.Errorf("DESCR1 is %v, want %v", got, want)
		}
	})

	// LOCAPT looks only at types and LOCAPV only at values, so each
	// misses what the other finds.
	t.Run("LOCAPT does not find a value", func(t *testing.T) {
		s := nodeMachine()
		setup(s)
		s.set(np3, 21, 0, 0)
		s.set(np1, 55, 56, 57)
		if err := s.LOCAPT(np1, np2, np3, specFail, specOK); err != nil {
			t.Fatal(err)
		}
		if s.PC != specFail {
			t.Errorf("PC is %d, want %d", s.PC, specFail)
		}
		if got, want := s.full(np1), [3]int{55, 56, 57}; got != want {
			t.Errorf("DESCR1 is %v, want %v; nothing is altered on FLOC", got, want)
		}
	})

	// All three fields have to match, which is the comparison the
	// figure draws.
	t.Run("all three fields", func(t *testing.T) {
		s := nodeMachine()
		setup(s)
		s.set(np3, 11, 1, 0) // the right address, the wrong flag
		if err := s.LOCAPT(np1, np2, np3, specFail, specOK); err != nil {
			t.Fatal(err)
		}
		if s.PC != specFail {
			t.Errorf("PC is %d, want %d", s.PC, specFail)
		}
	})
}

// TOP (6.124) walks back to the first descriptor carrying TTL and
// reports how far it went.
func TestTOP(t *testing.T) {
	const ttl = 1
	for _, tt := range []struct {
		name  string
		from  int
		title int
	}{
		// Note 1: N may be zero.
		{"already at the title", ndA, ndA},
		{"three descriptors down", ndA + 3, ndA},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := nodeMachine()
			s.set(tt.title, 0, ttl, 0)
			s.set(np3, tt.from, 5, 6)
			if err := s.TOP(np1, np2, np3); err != nil {
				t.Fatalf("TOP: %v", err)
			}
			if got, want := s.full(np1), [3]int{tt.title, 5, 6}; got != want {
				t.Errorf("DESCR1 is %v, want %v", got, want)
			}
			if got, want := s.full(np2), [3]int{tt.from - tt.title, 0, 0}; got != want {
				t.Errorf("DESCR2 is %v, want %v", got, want)
			}
		})
	}

	// A block with no title runs off the bottom of core.
	t.Run("no title", func(t *testing.T) {
		s := nodeMachine()
		s.set(np3, ndA, 0, 0)
		if err := s.TOP(np1, np2, np3); err == nil {
			t.Error("no error walking off the bottom of core")
		}
	})

	// TTL is a flag PARMS chooses, so an assembly without it cannot
	// run TOP.
	t.Run("TTL undefined", func(t *testing.T) {
		s := nodeMachine()
		delete(s.Symbols, "TTL")
		if err := s.TOP(np1, np2, np3); err == nil {
			t.Error("no error with TTL undefined")
		}
	})
}

// RPLACE (6.94) rewrites the characters of SPEC1 in place.
func TestRPLACE(t *testing.T) {
	for _, tt := range []struct {
		name             string
		text, from, with string
		want             string
	}{
		{"replace", "ABCABC", "AB", "XY", "XYCXYC"},
		{"no match", "ABC", "Z", "Q", "ABC"},
		{"empty table", "ABC", "", "", "ABC"},
		// Note 1: L may be zero.
		{"empty subject", "", "AB", "XY", ""},
		// Note 2: the last instance of a duplicate wins.
		{"duplicates", "AAA", "AA", "XY", "YYY"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := specMachine()
			s.chars(sc1, tt.text)
			s.fullSpec(sp1, sc1, 0, 0, 0, len(tt.text))
			s.chars(sc2, tt.from)
			s.fullSpec(sp2, sc2, 0, 0, 0, len(tt.from))
			s.chars(sc3, tt.with)
			s.fullSpec(sp3, sc3, 0, 0, 0, len(tt.with))

			if err := s.RPLACE(sp1, sp2, sp3); err != nil {
				t.Fatalf("RPLACE: %v", err)
			}
			if got := string(s.Text(sp1)); got != tt.want {
				t.Errorf("RPLACE gives %q, want %q", got, tt.want)
			}
			// The tables are input only.
			if got := string(s.Text(sp2)); got != tt.from {
				t.Errorf("SPEC2 is now %q, want %q", got, tt.from)
			}
		})
	}
}

// RPLACE reads and writes through offsets the program computed.
func TestRPLACEChecksItsAddresses(t *testing.T) {
	s := specMachine()
	s.fullSpec(sp1, specCore+9, 0, 0, 0, 3)
	s.fullSpec(sp2, sc2, 0, 0, 0, 0)
	s.fullSpec(sp3, sc3, 0, 0, 0, 0)
	if err := s.RPLACE(sp1, sp2, sp3); err == nil {
		t.Error("no error reaching outside core")
	}
}

// SPCINT (6.109) converts a specified string to an integer, and takes
// FLOC for anything else.
func TestSPCINT(t *testing.T) {
	for _, tt := range []struct {
		text string
		want int
		ok   bool
	}{
		{"123", 123, true},
		{"-123", -123, true},
		{"+123", 123, true},
		{"000123", 123, true}, // note 2: leading zeros
		{"", 0, true},         // note 4
		{"-", 0, false},       // note 3: a sign alone
		{"+", 0, false},
		{"1.5", 0, false}, // a real, which is SPREAL's business
		{"12A", 0, false},
		{" 12", 0, false},
		// Note 2: L does not determine whether it fits, so the range
		// is checked on the number.
		{"99999999999999", 0, false},
	} {
		s := realMachine()
		s.chars(sc1, tt.text)
		s.fullSpec(sp1, sc1, 0, 0, 0, len(tt.text))
		s.set(sd1, 55, 56, 57)

		if err := s.SPCINT(sd1, sp1, specFail, specOK); err != nil {
			t.Fatalf("SPCINT(%q): %v", tt.text, err)
		}
		if !tt.ok {
			if s.PC != specFail {
				t.Errorf("SPCINT(%q) went to %d, want %d", tt.text, s.PC, specFail)
			}
			if got, want := s.full(sd1), [3]int{55, 56, 57}; got != want {
				t.Errorf("SPCINT(%q) altered DESCR to %v", tt.text, got)
			}
			continue
		}
		if s.PC != specOK {
			t.Errorf("SPCINT(%q) went to %d, want %d", tt.text, s.PC, specOK)
		}
		if got, want := s.full(sd1), [3]int{tt.want, 0, intCode}; got != want {
			t.Errorf("SPCINT(%q) gives %v, want %v", tt.text, got, want)
		}
	}
}

// VARID (6.127) hashes a string into a chain offset and an ordering
// number. The values themselves are this machine's choice; what the
// document requires is the ranges, that the same string always gives
// the same pair, and that different strings mostly do not.
func TestVARID(t *testing.T) {
	const bins = 16
	machineFor := func(text string) *VM {
		s := realMachine()
		s.Symbols["OBSIZ"] = bins
		s.chars(sc1, text)
		s.fullSpec(sp1, sc1, 0, 0, 0, len(text))
		s.set(sd1, 0, 42, 0)
		return s
	}

	seen := map[[2]int]string{}
	for _, text := range []string{"A", "B", "AB", "BA", "ABC", "X", "OUTPUT", "INPUT"} {
		s := machineFor(text)
		if err := s.VARID(sd1, sp1); err != nil {
			t.Fatalf("VARID(%q): %v", text, err)
		}
		k, m := s.Core[sd1].A, s.Core[sd1].V
		if k < 0 || k > (bins-1)*s.Descr {
			t.Errorf("VARID(%q) gives K = %d, outside 0..%d", text, k, (bins-1)*s.Descr)
		}
		if k%s.Descr != 0 {
			t.Errorf("VARID(%q) gives K = %d, not on a descriptor boundary", text, k)
		}
		if limit := s.Symbols["SIZLIM"]; m < 0 || m > limit {
			t.Errorf("VARID(%q) gives M = %d, outside 0..%d", text, m, limit)
		}
		// The flag field is not named in the figure.
		if f := s.Core[sd1].F; f != 42 {
			t.Errorf("VARID(%q) set the flag field to %d, want 42 unchanged", text, f)
		}

		// The same string, twice.
		again := machineFor(text)
		if err := again.VARID(sd1, sp1); err != nil {
			t.Fatal(err)
		}
		if again.Core[sd1].A != k || again.Core[sd1].V != m {
			t.Errorf("VARID(%q) is not reproducible", text)
		}

		if other, ok := seen[[2]int{k, m}]; ok {
			t.Errorf("VARID(%q) and VARID(%q) give the same pair %d,%d", text, other, k, m)
		}
		seen[[2]int{k, m}] = text
	}
}

// The two numbers have to be different functions (6.127), which an
// anagram is the cheapest check of.
func TestVARIDsTwoNumbersAreDifferentFunctions(t *testing.T) {
	value := func(text string) [2]int {
		s := realMachine()
		s.Symbols["OBSIZ"] = 256
		s.chars(sc1, text)
		s.fullSpec(sp1, sc1, 0, 0, 0, len(text))
		if err := s.VARID(sd1, sp1); err != nil {
			t.Fatal(err)
		}
		return [2]int{s.Core[sd1].A, s.Core[sd1].V}
	}
	if a, b := value("AB"), value("BA"); a == b {
		t.Errorf("AB and BA both give %v", a)
	}
}

// VARID needs two program symbols to know its ranges.
func TestVARIDNeedsItsRanges(t *testing.T) {
	for _, name := range []string{"OBSIZ", "SIZLIM"} {
		s := realMachine()
		s.Symbols["OBSIZ"] = 16
		delete(s.Symbols, name)
		s.fullSpec(sp1, sc1, 0, 0, 0, 1)
		if err := s.VARID(sd1, sp1); err == nil {
			t.Errorf("no error with %s undefined", name)
		}
	}
}

// ORDVST is 6.74 note 1's documented alternative: it performs no
// operation, which leaves a post-mortem dump unalphabetized and
// otherwise correct.
func TestORDVST(t *testing.T) {
	s := machine()
	before := append([]Cell(nil), s.Core...)
	s.PC = 7
	s.ORDVST()
	if s.PC != 7 {
		t.Errorf("PC is %d; ORDVST does not branch", s.PC)
	}
	for a := range s.Core {
		if s.full(a) != [3]int{before[a].A, before[a].F, before[a].V} {
			t.Fatalf("ORDVST altered %d", a)
		}
	}
}
