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
)

// A place in core well away from the scratch descriptors, used as the
// block these operations reach through.
const blk = 24

// descr reads the three descriptor fields, for comparing against a
// figure.
func (s *VM) descr(a int) [3]int { return [3]int{s.Core[a].A, s.Core[a].F, s.Core[a].V} }

// GETD, GETDC, PUTD, PUTDC and MOVDIC are the same figure with the
// offset written four different ways, so they are tested together:
// each moves a whole descriptor to or from an address computed from
// another descriptor.
func TestGetAndPutDescriptors(t *testing.T) {
	want := [3]int{7, 8, 9}

	t.Run("GETD", func(t *testing.T) {
		s := machine()
		s.set(d2, blk, 0, 0)
		s.set(d3, 2, 0, 0)
		s.set(blk+2, 7, 8, 9)
		if err := s.GETD(d1, d2, d3); err != nil {
			t.Fatal(err)
		}
		if got := s.descr(d1); got != want {
			t.Errorf("DESCR1 is %v, want %v", got, want)
		}
	})

	t.Run("GETDC", func(t *testing.T) {
		s := machine()
		s.set(d2, blk, 0, 0)
		s.set(blk+2, 7, 8, 9)
		if err := s.GETDC(d1, d2, 2); err != nil {
			t.Fatal(err)
		}
		if got := s.descr(d1); got != want {
			t.Errorf("DESCR1 is %v, want %v", got, want)
		}
	})

	t.Run("PUTD", func(t *testing.T) {
		s := machine()
		s.set(d1, blk, 0, 0)
		s.set(d2, 2, 0, 0)
		s.set(d3, 7, 8, 9)
		if err := s.PUTD(d1, d2, d3); err != nil {
			t.Fatal(err)
		}
		if got := s.descr(blk + 2); got != want {
			t.Errorf("A1+A2 is %v, want %v", got, want)
		}
	})

	t.Run("PUTDC", func(t *testing.T) {
		s := machine()
		s.set(d1, blk, 0, 0)
		s.set(d2, 7, 8, 9)
		if err := s.PUTDC(d1, 2, d2); err != nil {
			t.Fatal(err)
		}
		if got := s.descr(blk + 2); got != want {
			t.Errorf("A1+N is %v, want %v", got, want)
		}
	})

	t.Run("MOVDIC", func(t *testing.T) {
		s := machine()
		s.set(d1, blk, 0, 0)
		s.set(d2, blk+4, 0, 0)
		s.set(blk+5, 7, 8, 9)
		if err := s.MOVDIC(d1, 2, d2, 1); err != nil {
			t.Fatal(err)
		}
		if got := s.descr(blk + 2); got != want {
			t.Errorf("A1+N1 is %v, want %v", got, want)
		}
	})

	// Every one of them reaches an address the program computed, so
	// every one of them can be pointed outside core.
	t.Run("outside core", func(t *testing.T) {
		for name, run := range map[string]func(*VM) error{
			"GETD":   func(s *VM) error { return s.GETD(d1, d2, d3) },
			"GETDC":  func(s *VM) error { return s.GETDC(d1, d2, 0) },
			"PUTD":   func(s *VM) error { return s.PUTD(d1, d2, d3) },
			"PUTDC":  func(s *VM) error { return s.PUTDC(d1, 0, d2) },
			"MOVDIC": func(s *VM) error { return s.MOVDIC(d1, 0, d2, 0) },
			"GETAC":  func(s *VM) error { return s.GETAC(d1, d2, 0) },
			"PUTAC":  func(s *VM) error { return s.PUTAC(d1, 0, d2) },
			"GETSIZ": func(s *VM) error { return s.GETSIZ(d1, d2) },
			"ADJUST": func(s *VM) error { return s.ADJUST(d1, d2, d3) },
			"BKSIZE": func(s *VM) error { return s.BKSIZE(d1, d2) },
		} {
			s := machine()
			s.set(d1, core+9, 0, 0)
			s.set(d2, core+9, 0, 0)
			s.set(d3, 0, 0, 0)
			if err := run(s); err == nil {
				t.Errorf("%s: no error reaching outside core", name)
			}
		}
	})
}

// MOVBLK copies N descriptors starting one past the title, and 6.66
// note 1 says the title itself is not touched.
func TestMOVBLK(t *testing.T) {
	const from, to = 30, blk
	s := machine()
	s.set(d1, to, 0, 0)
	s.set(d2, from, 0, 0)
	s.set(d3, 3, 0, 0) // D*N with D = 1
	s.set(from, 99, 99, 99)
	s.set(to, 11, 11, 11)
	for i := 1; i <= 3; i++ {
		s.set(from+i, i, i*10, i*100)
	}

	if err := s.MOVBLK(d1, d2, d3); err != nil {
		t.Fatal(err)
	}
	if got, want := s.descr(to), [3]int{11, 11, 11}; got != want {
		t.Errorf("the title at A1 is %v, want %v: 6.66 note 1", got, want)
	}
	for i := 1; i <= 3; i++ {
		if got, want := s.descr(to+i), [3]int{i, i * 10, i * 100}; got != want {
			t.Errorf("A1+%d is %v, want %v", i, got, want)
		}
	}
}

// 6.66 note 2: the areas may overlap, and only when A1 is less than
// A2. Moving down by one is the case that a naive forward copy gets
// right and a naive backward copy gets wrong.
func TestMOVBLKOverlaps(t *testing.T) {
	for _, tt := range []struct {
		name     string
		from, to int
	}{
		{"moving down", 21, 20},
		{"moving up", 20, 21},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, tt.to, 0, 0)
			s.set(d2, tt.from, 0, 0)
			s.set(d3, 4, 0, 0)
			for i := 1; i <= 4; i++ {
				s.set(tt.from+i, i, 0, 0)
			}
			if err := s.MOVBLK(d1, d2, d3); err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= 4; i++ {
				if got := s.Core[tt.to+i].A; got != i {
					t.Errorf("A1+%d is %d, want %d", i, got, i)
				}
			}
		})
	}
}

// ZERBLK zeroes I+1 descriptors and stops (6.131).
func TestZERBLK(t *testing.T) {
	s := machine()
	s.set(d1, blk, 0, 0)
	s.set(d2, 2, 0, 0) // D*I with D = 1, so three descriptors
	for i := 0; i < 4; i++ {
		s.set(blk+i, 9, 9, 9)
	}
	if err := s.ZERBLK(d1, d2); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if got := s.descr(blk + i); got != [3]int{0, 0, 0} {
			t.Errorf("%d is %v, want zeroed", blk+i, got)
		}
	}
	if got := s.descr(blk + 3); got != [3]int{9, 9, 9} {
		t.Errorf("%d is %v, want untouched", blk+3, got)
	}
	s.set(d2, core, 0, 0)
	if err := s.ZERBLK(d1, d2); err == nil {
		t.Error("no error zeroing a block that runs off the end of core")
	}
}

// PUSH and POP are inverses, and PUSH overflows to OVER rather than to
// INTR10 (6.80 note 1).
func TestPUSH(t *testing.T) {
	s := machine()
	s.Symbols["STSIZE"] = 8
	s.CStack = stack
	s.set(d1, 1, 2, 3)
	s.set(d2, 4, 5, 6)

	if err := s.PUSH([]int{d1, d2}); err != nil {
		t.Fatal(err)
	}
	if s.CStack != stack+2 {
		t.Errorf("CSTACK is %d, want %d", s.CStack, stack+2)
	}
	if got, want := s.descr(stack+1), [3]int{1, 2, 3}; got != want {
		t.Errorf("A+D is %v, want %v", got, want)
	}
	if got, want := s.descr(stack+2), [3]int{4, 5, 6}; got != want {
		t.Errorf("A+2D is %v, want %v", got, want)
	}

	// What went on comes back off, in reverse.
	s.set(d1, 0, 0, 0)
	s.set(d2, 0, 0, 0)
	if err := s.POP([]int{d1, d2}); err != nil {
		t.Fatal(err)
	}
	if got, want := s.descr(d1), [3]int{4, 5, 6}; got != want {
		t.Errorf("the first POP got %v, want %v", got, want)
	}
	if got, want := s.descr(d2), [3]int{1, 2, 3}; got != want {
		t.Errorf("the second POP got %v, want %v", got, want)
	}
	if s.CStack != stack {
		t.Errorf("CSTACK is %d, want %d", s.CStack, stack)
	}
}

func TestPUSHOverflow(t *testing.T) {
	s := machine()
	s.Symbols["STSIZE"] = 2
	s.CStack = stack + 2

	if err := s.PUSH([]int{d1}); err == nil {
		t.Fatal("no error pushing past the top of the stack")
	} else if !strings.Contains(err.Error(), "OVER is not defined") {
		t.Errorf("reported %v", err)
	}

	s = machine()
	s.Symbols["STSIZE"] = 2
	s.Symbols[Overflow] = 44
	s.CStack = stack + 2
	if err := s.PUSH([]int{d1}); err != nil {
		t.Fatalf("with %s defined: %v", Overflow, err)
	}
	if s.PC != 44 {
		t.Errorf("PC is %d, want %s at 44", s.PC, Overflow)
	}
}

// The operations that read or write one field.
func TestAddressFields(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*VM) error
		// The state before, as three descriptors and one block cell.
		d1, d2, d3, at [3]int
		// What d1 (or, for PUTAC, the block cell) should be after.
		want [3]int
		// where says which cell want describes.
		where int
	}{
		{
			name: "ADJUST adds an address integer to the address it points at",
			run:  func(s *VM) error { return s.ADJUST(d1, d2, d3) },
			d1:   [3]int{0, 5, 6}, d2: [3]int{blk, 0, 0}, d3: [3]int{100, 0, 0},
			at:   [3]int{7, 0, 0},
			want: [3]int{107, 5, 6}, where: d1,
		},
		{
			name:  "DECRA subtracts from the address field only",
			run:   func(s *VM) error { s.DECRA(d1, 4); return nil },
			d1:    [3]int{10, 5, 6},
			want:  [3]int{6, 5, 6},
			where: d1,
		},
		{
			name:  "DECRA may make the address negative",
			run:   func(s *VM) error { s.DECRA(d1, 14); return nil },
			d1:    [3]int{10, 0, 0},
			want:  [3]int{-4, 0, 0},
			where: d1,
		},
		{
			name:  "INCRA adds to the address field only",
			run:   func(s *VM) error { s.INCRA(d1, 4); return nil },
			d1:    [3]int{10, 5, 6},
			want:  [3]int{14, 5, 6},
			where: d1,
		},
		{
			name: "GETAC takes an address field through an offset",
			run:  func(s *VM) error { return s.GETAC(d1, d2, 1) },
			d1:   [3]int{0, 5, 6}, d2: [3]int{blk - 1, 0, 0},
			at:   [3]int{7, 8, 9},
			want: [3]int{7, 5, 6}, where: d1,
		},
		{
			name:  "GETAC takes a negative offset",
			run:   func(s *VM) error { return s.GETAC(d1, d2, -1) },
			d2:    [3]int{blk + 1, 0, 0},
			at:    [3]int{7, 8, 9},
			want:  [3]int{7, 0, 0},
			where: d1,
		},
		{
			name: "PUTAC puts an address field through an offset",
			run:  func(s *VM) error { return s.PUTAC(d1, 1, d2) },
			d1:   [3]int{blk - 1, 0, 0}, d2: [3]int{7, 0, 0},
			at:   [3]int{0, 8, 9},
			want: [3]int{7, 8, 9}, where: blk,
		},
		{
			name: "GETSIZ takes the value field of a title",
			run:  func(s *VM) error { return s.GETSIZ(d1, d2) },
			d1:   [3]int{1, 2, 3}, d2: [3]int{blk, 0, 0},
			at:   [3]int{0, 0, 42},
			want: [3]int{42, 0, 0}, where: d1,
		},
		{
			name: "MOVA moves the address field only",
			run:  func(s *VM) error { s.MOVA(d1, d2); return nil },
			d1:   [3]int{1, 2, 3}, d2: [3]int{40, 50, 60},
			want:  [3]int{40, 2, 3},
			where: d1,
		},
		{
			name: "SETAV takes the address from a value field and clears the rest",
			run:  func(s *VM) error { s.SETAV(d1, d2); return nil },
			d1:   [3]int{1, 2, 3}, d2: [3]int{40, 50, 60},
			want:  [3]int{60, 0, 0},
			where: d1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, tt.d1[0], tt.d1[1], tt.d1[2])
			s.set(d2, tt.d2[0], tt.d2[1], tt.d2[2])
			s.set(d3, tt.d3[0], tt.d3[1], tt.d3[2])
			s.set(blk, tt.at[0], tt.at[1], tt.at[2])
			if err := tt.run(s); err != nil {
				t.Fatal(err)
			}
			if got := s.descr(tt.where); got != tt.want {
				t.Errorf("%d is %v, want %v", tt.where, got, tt.want)
			}
		})
	}
}

// GETLG reads the length out of the second half of a specifier.
func TestGETLG(t *testing.T) {
	s := machine()
	s.Core[spec1] = Cell{Kind: Data, A: str1}
	s.Core[spec1+1] = Cell{Kind: Data, A: 2, V: 5}
	s.set(d1, 1, 2, 3)
	s.GETLG(d1, spec1)
	if got, want := s.descr(d1), [3]int{5, 0, 0}; got != want {
		t.Errorf("DESCR is %v, want %v", got, want)
	}
}

// The two size formulas of 6.13 and 6.41, which differ by the one
// descriptor a string structure's title adds.
func TestBlockSizes(t *testing.T) {
	// With DESCR = 1 and CPA = 1 there is one character per
	// descriptor, so a string of L characters takes L of them.
	for _, tt := range []struct {
		chars  int
		bksize int // D*(4 + descriptors)
		getlth int // D*(3 + descriptors)
	}{
		{0, 4, 3},
		{1, 5, 4},
		{7, 11, 10},
	} {
		s := machine()
		s.set(d2, blk, 0, 0)
		s.set(blk, 0, s.Symbols["STTL"], tt.chars)
		if err := s.BKSIZE(d1, d2); err != nil {
			t.Fatal(err)
		}
		if got := s.Core[d1].A; got != tt.bksize {
			t.Errorf("BKSIZE of a %d-character string structure is %d, want %d", tt.chars, got, tt.bksize)
		}

		s = machine()
		s.set(d2, tt.chars, 0, 0)
		s.GETLTH(d1, d2)
		if got := s.Core[d1].A; got != tt.getlth {
			t.Errorf("GETLTH of %d characters is %d, want %d", tt.chars, got, tt.getlth)
		}
	}

	// Without STTL the block is its own size plus one descriptor for
	// the title.
	s := machine()
	s.set(d2, blk, 0, 0)
	s.set(blk, 0, 0, 40)
	if err := s.BKSIZE(d1, d2); err != nil {
		t.Fatal(err)
	}
	if got, want := s.Core[d1].A, 40+s.Descr; got != want {
		t.Errorf("BKSIZE of a block is %d, want V+D = %d", got, want)
	}

	// STTL comes from PARMS, and BKSIZE cannot tell a string structure
	// from a block without it.
	s = machine()
	delete(s.Symbols, "STTL")
	s.set(d2, blk, 0, 0)
	if err := s.BKSIZE(d1, d2); err == nil {
		t.Error("no error with STTL undefined")
	}
}
