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
	"fmt"
	"testing"
)

// The integer arithmetic of S4D58 7.5's tenth group. Every one of them
// writes the result into the address field of DESCR1, keeps the flag
// and value fields of DESCR2, and takes SLOC; a result out of range
// takes FLOC and writes nothing at all.
func TestIntegerArithmetic(t *testing.T) {
	const (
		floc = 40
		sloc = 50
	)
	for _, tt := range []struct {
		name string
		run  func(*VM)
		a2   int // the address field of DESCR2
		a3   int // the address field of DESCR3
		want int // the address field DESCR1 should end with
		fail bool
	}{
		{
			name: "SUM", run: func(s *VM) { s.SUM(d1, d2, d3, floc, sloc) },
			a2: 40, a3: 2, want: 42,
		},
		{
			name: "SUM out of range", run: func(s *VM) { s.SUM(d1, d2, d3, floc, sloc) },
			a2: 1 << 24, a3: 1, fail: true,
		},
		{
			name: "SUBTRT", run: func(s *VM) { s.SUBTRT(d1, d2, d3, floc, sloc) },
			a2: 40, a3: 2, want: 38,
		},
		{
			name: "SUBTRT below the range", run: func(s *VM) { s.SUBTRT(d1, d2, d3, floc, sloc) },
			a2: -(1 << 24), a3: 1, fail: true,
		},
		{
			name: "MULT", run: func(s *VM) { s.MULT(d1, d2, d3, floc, sloc) },
			a2: 6, a3: 7, want: 42,
		},
		{
			name: "MULT by zero", run: func(s *VM) { s.MULT(d1, d2, d3, floc, sloc) },
			a2: 6, a3: 0, want: 0,
		},
		{
			name: "MULT out of range", run: func(s *VM) { s.MULT(d1, d2, d3, floc, sloc) },
			a2: 1 << 20, a3: 1 << 20, fail: true,
		},
		{
			name: "DIVIDE", run: func(s *VM) { s.DIVIDE(d1, d2, d3, floc, sloc) },
			a2: 42, a3: 6, want: 7,
		},
		{
			// 6.26: "the result is truncated, not rounded".
			name: "DIVIDE truncates toward zero", run: func(s *VM) { s.DIVIDE(d1, d2, d3, floc, sloc) },
			a2: 7, a3: 2, want: 3,
		},
		{
			name: "DIVIDE truncates a negative toward zero", run: func(s *VM) { s.DIVIDE(d1, d2, d3, floc, sloc) },
			a2: -7, a3: 2, want: -3,
		},
		{
			name: "DIVIDE by zero", run: func(s *VM) { s.DIVIDE(d1, d2, d3, floc, sloc) },
			a2: 42, a3: 0, fail: true,
		},
		{
			name: "EXPINT", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 2, a3: 10, want: 1024,
		},
		{
			name: "EXPINT to the zeroth power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 7, a3: 0, want: 1,
		},
		{
			name: "EXPINT of a negative base", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: -2, a3: 3, want: -8,
		},
		{
			// 6.32: FLOC "if I1 = 0 and I2 is not positive".
			name: "EXPINT of zero to the zeroth", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 0, a3: 0, fail: true,
		},
		{
			name: "EXPINT of zero to a negative power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 0, a3: -1, fail: true,
		},
		{
			name: "EXPINT of zero to a positive power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 0, a3: 3, want: 0,
		},
		{
			// A proper fraction is not an integer, so it is out of the
			// range available for them.
			name: "EXPINT to a negative power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 2, a3: -1, fail: true,
		},
		{
			name: "EXPINT of one to a negative power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 1, a3: -5, want: 1,
		},
		{
			name: "EXPINT of minus one to an odd negative power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: -1, a3: -5, want: -1,
		},
		{
			name: "EXPINT of minus one to an even negative power", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: -1, a3: -4, want: 1,
		},
		{
			name: "EXPINT out of range", run: func(s *VM) { s.EXPINT(d1, d2, d3, floc, sloc) },
			a2: 3, a3: 40, fail: true,
		},
		{
			name: "MNSINT", run: func(s *VM) { s.MNSINT(d1, d2, floc, sloc) },
			a2: 42, want: -42,
		},
		{
			// 6.64 note 1: I may be negative.
			name: "MNSINT of a negative", run: func(s *VM) { s.MNSINT(d1, d2, floc, sloc) },
			a2: -42, want: 42,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, 99, 98, 97)
			s.set(d2, tt.a2, 7, 9)
			s.set(d3, tt.a3, 0, 0)

			tt.run(s)

			if tt.fail {
				if s.PC != floc {
					t.Errorf("PC is %d, want FLOC %d", s.PC, floc)
				}
				if got, want := s.descr(d1), [3]int{99, 98, 97}; got != want {
					t.Errorf("DESCR1 is %v, want %v: nothing is written on failure", got, want)
				}
				return
			}
			if s.PC != sloc {
				t.Errorf("PC is %d, want SLOC %d", s.PC, sloc)
			}
			if got, want := s.descr(d1), [3]int{tt.want, 7, 9}; got != want {
				t.Errorf("DESCR1 is %v, want %v: the flag and value fields come from DESCR2", got, want)
			}
		})
	}
}

// MULTC is the one operation in the group that does not branch: 6.73
// note 1 says the product is always in range. It clears the flag and
// value fields, unlike the rest.
func TestMULTC(t *testing.T) {
	for _, tt := range []struct{ i, n, want int }{
		{6, 7, 42},
		{0, 7, 0},
		{-6, 7, -42},
		{6, 1, 6},
	} {
		t.Run(fmt.Sprintf("%d by %d", tt.i, tt.n), func(t *testing.T) {
			s := machine()
			s.set(d1, 99, 98, 97)
			s.set(d2, tt.i, 7, 9)
			s.PC = 99
			s.MULTC(d1, d2, tt.n)
			if got, want := s.descr(d1), [3]int{tt.want, 0, 0}; got != want {
				t.Errorf("DESCR1 is %v, want %v", got, want)
			}
			if s.PC != 99 {
				t.Errorf("MULTC moved the counter to %d", s.PC)
			}
		})
	}
}

// 6.72 note 2, 6.73 note 2, 6.119 note 3: DESCR1 and DESCR2 are often
// the same descriptor, so an operation must read before it writes.
func TestArithmeticInPlace(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*VM)
		want int
	}{
		{"SUM", func(s *VM) { s.SUM(d1, d1, d3, 40, 50) }, 12},
		{"SUBTRT", func(s *VM) { s.SUBTRT(d1, d1, d3, 40, 50) }, 8},
		{"MULT", func(s *VM) { s.MULT(d1, d1, d3, 40, 50) }, 20},
		{"MULTC", func(s *VM) { s.MULTC(d1, d1, 2) }, 20},
		{"DIVIDE", func(s *VM) { s.DIVIDE(d1, d1, d3, 40, 50) }, 5},
		{"MNSINT", func(s *VM) { s.MNSINT(d1, d1, 40, 50) }, -10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := machine()
			s.set(d1, 10, 0, 0)
			s.set(d3, 2, 0, 0)
			tt.run(s)
			if got := s.Core[d1].A; got != tt.want {
				t.Errorf("DESCR1 addresses %d, want %d", got, tt.want)
			}
		})
	}
}
