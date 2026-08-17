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

// The operations that modify the flag field of a descriptor, S4D58
// 7.5's ninth group, and those that modify the value field, its
// eighth.

// SETF (set flag) is used to set (add) a flag in the flag field of
// DESCR.
//
// Data Input:
//
//	DESCR F
//
// Data Altered:
//
//	DESCR F+FLAG
//
// Programming Notes:
//  1. FLAG is added to the flags already present in F. The other flags
//     are left unchanged.
//  2. If F already contains FLAG, no data is altered.
//  3. See also SETFI.
//
// Note 1 is why this is a bitwise or rather than the addition the
// figure writes: adding a flag that is already there would carry into
// the next one. 3.1.2 calls the flag field "a set of bits that are
// individually tested, turned on, and turned off", and PARMS gives the
// five SNOBOL4 flags distinct bits.
//
// S4D58.PDF: 6.101
func (s *VM) SETF(descr, flag int) { s.Core[descr].F |= flag }

// SETFI (set flag indirect) is used to set (add) a flag in the flag
// field of a descriptor specified indirectly.
//
// Data Input:
//
//	DESCR A
//	A     F
//
// Data Altered:
//
//	A F+FLAG
//
// Programming Notes:
//  1. FLAG is added to the flags already present in F. The other flags
//     are left unchanged.
//  2. If F already contains FLAG, no data is altered.
//  3. See also SETF and RSETFI.
//
// S4D58.PDF: 6.102
func (s *VM) SETFI(descr, flag int) error {
	at := s.Core[descr].A
	if !s.inCore(at) {
		return s.fault("SETFI: %d is outside core", at)
	}
	s.Core[at].F |= flag
	return nil
}

// RESETF (reset flag) is used to reset (delete) a flag from a
// descriptor.
//
// Data Input:
//
//	DESCR F
//
// Data Altered:
//
//	DESCR F-FLAG
//
// Programming Notes:
//  1. Only FLAG is removed from the flags in F. Any other flags are
//     left unchanged.
//  2. If F does not contain FLAG, no data is altered.
//  3. See also RSETFI and SETFI.
//
// S4D58.PDF: 6.91
func (s *VM) RESETF(descr, flag int) { s.Core[descr].F &^= flag }

// RSETFI (reset flag indirect) is used to reset (delete) a flag from a
// descriptor that is specified indirectly.
//
// Data Input:
//
//	DESCR A
//	A     F
//
// Data Altered:
//
//	A F-FLAG
//
// Programming Notes:
//  1. Only FLAG is removed from the flags in F. Any other flags are
//     left unchanged.
//  2. If F does not contain FLAG, no data is altered.
//  3. See also RESETF and SETFI.
//
// S4D58.PDF: 6.96
func (s *VM) RSETFI(descr, flag int) error {
	at := s.Core[descr].A
	if !s.inCore(at) {
		return s.fault("RSETFI: %d is outside core", at)
	}
	s.Core[at].F &^= flag
	return nil
}

// INCRV (increment value field) is used to increment the value field
// of a descriptor. I is considered as an unsigned (nonnegative)
// integer.
//
// Data Input:
//
//	DESCR I
//
// Data Altered:
//
//	DESCR I+N
//
// Programming Notes:
//  1. N is always positive.
//  2. N is often 1.
//  3. See also INCRA.
//
// S4D58.PDF: 6.45
func (s *VM) INCRV(descr, n int) { s.Core[descr].V += n }

// MOVV (move value field) is used to move a value field from one
// descriptor to another.
//
// Data Input:
//
//	DESCR2 V
//
// Data Altered:
//
//	DESCR1 V
//
// Programming Notes:
//  1. See also MOVA and MOVD.
//
// S4D58.PDF: 6.69
func (s *VM) MOVV(descr1, descr2 int) { s.Core[descr1].V = s.Core[descr2].V }

// PUTVC (put value field with offset constant) is used to put a value
// field into a descriptor at a location with a constant offset.
//
// Data Input:
//
//	DESCR1 A
//	DESCR2 V
//
// Data Altered:
//
//	A+N V
//
// Programming Notes:
//  1. See also PUTAC, PUTDC, and PUTD.
//
// S4D58.PDF: 6.86
func (s *VM) PUTVC(descr1, n, descr2 int) error {
	at := s.Core[descr1].A + n
	if !s.inCore(at) {
		return s.fault("PUTVC: %d is outside core", at)
	}
	s.Core[at].V = s.Core[descr2].V
	return nil
}

// SETSIZ (set size) is used to set the size into the value field of a
// title descriptor.
//
// Data Input:
//
//	DESCR1 A
//	DESCR2 I
//
// Data Altered:
//
//	A I
//
// Programming Notes:
//  1. See also GETSIZ.
//
// S4D58.PDF: 6.104
func (s *VM) SETSIZ(descr1, descr2 int) error {
	at := s.Core[descr1].A
	if !s.inCore(at) {
		return s.fault("SETSIZ: %d is outside core", at)
	}
	s.Core[at].V = s.Core[descr2].A
	return nil
}

// SETVA (set value field from address) is used to set the value field
// of one descriptor from the address field of another.
//
// Data Input:
//
//	DESCR2 I
//
// Data Altered:
//
//	DESCR1 I
//
// Programming Notes:
//  1. See also SETAV.
//
// S4D58.PDF: 6.106
func (s *VM) SETVA(descr1, descr2 int) { s.Core[descr1].V = s.Core[descr2].A }

// SETVC (set value to constant) is used to set the value field of a
// descriptor to a constant.
//
// Data Altered:
//
//	DESCR N
//
// Programming Notes:
//  1. See also SETAC.
//
// S4D58.PDF: 6.107
func (s *VM) SETVC(descr, n int) { s.Core[descr].V = n }
