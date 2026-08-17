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

import "bytes"

// The comparison operations, S4D58 7.5's fourth group, less ACOMP and
// ACOMPC, which are in macros.go, and RCOMP, which waits for real
// numbers.
//
// Every one of them is a three-way or two-way branch and nothing else:
// none alters any data. RCALL's frame, a descriptor field, a specifier
// length -- whatever is being compared, the only effect is on the
// program counter.

// order sets the counter from the sign of a comparison, which is the
// shape every three-way branch in this group has.
func (s *VM) order(cmp int, gtloc, eqloc, ltloc int) {
	switch {
	case cmp > 0:
		s.PC = gtloc
	case cmp == 0:
		s.PC = eqloc
	default:
		s.PC = ltloc
	}
}

// equal sets the counter from a two-way test.
func (s *VM) equal(eq bool, neloc, eqloc int) {
	if eq {
		s.PC = eqloc
		return
	}
	s.PC = neloc
}

// AEQL (addresses equal test) is used to compare the address fields of
// two descriptors. The comparison is arithmetic with A1 and A2 being
// considered as signed integers: If A1 = A2, transfer is to EQLOC.
// Otherwise transfer is to NELOC.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 A2
//
// Programming Notes:
//  1. A1 and A2 may be relocatable addresses.
//  2. See also VEQL, AEQLC, LEQLC, AEQLIC, ACOMP, and ACOMPC.
//
// S4D58.PDF: 6.8
func (s *VM) AEQL(descr1, descr2, neloc, eqloc int) {
	s.equal(s.Core[descr1].A == s.Core[descr2].A, neloc, eqloc)
}

// AEQLC (address equal to constant test) is used to compare the
// address field of a descriptor to a constant. The comparison is
// arithmetic with A being considered as a signed integer. If A = N,
// transfer is to EQLOC. Otherwise transfer is to NELOC.
//
// Data Input:
//
//	DESCR A
//
// Programming Notes:
//  1. A may be a relocatable address.
//  2. N is never negative.
//  3. N is often 0.
//  4. See also LEQLC, AEQL, AEQLIC, ACOMP, and ACOMPC.
//
// S4D58.PDF: 6.9
func (s *VM) AEQLC(descr, n, neloc, eqloc int) {
	s.equal(s.Core[descr].A == n, neloc, eqloc)
}

// AEQLIC (address equal to constant indirect test) is used to compare
// an indirectly specified address field of a descriptor to a constant.
// The comparison is arithmetic with A1 being considered as a signed
// integer. If A2 = N2, transfer is to EQLOC. Otherwise transfer is to
// NELOC.
//
// Data Input:
//
//	DESCR  A1
//	A1+N1  A2
//
// Programming Notes:
//  1. A2 may be a relocatable address.
//  2. N2 is never negative.
//  3. N1 is always zero.
//  4. See also AEQL, AEQLC, LEQLC, ACOMP, and ACOMPC.
//
// Note 3 is not relied on. It holds in the SNOBOL4 source -- all ten
// sites write a zero -- but nothing here needs it to.
//
// S4D58.PDF: 6.10
func (s *VM) AEQLIC(descr, n1, n2, neloc, eqloc int) error {
	at := s.Core[descr].A + n1
	if !s.inCore(at) {
		return s.fault("AEQLIC: %d is outside core", at)
	}
	s.equal(s.Core[at].A == n2, neloc, eqloc)
	return nil
}

// CHKVAL (check value) is used to compare an integer to the length of
// a specifier plus another integer. If L+I2 > I1, transfer is to
// GTLOC. If L+I2 = I1, transfer is to EQLOC. If L+I2 < I1, transfer
// is to LTLOC.
//
// Data Input:
//
//	SPEC   L
//	DESCR1 I1
//	DESCR2 I2
//
// Programming Notes:
//  1. I1, I2, and L are always positive integers.
//  2. CHKVAL is used only in pattern matching.
//
// S4D58.PDF: 6.18
func (s *VM) CHKVAL(descr1, descr2, spec, gtloc, eqloc, ltloc int) {
	_, _, _, _, length := s.Specifier(spec)
	s.order(length+s.Core[descr2].A-s.Core[descr1].A, gtloc, eqloc, ltloc)
}

// DEQL (descriptor equal test) is used to compare two descriptors. If
// A1 = A2, F1 = F2, and V1 = V2, transfer is to EQLOC. Otherwise
// transfer is to NELOC.
//
// Data Input:
//
//	DESCR1 A1,F1,V1
//	DESCR2 A2,F2,V2
//
// Programming Notes:
//  1. All fields of the two descriptors must be identical for transfer
//     to EQLOC.
//
// S4D58.PDF: 6.24
func (s *VM) DEQL(descr1, descr2, neloc, eqloc int) {
	a, b := s.Core[descr1], s.Core[descr2]
	s.equal(a.A == b.A && a.F == b.F && a.V == b.V, neloc, eqloc)
}

// LCOMP (length comparison) is used to compare the lengths of two
// specifiers. If L1 > L2, transfer is to GTLOC. If L1 = L2, transfer
// is to EQLOC. If L1 < L2, transfer is to LTLOC.
//
// Data Input:
//
//	SPEC1 L1
//	SPEC2 L2
//
// Programming Notes:
//  1. See also ACOMP, RCOMP, and LEQLC.
//
// S4D58.PDF: 6.51
func (s *VM) LCOMP(spec1, spec2, gtloc, eqloc, ltloc int) {
	_, _, _, _, l1 := s.Specifier(spec1)
	_, _, _, _, l2 := s.Specifier(spec2)
	s.order(l1-l2, gtloc, eqloc, ltloc)
}

// LEQLC (length equal to constant test) is used to compare the length
// of a specifier to a constant. If L = N, transfer is to EQLOC.
// Otherwise transfer is to NELOC.
//
// Data Input:
//
//	SPEC L
//
// Programming Notes:
//  1. L and N are never negative.
//  2. See also LCOMP, AEQLC, and AEQLIC.
//
// S4D58.PDF: 6.52
func (s *VM) LEQLC(spec, n, neloc, eqloc int) {
	_, _, _, _, length := s.Specifier(spec)
	s.equal(length == n, neloc, eqloc)
}

// LEXCMP (lexical comparison of strings) is used to compare two
// strings lexicographically (i.e. according to their alphabetical
// ordering).
//
// Data Input:
//
//	SPEC1 A1,O1,N
//	SPEC2 A2,O2,M
//	A1+O1 C11...C1N
//	A2+O2 C21...C2M
//
// Programming Notes:
//  1. The lexicographical ordering is machine dependent and is
//     determined by the numerical order of the internal representation
//     of the characters for a particular machine.
//  2. A string that is an initial substring of another string is
//     lexicographically less than that string. That is ABC is less
//     than ABCA.
//  3. The null (zero-length) string is lexicographically less than any
//     other string.
//  4. Two strings are equal if and only if they are of the same length
//     and are identical character by character.
//  5. By far the most frequent use of LEXCMP is to determine whether
//     two strings are the same or different. In these cases GTLOC and
//     LTLOC will specify the same location or both be omitted.
//
// # A deviation from the document
//
// 6.53's own sentence reads: "If C11...C1N1 < C21...C2M, transfer is
// to GTLOC. ... If C11...C1N1 > C21...C2M, transfer is to LTLOC."
// That is backwards, and this implementation does the opposite: SPEC1
// greater takes GTLOC.
//
// The evidence is LGT at line 4485 of the SNOBOL4 source, which is the
// one site out of twelve where GTLOC and LTLOC differ:
//
//	LGT    PROC    ,
//	       ...
//	       LEXCMP  XSP,YSP,RETNUL,FAIL,FAIL
//
// LGT(X,Y) succeeds when X is lexically greater than Y, and RETNUL is
// how a SNOBOL4 primitive succeeds. The line just above it makes the
// same point from the other side: AEQLC YPTR,0,,RETNUL succeeds when
// the second argument is null, because anything is greater than the
// null string (note 3). Note 5 explains why the error survived: the
// other eleven sites cannot tell the difference.
//
// S4D58.PDF: 6.53
func (s *VM) LEXCMP(spec1, spec2, gtloc, eqloc, ltloc int) {
	s.order(bytes.Compare(s.Text(spec1), s.Text(spec2)), gtloc, eqloc, ltloc)
}

// TESTF (test flag) is used to test a flag field for the presence of a
// flag. If F contains FLAG, transfer is to SLOC. Otherwise transfer is
// to FLOC.
//
// Data Input:
//
//	DESCR F
//
// Programming Notes:
//  1. See also TESTFI.
//
// S4D58.PDF: 6.121
func (s *VM) TESTF(descr, flag, floc, sloc int) {
	s.equal(s.Core[descr].F&flag == flag, floc, sloc)
}

// TESTFI (test flag indirect) is used to test an indirectly specified
// flag field for the presence of a flag. If F contains FLAG, transfer
// is to SLOC. Otherwise transfer is to FLOC.
//
// Data Input:
//
//	DESCR A
//	A     F
//
// Programming Notes:
//  1. See also TESTF.
//
// S4D58.PDF: 6.122
func (s *VM) TESTFI(descr, flag, floc, sloc int) error {
	at := s.Core[descr].A
	if !s.inCore(at) {
		return s.fault("TESTFI: %d is outside core", at)
	}
	s.equal(s.Core[at].F&flag == flag, floc, sloc)
	return nil
}

// VCMPIC (value field compare indirect with offset constant) is used
// to compare a value field, indirectly specified with an offset
// constant, with another value field. V1 and V2 are considered as
// unsigned integers. If V1 > V2, transfer is to GTLOC. If V1 = V2,
// transfer is to EQLOC. If V1 < V2, transfer is to LTLOC.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 V2
//	A1+N   V1
//
// S4D58.PDF: 6.128
func (s *VM) VCMPIC(descr1, n, descr2, gtloc, eqloc, ltloc int) error {
	at := s.Core[descr1].A + n
	if !s.inCore(at) {
		return s.fault("VCMPIC: %d is outside core", at)
	}
	s.order(s.Core[at].V-s.Core[descr2].V, gtloc, eqloc, ltloc)
	return nil
}

// VEQL (value fields equal test) is used to compare the value fields
// of two descriptors. V1 and V2 are considered as unsigned integers.
// If V1 = V2, transfer is to EQLOC. Otherwise transfer is to NELOC.
//
// Data Input:
//
//	DESCR1 V1
//	DESCR2 V2
//
// Programming Notes:
//  1. See also AEQL and VEQLC.
//
// S4D58.PDF: 6.129
func (s *VM) VEQL(descr1, descr2, neloc, eqloc int) {
	s.equal(s.Core[descr1].V == s.Core[descr2].V, neloc, eqloc)
}

// VEQLC (value field equal to constant test) is used to compare the
// value field of a descriptor to a constant. V is considered as an
// unsigned integer. If V = N, transfer is to EQLOC. Otherwise transfer
// is to NELOC.
//
// Data Input:
//
//	DESCR V
//
// Programming Notes:
//  1. N is never negative.
//  2. See also AEQLC and VEQL.
//
// S4D58.PDF: 6.130
func (s *VM) VEQLC(descr, n, neloc, eqloc int) {
	s.equal(s.Core[descr].V == n, neloc, eqloc)
}
