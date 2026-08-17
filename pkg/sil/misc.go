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

// S4D58 7.5's miscellaneous group: the searches over attribute lists
// and blocks, the two operations that walk a pattern's alternative
// chain, string replacement, string-to-integer conversion and the
// variable hash.
//
// # The alternative chain
//
// LINKOR and LVALUE both walk it, and between them they say what it
// is. 6.61's figure is the one that settles the arithmetic: from a
// pattern at A the first alternative field is at A+2D and holds N1,
// and the next field is at A+N1+2D. So a field holds the offset of the
// next node from A, the base of the pattern, and not from itself.
//
// CPYPAT agrees from the other side. Copying X | Y (the source's ORPP,
// line 2171) copies X with A4 = 0 and then Y with A4 = XSIZ, and 6.21
// relocates each alternative field by F1(X) = X+A4. A self-relative
// link would need no relocation when Y moves as a unit; one measured
// from the base of the block needs exactly XSIZ. The LINKOR that
// follows, at line 2180, then writes XSIZ into the end of X's chain,
// which is the offset of Y's first node from that same base.

// chainStep is the address of the alternative field of the node an
// alternative field points at, from a pattern based at a.
func (s *VM) chainStep(a, offset int) int { return a + offset + 2*s.Descr }

// LINKOR (link "or" fields of pattern nodes) links through "or"
// (alternative) fields of pattern nodes until the end, indicated by a
// zero field, is reached. This zero field is replaced by I.
//
// Data Input:
//
//	DESCR1  A
//	DESCR2  I
//	A+2D    I1
//	A+2D+I1 I2
//	...
//	A+2D+IN 0
//
// Data Altered:
//
//	A+2D+IN I
//
// S4D58.PDF: 6.56
func (s *VM) LINKOR(descr1, descr2 int) error {
	a, i := s.Core[descr1].A, s.Core[descr2].A
	at := a + 2*s.Descr
	for seen := 0; ; seen++ {
		if !s.inCore(at) {
			return s.fault("LINKOR: %d is outside core", at)
		}
		next := s.Core[at].A
		if next == 0 {
			s.Core[at].A = i
			return nil
		}
		if seen > len(s.Core) {
			return s.fault("LINKOR: the alternative chain from %d does not end", a)
		}
		at = s.chainStep(a, next)
	}
}

// LVALUE (get least length value) is used to get the least value of
// address fields in a chain of pattern nodes. The address field of
// DESCR1 is set to I where I = min(I0,...,IK).
//
// Data Input:
//
//	DESCR2  A
//	A+2D    N1
//	A+3D    I0
//	A+N1+2D N2
//	A+N1+3D I1
//	...
//	A+NK+2D 0
//	A+NK+3D IK
//
// Data Altered:
//
//	DESCR1 I,0,0
//
// Programming Notes:
//  1. I0,...,IK are all nonnegative.
//  2. A is never zero, but N1 may be.
//
// Note 2's second half is the terminal case of the walk rather than a
// special one: a zero alternative field ends the chain wherever it is
// found, so N1 = 0 gives a chain of the one node and I = I0.
//
// S4D58.PDF: 6.61
func (s *VM) LVALUE(descr1, descr2 int) error {
	a := s.Core[descr2].A
	at := a + 2*s.Descr
	least := 0
	for seen := 0; ; seen++ {
		if !s.inCore(at) || !s.inCore(at+s.Descr) {
			return s.fault("LVALUE: %d is outside core", at)
		}
		if i := s.Core[at+s.Descr].A; seen == 0 || i < least {
			least = i
		}
		next := s.Core[at].A
		if next == 0 {
			break
		}
		if seen > len(s.Core) {
			return s.fault("LVALUE: the alternative chain from %d does not end", a)
		}
		at = s.chainStep(a, next)
	}
	s.Core[descr1] = Cell{Kind: Data, A: least}
	return nil
}

// LOCAPT (locate attribute pair by type) is used to locate the "type"
// descriptor of a descriptor pair on an attribute list. Descriptors on
// an attribute list are in "type-value" pairs. Odd-numbered
// descriptors are "type" descriptors. The list starting at A+D is
// searched, comparing descriptors at A+D, A+3D, ... for the first
// descriptor whose value is equal to the value of DESCR3. If a
// descriptor equal to DESCR3 is not found, transfer is to FLOC.
// Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2    A,F,V
//	DESCR3    A3,F3,V3
//	A         2K*D (the value field: the size of the block)
//	A+D       A11,F11,V11
//	...
//	A+D+2I*D  A3,F3,V3
//	...
//
// Data Altered:
//
//	DESCR1 A+2I*D,F,V
//
// Programming Notes:
//  1. Note that the address of DESCR1 is set to one descriptor less
//     than the descriptor that is located.
//  2. See also LOCAPV.
//
// The prose says "whose value is equal to the value of DESCR3" and
// then "if a descriptor equal to DESCR3 is not found"; the figure
// draws the match as A3,F3,V3 against DESCR3's A3,F3,V3. All three
// fields, then -- the same comparison DEQL makes.
//
// S4D58.PDF: 6.58
func (s *VM) LOCAPT(descr1, descr2, descr3, floc, sloc int) error {
	return s.locateAttribute("LOCAPT", descr1, descr2, descr3, s.Descr, floc, sloc)
}

// LOCAPV (locate attribute pair by value) is used to locate the
// "value" descriptor of a descriptor pair on an attribute list.
// Descriptors on an attribute list are in "type-value" pairs.
// Even-numbered descriptors are "value" descriptors. The list starting
// at A+D is searched, comparing descriptors at A+2D, A+4D, ... for the
// first descriptor whose value is equal to the value of DESCR3. If a
// descriptor equal to DESCR3 is not found, transfer is to FLOC.
// Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2     A,F,V
//	DESCR3     A3,F3,V3
//	A          2K*D (the value field: the size of the block)
//	A+2D       A12,F12,V12
//	...
//	A+2D+2I*D  A3,F3,V3
//	...
//
// Data Altered:
//
//	DESCR1 A+2I*D,F,V
//
// Programming Notes:
//  1. Note that the address of DESCR1 is set to two descriptors less
//     than the descriptor that is located.
//  2. See also LOCAPT.
//
// S4D58.PDF: 6.59
func (s *VM) LOCAPV(descr1, descr2, descr3, floc, sloc int) error {
	return s.locateAttribute("LOCAPV", descr1, descr2, descr3, 2*s.Descr, floc, sloc)
}

// locateAttribute is 6.58 and 6.59, which differ only in which of each
// pair is examined -- the first for LOCAPT and the second for LOCAPV.
// Both leave DESCR1 addressing the start of the pair rather than the
// descriptor found, which is what their note 1 is pointing out and why
// one subtraction serves both.
func (s *VM) locateAttribute(of string, descr1, descr2, descr3, first, floc, sloc int) error {
	block := s.Core[descr2]
	a, want := block.A, descriptorAt(s.Core, descr3)
	if !s.inCore(a) {
		return s.fault("%s: %d is outside core", of, a)
	}
	// The block's title holds 2K*D, its size in address units. A block
	// is a title and then that much storage -- the source writes
	// BLOCK DESCR BLOCK,TTL+MARK,LEN*DESCR followed by ARRAY LEN -- so
	// the 2K descriptors of the list run from A+D through A+2K*D, and
	// the last of them is included.
	end := a + s.Core[a].V
	pair := 2 * s.Descr

	for at := a + first; at <= end; at += pair {
		if !s.inCore(at) {
			return s.fault("%s: %d is outside core", of, at)
		}
		if got := descriptorAt(s.Core, at); got.A == want.A && got.F == want.F && got.V == want.V {
			s.Core[descr1] = Cell{Kind: Data, A: at - first, F: block.F, V: block.V}
			s.PC = sloc
			return nil
		}
	}
	s.PC = floc
	return nil
}

// TOP (get to top of block) is used to get to the top of a block of
// descriptors. Descriptors at A, A-D,...,A-(N*D) are examined
// successively for the first descriptor whose flag field contains the
// flag TTL. Data is altered as indicated, where F3N is the first field
// to contain TTL.
//
// Data Input:
//
//	DESCR3  A,F,V
//	A       F30
//	A-D     F31
//	...
//	A-(N*D) F3N
//
// Data Altered:
//
//	DESCR1 A-(N*D),F,V
//	DESCR2 N*D,0,0
//
// Programming Notes:
//  1. N may be 0. That is, F30 may contain TTL.
//
// TTL is one of the five flags PARMS chooses (6.20), so the machine
// reads it out of the assembly.
//
// S4D58.PDF: 6.124
func (s *VM) TOP(descr1, descr2, descr3 int) error {
	ttl, ok := s.Symbols["TTL"]
	if !ok {
		return s.fault("TOP: the flag TTL is not defined")
	}
	from := s.Core[descr3]
	for at := from.A; ; at -= s.Descr {
		if !s.inCore(at) {
			return s.fault("TOP: no descriptor at or below %d carries TTL", from.A)
		}
		if s.Core[at].F&ttl == ttl {
			s.Core[descr1] = Cell{Kind: Data, A: at, F: from.F, V: from.V}
			s.Core[descr2] = Cell{Kind: Data, A: from.A - at}
			return nil
		}
	}
}

// RPLACE (replace characters) is used to replace characters in a
// string. SPEC2 specifies a set of characters to be replaced. SPEC3
// specifies the replacement to be made for the characters specified by
// SPEC2. The replacement is described by the following rules. For
// I = 1,...,L
//
//	F(CI) = CI  if CI != C2J for any J (1 <= J <= L2)
//	F(CI) = C3J if CI  = C2J for some J (1 <= J <= L2)
//
// Data Input:
//
//	SPEC1 A1,O1,L
//	SPEC2 A2,O2,L2
//	SPEC3 A3,O3,L2
//	A1+O1 C1...CL
//	A2+O2 C21...C2L2
//	A3+O3 C31...C3L2
//
// Data Altered:
//
//	A1+O1 F(C1)...F(CL)
//
// Programming Notes:
//  1. L may be zero.
//  2. If there are duplicate characters in C21...C2L2, replacement
//     should be made corresponding to the last instance of the
//     character.
//  3. RPLACE is used only in the SNOBOL4 REPLACE function.
//
// Note 2 is why the table is built by running over SPEC2 forwards and
// letting a later pair overwrite an earlier one.
//
// S4D58.PDF: 6.94
func (s *VM) RPLACE(spec1, spec2, spec3 int) error {
	a1, _, _, o1, l := s.Specifier(spec1)
	a2, _, _, o2, l2 := s.Specifier(spec2)
	a3, _, _, o3, _ := s.Specifier(spec3)

	for _, r := range [][2]int{{a1 + o1, l}, {a2 + o2, l2}, {a3 + o3, l2}} {
		if !s.charsInCore(r[0], r[1]) {
			return s.fault("RPLACE: %d characters at %d are outside core", r[1], r[0])
		}
	}

	var to [256]byte
	var replaced [256]bool
	from, with := s.Chars(a2+o2, l2), s.Chars(a3+o3, l2)
	for i, c := range from {
		to[c], replaced[c] = with[i], true
	}

	text := s.Chars(a1+o1, l)
	for i, c := range text {
		if replaced[c] {
			text[i] = to[c]
		}
	}
	s.putChars(a1+o1, text)
	return nil
}

// SPCINT (convert specifier to integer) is used to convert a specified
// string to an integer. I(S) is a signed integer resulting from the
// conversion of the string C1...CL. If C1...CL does not represent an
// integer or if the integer it represents is too large to fit the
// address field, transfer is to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	SPEC A,O,L
//	A+O  C1...CL
//
// Data Altered:
//
//	DESCR I(S),0,I
//
// Programming Notes:
//  1. I is a symbol defined in the source program and is the code for
//     the integer data type.
//  2. C1...CL may begin with a sign (plus or minus) and may contain an
//     indefinite number of leading zeros. Consequently the value of L
//     itself does not determine whether the integer represented is too
//     large to fit into an address field.
//  3. A sign alone is not a valid integer.
//  4. If L = 0, I(S) should be the integer 0.
//  5. See also INTSPC and SPREAL.
//
// S4D58.PDF: 6.109
func (s *VM) SPCINT(descr, spec, floc, sloc int) error {
	integer, err := s.typeCode("SPCINT", "I")
	if err != nil {
		return err
	}
	addr, _, _, offset, length := s.Specifier(spec)
	if !s.charsInCore(addr+offset, length) {
		return s.fault("SPCINT: %d characters at %d are outside core", length, addr+offset)
	}

	n, ok := s.integerOf(s.Chars(addr+offset, length))
	if !ok {
		s.PC = floc
		return nil
	}
	s.Core[descr] = Cell{Kind: Data, A: n, V: integer}
	s.PC = sloc
	return nil
}

// integerOf is I(S), and whether the string represents an integer that
// fits. A decimal point is not accepted here as it is in SPREAL: 6.109
// note 2 lists a sign and leading zeros as what may appear and nothing
// else, and the source calls SPCINT first and SPREAL second precisely
// to tell the two apart (line 4716).
func (s *VM) integerOf(text []byte) (int, bool) {
	if len(text) == 0 {
		return 0, true // note 4
	}
	i := 0
	if text[0] == '+' || text[0] == '-' {
		i++
	}
	if i == len(text) {
		return 0, false // note 3: a sign alone is not a valid integer
	}
	for ; i < len(text); i++ {
		if c := text[i]; c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(string(text))
	if err != nil || s.outOfRange(n) {
		return 0, false
	}
	return n, true
}

// VARID (compute variable identification numbers) is used to compute
// two variable identification numbers from a specified string. K and M
// are computed by
//
//	K = F1(C1...CL)
//	M = F2(C1...CL)
//
// where F1 and F2 are two (different) functions that compute
// pseudo-random numbers from the characters C1...CL. The numbers
// computed should be in the ranges
//
//	0 <= K <= (OBSIZ-1)*D
//	0 <= M <= SIZLIM
//
// where OBSIZ is a program symbol defining the number of chains in
// variable storage and SIZLIM is a program symbol defining the largest
// integer that can be stored in the value field of a descriptor.
//
// Data Input:
//
//	SPEC A,O,L
//	A+O  C1...CL
//
// Data Altered:
//
//	DESCR K,M (the address and value fields)
//
// Programming Notes:
//  1. K is used to select one of a number of chains in variable
//     storage. The K are address offsets that must fall on descriptor
//     boundaries.
//  2. M is used to order variables (string structures) within a chain.
//     See ORDVST.
//  3. The values of K and M should have as little correlation as
//     possible with the characters C1...CL, since the "randomness" of
//     the results determines the efficiency of variable access.
//  4. One simple algorithm consists of multiplying the first part of
//     C1...CL by the last part, and separating the central portion of
//     the result into K and M.
//  5. L is always greater than zero.
//
// # The choice of hash
//
// Note 4 offers an algorithm rather than requiring one, and notes 1 to
// 3 are the specification: two functions, uncorrelated with the
// characters and with each other, K a descriptor-aligned offset below
// (OBSIZ-1)*D and M no larger than SIZLIM. FNV-1a over the characters
// gives the first, and FNV-1a over the characters reversed with a
// different offset basis gives a second that is independent of it.
// Both are fixed arithmetic with no seed, so a run is reproducible and
// a core listing is stable, which 6.127 does not ask for but AGENTS.md
// does.
//
// The flag field is not named in the figure, so it is not touched;
// 6.100 shows that this document writes a zero when it means one.
//
// S4D58.PDF: 6.127
func (s *VM) VARID(descr, spec int) error {
	bins, ok := s.Symbols["OBSIZ"]
	if !ok || bins <= 0 {
		return s.fault("VARID: OBSIZ is not defined, and it is the number of chains in variable storage")
	}
	limit, ok := s.Symbols["SIZLIM"]
	if !ok || limit <= 0 {
		return s.fault("VARID: SIZLIM is not defined, and M may not exceed it")
	}
	addr, _, _, offset, length := s.Specifier(spec)
	if !s.charsInCore(addr+offset, length) {
		return s.fault("VARID: %d characters at %d are outside core", length, addr+offset)
	}

	text := s.Chars(addr+offset, length)
	// Note 1: K is an address offset on a descriptor boundary, so it
	// counts bins and is then scaled, which is also what keeps it
	// inside (OBSIZ-1)*D.
	k := int(fnv(text, false)%uint64(bins)) * s.Descr
	m := int(fnv(text, true) % uint64(limit+1))

	s.Core[descr].A, s.Core[descr].V = k, m
	return nil
}

// fnv is FNV-1a over the characters, forwards or backwards. The two
// directions are the two functions 6.127 asks for; running the same
// mixing over a different order of the same bytes decorrelates them
// about as well as two separate hashes would.
func fnv(text []byte, backwards bool) uint64 {
	const (
		basis = 14695981039346656037
		prime = 1099511628211
	)
	h := uint64(basis)
	for i := range text {
		c := text[i]
		if backwards {
			c = text[len(text)-1-i]
		}
		h ^= uint64(c)
		h *= prime
	}
	return h
}

// ORDVST (order variable storage) is used to alphabetically order
// variables in SNOBOL4 dynamic storage.
//
// Programming Notes:
//  1. ORDVST is used only in ordering variables for a
//     programmer-requested post-mortem dump of variable storage.
//     ORDVST need not be implemented as such, but may simply perform
//     no operation. In this case, the post-mortem dump will not be
//     alphabetized, but will be otherwise correct.
//  2. If ORDVST is implemented, it is easiest to put all variables in
//     one long chain starting at OBSTRT. The address fields of the
//     descriptors OBSTRT+D,...,OBSTRT+(OBSIZ-1)*D should then be set
//     to zero.
//  3. Since dynamic storage may contain many variables, some care must
//     be taken to assure that the sorting procedure is not excessively
//     slow. Variables whose values are the null string (zero address
//     field and value field containing the program symbol S) should be
//     omitted from the sort.
//  4. Since any character may appear in a string, the value of I must
//     be used to determine the length of the string in a variable --
//     characters following the string in the last descriptor are
//     undefined.
//
// # This is the documented alternative, not the operation
//
// Note 1 offers "perform no operation" as an implementation, and 7.1
// lists ORDVST with that alternative and names the feature it disables
// as "alphabetization of post-run dump". That is what is here, and it
// is a deviation stated rather than hidden.
//
// The reason for taking it is note 3. Sorting needs the variable's
// value descriptor, to leave out the ones whose value is the null
// string, and 6.74 draws only the parts of a variable it calls
// relevant: the length at A, the link at A+3D and the characters at
// A+4D. Where the value lives is not in this section, so implementing
// note 3 would mean deciding it from the source. The one call site,
// line 5118, runs once at the end of a run and only under &DUMP.
//
// S4D58.PDF: 6.74
func (s *VM) ORDVST() {}
