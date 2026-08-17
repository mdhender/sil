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

// The operations that move and set whole descriptors, S4D58 7.5's
// sixth group, and the ones that modify address fields, its seventh.
// MOVD and POP are in macros.go with the rest of the vertical slice.

// GETD (get descriptor) is used to get a descriptor.
//
// Data Input:
//
//	DESCR2 A2
//	DESCR3 A3
//	A2+A3  A,F,V
//
// Data Altered:
//
//	DESCR1 A,F,V
//
// Programming Notes:
//  1. See also GETDC, PUTD, and PUTDC.
//
// S4D58.PDF: 6.38
func (s *VM) GETD(descr1, descr2, descr3 int) error {
	at := s.Core[descr2].A + s.Core[descr3].A
	if !s.inCore(at) {
		return s.fault("GETD: %d is outside core", at)
	}
	s.Core[descr1] = descriptorAt(s.Core, at)
	return nil
}

// GETDC (get descriptor with offset constant) is used to get a
// descriptor with an offset constant.
//
// Data Input:
//
//	DESCR2 A2
//	A2+N   A,F,V
//
// Data Altered:
//
//	DESCR1 A,F,V
//
// Programming Notes:
//  1. See also GETD, PUTDC, and PUTD.
//
// S4D58.PDF: 6.39
func (s *VM) GETDC(descr1, descr2, n int) error {
	at := s.Core[descr2].A + n
	if !s.inCore(at) {
		return s.fault("GETDC: %d is outside core", at)
	}
	s.Core[descr1] = descriptorAt(s.Core, at)
	return nil
}

// PUTD (put descriptor) is used to put a descriptor.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 A2
//	DESCR3 A,F,V
//
// Data Altered:
//
//	A1+A2 A,F,V
//
// Programming Notes:
//  1. See also PUTDC, PUTAC, PUTVC, and GETD.
//
// S4D58.PDF: 6.82
func (s *VM) PUTD(descr1, descr2, descr3 int) error {
	at := s.Core[descr1].A + s.Core[descr2].A
	if !s.inCore(at) {
		return s.fault("PUTD: %d is outside core", at)
	}
	s.Core[at] = descriptorAt(s.Core, descr3)
	return nil
}

// PUTDC (put descriptor with constant offset) is used to put a
// descriptor at a location with a constant offset.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 A,F,V
//
// Data Altered:
//
//	A1+N A,F,V
//
// Programming Notes:
//  1. See also PUTD, PUTAC, PUTVC, and GETD.
//
// S4D58.PDF: 6.83
func (s *VM) PUTDC(descr1, n, descr2 int) error {
	at := s.Core[descr1].A + n
	if !s.inCore(at) {
		return s.fault("PUTDC: %d is outside core", at)
	}
	s.Core[at] = descriptorAt(s.Core, descr2)
	return nil
}

// MOVDIC (move descriptor indirect with constant offset) is used to
// move a descriptor that is indirectly specified with an offset
// constant.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 A2
//	A2+N2  A,F,V
//
// Data Altered:
//
//	A1+N1 A,F,V
//
// Programming Notes:
//  1. See also MOVD, GETDC, and PUTDC.
//
// S4D58.PDF: 6.68
func (s *VM) MOVDIC(descr1, n1, descr2, n2 int) error {
	from, to := s.Core[descr2].A+n2, s.Core[descr1].A+n1
	if !s.inCore(from) || !s.inCore(to) {
		return s.fault("MOVDIC: %d to %d, outside core", from, to)
	}
	s.Core[to] = descriptorAt(s.Core, from)
	return nil
}

// MOVBLK (move block of descriptors) is used to move (copy) a block of
// descriptors.
//
// Data Input:
//
//	DESCR1    A1
//	DESCR2    A2
//	DESCR3    D*N
//	A2+D      A21,F21,V21
//	...
//	A2+(D*N)  A2N,F2N,V2N
//
// Data Altered:
//
//	A1+D      A21,F21,V21
//	...
//	A1+(D*N)  A2N,F2N,V2N
//
// Programming Notes:
//  1. Note that the descriptor at A1 is not altered.
//  2. The area into which the move is made may overlap the area from
//     which the move is made. This only occurs when A1 is less than
//     A2. Care must be taken to handle this case correctly.
//
// Note 2 is why this is a slice copy: Go's copy is defined for
// overlapping slices of the same array, in either direction.
//
// S4D58.PDF: 6.66
func (s *VM) MOVBLK(descr1, descr2, descr3 int) error {
	a1, a2, size := s.Core[descr1].A, s.Core[descr2].A, s.Core[descr3].A
	if size <= 0 {
		return nil
	}
	// Note 1: the block starts one descriptor past the title.
	from, to := a2+s.Descr, a1+s.Descr
	n := size / s.Descr
	if !s.inCore(from) || !s.inCore(from+n-1) || !s.inCore(to) || !s.inCore(to+n-1) {
		return s.fault("MOVBLK: %d descriptors from %d to %d, outside core", n, from, to)
	}
	copy(s.Core[to:to+n], s.Core[from:from+n])
	return nil
}

// ZERBLK (zero block) is used to zero a block of I+1 descriptors.
//
// Data Input:
//
//	DESCR1 A
//	DESCR2 D*I
//
// Data Altered:
//
//	A        0,0,0
//	...
//	A+(D*I)  0,0,0
//
// Programming Notes:
//  1. I is always positive.
//
// S4D58.PDF: 6.131
func (s *VM) ZERBLK(descr1, descr2 int) error {
	a, size := s.Core[descr1].A, s.Core[descr2].A
	n := size/s.Descr + 1
	if !s.inCore(a) || !s.inCore(a+n-1) {
		return s.fault("ZERBLK: %d descriptors at %d, outside core", n, a)
	}
	for i := 0; i < n; i++ {
		c := &s.Core[a+i]
		c.A, c.F, c.V = 0, 0, 0
	}
	return nil
}

// PUSH (push descriptors onto stack) is used to push a list of
// descriptors onto the system stack.
//
// Data Input:
//
//	CSTACK A
//	DESCR1 A1,F1,V1
//	...
//	DESCRN AN,FN,VN
//
// Data Altered:
//
//	CSTACK  A+(D*N)
//	A+D     A1,F1,V1
//	...
//	A+(D*N) AN,FN,VN
//
// Programming Notes:
//  1. If A+(D*N) > STACK+STSIZE, stack overflow occurs. Transfer
//     should be made to the program location OVER, which will result
//     in an appropriate error termination.
//  2. See also SPUSH, POP, and SPOP.
//
// Note 1 writes the top of the stack as STACK+STSIZE. STSIZE is the
// number of descriptors the SNOBOL4 source reserves -- STACK DESCR
// STACK,TTL+MARK,STSIZE*DESCR and then ARRAY STSIZE -- so the top is
// STACK+STSIZE*DESCR, which is what is checked here.
//
// S4D58.PDF: 6.80
func (s *VM) PUSH(descrs []int) error {
	top := s.CStack + len(descrs)*s.Descr
	if limit, ok := s.stackTop(); ok && top > limit {
		return s.overflow("PUSH: %d descriptors would take the stack to %d, past %d", len(descrs), top, limit)
	}
	if !s.inCore(top) {
		return s.overflow("PUSH: %d descriptors would take the stack to %d, outside core", len(descrs), top)
	}
	for i, descr := range descrs {
		s.Core[s.CStack+(i+1)*s.Descr] = descriptorAt(s.Core, descr)
	}
	s.CStack = top
	return nil
}

// stackTop is the last address the system stack occupies, when the
// program says how big it is.
func (s *VM) stackTop() (int, bool) {
	stack, ok := s.Symbols["STACK"]
	if !ok {
		return 0, false
	}
	size, ok := s.Symbols["STSIZE"]
	if !ok {
		return 0, false
	}
	return stack + size*s.Descr, true
}

// ADJUST (compute adjusted address) is used to adjust the address
// field of a descriptor.
//
// Data Input:
//
//	DESCR2 A2
//	DESCR3 A3
//	A2     A4
//
// Data Altered:
//
//	DESCR1 A3+A4
//
// Programming Notes:
//  1. A3 is always an address integer.
//
// S4D58.PDF: 6.6
func (s *VM) ADJUST(descr1, descr2, descr3 int) error {
	at := s.Core[descr2].A
	if !s.inCore(at) {
		return s.fault("ADJUST: %d is outside core", at)
	}
	s.Core[descr1].A = s.Core[descr3].A + s.Core[at].A
	return nil
}

// BKSIZE (get block size) is used to determine the amount of storage
// occupied by a block or string structure. The flag field of the
// descriptor at A distinguishes between string structures and blocks.
// If F contains the flag STTL, then
//
//	F(V)=D*(4+[(V-1)/CPD+1])
//
// where [V] is the integer part of V and CPD is the number of
// characters stored per descriptor. The constant 4 occurs because
// there are 4 descriptors (including the title) in a string structure
// in addition to the string itself. The expression in brackets
// represents the number of descriptors required for a string of V
// characters. If F does not contain the flag STTL, then F(V) = V+D.
//
// Data Input:
//
//	DESCR2 A
//	A      F,V
//
// Data Altered:
//
//	DESCR1 F(V),0,0
//
// Programming Notes:
//  1. See also GETLTH.
//
// S4D58.PDF: 6.13
func (s *VM) BKSIZE(descr1, descr2 int) error {
	at := s.Core[descr2].A
	if !s.inCore(at) {
		return s.fault("BKSIZE: %d is outside core", at)
	}
	title := s.Core[at]

	size := title.V + s.Descr
	sttl, ok := s.Symbols["STTL"]
	if !ok {
		return s.fault("BKSIZE: the flag STTL is not defined; COPY PARMS must define it (6.20)")
	}
	if title.F&sttl != 0 {
		size = s.Descr * (4 + s.descriptorsFor(title.V))
	}
	s.Core[descr1] = Cell{Kind: Data, A: size}
	return nil
}

// GETLTH (get length for string structure) is used to determine the
// amount of storage required for a string structure. The amount of
// storage is given by the formula
//
//	F(L)=D*(3+[(L-1)/CPD+1])
//
// where [L] is the integer part of L and CPD is the number of
// characters stored per descriptor. The constant 3 accounts for the
// three descriptors in a string structure in addition to the string
// itself.
//
// Data Input:
//
//	DESCR2 L
//
// Data Altered:
//
//	DESCR1 F(L),0,0
//
// Programming Notes:
//  1. See also BKSIZE.
//
// S4D58.PDF: 6.41
func (s *VM) GETLTH(descr1, descr2 int) {
	l := s.Core[descr2].A
	s.Core[descr1] = Cell{Kind: Data, A: s.Descr * (3 + s.descriptorsFor(l))}
}

// descriptorsFor is [(L-1)/CPD+1], the number of descriptors a string
// of L characters occupies, where CPD is the number of characters
// stored per descriptor (S4D58 5.3). A string of no characters
// occupies none.
func (s *VM) descriptorsFor(l int) int {
	if l <= 0 {
		return 0
	}
	cpd := s.Descr * s.CPA
	return (l-1)/cpd + 1
}

// DECRA (decrement address) is used to decrement the address field of
// a descriptor. A is considered as a signed integer.
//
// Data Altered:
//
//	DESCR A-N
//
// Programming Notes:
//  1. A may be a relocatable address.
//  2. N is always positive.
//  3. N is often 1 or D.
//  4. A-N may be negative.
//  5. See also INCRA.
//
// S4D58.PDF: 6.23
func (s *VM) DECRA(descr, n int) { s.Core[descr].A -= n }

// INCRA (increment address) is used to increment the address field of
// a descriptor.
//
// Data Altered:
//
//	DESCR A+N
//
// Programming Notes:
//  1. A may be a relocatable address.
//  2. A is never negative.
//  3. N is always positive.
//  4. N is often 1 or D.
//  5. See also DECRA and INCRV.
//
// S4D58.PDF: 6.44
func (s *VM) INCRA(descr, n int) { s.Core[descr].A += n }

// GETAC (get address with offset constant) is used to get an address
// field with an offset constant.
//
// Data Input:
//
//	DESCR2 A2
//	A2+N   A
//
// Data Altered:
//
//	DESCR1 A
//
// Programming Notes:
//  1. N may be negative.
//  2. See also PUTAC, GETDC, and PUTDC.
//
// S4D58.PDF: 6.36
func (s *VM) GETAC(descr1, descr2, n int) error {
	at := s.Core[descr2].A + n
	if !s.inCore(at) {
		return s.fault("GETAC: %d is outside core", at)
	}
	s.Core[descr1].A = s.Core[at].A
	return nil
}

// PUTAC (put address with offset constant) is used to put an address
// field into a descriptor located at a constant offset.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 A2
//
// Data Altered:
//
//	A1+N A2
//
// Programming Notes:
//  1. See also GETAC, PUTVC, PUTD, and PUTDC.
//
// S4D58.PDF: 6.81
func (s *VM) PUTAC(descr1, n, descr2 int) error {
	at := s.Core[descr1].A + n
	if !s.inCore(at) {
		return s.fault("PUTAC: %d is outside core", at)
	}
	s.Core[at].A = s.Core[descr2].A
	return nil
}

// GETSIZ (get size) is used to get the size from the value field of a
// title descriptor.
//
// Data Input:
//
//	DESCR2 A
//	A      V
//
// Data Altered:
//
//	DESCR1 V,0,0
//
// Programming Notes:
//  1. See also SETSIZ.
//
// S4D58.PDF: 6.42
func (s *VM) GETSIZ(descr1, descr2 int) error {
	at := s.Core[descr2].A
	if !s.inCore(at) {
		return s.fault("GETSIZ: %d is outside core", at)
	}
	s.Core[descr1] = Cell{Kind: Data, A: s.Core[at].V}
	return nil
}

// GETLG (get length of specifier) is used to get the length of a
// specifier.
//
// Data Input:
//
//	SPEC L
//
// Data Altered:
//
//	DESCR L,0,0
//
// Programming Notes:
//  1. See also PUTLG.
//
// S4D58.PDF: 6.40
func (s *VM) GETLG(descr, spec int) {
	_, _, _, _, length := s.Specifier(spec)
	s.Core[descr] = Cell{Kind: Data, A: length}
}

// MOVA (move address) is used to move an address field from one
// descriptor to another.
//
// Data Input:
//
//	DESCR2 A
//
// Data Altered:
//
//	DESCR1 A
//
// Programming Notes:
//  1. See also MOVD and MOVV.
//
// S4D58.PDF: 6.65
func (s *VM) MOVA(descr1, descr2 int) { s.Core[descr1].A = s.Core[descr2].A }

// SETAV (set address from value field) sets the address field of one
// descriptor from the value field of another.
//
// Data Input:
//
//	DESCR2 V
//
// Data Altered:
//
//	DESCR1 V,0,0
//
// Programming Notes:
//  1. See also SETAC.
//
// S4D58.PDF: 6.100
func (s *VM) SETAV(descr1, descr2 int) {
	s.Core[descr1] = Cell{Kind: Data, A: s.Core[descr2].V}
}
