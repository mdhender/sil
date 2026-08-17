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

package scanner_test

import (
	"slices"
	"testing"

	"github.com/mdhender/sil/pkg/sil/scanner"
)

// The lines below are transcribed from S4D58 7.6's sample program and
// from the shapes the historical source actually contains. Column
// positions matter, so each is written out in full rather than built
// from parts.
func TestScanLine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		text    string
		comment bool
		label   string
		op      string
		operand string
		trailer string
	}{
		{
			name:    "label, operation, operands and a comment",
			text:    "GCMA1  GETSIZ  BKDX,BK1CL           Get size of block",
			label:   "GCMA1",
			op:      "GETSIZ",
			operand: "BKDX,BK1CL",
			trailer: "           Get size of block",
		},
		{
			name:    "no label",
			text:    "       POP     BK1CL               Restore block to mark from",
			op:      "POP",
			operand: "BK1CL",
			trailer: "               Restore block to mark from",
		},
		{
			// S4D58 7.6: "If there are no operands, there is a
			// comma in column 16 and a blank in column 17."
			name:    "no operands is a lone comma",
			text:    "GCM    PROC    ,                   Procedure to mark blocks",
			label:   "GCM",
			op:      "PROC",
			operand: ",",
			trailer: "                   Procedure to mark blocks",
		},
		{
			name:    "omitted branch points are empty operands",
			text:    "       AEQLC   DESCL,0,,GCMA3      Is address zero?",
			op:      "AEQLC",
			operand: "DESCL,0,,GCMA3",
			trailer: "      Is address zero?",
		},
		{
			// A blank inside a literal does not end the operand
			// field. This is the rule S4D58 7.6 calls out.
			name:    "blanks inside a character literal",
			text:    "TRSTSP STRING  '    STATEMENT '",
			label:   "TRSTSP",
			op:      "STRING",
			operand: "'    STATEMENT '",
		},
		{
			name:    "commas and parens inside a literal",
			text:    "EXNO   FORMAT  '(1H0,I15,21H STATEMENTS EXECUTED,,I8,7H FAILED)'",
			label:   "EXNO",
			op:      "FORMAT",
			operand: "'(1H0,I15,21H STATEMENTS EXECUTED,,I8,7H FAILED)'",
		},
		{
			name:    "parenthesised list with an empty slot",
			text:    "       SELBRA  BRTYPE,(,RTN2,RTN2,,,RTN2,RTN2)",
			op:      "SELBRA",
			operand: "BRTYPE,(,RTN2,RTN2,,,RTN2,RTN2)",
		},
		{
			name:    "an expression operand",
			text:    "ATTRIB EQU     2*DESCR",
			label:   "ATTRIB",
			op:      "EQU",
			operand: "2*DESCR",
		},
		{
			// The last line of the historical source, and the
			// only statement in it with no operand field at all.
			name: "END carries no operand",
			text: "       END",
			op:   "END",
		},
		{
			name:    "a one-character label",
			text:    "S      EQU     1",
			label:   "S",
			op:      "EQU",
			operand: "1",
		},
		{
			name:    "a tab before the comment",
			text:    "BEGIN  INIT    ,\t\t   Initialize system",
			label:   "BEGIN",
			op:      "INIT",
			operand: ",",
			trailer: "\t\t   Initialize system",
		},
		{name: "comment card", text: "*      Block Marking", comment: true},
		{name: "no-fall-through marker", text: "*_", comment: true},
		{name: "bare asterisk", text: "*", comment: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines, ds := scanner.Scan("t.sil", []byte(tc.text))
			if err := ds.Err(); err != nil {
				t.Fatalf("unexpected diagnostics:\n%v", err)
			}
			if len(lines) != 1 {
				t.Fatalf("got %d lines, want 1", len(lines))
			}
			got := lines[0]
			if got.Comment != tc.comment {
				t.Errorf("Comment = %v, want %v", got.Comment, tc.comment)
			}
			if got.Label != tc.label {
				t.Errorf("Label = %q, want %q", got.Label, tc.label)
			}
			if got.Op != tc.op {
				t.Errorf("Op = %q, want %q", got.Op, tc.op)
			}
			if got.Operand != tc.operand {
				t.Errorf("Operand = %q, want %q", got.Operand, tc.operand)
			}
			if got.Trailer != tc.trailer {
				t.Errorf("Trailer = %q, want %q", got.Trailer, tc.trailer)
			}
			if rejoined := got.Rejoin(); rejoined != tc.text {
				t.Errorf("Rejoin() = %q, want %q", rejoined, tc.text)
			}
		})
	}
}

// A label and an operation may be spelled the same. Fourteen names in
// the historical source are, and the columns are the only thing that
// tells them apart.
func TestOperationAndLabelCollide(t *testing.T) {
	lines, ds := scanner.Scan("t.sil", []byte(
		"COPY   PROC    ,\n"+
			"       COPY    MLINK\n"))
	if err := ds.Err(); err != nil {
		t.Fatalf("unexpected diagnostics:\n%v", err)
	}
	if lines[0].Label != "COPY" || lines[0].Op != "PROC" {
		t.Errorf("line 1: got label %q op %q, want COPY/PROC", lines[0].Label, lines[0].Op)
	}
	if lines[1].Label != "" || lines[1].Op != "COPY" {
		t.Errorf("line 2: got label %q op %q, want \"\"/COPY", lines[1].Label, lines[1].Op)
	}
}

func TestScanDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want []string
	}{
		{
			name: "unterminated literal",
			text: "BLSP   STRING  'oops",
			want: []string{"t.sil:1:16: unterminated character literal"},
		},
		{
			name: "empty operation field",
			text: "LABEL          ,",
			want: []string{"t.sil:1:8: empty operation field"},
		},
		{
			// A character in column 7 shifts the operation field as
			// well, so it draws a second diagnostic. Both are true
			// and both are worth reporting: the stage accumulates.
			name: "column 7 not blank",
			text: "LABELXX GETD   A,B",
			want: []string{
				`t.sil:1:7: column 7 must be blank: found "X"`,
				`t.sil:1:8: operation " GETD": embedded blank in the operation field`,
			},
		},
		{
			name: "line ends before the operation field",
			text: "AB",
			want: []string{"t.sil:1:1: line ends before the operation field at column 8"},
		},
		{
			name: "blank line",
			text: "   ",
			want: []string{"t.sil:1: blank line: expected a comment or a statement"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ds := scanner.Scan("t.sil", []byte(tc.text))
			var got []string
			for _, d := range ds {
				got = append(got, d.String())
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

// A file whose last line has no newline scans the same as one that
// does, and a trailing newline does not invent an empty line.
func TestScanTrailingNewline(t *testing.T) {
	const src = "       END"
	with, _ := scanner.Scan("t.sil", []byte(src+"\n"))
	without, _ := scanner.Scan("t.sil", []byte(src))
	if len(with) != 1 || len(without) != 1 {
		t.Fatalf("got %d and %d lines, want 1 each", len(with), len(without))
	}
	if with[0] != without[0] {
		t.Errorf("trailing newline changed the line: %+v vs %+v", with[0], without[0])
	}
}
