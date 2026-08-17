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

// Package fortran interprets the FORTRAN IV formats that the SNOBOL4
// system prints under.
//
// S4D58 does not define them. 6.114 note 1 says only that "the format
// C11...C1M is a FORTRAN IV format in ”'undigested”' form", 6.34
// says the same of what FORMAT assembles, and 2.1 explains why the
// machine must not look inside one: "Formats used by STPRNT are
// strings that may be formed during program execution and hence must
// be accepted in their undigested form." So the machine carries the
// characters across the host boundary untouched, and this package is
// what the far side does with them. It is the one part of the system
// whose specification is somewhere else -- ANSI X3.9-1966, section 7,
// and the FORTRAN IV manual of whatever machine the system was being
// moved to.
//
// # What it has to handle
//
// The SNOBOL4 source has twenty-six FORMAT statements and two format
// strings, and between them they use six things: Hollerith literals
// (nH), integers (Iw), reals (Fw.d), characters (Aw), blanks (nX) and
// the record separator (/). Repeat counts and parenthesised groups
// come with them. Ew.d and quoted literals are here too, since a
// format may also be built by a SNOBOL4 program at run time (2.1) and
// those are the rest of what a program is likely to write; anything
// else is an error that names itself rather than a field quietly
// printed wrong.
//
// # Numbers arrive as bits
//
// 6.75 gives OUTPUT "the conversion of integers and real numbers given
// in the address fields A1,...,AN", and 3.1.1 puts both in the same
// field: "the address field must also be large enough to contain any
// integer or real number (including sign)". So which one a value is
// cannot be read off the value -- it is the format that says, I or F
// -- and Numbers takes the address fields as they stand and lets each
// descriptor decide. See Value.
//
// # Carriage control
//
// A record's first character is not printed. It says how far to
// advance first, and the four values are a blank for one line, 0 for
// two, 1 for a new page and + for none at all. That is a property of
// the printer rather than of the format, so it is applied by Lines
// rather than by Numbers and Chars, and only for a unit that is a
// printer: the SNOBOL4 system prints PUNCH output under (80A1) on unit
// 7, where the first character is a character like any other.
package fortran

import (
	"fmt"
	"math"
)

// A Record is one output record: the characters one pass of the format
// produced, before a printer has taken the first of them as carriage
// control.
type Record string

// Control is the carriage control character of a record, which a
// printer acts on and does not print. An empty record advances one
// line, like a blank.
func (r Record) Control() byte {
	if r == "" {
		return ' '
	}
	return r[0]
}

// Text is what a printer puts on the page: the record without its
// carriage control character.
func (r Record) Text() string {
	if r == "" {
		return ""
	}
	return string(r[1:])
}

// A Value is one number from an output list, as it sits in the address
// field of a descriptor. Whether it is an integer or a real number is
// the format's business and not the value's; see the package comment.
type Value int

// Int reads the value as a signed integer, which is what Iw wants.
func (v Value) Int() int { return int(v) }

// Real reads the value as a real number, which is what Fw.d and Ew.d
// want. The machine holds a real in the address field as its IEEE 754
// bit pattern; see pkg/sil/reals.go.
func (v Value) Real() float64 { return math.Float64frombits(uint64(v)) }

// Numbers formats a list of values under a format, as OUTPUT does
// (6.75). The values are address fields; each descriptor decides how
// to read the one it takes.
func Numbers(format []byte, values []int) ([]Record, error) {
	items, err := parse(format)
	if err != nil {
		return nil, err
	}
	return emit(items, &numberList{values: values})
}

// Chars formats the characters of a string under a format, as STPRNT
// does (6.114).
//
// The list is the characters, one item each, which is what the SNOBOL4
// system's own formats expect: (1X,132A1) is a blank and then 132
// one-character fields. A wider Aw takes w characters from the same
// stream, which is the same thing said for w = 1 and the only reading
// that makes sense of a list that is a string rather than an array of
// variables.
func Chars(format []byte, s []byte) ([]Record, error) {
	items, err := parse(format)
	if err != nil {
		return nil, err
	}
	return emit(items, &charList{chars: s})
}

// Lines renders records for a device that advances a line at a time,
// applying carriage control.
//
// A new page is a form feed at the head of the line, which is what a
// printer would have been sent. Overprinting has no equivalent -- two
// records on one line is what a printer does with ink and a stream of
// text cannot -- so the record is given a line of its own, which is
// what the one place the SNOBOL4 system uses it wants anyway: the
// banner underlines SNOBOL4 by overprinting seven underscores.
func Lines(records []Record) []string {
	var out []string
	for _, r := range records {
		switch r.Control() {
		case '0': // two lines: one blank, then the text
			out = append(out, "", r.Text())
		case '1': // a new page
			out = append(out, "\f"+r.Text())
		default: // a blank advances one line, and + cannot be done
			out = append(out, r.Text())
		}
	}
	return out
}

// A list is what the format takes its data from. It is the characters
// of a string for STPRNT and the address fields of descriptors for
// OUTPUT, and a descriptor that asks for the wrong one of those is an
// error rather than a guess.
type list interface {
	// left is how many items have not been transmitted. Format
	// control stops when a data descriptor is reached and this is
	// zero.
	left() int
	number() (Value, error)
	char() (byte, error)
}

type numberList struct {
	values []int
	at     int
}

func (l *numberList) left() int { return len(l.values) - l.at }

func (l *numberList) number() (Value, error) {
	v := l.values[l.at]
	l.at++
	return Value(v), nil
}

func (l *numberList) char() (byte, error) {
	return 0, fmt.Errorf("A: the output list is numbers, not characters")
}

type charList struct {
	chars []byte
	at    int
}

func (l *charList) left() int { return len(l.chars) - l.at }

func (l *charList) number() (Value, error) {
	return 0, fmt.Errorf("the output list is a string, so it has no number for this field")
}

func (l *charList) char() (byte, error) {
	c := l.chars[l.at]
	l.at++
	return c, nil
}
