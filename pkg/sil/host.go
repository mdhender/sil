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
	"bufio"
	"fmt"
	"io"
)

// Host is what the machine asks the outside world for.
//
// S4D58 2.1 says SNOBOL4 does all its input and output through FORTRAN
// IV routines and refers to files by FORTRAN unit reference numbers,
// so the boundary is drawn where the document draws it: the machine
// resolves a unit number and hands over bytes, and whatever is on the
// other side decides what a unit is.
//
// It grows one operation at a time, as an operation that needs it is
// implemented. Today that is 7.5's input/output group -- STPRNT,
// OUTPUT, STREAD, BKSPCE, ENFILE and REWIND -- and the two
// system-dependent operations that ask the machine's environment
// rather than a file, MSTIME and DATE.
//
// Nothing here interprets a FORTRAN IV format, and nothing here should.
// 2.1: "Formats used by STPRNT are strings that may be formed during
// program execution and hence must be accepted in their undigested
// form", so the machine's job is to find the bytes and a host's is to
// read them. pkg/fortran is what reads them; cmd/sil is a host that
// uses it, and WriterHost below is one that does not.
type Host interface {
	// Print writes a string to a unit under a format (6.114).
	//
	// It returns the condition the output routine signals, which
	// STPRNT puts in the address field of its first operand. 6.114
	// note 3 says that condition is not used.
	Print(unit int, format, s []byte) (condition int, err error)

	// Output writes a list of values to a unit under a format (6.75).
	// The values are the address fields of the descriptors the
	// operation was given; which of them are integers and which are
	// real numbers is what the format says, not what the descriptors
	// say.
	Output(unit int, format []byte, values []int) error

	// Read reads one record of up to n characters from a unit
	// (6.115). A record shorter than n is padded with blanks by the
	// machine, following the FORTRAN IV convention 6.115 note 1
	// points at; a host that wants to read additional records to fill
	// n may do so instead.
	Read(unit, n int) (s []byte, eof bool, err error)

	// Backspace moves back one record (6.14), EndFile writes an
	// end-of-file and closes (6.30), and Rewind rewinds (6.92). None
	// of the three has a failure branch, so an error from any of them
	// stops the machine.
	Backspace(unit int) error
	EndFile(unit int) error
	Rewind(unit int) error

	// Time is the millisecond time (6.71). 6.71 note 1 says the origin
	// does not matter, because the system uses only differences, and
	// note 4 says zero is an acceptable answer.
	Time() int

	// Date is a character representation of the current date (6.22).
	// 6.22 note 1 says the representation does not matter; note 4 says
	// no date at all is acceptable, and returning nothing is how a
	// host says so.
	Date() []byte
}

// WriterHost is a Host with one writer for every output unit and an
// optional reader for every input one. It is enough to see what a
// program prints and to feed one a few lines, and it is what the tests
// use.
//
// Everything it cannot do it does in the way S4D58 licenses: it
// ignores formats, has no clock (6.71 note 4) and no calendar (6.22
// note 4), and positioning a stream is not something a writer can do,
// so Backspace, EndFile and Rewind do nothing rather than fail.
type WriterHost struct {
	W io.Writer
	R io.Reader

	lines *bufio.Reader
}

// Print writes s and a newline, ignoring the unit and the format.
func (h *WriterHost) Print(unit int, format, s []byte) (int, error) {
	if _, err := h.W.Write(append(append([]byte{}, s...), '\n')); err != nil {
		return 0, err
	}
	return 0, nil
}

// Output writes the characters of the format and then the values, in
// decimal, separated by blanks. This is not a FORTRAN IV interpreter
// and does not pretend to be one -- pkg/fortran is, and cmd/sil uses
// it. This is legible, which is what a test needs from it.
func (h *WriterHost) Output(unit int, format []byte, values []int) error {
	line := append([]byte{}, format...)
	for _, v := range values {
		line = append(line, ' ')
		line = append(line, fmt.Appendf(nil, "%d", v)...)
	}
	_, err := h.W.Write(append(line, '\n'))
	return err
}

// Read returns one line, without its newline, truncated to n
// characters. With no reader there is nothing to read, so every unit
// is at end of file.
func (h *WriterHost) Read(unit, n int) ([]byte, bool, error) {
	if h.R == nil {
		return nil, true, nil
	}
	if h.lines == nil {
		h.lines = bufio.NewReader(h.R)
	}
	line, err := h.lines.ReadBytes('\n')
	if len(line) == 0 && err == io.EOF {
		return nil, true, nil
	}
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	if i := len(line) - 1; i >= 0 && line[i] == '\n' {
		line = line[:i]
	}
	if len(line) > n {
		line = line[:n]
	}
	return line, false, nil
}

func (h *WriterHost) Backspace(unit int) error { return nil }
func (h *WriterHost) EndFile(unit int) error   { return nil }
func (h *WriterHost) Rewind(unit int) error    { return nil }
func (h *WriterHost) Time() int                { return 0 }
func (h *WriterHost) Date() []byte             { return nil }
