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

// Integer arithmetic on address fields, S4D58 7.5's tenth group, less
// INCRA and DECRA, which are with the other address-field operations,
// and SUM, which is in macros.go with the vertical slice.
//
// Every one of them computes into the address field of DESCR1, keeps
// the flag and value fields of DESCR2, and branches: FLOC when the
// result is out of the range available for integers, SLOC otherwise.
// The range is SIZLIM, which PARMS chooses (6.20).

// arith writes a result and takes SLOC, or takes FLOC and writes
// nothing. It is the shape every operation in this group has, and
// keeping it in one place is what stops one of them silently leaving
// a half-computed descriptor behind on overflow.
func (s *VM) arith(descr1, descr2, result int, ok bool, floc, sloc int) {
	if !ok || s.outOfRange(result) {
		s.PC = floc
		return
	}
	src := s.Core[descr2]
	dst := &s.Core[descr1]
	dst.A, dst.F, dst.V = result, src.F, src.V
	s.PC = sloc
}

// outOfRange reports whether an integer is outside what the address
// field can hold. SIZLIM is "the value of the largest integer that can
// be stored in the value field of a descriptor" (6.20), and it is what
// the SNOBOL4 source uses for the limit on integers throughout.
func (s *VM) outOfRange(n int) bool {
	limit, ok := s.Symbols["SIZLIM"]
	return ok && (n > limit || n < -limit)
}

// SUBTRT (subtract addresses) is used to subtract one address field
// from another. A2 and A3 are considered as signed integers. If A2-A3
// is out of the range available for integers, transfer is to FLOC.
// Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 A2,F2,V2
//	DESCR3 A3
//
// Data Altered:
//
//	DESCR1 A2-A3,F2,V2
//
// Programming Notes:
//  1. A2 and A3 may be relocatable addresses.
//  2. The test for success and failure is used in only one call of
//     this macro.
//  3. DESCR1 and DESCR2 are often the same.
//  4. See also SUM.
//
// S4D58.PDF: 6.119
func (s *VM) SUBTRT(descr1, descr2, descr3, floc, sloc int) {
	s.arith(descr1, descr2, s.Core[descr2].A-s.Core[descr3].A, true, floc, sloc)
}

// MULT (multiply integers) is used to multiply two integers. In the
// event of overflow, transfer is to FLOC. Otherwise, transfer is to
// SLOC.
//
// Data Input:
//
//	DESCR2 I2,F2,V2
//	DESCR3 I3
//
// Data Altered:
//
//	DESCR1 I2*I3,F2,V2
//
// Programming Notes:
//  1. The test for success and failure is used in only two calls of
//     this macro.
//  2. DESCR1 and DESCR2 are often the same.
//  3. See also MULTC and DIVIDE.
//
// S4D58.PDF: 6.72
func (s *VM) MULT(descr1, descr2, descr3, floc, sloc int) {
	i2, i3 := s.Core[descr2].A, s.Core[descr3].A
	product := i2 * i3
	// A Go int is wide enough that SIZLIM catches every overflow the
	// SNOBOL4 system can produce, but the product itself must not wrap
	// on the way to the check.
	ok := i2 == 0 || (product/i2 == i3)
	s.arith(descr1, descr2, product, ok, floc, sloc)
}

// MULTC (multiply address by constant) is used to multiply an integer
// by a constant.
//
// Data Input:
//
//	DESCR2 I
//
// Data Altered:
//
//	DESCR1 I*N,0,0
//
// Programming Notes:
//  1. I*N never exceeds the range available for integers.
//  2. DESCR1 and DESCR2 are often the same.
//  3. N is often D, which typically may be implemented by a shift, or
//     simply by no operation if D is 1 for a particular machine.
//  4. See also MULT.
//
// Unlike the rest of the group MULTC clears the flag and value fields
// and does not branch: note 1 says the product is always in range, so
// there is nothing to branch on.
//
// S4D58.PDF: 6.73
func (s *VM) MULTC(descr1, descr2, n int) {
	s.Core[descr1] = Cell{Kind: Data, A: s.Core[descr2].A * n}
}

// DIVIDE (divide integers) is used to divide one integer by another.
// Any remainder is discarded. That is, the result is truncated, not
// rounded. If I = 0, transfer is to FLOC. Otherwise transfer is to
// SLOC.
//
// Data Input:
//
//	DESCR2 A,F,V
//	DESCR3 I
//
// Data Altered:
//
//	DESCR1 A/I,F,V
//
// Programming Notes:
//  1. A may be a relocatable address.
//
// Go truncates toward zero, which is what "truncated, not rounded"
// asks for.
//
// S4D58.PDF: 6.26
func (s *VM) DIVIDE(descr1, descr2, descr3, floc, sloc int) {
	i := s.Core[descr3].A
	if i == 0 {
		s.PC = floc
		return
	}
	s.arith(descr1, descr2, s.Core[descr2].A/i, true, floc, sloc)
}

// EXPINT (exponentiate integers) is used to raise an integer to an
// integer power. If I1 = 0 and I2 is not positive, or if the result is
// out of the range available for integers, transfer is to FLOC.
// Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 I1,F,V
//	DESCR3 I2
//
// Data Altered:
//
//	DESCR1 I1**I2,F,V
//
// A nonzero base raised to a negative power is not a special case the
// document spells out, but its own words settle it: FLOC is taken "if
// the result is out of the range available for integers", and a proper
// fraction is not an integer at all. The exceptions are the two bases
// whose negative powers are still integers, 1 and -1.
//
// S4D58.PDF: 6.32
func (s *VM) EXPINT(descr1, descr2, descr3, floc, sloc int) {
	base, power := s.Core[descr2].A, s.Core[descr3].A
	if base == 0 && power <= 0 {
		s.PC = floc
		return
	}
	if power < 0 {
		switch {
		case base == 1:
			s.arith(descr1, descr2, 1, true, floc, sloc)
		case base == -1 && power%2 != 0:
			s.arith(descr1, descr2, -1, true, floc, sloc)
		case base == -1:
			s.arith(descr1, descr2, 1, true, floc, sloc)
		default:
			s.PC = floc
		}
		return
	}
	result, ok := 1, true
	for i := 0; i < power && ok; i++ {
		next := result * base
		ok = base == 0 || next/base == result
		result = next
		if s.outOfRange(result) {
			ok = false
		}
	}
	s.arith(descr1, descr2, result, ok, floc, sloc)
}

// MNSINT (minus integer) is used to change the sign of an integer. If
// -I exceeds the maximum integer, transfer is to FLOC. Otherwise
// transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 I,F,V
//
// Data Altered:
//
//	DESCR1 -I,F,V
//
// Programming Notes:
//  1. I may be negative.
//  2. See also MNREAL.
//
// S4D58.PDF: 6.64
func (s *VM) MNSINT(descr1, descr2, floc, sloc int) {
	s.arith(descr1, descr2, -s.Core[descr2].A, true, floc, sloc)
}
