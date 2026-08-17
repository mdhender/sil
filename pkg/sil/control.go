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

// The rest of S4D58 7.5's stack group -- PSTACK, SPUSH and SPOP -- and
// the two branch operations the vertical slice did not need. ISTACK,
// PUSH, POP, RCALL and RRTURN are in macros.go and descriptors.go with
// the rest of that slice, and PROC assembles nothing (6.78 note 2).
//
// A specifier on the stack occupies S = 2D address units and is
// stacked exactly as a pair of descriptors would be, which is what
// 3.2 means by "specifiers and descriptors may be stored in the same
// area indiscriminately". SPUSH and SPOP are therefore PUSH and POP
// with S in place of D.

// PSTACK (post stack position) is used to post the current stack
// position.
//
// Data Input:
//
//	CSTACK A
//
// Data Altered:
//
//	DESCR A-D,0,0
//
// Programming Notes:
//  1. See also ISTACK.
//
// A-D rather than A because CSTACK addresses the descriptor on top of
// the stack, so A-D is the position the stack was in before whatever
// is on top was put there.
//
// S4D58.PDF: 6.79
func (s *VM) PSTACK(descr int) {
	s.Core[descr] = Cell{Kind: Data, A: s.CStack - s.Descr}
}

// SPUSH (push specifiers onto stack) is used to push a list of
// specifiers onto the system stack.
//
// Data Input:
//
//	CSTACK A
//	SPEC1  A1,F1,V1,O1,L1
//	...
//	SPECN  AN,FN,VN,ON,LN
//
// Data Altered:
//
//	CSTACK      A+(S*N)
//	A+D         A1,F1,V1,O1,L1
//	...
//	A+D+S*N-S   AN,FN,VN,ON,LN
//
// Programming Notes:
//  1. If A+(S*N) > STACK+STSIZE, stack overflow occurs. Transfer
//     should be made to the program location OVER, which will result
//     in an appropriate error termination.
//  2. See also PUSH, POP, and SPOP.
//
// Note 1 writes the top of the stack as STACK+STSIZE, which is
// STACK+STSIZE*DESCR in address units; see PUSH.
//
// S4D58.PDF: 6.113
func (s *VM) SPUSH(specs []int) error {
	top := s.CStack + len(specs)*s.Spec
	if limit, ok := s.stackTop(); ok && top > limit {
		return s.overflow("SPUSH: %d specifiers would take the stack to %d, past %d", len(specs), top, limit)
	}
	if !s.inCore(top) {
		return s.overflow("SPUSH: %d specifiers would take the stack to %d, outside core", len(specs), top)
	}
	for i, spec := range specs {
		addr, flag, value, offset, length := s.Specifier(spec)
		s.putSpecifier(s.CStack+s.Descr+i*s.Spec, addr, flag, value, offset, length)
	}
	s.CStack = top
	return nil
}

// SPOP (pop specifier from stack) is used to pop a list of specifiers
// from the system stack.
//
// Data Input:
//
//	CSTACK      A
//	A+D-S       A1,F1,V1,O1,L1
//	...
//	A+D-(N*S)   AN,FN,VN,ON,LN
//
// Data Altered:
//
//	CSTACK A-(N*S)
//	SPEC1  A1,F1,V1,O1,L1
//	...
//	SPECN  AN,FN,VN,ON,LN
//
// Programming Notes:
//  1. If A-(N*S) < STACK, stack underflow occurs. This condition
//     indicates a programming error in the implementation of the macro
//     language. An appropriate error termination for this error may be
//     obtained by transferring to the program location INTR10 if the
//     condition is detected.
//  2. See also POP, SPUSH, and PUSH.
//
// The first specifier in the list comes off the top, as in POP: the
// topmost specifier occupies A+D-S through A+D-1, so its second half
// is the cell CSTACK addresses.
//
// S4D58.PDF: 6.111
func (s *VM) SPOP(specs []int) error {
	bottom, ok := s.Symbols["STACK"]
	if !ok {
		return s.fault("SPOP: the program symbol STACK is not defined")
	}
	// Note 1's condition exactly: A-(N*S) < STACK.
	if s.CStack-len(specs)*s.Spec < bottom {
		return s.interrupt("SPOP: stack underflow popping %d specifiers from %d", len(specs), s.CStack)
	}
	for i, spec := range specs {
		at := s.CStack + s.Descr - (i+1)*s.Spec
		addr, flag, value, offset, length := s.Specifier(at)
		s.putSpecifier(spec, addr, flag, value, offset, length)
	}
	s.CStack -= len(specs) * s.Spec
	return nil
}

// BRANIC (branch indirect with offset constant) is used to alter the
// flow of program control by branching indirectly to the operation at
// LOC.
//
// Data Input:
//
//	DESCR A
//	A+N   LOC
//
// Programming Notes:
//  1. N is always zero.
//
// Note 1 is a statement about the SNOBOL4 source rather than a
// restriction on the operation -- 6.16's box gives N as an operand
// like any other -- so N is added rather than asserted to be zero.
// The arithmetic is the same either way, and faulting on an operand
// the document allows would be a restriction this machine invented.
//
// S4D58.PDF: 6.16
func (s *VM) BRANIC(descr, n int) error {
	at := s.Core[descr].A + n
	if !s.inCore(at) {
		return s.fault("BRANIC: %d is outside core", at)
	}
	s.PC = s.Core[at].A
	return nil
}

// SELBRA (select branch point) is used to alter the flow of program
// control by selecting a location from a list and branching to it.
// Transfer is to LOCI corresponding to I.
//
// Data Input:
//
//	DESCR I
//
// Programming Notes:
//  1. Any of the locations may be omitted. As in the case of
//     operations with omitted conditional branches, control then
//     passes to the operation following SELBRA.
//  2. If I = N+1, control is passed to the operation following SELBRA.
//  3. I is always in the range 1 <= I <= N+1. For debugging purposes,
//     it may be useful to verify that I is within this range.
//
// The whole of notes 1 and 2 is arithmetic, because the assembler
// emits the locations as N BRANCH instructions following the SELBRA
// and resolves an omitted one to the operation after the last of them
// (6.98 note 1). Setting the counter to the SELBRA's own address plus
// I lands on a branch for I <= N and on the following operation for
// I = N+1, with no case to write for either note. RRTURN does the same
// thing with the same emitter; see 6.87.
//
// Step has already advanced the counter past the SELBRA cell, so the
// SELBRA's own address is one back. N is the second operand: the
// assembler puts it there because a BRANCH belonging to a SELBRA is
// indistinguishable from any other, so note 3's check cannot be made
// from core alone.
//
// S4D58.PDF: 6.98
func (s *VM) SELBRA(descr, n int) error {
	i := s.Core[descr].A
	if i < 1 || i > n+1 {
		return s.fault("SELBRA: %d is outside the range 1 to %d that 6.98 note 3 requires", i, n+1)
	}
	s.PC += i - 1
	return nil
}
