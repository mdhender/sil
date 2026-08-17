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

package parser_test

import (
	"slices"
	"testing"

	"github.com/mdhender/sil/pkg/sil/parser"
)

// The single most important test in the package.
//
// OBEND at sil-v3.11.sil:5475 is the only statement in 6580 lines that
// mixes `*` with `+`. OBLIST is a relocatable address (OBSTRT-LNKFLD)
// and OBOFF is 254, so left-to-right evaluation would multiply an
// address by 254 rather than offsetting it -- and would do so
// silently, since the result is still a number. No SIL test program
// will ever catch this; only this test will.
func TestMultiplicationBindsTighter(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"OBLIST+DESCR*OBOFF", "(OBLIST+(DESCR*OBOFF))"},
		{"DESCR*OBOFF+OBLIST", "((DESCR*OBOFF)+OBLIST)"},
		{"A-B*C", "(A-(B*C))"},
		{"A*B-C", "((A*B)-C)"},
		// Addition and subtraction associate to the left, which is
		// what DTLIST's size computation depends on.
		{"DTLEND-DTLIST-DESCR", "((DTLEND-DTLIST)-DESCR)"},
		{"TTL+MARK", "(TTL+MARK)"},
		{"2*DESCR", "(2*DESCR)"},
		// Unary minus, which appears twice: GETAC TVAL,PDLPTR,-2*DESCR
		{"-2*DESCR", "(-2*DESCR)"},
		{"CARDSZ+STNOSZ-SEQSIZ+1", "(((CARDSZ+STNOSZ)-SEQSIZ)+1)"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			items := mustParse(t, tc.in)
			if len(items) != 1 || items[0].Kind != parser.ItemExpr {
				t.Fatalf("got %d items of kind %v, want 1 expression", len(items), items[0].Kind)
			}
			if got := items[0].Expr.String(); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestParseOperands(t *testing.T) {
	for _, tc := range []struct {
		name  string
		in    string
		kinds []parser.ItemKind
		want  string // Format() of the parse
	}{
		{
			// S4D58 7.6: a lone comma in column 16 means the
			// operation has no operands at all. PROC, INIT, ISTACK
			// and LHERE are all written this way.
			name: "lone comma is no operands",
			in:   ",",
		},
		{
			name: "no operand field at all",
			in:   "",
		},
		{
			name:  "one symbol",
			in:    "XLATRN",
			kinds: []parser.ItemKind{parser.ItemExpr},
			want:  "XLATRN",
		},
		{
			// An omitted branch point. AEQLC's operands are
			// DESCR,N,NELOC,EQLOC; here NELOC is omitted, so an
			// unequal comparison falls through (S4D58 5.2, 6.9).
			name:  "omitted interior operand",
			in:    "ZPTR,0,,SPCNV2",
			kinds: []parser.ItemKind{parser.ItemExpr, parser.ItemExpr, parser.ItemNull, parser.ItemExpr},
			want:  "ZPTR,0,,SPCNV2",
		},
		{
			name:  "two omitted operands in a row",
			in:    "INITB,INITE,,,INITD1",
			kinds: []parser.ItemKind{parser.ItemExpr, parser.ItemExpr, parser.ItemNull, parser.ItemNull, parser.ItemExpr},
			want:  "INITB,INITE,,,INITD1",
		},
		{
			name:  "leading omitted operand",
			in:    ",NEWCRD,,(XLATRD,,)",
			kinds: []parser.ItemKind{parser.ItemNull, parser.ItemExpr, parser.ItemNull, parser.ItemList},
			want:  ",NEWCRD,,(XLATRD,,)",
		},
		{
			name:  "list with a leading empty slot",
			in:    "BRTYPE,(,RTN2,RTN2,,,RTN2,RTN2)",
			kinds: []parser.ItemKind{parser.ItemExpr, parser.ItemList},
			want:  "BRTYPE,(,RTN2,RTN2,,,RTN2,RTN2)",
		},
		{
			name:  "twelve-way select",
			in:    "SCL,(AD,DV,EX,MP,SB,CEQ,CGE,CGT,CLE,CLT,CNE,RM)",
			kinds: []parser.ItemKind{parser.ItemExpr, parser.ItemList},
			want:  "SCL,(AD,DV,EX,MP,SB,CEQ,CGE,CGT,CLE,CLT,CNE,RM)",
		},
		{
			name:  "two lists",
			in:    ",GC,(ARG1CL),(ALOC2,BLOCK1)",
			kinds: []parser.ItemKind{parser.ItemNull, parser.ItemExpr, parser.ItemList, parser.ItemList},
			want:  ",GC,(ARG1CL),(ALOC2,BLOCK1)",
		},
		{
			// Blanks inside a literal are data, not the end of the
			// operand field.
			name:  "literal with blanks",
			in:    "'    STATEMENT '",
			kinds: []parser.ItemKind{parser.ItemLiteral},
			want:  "'    STATEMENT '",
		},
		{
			// Commas and parens inside a literal are data too.
			name:  "literal with commas and parens",
			in:    "'(1H0,I15,21H STATEMENTS EXECUTED,,I8,7H FAILED)'",
			kinds: []parser.ItemKind{parser.ItemLiteral},
			want:  "'(1H0,I15,21H STATEMENTS EXECUTED,,I8,7H FAILED)'",
		},
		{
			name:  "single character literal",
			in:    "' '",
			kinds: []parser.ItemKind{parser.ItemLiteral},
			want:  "' '",
		},
		{
			name:  "descriptor with a self-reference and a computed size",
			in:    "DTLIST,TTL+MARK,DTLEND-DTLIST-DESCR",
			kinds: []parser.ItemKind{parser.ItemExpr, parser.ItemExpr, parser.ItemExpr},
			want:  "DTLIST,(TTL+MARK),((DTLEND-DTLIST)-DESCR)",
		},
		{
			// A one-character symbol: the data type codes at the head
			// of the source are single letters.
			name:  "one-character symbol",
			in:    "A4PTR,B",
			kinds: []parser.ItemKind{parser.ItemExpr, parser.ItemExpr},
			want:  "A4PTR,B",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := mustParse(t, tc.in)
			var kinds []parser.ItemKind
			for _, it := range items {
				kinds = append(kinds, it.Kind)
			}
			if !slices.Equal(kinds, tc.kinds) {
				t.Errorf("kinds = %v, want %v", kinds, tc.kinds)
			}
			if got := parser.Format(items); got != tc.want {
				t.Errorf("Format() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "nested list",
			in:   "A,(B,(C))",
			want: []string{"t.sil:1:21: nested parentheses: lists do not nest"},
		},
		{
			name: "unclosed list",
			in:   "A,(B,C",
			want: []string{"t.sil:1:18: unclosed parenthesis"},
		},
		{
			name: "unterminated literal",
			in:   "'oops",
			want: []string{"t.sil:1:16: unterminated character literal"},
		},
		{
			name: "a symbol may not begin with a digit",
			in:   "1ABC",
			want: []string{`t.sil:1:17: unexpected "A" in the operand field`},
		},
		{
			name: "dangling operator",
			in:   "A+",
			want: []string{"t.sil:1:18: expected a symbol or an integer, found end of operand"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ds = newList()
			parser.ParseOperands("t.sil", 1, tc.in, ds)
			var got []string
			for _, d := range *ds {
				got = append(got, d.String())
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestSymbols(t *testing.T) {
	items := mustParse(t, "DTLIST,TTL+MARK,DTLEND-DTLIST-DESCR")
	var got []string
	for _, it := range items {
		if it.Kind == parser.ItemExpr {
			got = append(got, parser.Symbols(it.Expr)...)
		}
	}
	slices.Sort(got)
	got = slices.Compact(got)
	want := []string{"DESCR", "DTLEND", "DTLIST", "MARK", "TTL"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A literal contributes no symbol references, which is what keeps
// FORMAT and STRING operands out of the symbol table.
func TestLiteralHasNoSymbols(t *testing.T) {
	items := mustParse(t, "'ARRAY'")
	if items[0].Kind != parser.ItemLiteral {
		t.Fatalf("kind = %v, want literal", items[0].Kind)
	}
	if items[0].Expr != nil {
		t.Errorf("literal carries an expression: %v", items[0].Expr)
	}
}

func mustParse(t *testing.T, s string) []parser.Item {
	t.Helper()
	ds := newList()
	items := parser.ParseOperands("t.sil", 1, s, ds)
	if err := ds.Err(); err != nil {
		t.Fatalf("unexpected diagnostics for %q:\n%v", s, err)
	}
	return items
}
