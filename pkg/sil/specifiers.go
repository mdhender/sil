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

import "strconv"

// The operations on specifiers, S4D58 7.5's first and second groups:
// the ones that move a whole specifier and the ones that alter the
// parts of one. STREAM is the third member of the second group and is
// not here; it needs the syntax tables, which are their own milestone.
//
// A specifier is two adjacent cells (3.2). The first is a descriptor
// -- address, flag and value -- and the second carries the offset in
// its address field and the length in its value field. VM.Specifier
// reads the pair; the operations here write it a field at a time when
// the document alters one field, and whole when it alters all five,
// because which fields are left alone is the whole content of several
// of these entries.

// offset and length are the two cells of a specifier that the
// document draws as a fourth and fifth field. Naming them keeps the
// arithmetic on Descr in one place.
func (s *VM) offsetOf(spec int) *int { return &s.Core[spec+s.Descr].A }
func (s *VM) lengthOf(spec int) *int { return &s.Core[spec+s.Descr].V }

// putSpecifier writes all five fields of a specifier (3.2).
func (s *VM) putSpecifier(spec, addr, flag, value, offset, length int) {
	s.Core[spec] = Cell{Kind: Data, A: addr, F: flag, V: value}
	s.Core[spec+s.Descr] = Cell{Kind: Data, A: offset, V: length}
}

// putChars writes characters into core. A character cell holds nothing
// but the character (3.3), so whatever descriptor fields the cells it
// overwrites were carrying go with them.
func (s *VM) putChars(a int, text []byte) {
	for i, c := range text {
		s.Core[a+i] = Cell{Kind: Data, Ch: c}
	}
}

// charsInCore reports whether a run of n characters starting at a lies
// inside core. One character per address unit, so this is the same
// question as whether the cells are there. See VM.Chars.
func (s *VM) charsInCore(a, n int) bool {
	return n >= 0 && a >= 0 && a+n <= len(s.Core)
}

// blank is the character TRIMSP trims. Characters are held one to a
// cell as their internal codes (3.3), which on this machine are ASCII,
// so a blank is a space.
const blank = ' '

// ADDLG (add to specifier length) is used to add an integer to the
// length of a specifier.
//
// Data Input:
//
//	SPEC  L
//	DESCR I
//
// Data Altered:
//
//	SPEC L+I
//
// Programming Notes:
//  1. I is always positive.
//
// S4D58.PDF: 6.3
func (s *VM) ADDLG(spec, descr int) { *s.lengthOf(spec) += s.Core[descr].A }

// APDSP (append specifier) is used to append one specified string to
// another specified string.
//
// Data Input:
//
//	SPEC1 A1,O1,L1
//	SPEC2 A2,O2,L2
//	A1+O1 C11...C1L1
//	A2+O2 C21...C2L2
//
// Data Altered:
//
//	SPEC1 A1,O1,L1+L2
//	A1+O1 C11...C1L1,C21...C2L2
//
// Programming Notes:
//  1. If L1 = 0, C21 is placed at A1+O1.
//  2. The storage following C1L1 is always adequate for C21...C2L2.
//
// Note 2 is a promise the SNOBOL4 source makes, not one the machine
// can check, so the only thing checked here is that the run of
// characters is inside core at all. The source is read before the
// destination is written, so an append of a string onto itself moves
// the characters that were there rather than the ones being written.
//
// S4D58.PDF: 6.11
func (s *VM) APDSP(spec1, spec2 int) error {
	a1, _, _, o1, l1 := s.Specifier(spec1)
	a2, _, _, o2, l2 := s.Specifier(spec2)
	if !s.charsInCore(a2+o2, l2) {
		return s.fault("APDSP: %d characters at %d are outside core", l2, a2+o2)
	}
	if !s.charsInCore(a1+o1+l1, l2) {
		return s.fault("APDSP: %d characters at %d are outside core", l2, a1+o1+l1)
	}
	s.putChars(a1+o1+l1, s.Chars(a2+o2, l2))
	*s.lengthOf(spec1) = l1 + l2
	return nil
}

// FSHRTN (foreshorten specifier) is used to exclude initial characters
// from a string specification.
//
// Data Input:
//
//	SPEC O,L
//
// Data Altered:
//
//	SPEC O+N,L-N
//
// Programming Notes:
//  1. L-N is never negative.
//  2. See also REMSP.
//
// S4D58.PDF: 6.35
func (s *VM) FSHRTN(spec, n int) {
	*s.offsetOf(spec) += n
	*s.lengthOf(spec) -= n
}

// GETBAL (get parenthesis balanced string) is used to get the
// specification of a balanced substring. The string starting at CL+1
// and ending at CL+N is examined to determine the shortest balanced
// substring CL+1,...,CL+J. J is determined according to the following
// rules:
//
// If CL+1 is not a parenthesis, then J = 1.
//
// If CL+1 is a left parenthesis, then J is the least integer such that
// CL+1...CL+J is balanced with respect to parentheses in the usual
// algebraic sense.
//
// If CL+1 is a right parenthesis, or if no such balanced string
// exists, transfer is to FLOC. Otherwise SPEC is modified as indicated
// and transfer is to SLOC.
//
// Data Input:
//
//	SPEC  A,O,L
//	DESCR N
//	A+O   C1...CL,CL+1...CL+N
//
// Data Altered:
//
//	SPEC A,O,L+J
//
// N = 0 leaves no CL+1 to examine, so no balanced string exists and
// transfer is to FLOC. That case is not drawn, but it is what "no such
// balanced string exists" means when the window is empty.
//
// S4D58.PDF: 6.37
func (s *VM) GETBAL(spec, descr, floc, sloc int) error {
	a, _, _, o, l := s.Specifier(spec)
	n := s.Core[descr].A
	if !s.charsInCore(a+o+l, n) {
		return s.fault("GETBAL: %d characters at %d are outside core", n, a+o+l)
	}

	j, ok := balance(s.Chars(a+o+l, n))
	if !ok {
		s.PC = floc
		return nil
	}
	*s.lengthOf(spec) = l + j
	s.PC = sloc
	return nil
}

// balance is the rule 6.37 states, over the window of characters
// GETBAL is given.
func balance(window []byte) (j int, ok bool) {
	if len(window) == 0 {
		return 0, false
	}
	switch window[0] {
	case ')':
		return 0, false
	case '(':
		depth := 0
		for i, c := range window {
			switch c {
			case '(':
				depth++
			case ')':
				depth--
			}
			if depth == 0 {
				return i + 1, true
			}
		}
		return 0, false
	}
	return 1, true
}

// GETSPC (get specifier with constant offset) is used to get a
// specifier.
//
// Data Input:
//
//	DESCR A1
//	A1+N  A,F,V,O,L
//
// Data Altered:
//
//	SPEC A,F,V,O,L
//
// Programming Notes:
//  1. See also PUTSPC.
//
// S4D58.PDF: 6.43
func (s *VM) GETSPC(spec, descr, n int) error {
	at := s.Core[descr].A + n
	if !s.inCore(at) || !s.inCore(at+s.Descr) {
		return s.fault("GETSPC: %d is outside core", at)
	}
	addr, flag, value, offset, length := s.Specifier(at)
	s.putSpecifier(spec, addr, flag, value, offset, length)
	return nil
}

// INTSPC (convert integer to specifier) is used to convert a (signed)
// integer to a specified string.
//
// Data Input:
//
//	DESCR I
//
// Data Altered:
//
//	SPEC     BUFFER,0,0,O,L
//	BUFFER+O C1...CL
//
// Programming Notes:
//  1. C1...CL should be a "normalized" string corresponding to the
//     integer I. That is, it should contain no leading zeroes and
//     should begin with a minus sign if I is negative.
//  2. BUFFER is local to INTSPC and its contents may be overwritten by
//     a subsequent use of INTSPC.
//  3. See also SPCINT.
//
// Note 2 is why the buffer is the machine's rather than the program's:
// nothing in the SNOBOL4 source names it, and its contents are
// promised to nobody past the next INTSPC. See VM.buffer. The offset
// is zero, which the figure leaves free.
//
// S4D58.PDF: 6.49
func (s *VM) INTSPC(spec, descr int) {
	text := []byte(strconv.Itoa(s.Core[descr].A))
	at := s.buffer(&s.intBuf, len(text))
	s.putChars(at, text)
	s.putSpecifier(spec, at, 0, 0, 0, len(text))
}

// LOCSP (locate specifier to string) is used to obtain a specifier to
// a string given in a string structure. CPD is the number of
// characters per descriptor.
//
// Data Input:
//
//	DESCR A,F,V
//	A     I
//
// Data Altered if A != 0:
//
//	SPEC A,F,V,4*CPD,I
//
// Data Altered if A = 0:
//
//	SPEC 0 (the length only)
//
// Programming Notes:
//  1. If A = 0, the value of DESCR represents the null (zero-length)
//     string and is handled as a special case as indicated. The other
//     fields of SPEC are unchanged in this case.
//
// The offset 4*CPD is where the characters of a string structure
// begin: 6.13 gives a string structure four descriptors including the
// title, and the SNOBOL4 source says the same thing as BCDFLD EQU
// 4*DESCR. CPD is DESCR*CPA, the characters a descriptor holds.
//
// S4D58.PDF: 6.60
func (s *VM) LOCSP(spec, descr int) error {
	d := s.Core[descr]
	if d.A == 0 {
		*s.lengthOf(spec) = 0
		return nil
	}
	if !s.inCore(d.A) {
		return s.fault("LOCSP: %d is outside core", d.A)
	}
	s.putSpecifier(spec, d.A, d.F, d.V, 4*s.Descr*s.CPA, s.Core[d.A].V)
	return nil
}

// PUTLG (put specifier length) is used to put a length into a
// specifier.
//
// Data Input:
//
//	DESCR I
//
// Data Altered:
//
//	SPEC I
//
// Programming Notes:
//  1. I is always nonnegative.
//  2. See also GETLG.
//
// S4D58.PDF: 6.84
func (s *VM) PUTLG(spec, descr int) { *s.lengthOf(spec) = s.Core[descr].A }

// PUTSPC (put specifier with offset constant) is used to put a
// specifier.
//
// Data Input:
//
//	DESCR A1
//	SPEC  A,F,V,O,L
//
// Data Altered:
//
//	A1+N A,F,V,O,L
//
// Programming Notes:
//  1. See also GETSPC.
//
// S4D58.PDF: 6.85
func (s *VM) PUTSPC(descr, n, spec int) error {
	at := s.Core[descr].A + n
	if !s.inCore(at) || !s.inCore(at+s.Descr) {
		return s.fault("PUTSPC: %d is outside core", at)
	}
	addr, flag, value, offset, length := s.Specifier(spec)
	s.putSpecifier(at, addr, flag, value, offset, length)
	return nil
}

// REMSP (specify remaining string) is used to obtain a remainder
// specifier resulting from the deletion of a specified length at the
// end.
//
// Data Input:
//
//	SPEC2 A2,F2,V2,O2,L2
//	SPEC3 L3
//
// Data Altered:
//
//	SPEC1 A2,F2,V2,O2+L3,L2-L3
//
// Programming Notes:
//  1. SPEC1 and SPEC3 may be the same.
//  2. L2-L3 is never negative.
//  3. See also FSHRTN.
//
// Note 1 is why both operands are read before either field of SPEC1
// is written.
//
// # The prose and the figure
//
// 6.90's sentence calls this "the deletion of a specified length at
// the end", which would leave the offset alone and give O2,L2-L3. The
// figure says O2+L3, and so does note 3's "see also FSHRTN", which is
// the operation that advances an offset. The source settles it: the
// six sites read "Get specifier to unscanned portion" (line 2532),
// "Remove part matched" (line 2800) and "Get tail of subject" (line
// 2272), and in each of them SPEC3 is what was just matched at the
// front. The figure is implemented.
//
// S4D58.PDF: 6.90
func (s *VM) REMSP(spec1, spec2, spec3 int) {
	a2, f2, v2, o2, l2 := s.Specifier(spec2)
	l3 := *s.lengthOf(spec3)
	s.putSpecifier(spec1, a2, f2, v2, o2+l3, l2-l3)
}

// SETLC (set length of specifier to constant) is used to set the
// length of a specifier to a constant.
//
// Data Altered:
//
//	SPEC N
//
// Programming Notes:
//  1. N is never negative.
//  2. N is often 0.
//  3. See also SETAC.
//
// S4D58.PDF: 6.103
func (s *VM) SETLC(spec, n int) { *s.lengthOf(spec) = n }

// SETSP (set specifier) is used to set one specifier equal to another.
//
// Data Input:
//
//	SPEC2 A,F,V,O,L
//
// Data Altered:
//
//	SPEC1 A,F,V,O,L
//
// S4D58.PDF: 6.105
func (s *VM) SETSP(spec1, spec2 int) {
	addr, flag, value, offset, length := s.Specifier(spec2)
	s.putSpecifier(spec1, addr, flag, value, offset, length)
}

// SHORTN (shorten specifier) is used to shorten the specification of a
// string.
//
// Data Input:
//
//	SPEC L
//
// Data Altered:
//
//	SPEC L-N
//
// Programming Notes:
//  1. L-N is never negative.
//
// S4D58.PDF: 6.108
func (s *VM) SHORTN(spec, n int) { *s.lengthOf(spec) -= n }

// SUBSP (substring specification) is used to specify an initial
// substring of a specified string. If L3 >= L2, transfer is to SLOC.
// Otherwise transfer is to FLOC and SPEC1 is not altered.
//
// Data Input:
//
//	SPEC2 L2
//	SPEC3 A3,F3,V3,O3,L3
//
// Data Altered if L3 >= L2:
//
//	SPEC1 A3,F3,V3,O3,L2
//
// SPEC1 and SPEC3 may be the same, as in REMSP, so both are read
// before either is written.
//
// S4D58.PDF: 6.118
func (s *VM) SUBSP(spec1, spec2, spec3, floc, sloc int) {
	l2 := *s.lengthOf(spec2)
	a3, f3, v3, o3, l3 := s.Specifier(spec3)
	if l3 < l2 {
		s.PC = floc
		return
	}
	s.putSpecifier(spec1, a3, f3, v3, o3, l2)
	s.PC = sloc
}

// TRIMSP (trim blanks from specifier) is used to obtain a specifier to
// the part of a specified string up to a trailing string of blanks.
//
// Data Input:
//
//	SPEC2 A,F,V,O,L
//	A+O   C1...CJ,CJ+1...CL
//
// Data Altered:
//
//	SPEC1 A,F,V,O,J
//
// Programming Notes:
//  1. If CL is not blank, J = L.
//  2. If L = 0, TRIMSP is equivalent to SETSP.
//
// S4D58.PDF: 6.125
func (s *VM) TRIMSP(spec1, spec2 int) error {
	a, f, v, o, l := s.Specifier(spec2)
	if !s.charsInCore(a+o, l) {
		return s.fault("TRIMSP: %d characters at %d are outside core", l, a+o)
	}
	j := l
	for j > 0 && s.Core[a+o+j-1].Ch == blank {
		j--
	}
	s.putSpecifier(spec1, a, f, v, o, j)
	return nil
}
