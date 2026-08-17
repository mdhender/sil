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

package fortran_test

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/fortran"
)

// real returns the address field a machine holds a real number in, so
// that a test can write 18.5 and hand over what OUTPUT would (3.1.1,
// and pkg/sil/reals.go).
func real(f float64) int { return int(math.Float64bits(f)) }

// The nine formats of the SNOBOL4 system that carry a value, run with
// the values the system passes them, against what a printer would put
// on the page.
//
// These are the check on the whole package: the formats are the
// source's own, at lines 6551 to 6579, and the output is what S4D58's
// system was documented to print.
func TestTheSystemsFormats(t *testing.T) {
	for _, tt := range []struct {
		format string
		values []int
		want   []string
	}{
		// A banner, a new page and then an overprinted underline: the
		// only place the system overprints.
		{
			"(37H1SNOBOL4 (VERSION 3.11, MAY 19, 1975)/8H+_______)", nil,
			[]string{"\fSNOBOL4 (VERSION 3.11, MAY 19, 1975)", "_______"},
		},
		// The 42 characters stop one short of the trailing comma, so
		// that comma is the separator before the / and not text. It
		// is the same trick as the ",,I8" below, read the other way.
		{
			"(42H0BELL TELEPHONE LABORATORIES, INCORPORATED,/1H1)", nil,
			[]string{"", "BELL TELEPHONE LABORATORIES, INCORPORATED", "\f"},
		},
		{
			"(37H0NO ERRORS DETECTED IN SOURCE PROGRAM/1H1)", nil,
			[]string{"", "NO ERRORS DETECTED IN SOURCE PROGRAM", "\f"},
		},
		{"(1H1)", nil, []string{"\f"}},
		{
			"(28H1NORMAL TERMINATION AT LEVEL,I3)", []int{0},
			[]string{"\fNORMAL TERMINATION AT LEVEL  0"},
		},
		{
			"(28H LAST STATEMENT EXECUTED WAS,I5)", []int{2},
			[]string{"LAST STATEMENT EXECUTED WAS    2"},
		},
		{
			"(1H0,I15,21H MS. COMPILATION TIME)", []int{37},
			[]string{"", "             37 MS. COMPILATION TIME"},
		},
		// The comma of ",,I8" is the last character of the twenty-one
		// the Hollerith field takes, not an empty item.
		{
			"(1H0,I15,21H STATEMENTS EXECUTED,,I8,7H FAILED)", []int{19, 1},
			[]string{"", "             19 STATEMENTS EXECUTED,       1 FAILED"},
		},
		{
			"(6H1ERROR,I3,13H IN STATEMENT,I5,9H AT LEVEL,I3)", []int{28, 1, 0},
			[]string{"\fERROR 28 IN STATEMENT    1 AT LEVEL  0"},
		},
		{
			"(1H0,F15.2,35H MS. AVERAGE PER STATEMENT EXECUTED/1H1)", []int{real(18.5)},
			[]string{"", "          18.50 MS. AVERAGE PER STATEMENT EXECUTED", "\f"},
		},
		{
			"(18H0NATURAL VARIABLES,/1H )", nil,
			[]string{"", "NATURAL VARIABLES", ""},
		},
	} {
		t.Run(tt.format, func(t *testing.T) {
			records, err := fortran.Numbers([]byte(tt.format), tt.values)
			if err != nil {
				t.Fatal(err)
			}
			if got := fortran.Lines(records); !slices.Equal(got, tt.want) {
				t.Errorf("printed\n\t%q\nwant\n\t%q", got, tt.want)
			}
		})
	}
}

// The two formats a SNOBOL4 program's own output goes under: (1X,132A1)
// on the printer and (80A1) on the punch. They are the whole of what
// STPRNT sees.
func TestTheProgramsFormats(t *testing.T) {
	t.Run("the printer", func(t *testing.T) {
		records, err := fortran.Chars([]byte("(1X,132A1)"), []byte("HELLO"))
		if err != nil {
			t.Fatal(err)
		}
		// 1X is the carriage control blank, so the record is a blank
		// and then the string, and the line is the string.
		if got, want := string(records[0]), " HELLO"; got != want {
			t.Errorf("the record is %q, want %q", got, want)
		}
		if got, want := fortran.Lines(records), []string{"HELLO"}; !slices.Equal(got, want) {
			t.Errorf("printed %q, want %q", got, want)
		}
	})

	// The punch has no carriage control, so the record is the string
	// and the first character of it is a character.
	t.Run("the punch", func(t *testing.T) {
		records, err := fortran.Chars([]byte("(80A1)"), []byte("PP"))
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 1 || string(records[0]) != "PP" {
			t.Errorf("punched %q, want one record of %q", records, "PP")
		}
	})

	// A string longer than the format wraps, because reaching the end
	// of a format with the list unfinished starts a new record.
	t.Run("a string longer than the record", func(t *testing.T) {
		records, err := fortran.Chars([]byte("(1X,4A1)"), []byte("ABCDEFGHIJ"))
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"ABCD", "EFGH", "IJ"}
		if got := fortran.Lines(records); !slices.Equal(got, want) {
			t.Errorf("printed %q, want %q", got, want)
		}
	})

	// The empty string is a record of nothing, which the printer turns
	// into a blank line.
	t.Run("the null string", func(t *testing.T) {
		records, err := fortran.Chars([]byte("(1X,132A1)"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := fortran.Lines(records), []string{""}; !slices.Equal(got, want) {
			t.Errorf("printed %q, want %q", got, want)
		}
	})
}

// Carriage control, which is the printer's and not the format's.
func TestCarriageControl(t *testing.T) {
	for _, tt := range []struct {
		record fortran.Record
		want   []string
	}{
		{" one line", []string{"one line"}},
		{"0two lines", []string{"", "two lines"}},
		{"1a new page", []string{"\fa new page"}},
		{"+no advance", []string{"no advance"}},
		{"", []string{""}},
	} {
		if got := fortran.Lines([]fortran.Record{tt.record}); !slices.Equal(got, tt.want) {
			t.Errorf("%q printed %q, want %q", tt.record, got, tt.want)
		}
		if got, want := tt.record.Control(), byte(' '); tt.record == "" && got != want {
			t.Errorf("an empty record controls %q, want %q", got, want)
		}
	}
}

// The numeric fields, in the w columns they are given.
func TestNumericFields(t *testing.T) {
	for _, tt := range []struct {
		format string
		value  int
		want   string
	}{
		{"(I5)", 42, "   42"},
		{"(I5)", -42, "  -42"},
		{"(I1)", 0, "0"},
		// A value that does not fit fills its columns with asterisks
		// rather than pushing every column after it along.
		{"(I2)", 12345, "**"},
		{"(I2)", -5, "-5"},
		{"(I1)", -5, "*"},

		{"(F8.2)", real(3.14159), "    3.14"},
		{"(F8.2)", real(-3.14159), "   -3.14"},
		{"(F8.2)", real(0), "    0.00"},
		{"(F4.2)", real(12345.6), "****"},
		// Rounding is to the decimal places asked for, and it is the
		// stored binary value that is rounded: 2.55 and 2.65 are both
		// a little under the decimal number they are written as, so
		// both round down, as they would on any binary machine.
		{"(F6.1)", real(2.55), "   2.5"},
		{"(F6.1)", real(2.65), "   2.6"},
		{"(F6.1)", real(2.66), "   2.7"},

		// Ew.d puts the first significant digit just after the point.
		{"(E12.4)", real(1234.5), "  0.1235E+04"},
		{"(E12.4)", real(-1234.5), " -0.1235E+04"},
		{"(E12.4)", real(0), "  0.0000E+00"},
		{"(E12.4)", real(0.00123), "  0.1230E-02"},
		// Rounding that carries into the next power of ten moves the
		// exponent rather than printing 1.0000.
		{"(E12.4)", real(0.99999999), "  0.1000E+01"},
	} {
		t.Run(tt.format+" "+tt.want, func(t *testing.T) {
			records, err := fortran.Numbers([]byte(tt.format), []int{tt.value})
			if err != nil {
				t.Fatal(err)
			}
			if got := string(records[0]); got != tt.want {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
		})
	}
}

// Repeat counts, groups and the record separator.
func TestRepeatsAndGroups(t *testing.T) {
	for _, tt := range []struct {
		format string
		values []int
		want   []string
	}{
		{"(3I3)", []int{1, 2, 3}, []string{"  1  2  3"}},
		{"(2(I2,1H;))", []int{1, 2}, []string{" 1; 2;"}},
		{"(I2/I2)", []int{1, 2}, []string{" 1", " 2"}},
		{"(2/I2)", []int{1}, []string{"", "", " 1"}},
		// Reversion goes back to the last group at the outer level,
		// not to the whole format.
		{"(1HA,2(I2))", []int{1, 2, 3, 4}, []string{"A 1 2", " 3 4"}},
		// With no group it goes back to the beginning, so the leading
		// literal comes round again.
		{"(1HA,I2)", []int{1, 2}, []string{"A 1", "A 2"}},
		// A list shorter than the format stops at the first field it
		// cannot fill.
		{"(I2,1HX,I2)", []int{1}, []string{" 1X"}},
		{"(4X,I2)", []int{7}, []string{"     7"}},
		// A quoted literal is a literal, and two quotes are one.
		{"('IT''S',I2)", []int{5}, []string{"IT'S 5"}},
		// Blanks between items are ignored; blanks inside a Hollerith
		// field are text.
		{"( 2 I3 , 4H A B )", []int{1, 2}, []string{"  1  2 A B"}},
	} {
		t.Run(tt.format, func(t *testing.T) {
			records, err := fortran.Numbers([]byte(tt.format), tt.values)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range records {
				got = append(got, string(r))
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
		})
	}
}

// A format this cannot read says so, rather than printing a field
// wrong. A SNOBOL4 program can build one at run time (2.1), so these
// are reachable.
func TestBadFormats(t *testing.T) {
	for _, tt := range []struct {
		format string
		want   string
	}{
		{"1H1", "does not start with ("},
		{"(1H1", "ends inside a list"},
		{"(1H1))", "follows the closing )"},
		{"(I)", "I has no width"},
		{"(F5)", "F5 has no decimal places"},
		{"(H)", "H has no count"},
		{"(9HSHORT)", "wants 9 characters"},
		{"(Q5)", `"Q" is not an edit descriptor`},
		{"((I2)", "ends inside a list"},
		{"(2(I2)", "ends inside a list"},
		{"('unclosed)", "not closed"},
		// A value with nowhere to go would otherwise go round the
		// format for ever.
		{"(1H1)", "no field for the 1 values left"},
		// The list is numbers, so A has nothing to take.
		{"(A1)", "the output list is numbers"},
	} {
		t.Run(tt.format, func(t *testing.T) {
			_, err := fortran.Numbers([]byte(tt.format), []int{1})
			if err == nil {
				t.Fatalf("no error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("reported %v, want it to mention %q", err, tt.want)
			}
		})
	}

	// And the other way round: a number field with a string to print.
	if _, err := fortran.Chars([]byte("(I5)"), []byte("X")); err == nil {
		t.Error("no error for a number field with a string")
	} else if !strings.Contains(err.Error(), "no number for this field") {
		t.Errorf("reported %v", err)
	}
}
