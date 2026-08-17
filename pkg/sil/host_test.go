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

import "errors"

// recorder is a Host that remembers what it was asked to do and gives
// back whatever it was told to. Every field the machine reads is
// settable, so a test can put the host into the state the operation
// under test is supposed to react to -- an end of file, a reading
// error, a clock, a calendar -- without any of it depending on the
// machine this runs on.
type recorder struct {
	unit      int
	format    []byte
	text      []byte
	values    []int
	condition int

	record []byte // what Read gives back
	asked  int    // and how many characters it was asked for
	atEOF  bool
	failed bool // Read reports a reading error
	moved  []string
	broken bool // the positioning operations fail
	time   int
	date   []byte
}

func (r *recorder) Print(unit int, format, s []byte) (int, error) {
	r.unit, r.format, r.text = unit, format, s
	return r.condition, nil
}

func (r *recorder) Output(unit int, format []byte, values []int) error {
	r.unit, r.format, r.values = unit, format, values
	return nil
}

func (r *recorder) Read(unit, n int) ([]byte, bool, error) {
	r.unit, r.asked = unit, n
	switch {
	case r.failed:
		return nil, false, errors.New("a reading error")
	case r.atEOF:
		return nil, true, nil
	}
	return r.record, false, nil
}

func (r *recorder) Backspace(unit int) error { return r.move("BACKSPACE", unit) }
func (r *recorder) EndFile(unit int) error   { return r.move("ENDFILE", unit) }
func (r *recorder) Rewind(unit int) error    { return r.move("REWIND", unit) }

func (r *recorder) move(what string, unit int) error {
	r.unit = unit
	if r.broken {
		return errors.New("the unit cannot be positioned")
	}
	r.moved = append(r.moved, what)
	return nil
}

func (r *recorder) Time() int    { return r.time }
func (r *recorder) Date() []byte { return r.date }
