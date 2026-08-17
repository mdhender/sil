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

// The three arms of a comparison, and the two of an equality test.
const (
	gt = 40
	eq = 50
	lt = 60
	ne = 70
)

// spec writes a specifier: the address, the offset, and the length.
func (s *VM) spec(at, addr, offset, length int) {
	s.Core[at] = Cell{Kind: Data, A: addr}
	s.Core[at+1] = Cell{Kind: Data, A: offset, V: length}
}

// chars writes characters.
func (s *VM) chars(at int, t string) {
	for i := 0; i < len(t); i++ {
		s.Core[at+i] = Cell{Kind: Data, Ch: t[i]}
	}
}

// Every comparison branches and alters nothing. The three-way ones
// take the same shape, so they are tabulated together: each case sets
// up two descriptors or two specifiers and says which arm the document
// requires.
func TestThreeWayComparisons(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(*VM)
		run   func(*VM) error
		want  int
	}{
		{
			// I1 = 1, I2 = 2, L = 5, so L+I2 > I1.
			name:  "CHKVAL: L+I2 against I1",
			setup: func(s *VM) { s.set(d1, 1, 0, 0); s.set(d2, 2, 0, 0); s.spec(spec1, str1, 0, 5) },
			run:   func(s *VM) error { s.CHKVAL(d1, d2, spec1, gt, eq, lt); return nil },
			want:  gt,
		},
		{
			name:  "CHKVAL when they are equal",
			setup: func(s *VM) { s.set(d1, 7, 0, 0); s.set(d2, 2, 0, 0); s.spec(spec1, str1, 0, 5) },
			run:   func(s *VM) error { s.CHKVAL(d1, d2, spec1, gt, eq, lt); return nil },
			want:  eq,
		},
		{
			name:  "CHKVAL when the integer is larger",
			setup: func(s *VM) { s.set(d1, 9, 0, 0); s.set(d2, 2, 0, 0); s.spec(spec1, str1, 0, 5) },
			run:   func(s *VM) error { s.CHKVAL(d1, d2, spec1, gt, eq, lt); return nil },
			want:  lt,
		},
		{
			name:  "LCOMP: the first specifier is longer",
			setup: func(s *VM) { s.spec(spec1, str1, 0, 5); s.spec(spec2, str1, 0, 2) },
			run:   func(s *VM) error { s.LCOMP(spec1, spec2, gt, eq, lt); return nil },
			want:  gt,
		},
		{
			name:  "LCOMP: the same length",
			setup: func(s *VM) { s.spec(spec1, str1, 0, 5); s.spec(spec2, str1, 0, 5) },
			run:   func(s *VM) error { s.LCOMP(spec1, spec2, gt, eq, lt); return nil },
			want:  eq,
		},
		{
			name:  "LCOMP: the second specifier is longer",
			setup: func(s *VM) { s.spec(spec1, str1, 0, 1); s.spec(spec2, str1, 0, 5) },
			run:   func(s *VM) error { s.LCOMP(spec1, spec2, gt, eq, lt); return nil },
			want:  lt,
		},
		{
			name:  "VCMPIC: the indirect value field is smaller",
			setup: func(s *VM) { s.set(d1, blk-1, 0, 0); s.set(d2, 0, 0, 9); s.set(blk, 0, 0, 4) },
			run:   func(s *VM) error { return s.VCMPIC(d1, 1, d2, gt, eq, lt) },
			want:  lt,
		},
		{
			name:  "VCMPIC: equal value fields",
			setup: func(s *VM) { s.set(d1, blk-1, 0, 0); s.set(d2, 0, 0, 4); s.set(blk, 0, 0, 4) },
			run:   func(s *VM) error { return s.VCMPIC(d1, 1, d2, gt, eq, lt) },
			want:  eq,
		},
		{
			name:  "VCMPIC: the indirect value field is larger",
			setup: func(s *VM) { s.set(d1, blk-1, 0, 0); s.set(d2, 0, 0, 1); s.set(blk, 0, 0, 4) },
			run:   func(s *VM) error { return s.VCMPIC(d1, 1, d2, gt, eq, lt) },
			want:  gt,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			tt.setup(s)
			s.PC = 99
			if err := tt.run(s); err != nil {
				t.Fatal(err)
			}
			if s.PC != tt.want {
				t.Errorf("PC is %d, want %d", s.PC, tt.want)
			}
		})
	}
}

// The equality tests, each with both arms.
func TestEqualityTests(t *testing.T) {
	for _, tt := range []struct {
		name string
		// setup writes the state; run makes the test.
		setup func(*VM)
		run   func(*VM) error
		want  int
	}{
		{
			name:  "AEQL on equal address fields",
			setup: func(s *VM) { s.set(d1, 7, 1, 2); s.set(d2, 7, 3, 4) },
			run:   func(s *VM) error { s.AEQL(d1, d2, ne, eq); return nil },
			want:  eq,
		},
		{
			name:  "AEQL on different address fields",
			setup: func(s *VM) { s.set(d1, 7, 0, 0); s.set(d2, 8, 0, 0) },
			run:   func(s *VM) error { s.AEQL(d1, d2, ne, eq); return nil },
			want:  ne,
		},
		{
			name:  "AEQLC against a constant",
			setup: func(s *VM) { s.set(d1, 7, 0, 0) },
			run:   func(s *VM) error { s.AEQLC(d1, 7, ne, eq); return nil },
			want:  eq,
		},
		{
			name:  "AEQLC against a different constant",
			setup: func(s *VM) { s.set(d1, 7, 0, 0) },
			run:   func(s *VM) error { s.AEQLC(d1, 8, ne, eq); return nil },
			want:  ne,
		},
		{
			name:  "AEQLIC through an offset",
			setup: func(s *VM) { s.set(d1, blk, 0, 0); s.set(blk+1, 7, 0, 0) },
			run:   func(s *VM) error { return s.AEQLIC(d1, 1, 7, ne, eq) },
			want:  eq,
		},
		{
			name:  "AEQLIC with a different constant",
			setup: func(s *VM) { s.set(d1, blk, 0, 0); s.set(blk+1, 7, 0, 0) },
			run:   func(s *VM) error { return s.AEQLIC(d1, 1, 8, ne, eq) },
			want:  ne,
		},
		{
			name:  "DEQL needs all three fields (6.24 note 1)",
			setup: func(s *VM) { s.set(d1, 1, 2, 3); s.set(d2, 1, 2, 3) },
			run:   func(s *VM) error { s.DEQL(d1, d2, ne, eq); return nil },
			want:  eq,
		},
		{
			name:  "DEQL with a different flag field",
			setup: func(s *VM) { s.set(d1, 1, 2, 3); s.set(d2, 1, 9, 3) },
			run:   func(s *VM) error { s.DEQL(d1, d2, ne, eq); return nil },
			want:  ne,
		},
		{
			name:  "DEQL with a different value field",
			setup: func(s *VM) { s.set(d1, 1, 2, 3); s.set(d2, 1, 2, 9) },
			run:   func(s *VM) error { s.DEQL(d1, d2, ne, eq); return nil },
			want:  ne,
		},
		{
			name:  "LEQLC on a specifier length",
			setup: func(s *VM) { s.spec(spec1, str1, 0, 5) },
			run:   func(s *VM) error { s.LEQLC(spec1, 5, ne, eq); return nil },
			want:  eq,
		},
		{
			name:  "LEQLC on a different length",
			setup: func(s *VM) { s.spec(spec1, str1, 0, 5) },
			run:   func(s *VM) error { s.LEQLC(spec1, 4, ne, eq); return nil },
			want:  ne,
		},
		{
			name:  "VEQL on equal value fields",
			setup: func(s *VM) { s.set(d1, 1, 2, 3); s.set(d2, 9, 9, 3) },
			run:   func(s *VM) error { s.VEQL(d1, d2, ne, eq); return nil },
			want:  eq,
		},
		{
			name:  "VEQL on different value fields",
			setup: func(s *VM) { s.set(d1, 1, 2, 3); s.set(d2, 1, 2, 4) },
			run:   func(s *VM) error { s.VEQL(d1, d2, ne, eq); return nil },
			want:  ne,
		},
		{
			name:  "VEQLC against a constant",
			setup: func(s *VM) { s.set(d1, 1, 2, 3) },
			run:   func(s *VM) error { s.VEQLC(d1, 3, ne, eq); return nil },
			want:  eq,
		},
		{
			name:  "VEQLC against a different constant",
			setup: func(s *VM) { s.set(d1, 1, 2, 3) },
			run:   func(s *VM) error { s.VEQLC(d1, 4, ne, eq); return nil },
			want:  ne,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			tt.setup(s)
			before := make([][3]int, len(s.Core))
			for a := range s.Core {
				before[a] = s.descr(a)
			}
			s.PC = 99
			if err := tt.run(s); err != nil {
				t.Fatal(err)
			}
			if s.PC != tt.want {
				t.Errorf("PC is %d, want %d", s.PC, tt.want)
			}
			for a := range before {
				if got := s.descr(a); got != before[a] {
					t.Errorf("a comparison altered %d: %v, was %v", a, got, before[a])
				}
			}
		})
	}
}

// TESTF and TESTFI take SLOC when the flag is present, FLOC when it is
// not, and a descriptor may carry more than one flag (3.1.2).
func TestFlagTests(t *testing.T) {
	const (
		floc = 40
		sloc = 50
		ttl  = 1
		mark = 2
		ptr  = 4
	)
	for _, tt := range []struct {
		name string
		f    int
		flag int
		want int
	}{
		{"the only flag", ttl, ttl, sloc},
		{"one of several", ttl + mark, mark, sloc},
		{"a flag that is not there", ttl + mark, ptr, floc},
		{"no flags at all", 0, ttl, floc},
		{"a sum of flags, both present", ttl + mark + ptr, ttl + mark, sloc},
		{"a sum of flags, one missing", ttl + ptr, ttl + mark, floc},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, 0, tt.f, 0)
			s.TESTF(d1, tt.flag, floc, sloc)
			if s.PC != tt.want {
				t.Errorf("TESTF put PC at %d, want %d", s.PC, tt.want)
			}

			s = machine()
			s.set(d1, blk, 0, 0)
			s.set(blk, 0, tt.f, 0)
			if err := s.TESTFI(d1, tt.flag, floc, sloc); err != nil {
				t.Fatal(err)
			}
			if s.PC != tt.want {
				t.Errorf("TESTFI put PC at %d, want %d", s.PC, tt.want)
			}
		})
	}
}

// LEXCMP, and the one place the document is wrong about it.
//
// 6.53's prose says a lesser SPEC1 takes GTLOC. LGT at line 4485 of
// the SNOBOL4 source says otherwise, and LGT is the only one of the
// twelve sites that can tell -- see the implementation's note.
func TestLEXCMP(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want int
	}{
		{"ABC", "ABD", lt},
		{"ABD", "ABC", gt},
		{"ABC", "ABC", eq},
		// 6.53 note 2: an initial substring is less than the string.
		{"ABC", "ABCA", lt},
		{"ABCA", "ABC", gt},
		// Note 3: the null string is less than any other.
		{"", "A", lt},
		{"A", "", gt},
		{"", "", eq},
	} {
		t.Run(tt.a+" against "+tt.b, func(t *testing.T) {
			s := machine()
			s.spec(spec1, str1, 0, len(tt.a))
			s.chars(str1, tt.a)
			s.spec(spec2, str2, 0, len(tt.b))
			s.chars(str2, tt.b)

			s.LEXCMP(spec1, spec2, gt, eq, lt)
			if s.PC != tt.want {
				t.Errorf("PC is %d, want %d", s.PC, tt.want)
			}
		})
	}
}

// LEXCMP reads through the offset, so a substring of a longer string
// compares as itself (3.2).
func TestLEXCMPReadsThroughTheOffset(t *testing.T) {
	s := machine()
	s.chars(str1, "xxABCxx")
	s.spec(spec1, str1, 2, 3)
	s.chars(str2, "ABC")
	s.spec(spec2, str2, 0, 3)
	s.LEXCMP(spec1, spec2, gt, eq, lt)
	if s.PC != eq {
		t.Errorf("PC is %d, want %d", s.PC, eq)
	}
}

// The comparisons that reach an address the program computed can be
// pointed outside core.
func TestComparisonsCheckTheirAddresses(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"AEQLIC": func(s *VM) error { return s.AEQLIC(d1, 0, 0, ne, eq) },
		"TESTFI": func(s *VM) error { return s.TESTFI(d1, 1, gt, lt) },
		"VCMPIC": func(s *VM) error { return s.VCMPIC(d1, 0, d2, gt, eq, lt) },
	} {
		s := machine()
		s.set(d1, core+9, 0, 0)
		if err := run(s); err == nil {
			t.Errorf("%s: no error reaching outside core", name)
		}
	}
}

// The flag and value operations, each of which writes one field and
// leaves the others alone.
func TestFieldOperations(t *testing.T) {
	const (
		ttl  = 1
		mark = 2
		ptr  = 4
	)
	for _, tt := range []struct {
		name  string
		run   func(*VM) error
		d1    [3]int
		d2    [3]int
		at    [3]int
		want  [3]int
		where int
	}{
		{
			name: "SETF adds a flag and leaves the others",
			run:  func(s *VM) error { s.SETF(d1, ptr); return nil },
			d1:   [3]int{1, ttl + mark, 3},
			want: [3]int{1, ttl + mark + ptr, 3}, where: d1,
		},
		{
			name: "SETF of a flag already there changes nothing (6.101 note 2)",
			run:  func(s *VM) error { s.SETF(d1, ttl); return nil },
			d1:   [3]int{1, ttl + mark, 3},
			want: [3]int{1, ttl + mark, 3}, where: d1,
		},
		{
			name: "RESETF removes one flag only",
			run:  func(s *VM) error { s.RESETF(d1, mark); return nil },
			d1:   [3]int{1, ttl + mark + ptr, 3},
			want: [3]int{1, ttl + ptr, 3}, where: d1,
		},
		{
			name: "RESETF of a flag that is not there changes nothing (6.91 note 2)",
			run:  func(s *VM) error { s.RESETF(d1, ptr); return nil },
			d1:   [3]int{1, ttl, 3},
			want: [3]int{1, ttl, 3}, where: d1,
		},
		{
			name: "SETFI sets through the address field",
			run:  func(s *VM) error { return s.SETFI(d1, ptr) },
			d1:   [3]int{blk, 0, 0}, at: [3]int{1, ttl, 3},
			want: [3]int{1, ttl + ptr, 3}, where: blk,
		},
		{
			name: "RSETFI resets through the address field",
			run:  func(s *VM) error { return s.RSETFI(d1, ttl) },
			d1:   [3]int{blk, 0, 0}, at: [3]int{1, ttl + ptr, 3},
			want: [3]int{1, ptr, 3}, where: blk,
		},
		{
			name: "INCRV adds to the value field only",
			run:  func(s *VM) error { s.INCRV(d1, 4); return nil },
			d1:   [3]int{1, 2, 3},
			want: [3]int{1, 2, 7}, where: d1,
		},
		{
			name: "MOVV moves the value field only",
			run:  func(s *VM) error { s.MOVV(d1, d2); return nil },
			d1:   [3]int{1, 2, 3}, d2: [3]int{40, 50, 60},
			want: [3]int{1, 2, 60}, where: d1,
		},
		{
			name: "SETVA takes the value from an address field",
			run:  func(s *VM) error { s.SETVA(d1, d2); return nil },
			d1:   [3]int{1, 2, 3}, d2: [3]int{40, 50, 60},
			want: [3]int{1, 2, 40}, where: d1,
		},
		{
			name: "SETVC sets the value field to a constant",
			run:  func(s *VM) error { s.SETVC(d1, 9); return nil },
			d1:   [3]int{1, 2, 3},
			want: [3]int{1, 2, 9}, where: d1,
		},
		{
			name: "PUTVC puts a value field through an offset",
			run:  func(s *VM) error { return s.PUTVC(d1, 1, d2) },
			d1:   [3]int{blk - 1, 0, 0}, d2: [3]int{0, 0, 9},
			at:   [3]int{1, 2, 3},
			want: [3]int{1, 2, 9}, where: blk,
		},
		{
			name: "SETSIZ puts an address field into a title's value field",
			run:  func(s *VM) error { return s.SETSIZ(d1, d2) },
			d1:   [3]int{blk, 0, 0}, d2: [3]int{9, 0, 0},
			at:   [3]int{1, 2, 3},
			want: [3]int{1, 2, 9}, where: blk,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, tt.d1[0], tt.d1[1], tt.d1[2])
			s.set(d2, tt.d2[0], tt.d2[1], tt.d2[2])
			s.set(blk, tt.at[0], tt.at[1], tt.at[2])
			if err := tt.run(s); err != nil {
				t.Fatal(err)
			}
			if got := s.descr(tt.where); got != tt.want {
				t.Errorf("%d is %v, want %v", tt.where, got, tt.want)
			}
		})
	}

	// The indirect ones reach an address the program computed.
	for name, run := range map[string]func(*VM) error{
		"SETFI":  func(s *VM) error { return s.SETFI(d1, 1) },
		"RSETFI": func(s *VM) error { return s.RSETFI(d1, 1) },
		"PUTVC":  func(s *VM) error { return s.PUTVC(d1, 0, d2) },
		"SETSIZ": func(s *VM) error { return s.SETSIZ(d1, d2) },
	} {
		s := machine()
		s.set(d1, core+9, 0, 0)
		if err := run(s); err == nil {
			t.Errorf("%s: no error reaching outside core", name)
		}
	}
}
