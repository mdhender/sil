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

// S4D58 7.5's input/output group, less STPRNT, which is in macros.go
// with the rest of the vertical slice.
//
// Every one of them takes a unit reference number out of an address
// field and hands it to the Host. 2.1 is what draws the boundary
// there: SNOBOL4 does its input and output through FORTRAN IV routines
// and names files by unit reference number, so the machine resolves
// the number and the host decides what a unit is.

// OUTPUT (output record) is used to output a list of items according
// to FORMAT. The output is put on the file associated with unit
// reference number I. The format C1...CL may specify literals and the
// conversion of integers and real numbers given in the address fields
// A1,...,AN.
//
// Data Input:
//
//	DESCR  I
//	FORMAT C1...CL
//	DESCR1 A1
//	...
//	DESCRN AN
//
// Programming Notes:
//  1. See also STPRNT.
//
// # Where L comes from
//
// 6.75's figure gives the format as characters at a location and never
// says how many, because FORMAT (6.34) assembles exactly the literal
// it was written with and a FORTRAN IV routine reads to the closing
// parenthesis. This machine does not read formats at all, so the
// assembler passes the count alongside the address: it is the one
// thing about a FORMAT operand that only the assembler knows.
//
// Which of A1...AN are integers and which are real numbers is the
// format's business, not the descriptors' -- a FORTRAN format says I
// or F per field -- so the address fields go across as they stand.
//
// S4D58.PDF: 6.75
func (s *VM) OUTPUT(descr, format, length int, descrs []int) error {
	if !s.charsInCore(format, length) {
		return s.fault("OUTPUT: the format of %d characters at %d is outside core", length, format)
	}
	values := make([]int, len(descrs))
	for i, d := range descrs {
		values[i] = s.Core[d].A
	}
	unit := s.Core[descr].A
	if err := s.Host.Output(unit, s.Chars(format, length), values); err != nil {
		return s.fault("OUTPUT: unit %d: %v", unit, err)
	}
	return nil
}

// STREAD (string read) is used to read a string. The string C1...CL is
// read from the file associated with unit reference number I. If an
// end-of-file is encountered, transfer is to EOF. If a reading error
// occurs, transfer is to ERROR. Otherwise transfer is to SLOC.
//
// Data Input:
//
//	DESCR I
//	SPEC  A,O,L
//
// Data Altered:
//
//	A+O C1...CL
//
// Programming Notes:
//  1. Note that the length of the string to be read is specified by
//     the data provided to STREAD. If the record read is not of length
//     L, FORTRAN IV conventions regarding truncation or reading of
//     additional records should be followed.
//  2. See also STPRNT.
//
// Note 1 is why L goes to the host rather than the host deciding how
// much to give: a record longer than L is truncated, and one shorter
// is padded with blanks, which is what a FORTRAN IV A-format read of L
// characters does. A host that would rather read additional records to
// fill L may return them joined; it is given L for that reason.
//
// A reading error is what the host reports as an error, and it is not
// a fault: 6.115 gives it a branch of its own.
//
// S4D58.PDF: 6.115
func (s *VM) STREAD(spec, descr, eof, errloc, sloc int) error {
	addr, _, _, offset, length := s.Specifier(spec)
	if !s.charsInCore(addr+offset, length) {
		return s.fault("STREAD: %d characters at %d are outside core", length, addr+offset)
	}
	unit := s.Core[descr].A

	text, atEOF, err := s.Host.Read(unit, length)
	if err != nil {
		s.PC = errloc
		return nil
	}
	if atEOF {
		s.PC = eof
		return nil
	}

	record := make([]byte, length)
	for i := range record {
		if i < len(text) {
			record[i] = text[i]
		} else {
			record[i] = blank
		}
	}
	s.putChars(addr+offset, record)
	s.PC = sloc
	return nil
}

// BKSPCE (backspace record) is used to back space one record on the
// file associated with unit reference number I.
//
// Data Input:
//
//	DESCR I
//
// Programming Notes:
//  1. See also ENFILE and REWIND.
//  2. Refer to Section 2.1 for a discussion of unit reference numbers.
//
// S4D58.PDF: 6.14
func (s *VM) BKSPCE(descr int) error {
	return s.position("BKSPCE", s.Core[descr].A, s.Host.Backspace)
}

// ENFILE (write end of file) is used to write an end-of-file on
// (close) the file associated with unit reference number I.
//
// Data Input:
//
//	DESCR I
//
// Programming Notes:
//  1. See also BKSPCE and REWIND.
//  2. Refer to Section 2.1 for a discussion of unit reference numbers.
//
// S4D58.PDF: 6.30
func (s *VM) ENFILE(descr int) error {
	return s.position("ENFILE", s.Core[descr].A, s.Host.EndFile)
}

// REWIND (rewind file) is used to rewind the file associated with the
// unit reference number I.
//
// Data Input:
//
//	DESCR I
//
// Programming Notes:
//  1. Refer to Section 2.1 for a discussion of unit reference numbers.
//  2. See also BKSPCE and ENFILE.
//
// S4D58.PDF: 6.92
func (s *VM) REWIND(descr int) error {
	return s.position("REWIND", s.Core[descr].A, s.Host.Rewind)
}

// position is the shape all three of 6.14, 6.30 and 6.92 have: a unit
// number out of an address field, nothing altered, and no branch, so
// a host that cannot do it stops the machine. The SNOBOL4 procedures
// that call them -- BKSPCE, ENFILE and REWIND at lines 3823 to 3845 --
// have no failure exit to offer either.
func (s *VM) position(of string, unit int, do func(int) error) error {
	if err := do(unit); err != nil {
		return s.fault("%s: unit %d: %v", of, unit, err)
	}
	return nil
}
