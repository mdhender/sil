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

// Package scanner splits SIL source into lines and each line into its
// fields.
//
// SIL source is a punched-card deck and its fields are columns, not
// whitespace-delimited tokens. S4D58 7.6 gives the format:
//
//	columns  1 - 6   label field
//	column   7       blank
//	columns  8 - 13  operation field
//	columns 14 - 15  blank
//	columns 16 - 71  variable (operand) field
//
// Columns are what make the source unambiguous. Fourteen names --
// ARRAY, BKSPCE, COPY, DATE, END, ENFILE, INIT, LOAD, LOCSP, OUTPUT,
// REMSP, REWIND, RPLACE, UNLOAD -- appear both as an operation and as
// a label, and DESCR and SPEC are simultaneously operations and
// symbols used in expressions (SPDR EQU SPEC+DESCR). A scanner that
// split on whitespace would confuse them; one that slices columns
// cannot.
//
// The scanner does not know what any operation means. It does not
// parse the operand field beyond finding where it ends.
package scanner

import (
	"strings"

	"github.com/mdhender/sil/pkg/sil/diag"
)

// Column positions from S4D58 7.6, one-based as the document gives
// them. Slicing is done with the zero-based constants below.
const (
	labelStart = 0  // column 1
	labelEnd   = 6  // through column 6
	blank7     = 6  // column 7
	opStart    = 7  // column 8
	opEnd      = 13 // through column 13
	blank14    = 13 // column 14
	blank15    = 14 // column 15
	operStart  = 15 // column 16
)

// Line is one source line with its fields located.
//
// A line is either a comment line or a statement; the fields other
// than Num and Text are meaningful only for statements.
type Line struct {
	File string
	Num  int    // one-based
	Text string // the raw line, newline removed

	Comment bool // an asterisk in column 1

	Label   string // columns 1-6, trailing blanks removed
	Op      string // columns 8-13, trailing blanks removed
	Operand string // column 16 through the end of the operand field
	Trailer string // everything after the operand field, verbatim
}

// Labeled reports whether the line defines a label.
func (l Line) Labeled() bool { return l.Label != "" }

// Rejoin reconstructs the raw line from the located fields.
//
// It is what makes the field boundaries testable: if Rejoin does not
// reproduce Text for every line of the historical source, the scanner
// has lost something. Statement lines in that source never carry
// trailing blanks, so padding the operation field out to column 15
// and then trimming is exact.
func (l Line) Rejoin() string {
	if l.Comment {
		return l.Text
	}
	var sb strings.Builder
	sb.WriteString(pad(l.Label, labelEnd-labelStart))
	sb.WriteString(" ")
	sb.WriteString(pad(l.Op, opEnd-opStart))
	sb.WriteString("  ")
	sb.WriteString(l.Operand)
	sb.WriteString(l.Trailer)
	return strings.TrimRight(sb.String(), " ")
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// Scan splits src into located lines, accumulating diagnostics rather
// than stopping at the first bad line.
//
// A returned line is always usable: a line that drew a diagnostic
// still has whatever fields could be located, so later stages see the
// whole file.
func Scan(file string, src []byte) ([]Line, diag.List) {
	var (
		lines []Line
		ds    diag.List
	)
	text := strings.TrimSuffix(string(src), "\n")
	if text == "" {
		return nil, ds
	}
	for i, raw := range strings.Split(text, "\n") {
		lines = append(lines, scanLine(file, i+1, strings.TrimSuffix(raw, "\r"), &ds))
	}
	return lines, ds
}

func scanLine(file string, num int, raw string, ds *diag.List) Line {
	l := Line{File: file, Num: num, Text: raw}

	// An asterisk in column 1 is a comment card (S4D58 7.6). This
	// covers the 516 "*_" markers and the 438 continuation-comment
	// lines, both of which are ordinary comments to a scanner.
	if strings.HasPrefix(raw, "*") {
		l.Comment = true
		return l
	}

	// A line too short to hold an operation field cannot be a
	// statement. The historical source has none.
	if len(raw) <= opStart {
		if strings.TrimSpace(raw) == "" {
			ds.Addf(file, num, 0, "blank line: expected a comment or a statement")
		} else {
			ds.Addf(file, num, 1, "line ends before the operation field at column %d", opStart+1)
		}
		return l
	}

	l.Label = strings.TrimRight(field(raw, labelStart, labelEnd), " ")
	if strings.ContainsAny(l.Label, " \t") {
		ds.Addf(file, num, 1, "label %q: embedded blank in the label field", l.Label)
	}
	if c := field(raw, blank7, blank7+1); c != " " {
		ds.Addf(file, num, blank7+1, "column %d must be blank: found %q", blank7+1, c)
	}

	l.Op = strings.TrimRight(field(raw, opStart, opEnd), " ")
	if l.Op == "" {
		ds.Addf(file, num, opStart+1, "empty operation field")
	} else if strings.ContainsAny(l.Op, " \t") {
		ds.Addf(file, num, opStart+1, "operation %q: embedded blank in the operation field", l.Op)
	}
	for _, col := range []int{blank14, blank15} {
		if c := field(raw, col, col+1); c != " " && c != "" {
			ds.Addf(file, num, col+1, "column %d must be blank: found %q", col+1, c)
		}
	}

	if len(raw) > operStart {
		n := operandEnd(raw[operStart:])
		if n < 0 {
			ds.Addf(file, num, operStart+1, "unterminated character literal")
			n = len(raw) - operStart
		}
		l.Operand = raw[operStart : operStart+n]
		l.Trailer = raw[operStart+n:]
	}
	return l
}

// field returns the half-open column range [lo, hi) of raw, short if
// the line ends first.
func field(raw string, lo, hi int) string {
	if lo >= len(raw) {
		return ""
	}
	if hi > len(raw) {
		hi = len(raw)
	}
	return raw[lo:hi]
}

// operandEnd returns the length of the operand field in s, which
// begins at column 16.
//
// The field ends at the first blank or tab that is not inside a
// character literal; everything after that is comment (S4D58 7.6:
// "Quotation marks do not occur within literals, but commas,
// parentheses, and blanks may. This fact must be taken into account
// in analyzing the variable field."). It returns -1 for a literal
// that is never closed.
func operandEnd(s string) int {
	quoted := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			quoted = !quoted
		case ' ', '\t':
			if !quoted {
				return i
			}
		}
	}
	if quoted {
		return -1
	}
	return len(s)
}
