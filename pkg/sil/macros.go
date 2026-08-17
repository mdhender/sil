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

// The operations, one method each, named exactly as the SIL mnemonic.
//
// Every operand is an address in core -- the address of a descriptor,
// of a specifier, or of an instruction -- or a constant the assembler
// resolved. A branch point is always a real address: the assembler
// turns an omitted one into the address of the next operation (S4D58
// 5.2), so no method has to know what "omitted" means.

// ACOMP (address comparison) is used to compare the address fields
// of two descriptors. The comparison is arithmetic with A1 and A2
// being considered as signed integers.
//
//	If A1 > A2, transfer is to GTLOC.
//	If A1 = A2, transfer is to EQLOC.
//	If A1 < A2, transfer is to LTLOC.
//
// Data Input:
//
//	DESCR1 A1
//	DESCR2 A2
//
// Programming Notes:
//  1. A1 and A2 may be relocatable addresses.
//  2. See also LCOMP, ACOMPC, AEQL, AEQLC, and AEQLIC.
//
// S4D58.PDF: 6.1
func (s *VM) ACOMP(descr1, descr2, gtloc, eqloc, ltloc int) {
	a1, a2 := s.Core[descr1].A, s.Core[descr2].A
	switch {
	case a1 > a2:
		s.PC = gtloc
	case a1 == a2:
		s.PC = eqloc
	default:
		s.PC = ltloc
	}
}

// ACOMPC (address comparison with constant) is used to compare the
// address field of a descriptor to a constant. The comparison is
// arithmetic with A being considered as a signed integer.
//
//	If A > N, transfer is to GTLOC.
//	If A = N, transfer is to EQLOC.
//	If A < N, transfer is to LTLOC.
//
// Data Input:
//
//	DESCR A
//
// Programming Notes:
//  1. A may be a relocatable address.
//  2. N is never negative.
//  3. N is often 0.
//  4. See also ACOMP, AEQL, AEQLC, and AEQLIC.
//
// S4D58.PDF: 6.2
func (s *VM) ACOMPC(descr, n, gtloc, eqloc, ltloc int) {
	a := s.Core[descr].A
	switch {
	case a > n:
		s.PC = gtloc
	case a == n:
		s.PC = eqloc
	default:
		s.PC = ltloc
	}
}

// BRANCH (branch to program location) is used to alter the flow of
// program control by branching to LOC. If PROC is given, it is the
// procedure in which LOC occurs. If PROC is omitted, LOC is in the
// current procedure.
//
// Programming Notes:
//  1. See also PROC.
//
// PROC reaches the machine as nothing at all. It exists for machines
// with a limited program basing range, and this one has none: the
// assembler checks that the name it gives is a procedure entry point
// and then discards it, so BRANCH takes one operand here.
//
// S4D58.PDF: 6.15
func (s *VM) BRANCH(loc int) { s.PC = loc }

// ENDEX (end execution of SNOBOL4 run) is used to terminate execution
// of a SNOBOL4 run. ENDEX is the last instruction executed and is
// responsible for returning properly to the environment that initiated
// the SNOBOL4 run. If I is nonzero, a post-mortem dump of user core
// should be given.
//
// Data Input:
//
//	DESCR I
//
// Programming Notes:
//  1. If a dump is not given, the keyword &ABEND will not have its
//     specified effect.
//  2. See also INIT.
//
// The dump is not given. I is kept in Status, so a caller that wants
// one can walk Core itself, which is the whole point of core doubling
// as its own listing.
//
// S4D58.PDF: 6.29
func (s *VM) ENDEX(descr int) {
	s.Status = s.Core[descr].A
	s.Halted = true
}

// INIT (initialize SNOBOL4 run) is used to initialize a SNOBOL4 run.
// INIT is the first instruction executed and is responsible for
// performing any initialization necessary. The operation is machine
// and system dependent.
//
// In addition to any initialization required for a particular system
// and machine, INIT also performs the following initialization for the
// SNOBOL4 system. Dynamic storage is initialized. The address fields
// of FRSGPT and HDSGPT are set to point to the first descriptor in
// dynamic storage. The address field of TLSGP1 is set to the first
// descriptor past the end of dynamic storage. Space for dynamic
// storage may be preallocated or obtained from the operating system by
// INIT. The timer is initialized for subsequent use by the MSTIME
// macro (q.v.).
//
// Programming Notes:
//  1. See also ENDEX.
//
// Incomplete, and deliberately so: this machine has no dynamic storage
// yet, so FRSGPT, HDSGPT and TLSGP1 are not set, and there is no timer
// because MSTIME is not implemented. A program that uses dynamic
// storage will fault on the first operation that reads one of those
// three rather than run against a region that was never allocated.
//
// S4D58.PDF: 6.46
func (s *VM) INIT() {}

// ISTACK (initialize stack) is used to initialize the system stack.
//
// Data Altered:
//
//	OSTACK 0
//	CSTACK STACK
//
// Programming Notes:
//  1. STACK is a program symbol whose value is the address of the
//     first descriptor of the system stack.
//
// S4D58.PDF: 6.50
func (s *VM) ISTACK() error {
	stack, ok := s.Symbols["STACK"]
	if !ok {
		return s.fault("ISTACK: the program symbol STACK is not defined (6.50 note 1)")
	}
	s.OStack = 0
	s.CStack = stack
	return nil
}

// MOVD (move descriptor) is used to move (copy) a descriptor from one
// location to another.
//
// Data Input:
//
//	DESCR2 A,F,V
//
// Data Altered:
//
//	DESCR1 A,F,V
//
// Programming Notes:
//  1. See also MOVA and MOVV.
//
// The character a cell may hold is not part of a descriptor, so it is
// not moved; a descriptor and a character never occupy the same cell
// in an assembled program.
//
// S4D58.PDF: 6.67
func (s *VM) MOVD(descr1, descr2 int) {
	src := s.Core[descr2]
	dst := &s.Core[descr1]
	dst.A, dst.F, dst.V = src.A, src.F, src.V
}

// POP (pop descriptors from stack) is used to pop a list of
// descriptors off the system stack.
//
// Data Input:
//
//	CSTACK    A
//	A         A1,F1,V1
//	...
//	A-D*(N-1) AN,FN,VN
//
// Data Altered:
//
//	CSTACK A-(N*D)
//	DESCR1 A1,F1,V1
//	...
//	DESCRN AN,FN,VN
//
// Programming Notes:
//  1. If A-(N*D) < STACK, stack underflow occurs. This condition
//     indicates a programming error in the implementation of the macro
//     language. An appropriate diagnostic message indicating an error
//     may be obtained by transferring to the program location INTR10
//     if the condition is detected.
//
// The stack grows upward and CSTACK addresses the descriptor on top,
// so the first descriptor in the list gets the top of the stack and
// each one after it gets the descriptor below.
//
// S4D58.PDF: 6.77
func (s *VM) POP(descrs []int) error {
	bottom, ok := s.Symbols["STACK"]
	if !ok {
		return s.fault("POP: the program symbol STACK is not defined")
	}
	// Note 1's condition exactly: A-(N*D) < STACK.
	if s.CStack-len(descrs)*s.Descr < bottom {
		return s.interrupt("POP: stack underflow popping %d descriptors from %d", len(descrs), s.CStack)
	}
	for i, descr := range descrs {
		s.Core[descr] = descriptorAt(s.Core, s.CStack-i*s.Descr)
	}
	s.CStack -= len(descrs) * s.Descr
	return nil
}

// RCALL (recursive call) is used to call a procedure recursively.
//
// Data Input:
//
//	CSTACK A
//	OSTACK A0
//	DESCR1 A1,F1,V1
//	...
//	DESCRN AN,FN,VN
//
// Data Altered:
//
//	A+D        A0,0,0
//	A+2D       LOC,0,0
//	A+3D       AN,FN,VN
//	...
//	A+D*(2+N)  A1,F1,V1
//	CSTACK     A+(2+N)*D
//	OSTACK     A
//
// Return Code at LOC:
//
//	LOC   OP      DESCR1
//	      BRANCH  LOC1
//	      ...
//	      BRANCH  LOCM
//
// Programming Notes:
//  1. RCALL and RRTURN are used in combination, and their relation to
//     each other must be thoroughly understood in order to implement
//     them correctly.
//  2. Ordinarily OP is an instruction to store the value returned by
//     RRTURN.
//  3. DESCR sometimes is omitted. In this case, any value returned by
//     RRTURN is ignored and OP should perform no operation.
//  4. (DESCR1,...,DESCRN) sometimes is entirely omitted. In this case
//     N should be taken to be zero.
//  5. Any of the locations LOC1,...,LOCM may be omitted. As in the
//     case of operations with omitted conditional branches, control
//     then passes to the operation following the RCALL.
//  6. The return indicated by RRTURN may be M+1, in which case control
//     is passed to the operation following the RCALL.
//  7. The return indicated by RRTURN is never greater than M+1.
//  8. RCALL typically must save program state information. ... The
//     flag fields of descriptors used to save program state
//     information must be set to zero.
//  9. See also SELBRA.
//
// Note 8 is why the two saved cells are written with their flag and
// value fields zeroed: the garbage collector walks the stack as
// descriptors and must not find a flag it would act on.
//
// The whole of the return dispatch is note 6's arithmetic. LOC is the
// cell after the RCALL, the M branch vector cells follow it, and
// RRTURN sets the program counter to LOC+N. N = M+1 therefore lands on
// the cell after the vector, which is the operation following the
// RCALL, with no special case anywhere. The machine never has to know
// M.
//
// S4D58.PDF: 6.87
func (s *VM) RCALL(descr, proc int, args []int) error {
	a := s.CStack
	loc := s.PC // Step has already advanced past the RCALL cell.
	top := a + (2+len(args))*s.Descr
	if !s.inCore(top) {
		return s.interrupt("RCALL: stack overflow at %d", top)
	}

	s.Core[a+s.Descr] = Cell{Kind: Data, A: s.OStack, Src: s.Core[a+s.Descr].Src}
	s.Core[a+2*s.Descr] = Cell{Kind: Data, A: loc, Src: s.Core[a+2*s.Descr].Src}
	// Arguments go on in reverse, so that the first POP in the
	// procedure gets the first argument.
	for i, arg := range args {
		at := a + (2+len(args)-i)*s.Descr
		s.Core[at] = descriptorAt(s.Core, arg)
	}
	s.CStack, s.OStack = top, a

	// The return cell holds the descriptor the value comes back in.
	// It is assembly-time data -- note 3's omitted DESCR is a zero
	// there -- so RCALL only checks that the assembler laid it down.
	if !s.inCore(loc) || s.Core[loc].Kind != Return {
		return s.fault("RCALL: %d is not a return point", loc)
	}
	if s.Core[loc].A != descr {
		return s.fault("RCALL: the return point at %d says %d, want %d", loc, s.Core[loc].A, descr)
	}
	s.PC = proc
	return nil
}

// RRTURN (recursive return) is used to return from a recursive call.
// DESCR is the descriptor whose value is returned. The stack pointers
// are repositioned as shown. At the location LOC, code similar to that
// shown is assembled by the RCALL to which return is to be made. OP
// represents an instruction that is used by RRTURN to return the value
// of DESCR. Control is transferred to LOCN corresponding to N given in
// the RRTURN.
//
// Data Input:
//
//	OSTACK A
//	A+D    A0
//	A+2D   LOC
//	DESCR  A1,F1,V1
//
// Data Altered:
//
//	CSTACK A
//	OSTACK A0
//	DESCR1 A1,F1,V1
//
// Programming Notes:
//  1. RCALL and RRTURN are used in combination, and their relation to
//     each other must be thoroughly understood.
//  2. DESCR may be omitted. In this case, OP should not be executed.
//
// S4D58.PDF: 6.95
func (s *VM) RRTURN(descr, n int) error {
	a := s.OStack
	if !s.inCore(a+2*s.Descr) || a == 0 {
		return s.interrupt("RRTURN: no frame to return from, OSTACK is %d", a)
	}
	a0 := s.Core[a+s.Descr].A
	loc := s.Core[a+2*s.Descr].A
	if !s.inCore(loc) || s.Core[loc].Kind != Return {
		return s.fault("RRTURN: the frame at %d says it was called from %d, which is not a return point", a, loc)
	}

	// Note 2: an omitted DESCR left the return cell zero, and nothing
	// is stored.
	if into := s.Core[loc].A; into != 0 {
		s.Core[into] = descriptorAt(s.Core, descr)
	}
	s.CStack, s.OStack = a, a0
	s.PC = loc + n
	return nil
}

// SETAC (set address to constant) is used to set the address field of
// a descriptor to a constant.
//
// Data Altered:
//
//	DESCR N
//
// Programming Notes:
//  1. N may be a relocatable address.
//  2. N is often 0, 1, or D.
//  3. N is never negative.
//  4. See also SETVC, SETLC, and SETAV.
//
// S4D58.PDF: 6.99
func (s *VM) SETAC(descr, n int) { s.Core[descr].A = n }

// STPRNT (string print) is used to print a string. The string
// C11...C1L is printed on the file associated with unit reference
// number I. C21...C2M is the output format. J is an integer specifying
// a condition signaled by the output routine.
//
// Data Input:
//
//	DESCR2 A
//	A+D    I
//	A+2D   A2
//	A2     ...,M
//	A2+4D  C21...C2M
//	SPEC   A1,...,O1,L
//	A1+O1  C11...C1L
//
// Data Altered:
//
//	DESCR1 J
//
// Programming Notes:
//  1. The format C21...C2M is a FORTRAN IV format in "undigested"
//     form. See FORMAT.
//  2. Both C11...C1L and C21...C2M begin at descriptor boundaries.
//  3. The condition J set in the address field of DESCR1 is not used.
//  4. See also OUTPUT and STREAD.
//
// A2 is the title of a string structure, not a specifier. Two things
// say so: the figure puts M in the title's value field and the
// characters at A2+4D, and the SNOBOL4 source defines BCDFLD EQU
// 4*DESCR, "offset of string in string structure" (line 232), against
// 6.13's "there are 4 descriptors (including the title) in a string
// structure in addition to the string itself". The formats the system
// uses reach this shape through initialization: line 322 converts each
// statically assembled specifier with GENVAR and stores the resulting
// structure, which is also how a format formed at run time arrives
// (2.1: formats used by STPRNT "are strings that may be formed during
// program execution").
//
// S4D58.PDF: 6.114
func (s *VM) STPRNT(descr1, descr2, spec int) error {
	a := s.Core[descr2].A
	if !s.inCore(a + 2*s.Descr) {
		return s.fault("STPRNT: the print block at %d is outside core", a)
	}
	unit := s.Core[a+s.Descr].A
	title := s.Core[a+2*s.Descr].A

	length := s.Core[title].V
	chars := title + bcdField*s.Descr
	if !s.inCore(chars+length-1) || length < 0 {
		return s.fault("STPRNT: the format at %d claims %d characters", title, length)
	}
	format := s.Chars(chars, length)

	condition, err := s.Host.Print(unit, format, s.Text(spec))
	if err != nil {
		return s.fault("STPRNT: unit %d: %v", unit, err)
	}
	s.Core[descr1].A = condition
	return nil
}

// bcdField is the offset of the characters in a string structure, in
// descriptors. The SNOBOL4 source calls it BCDFLD and equates it to
// 4*DESCR (line 232); S4D58 6.13 explains the 4.
const bcdField = 4

// SUM (sum addresses) is used to add two address fields. A and I are
// considered as signed integers. If A+I is out of the range available
// for integers, transfer is to FLOC. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR2 A,F,V
//	DESCR3 I
//
// Data Altered:
//
//	DESCR1 A+I,F,V
//
// Programming Notes:
//  1. A may be a relocatable address.
//  2. See also SUBTRT.
//
// The range available for integers is SIZLIM, which PARMS chooses. A
// sum outside it takes FLOC.
//
// S4D58.PDF: 6.120
func (s *VM) SUM(descr1, descr2, descr3, floc, sloc int) {
	src := s.Core[descr2]
	sum := src.A + s.Core[descr3].A
	if limit, ok := s.Symbols["SIZLIM"]; ok && (sum > limit || sum < -limit) {
		s.PC = floc
		return
	}
	dst := &s.Core[descr1]
	dst.A, dst.F, dst.V = sum, src.F, src.V
	s.PC = sloc
}

// descriptorAt reads the three descriptor fields of a cell, leaving
// behind anything that is not part of a descriptor.
func descriptorAt(core []Cell, a int) Cell {
	c := core[a]
	return Cell{Kind: Data, Op: c.Op, A: c.A, F: c.F, V: c.V, Src: c.Src}
}
