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
	"strconv"
	"strings"
)

// The real-number group of S4D58 7.5, and RCOMP, which is the last of
// the comparisons and belongs here because it is the only one that
// reads a real.
//
// # Where a real number lives
//
// In the address field, as every figure in this group draws it: 6.7
// prints DESCR2 as "R2 F2 V2" with the real in the first position.
// 3.1.1 says so in words -- "the address field must also be large
// enough to contain any integer or real number (including sign)" --
// and 5.3 confirms that the R in a figure is a real number while the R
// in a value field is the data type code the source program defines
// at line 298.
//
// The address field is a Go int, so a real is held there as its
// IEEE 754 bit pattern and read back with math.Float64frombits. That
// is what makes a real travel with its descriptor: MOVD, GETD, PUSH
// and the garbage collector move descriptors without knowing what is
// in them, exactly as they do on a machine where the address field is
// a word that sometimes holds an address, sometimes an integer and
// sometimes a floating-point number.
//
// The one thing this requires is a 64-bit int, which the constant
// below checks: on a machine with a 32-bit int it is a division by
// zero and the package does not compile.
const _ = 1 / (^uint(0) >> 63)

// realIn reads the real number in the address field of a descriptor.
func realIn(c Cell) float64 { return math.Float64frombits(uint64(c.A)) }

// realBits is the inverse: the address field a real number occupies.
func realBits(r float64) int { return int(math.Float64bits(r)) }

// realArith writes a result and takes SLOC, or takes FLOC and writes
// nothing. Every operation in this group has that shape, and the
// result "out of the range available for real numbers" is an infinity
// or a not-a-number, which is what the hardware produces where the
// System/360 would have raised an exponent overflow.
func (s *VM) realArith(descr1, descr2 int, result float64, ok bool, floc, sloc int) {
	if !ok || math.IsInf(result, 0) || math.IsNaN(result) {
		s.PC = floc
		return
	}
	src := s.Core[descr2]
	dst := &s.Core[descr1]
	dst.A, dst.F, dst.V = realBits(result), src.F, src.V
	s.PC = sloc
}

// typeCode reads one of the two data type codes 5.3 says "are defined
// in the SIL source program": I for integers, at line 293, and R for
// reals, at line 298.
func (s *VM) typeCode(of, name string) (int, error) {
	v, ok := s.Symbols[name]
	if !ok {
		return 0, s.fault("%s: the data type code %s is not defined", of, name)
	}
	return v, nil
}

// ADREAL (add real numbers) is used to add two real numbers. If the
// result is out of the range available for real numbers, transfer is
// to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 R2,F2,V2
//	DESCR3 R3
//
// Data Altered:
//
//	DESCR1 R2+R3,F2,V2
//
// Programming Notes:
//  1. See also DVREAL, EXREAL, MNREAL, MPREAL, and SBREAL.
//
// S4D58.PDF: 6.7
func (s *VM) ADREAL(descr1, descr2, descr3, floc, sloc int) {
	r2, r3 := realIn(s.Core[descr2]), realIn(s.Core[descr3])
	s.realArith(descr1, descr2, r2+r3, true, floc, sloc)
}

// SBREAL (subtract real numbers) is used to subtract one real number
// from another. If the result is out of the range available for real
// numbers, transfer is to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 R2,F2,V2
//	DESCR3 R3
//
// Data Altered:
//
//	DESCR1 R2-R3,F2,V2
//
// Programming Notes:
//  1. See also ADREAL, DVREAL, EXREAL, MNREAL, and MPREAL.
//
// S4D58.PDF: 6.97
func (s *VM) SBREAL(descr1, descr2, descr3, floc, sloc int) {
	r2, r3 := realIn(s.Core[descr2]), realIn(s.Core[descr3])
	s.realArith(descr1, descr2, r2-r3, true, floc, sloc)
}

// MPREAL (multiply real numbers) is used to multiply two real
// numbers. If the result is out of the range available for real
// numbers, transfer is to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 R2,F2,V2
//	DESCR3 R3
//
// Data Altered:
//
//	DESCR1 R2*R3,F2,V2
//
// Programming Notes:
//  1. See also ADREAL, DVREAL, EXREAL, MNREAL, and SBREAL.
//
// S4D58.PDF: 6.70
func (s *VM) MPREAL(descr1, descr2, descr3, floc, sloc int) {
	r2, r3 := realIn(s.Core[descr2]), realIn(s.Core[descr3])
	s.realArith(descr1, descr2, r2*r3, true, floc, sloc)
}

// DVREAL (divide real numbers) is used to divide one real number by
// another. If R3 = 0 or the result is out of the range available for
// real numbers, transfer is to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 R2,F2,V2
//	DESCR3 R3
//
// Data Altered:
//
//	DESCR1 R2/R3,F2,V2
//
// Programming Notes:
//  1. In addition to use in source-language arithmetic, DVREAL is used
//     in the computation of statistics published at the end of a
//     SNOBOL4 run.
//  2. See also ADREAL, EXREAL, MNREAL, MPREAL, and SBREAL.
//
// S4D58.PDF: 6.27
func (s *VM) DVREAL(descr1, descr2, descr3, floc, sloc int) {
	r2, r3 := realIn(s.Core[descr2]), realIn(s.Core[descr3])
	s.realArith(descr1, descr2, r2/r3, r3 != 0, floc, sloc)
}

// EXREAL (exponentiate real numbers) is used to raise a real number to
// a real power. If the result is not a real number or is out of the
// range available for real numbers, transfer is to FLOC. Otherwise
// transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 R1,F,V
//	DESCR3 R2
//
// Data Altered:
//
//	DESCR1 R1**R2,F,V
//
// "Not a real number" is the case the other five do not have: a
// negative base raised to a fractional power has no real value, and
// math.Pow gives a not-a-number for it, which realArith sends to FLOC.
//
// S4D58.PDF: 6.33
func (s *VM) EXREAL(descr1, descr2, descr3, floc, sloc int) {
	r1, r2 := realIn(s.Core[descr2]), realIn(s.Core[descr3])
	s.realArith(descr1, descr2, math.Pow(r1, r2), true, floc, sloc)
}

// MNREAL (minus real number) is used to change the sign of a real
// number.
//
// Data Input:
//
//	DESCR2 R,F,V
//
// Data Altered:
//
//	DESCR1 -R,F,V
//
// MNREAL has no FLOC: negating a real cannot leave the range, which is
// why 6.63's box has two operands where the other five have four.
//
// S4D58.PDF: 6.63
func (s *VM) MNREAL(descr1, descr2 int) {
	src := s.Core[descr2]
	dst := &s.Core[descr1]
	dst.A, dst.F, dst.V = realBits(-realIn(src)), src.F, src.V
}

// RCOMP (real comparison) is used to compare two real numbers. If
// R1 > R2, transfer is to GTLOC. If R1 = R2, transfer is to EQLOC. If
// R1 < R2, transfer is to LTLOC.
//
// Data Input:
//
//	DESCR1 R1
//	DESCR2 R2
//
// Programming Notes:
//  1. See also ACOMP and LCOMP.
//
// 6.88's second sentence reads "If R1 = R2, transfer is to GTLOC",
// which is a slip: the operand list has an EQLOC, and no other
// three-way comparison in 6 sends two of its three arms to the same
// place. Compare 6.1, whose wording is the same shape and correct.
//
// S4D58.PDF: 6.88
func (s *VM) RCOMP(descr1, descr2, gtloc, eqloc, ltloc int) {
	r1, r2 := realIn(s.Core[descr1]), realIn(s.Core[descr2])
	cmp := 0
	switch {
	case r1 > r2:
		cmp = 1
	case r1 < r2:
		cmp = -1
	}
	s.order(cmp, gtloc, eqloc, ltloc)
}

// INTRL (convert integer to real number) is used to convert a (signed)
// integer to a real number. R(I) is the real number corresponding to
// I.
//
// Data Input:
//
//	DESCR2 I
//
// Data Altered:
//
//	DESCR1 R(I),0,R
//
// Programming Notes:
//  1. R is a symbol defined in the source program and is the code for
//     the real data type.
//
// S4D58.PDF: 6.48
func (s *VM) INTRL(descr1, descr2 int) error {
	real, err := s.typeCode("INTRL", "R")
	if err != nil {
		return err
	}
	i := s.Core[descr2].A
	s.Core[descr1] = Cell{Kind: Data, A: realBits(float64(i)), V: real}
	return nil
}

// RLINT (convert real number to integer) is used to convert a real
// number to an integer. If the magnitude of R exceeds the magnitude of
// the largest integer, transfer is to FLOC. Otherwise transfer is to
// SLOC.
//
// Data Input:
//
//	DESCR2 R
//
// Data Altered:
//
//	DESCR1 I(R),0,I
//
// Programming Notes:
//  1. I(R) is the integer equivalent of the real number R.
//  2. The fractional part of R is discarded.
//  3. I is a symbol defined in the source program and is the code for
//     the integer data type.
//
// S4D58.PDF: 6.93
func (s *VM) RLINT(descr1, descr2, floc, sloc int) error {
	integer, err := s.typeCode("RLINT", "I")
	if err != nil {
		return err
	}
	n, ok := s.intOf(realIn(s.Core[descr2]))
	if !ok {
		s.PC = floc
		return nil
	}
	s.Core[descr1] = Cell{Kind: Data, A: n, V: integer}
	s.PC = sloc
	return nil
}

// intOf is I(R): the integer equivalent of a real number with its
// fractional part discarded (6.93 notes 1 and 2), and whether its
// magnitude is inside the range available for integers, which is
// SIZLIM.
func (s *VM) intOf(r float64) (int, bool) {
	t := math.Trunc(r)
	if math.IsNaN(t) || math.Abs(t) >= math.MaxInt64 {
		return 0, false
	}
	n := int(t)
	return n, !s.outOfRange(n)
}

// REALST (convert real number to string) is used to convert a real
// number into a specified string.
//
// Data Input:
//
//	DESCR R
//
// Data Altered:
//
//	SPEC   BUFFER,0,0,0,L
//	BUFFER C1...CL
//
// Programming Notes:
//  1. C1...CL should represent the real number R in the SNOBOL4
//     fashion, containing a decimal point and having at least one
//     digit before the decimal point, zeroes being added as necessary.
//     If R is negative, the string should begin with a minus sign. For
//     compatibility with real literals and data type conversions, the
//     real number should not be represented in exponent form, although
//     very large or small real numbers may require a large number of
//     characters for their representation otherwise.
//  2. The number of digits (and hence the size of BUFFER) required is
//     machine dependent and depends on the range available for real
//     numbers.
//  3. BUFFER is local to REALST and its contents may be overwritten by
//     a subsequent use of REALST.
//  4. See also INTSPC and SPREAL.
//
// Note 2 is why the buffer is sized from the text rather than fixed,
// and note 3 is why it is the machine's and separate from the one
// INTSPC uses. See VM.buffer.
//
// S4D58.PDF: 6.89
func (s *VM) REALST(spec, descr int) {
	text := []byte(realText(realIn(s.Core[descr])))
	at := s.buffer(&s.realBuf, len(text))
	s.putChars(at, text)
	s.putSpecifier(spec, at, 0, 0, 0, len(text))
}

// realText writes a real number the way note 1 asks: a decimal point,
// a digit before it, a minus sign for a negative, and never an
// exponent. The 'f' format never uses an exponent and a precision of
// -1 gives the shortest text that reads back as the same number; the
// only thing it leaves out is the point itself, when the number is a
// whole one.
func realText(r float64) string {
	t := strconv.FormatFloat(r, 'f', -1, 64)
	if !strings.ContainsRune(t, '.') {
		t += ".0"
	}
	return t
}

// SPREAL (convert specified string to real number) is used to convert
// a specified string into a real number. R(S) is a signed real number
// resulting from the conversion of the string S = C1...CL. If
// C1...CL does not represent a real number, or if the real number it
// represents is out of the range available for real numbers, transfer
// is to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	SPEC A,O,L
//	A+O  C1...CL
//
// Data Altered:
//
//	DESCR R(S),0,R
//
// Programming Notes:
//  1. R is a symbol defined in the source program and is the code for
//     the real data type.
//  2. C1,...,CL may begin with a sign (plus or minus) and may contain
//     an indefinite number of leading zeros. C1,...,CL will contain a
//     decimal point if it represents a real number, and have at least
//     one digit before the decimal point.
//  3. If L = 0, R(S) should be the real number 0.0.
//  4. See also SPCINT and INTRL.
//
// # The decimal point is not required
//
// Note 2 says what the caller supplies, not what the operation must
// reject, and note 3 -- an empty string is 0.0 -- shows that this is
// not a strict literal parser. The source settles it the other way as
// well: CNVVI at line 4716 tries SPCINT first and only reaches SPREAL
// when the string is not an integer, and CONVR at line 4703 does the
// same, so a string of digits alone never gets here in the first
// place. Accepting one costs nothing and rejecting it would be a
// guess.
//
// What is rejected is anything outside a sign, digits and one point:
// the exponent, infinity and not-a-number forms Go's own parser
// accepts are not SNOBOL4 reals.
//
// S4D58.PDF: 6.112
func (s *VM) SPREAL(descr, spec, floc, sloc int) error {
	real, err := s.typeCode("SPREAL", "R")
	if err != nil {
		return err
	}
	addr, _, _, offset, length := s.Specifier(spec)
	if !s.charsInCore(addr+offset, length) {
		return s.fault("SPREAL: %d characters at %d are outside core", length, addr+offset)
	}

	r, ok := realOf(s.Chars(addr+offset, length))
	if !ok {
		s.PC = floc
		return nil
	}
	s.Core[descr] = Cell{Kind: Data, A: realBits(r), V: real}
	s.PC = sloc
	return nil
}

// realOf is R(S), and whether the string represents a real number at
// all.
func realOf(text []byte) (float64, bool) {
	if len(text) == 0 {
		return 0, true // note 3
	}
	i := 0
	if text[0] == '+' || text[0] == '-' {
		i++
	}
	digits, points := 0, 0
	for ; i < len(text); i++ {
		switch c := text[i]; {
		case c >= '0' && c <= '9':
			digits++
		case c == '.':
			points++
		default:
			return 0, false
		}
	}
	if digits == 0 || points > 1 {
		return 0, false
	}
	// ParseFloat reports a range error rather than an infinity, which
	// is the "out of the range available for real numbers" arm.
	r, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return 0, false
	}
	return r, true
}
