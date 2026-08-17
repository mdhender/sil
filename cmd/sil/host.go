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

package main

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/mdhender/sil/pkg/fortran"
)

// host is what the machine asks the outside world for (S4D58 2.1).
//
// One input stream and two output streams. It reads the unit reference
// number for one thing only: whether the unit is the printer, which is
// what decides if the first character of a record is carriage control
// or a character. 2.1 gives the SNOBOL4 system three units -- input,
// print output and punch output -- and a command-line runner has one
// place for each direction, so which file a unit is is not a thing
// this host distinguishes. A host that wanted files per unit would
// implement the same interface.
//
// Both output operations hand over a FORTRAN IV format, which
// pkg/fortran reads. This is where 2.1 puts that: "Formats used by
// STPRNT are strings that may be formed during program execution and
// hence must be accepted in their undigested form", so the machine
// passes them across untouched and the host is what digests them.
type host struct {
	out    io.Writer // what the program printed
	system io.Writer // what the SNOBOL4 system printed about itself
	in     io.Reader

	printer int // the unit that carriage control applies to
	deck    *bufio.Reader
	started time.Time
}

// Print writes what a program printed, and what the compiler listed.
// The string goes under the format, which for the SNOBOL4 system's own
// two is (1X,132A1) on the printer and (80A1) on the punch (6.114).
func (h *host) Print(unit int, format, s []byte) (int, error) {
	records, err := fortran.Chars(format, s)
	if err != nil {
		return 0, fmt.Errorf("STPRNT on unit %d: %w", unit, err)
	}
	if err := h.write(h.out, unit, records); err != nil {
		return 0, err
	}
	// 6.114 note 3: the condition the output routine signals is not
	// used.
	return 0, nil
}

// Output writes what the SNOBOL4 system says about itself: its banner,
// whether the program compiled, and the statistics (6.75).
func (h *host) Output(unit int, format []byte, values []int) error {
	records, err := fortran.Numbers(format, values)
	if err != nil {
		return fmt.Errorf("OUTPUT on unit %d: %w", unit, err)
	}
	return h.write(h.system, unit, records)
}

// write puts records on a stream, taking the first character of each
// as carriage control when the unit is the printer and as a character
// when it is not. The punch has no carriage; the SNOBOL4 PUNCH
// variable goes out under (80A1), where every column is data.
func (h *host) write(w io.Writer, unit int, records []fortran.Record) error {
	if unit != h.printer {
		for _, r := range records {
			if err := line(w, []byte(r)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, text := range fortran.Lines(records) {
		if err := line(w, []byte(text)); err != nil {
			return err
		}
	}
	return nil
}

// Read hands over one line of the deck, truncated to what STREAD asked
// for. A record shorter than that is padded with blanks by the machine
// (6.115 note 1), so a short line is a short card and not an error.
func (h *host) Read(unit, n int) ([]byte, bool, error) {
	if h.deck == nil {
		h.deck = bufio.NewReader(h.in)
	}
	text, err := h.deck.ReadBytes('\n')
	if len(text) == 0 && err == io.EOF {
		return nil, true, nil
	}
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	text = trimNewline(text)
	if len(text) > n {
		text = text[:n]
	}
	return text, false, nil
}

// A stream of lines cannot be positioned, and 6.14, 6.30 and 6.92 have
// no failure branch, so failing here would stop the machine over
// something the SNOBOL4 functions BACKSPACE, ENDFILE and REWIND are
// entitled to ask for and not entitled to be told about.
func (h *host) Backspace(unit int) error { return nil }
func (h *host) EndFile(unit int) error   { return nil }
func (h *host) Rewind(unit int) error    { return nil }

// Time is milliseconds since the run started. 6.71 note 1: "The origin
// with respect to which the time is obtained is not important. The
// SNOBOL4 system deals only with differences in times."
func (h *host) Time() int {
	if h.started.IsZero() {
		h.started = time.Now()
	}
	return int(time.Since(h.started).Milliseconds())
}

// Date is the current date. 6.22 note 1 says the representation does
// not matter and gives 04/01/81 as one of four acceptable ones.
func (h *host) Date() []byte {
	return []byte(time.Now().Format("01/02/06"))
}

func line(w io.Writer, s []byte) error {
	_, err := w.Write(append(append(make([]byte, 0, len(s)+1), s...), '\n'))
	return err
}

func trimNewline(s []byte) []byte {
	s, _ = cutSuffix(s, '\n')
	s, _ = cutSuffix(s, '\r')
	return s
}

func cutSuffix(s []byte, c byte) ([]byte, bool) {
	if n := len(s) - 1; n >= 0 && s[n] == c {
		return s[:n], true
	}
	return s, false
}
