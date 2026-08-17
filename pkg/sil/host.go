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

import "io"

// Host is what the machine asks the outside world for.
//
// S4D58 2.1 says SNOBOL4 does all its input and output through FORTRAN
// IV routines and refers to files by FORTRAN unit reference numbers,
// so the boundary is drawn where the document draws it: the machine
// resolves a unit number and hands over bytes, and whatever is on the
// other side decides what a unit is.
//
// It grows one operation at a time, as an operation that needs it is
// implemented. Today that is STPRNT.
type Host interface {
	// Print writes a string to a unit under a format (6.114, and 6.75
	// for OUTPUT).
	//
	// The format is passed through undigested. 2.1: "Formats used by
	// STPRNT are strings that may be formed during program execution
	// and hence must be accepted in their undigested form" -- so a
	// FORTRAN IV format interpreter belongs on this side of the
	// boundary, and until there is one a host may ignore the format.
	// The machine's job is to find the bytes, not to read them.
	//
	// It returns the condition the output routine signals, which
	// STPRNT puts in the address field of its first operand. 6.114
	// note 3 says that condition is not used.
	Print(unit int, format, s []byte) (condition int, err error)
}

// WriterHost is a Host that sends every unit to one writer and ignores
// the format, which is enough to see what a program prints.
type WriterHost struct {
	W io.Writer
}

// Print writes s and a newline, ignoring the unit and the format.
func (h WriterHost) Print(unit int, format, s []byte) (int, error) {
	if _, err := h.W.Write(append(append([]byte{}, s...), '\n')); err != nil {
		return 0, err
	}
	return 0, nil
}
