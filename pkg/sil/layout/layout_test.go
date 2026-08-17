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

package layout_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/layout"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

// narrow is the assembly prologue every test here starts with: a
// descriptor is one address unit, a specifier two, one character per
// unit. It is what PARMS chooses, so the sizes below read the way the
// listing of a real assembly would.
var narrow = []string{
	stmt("DESCR", "EQU", "1"),
	stmt("SPEC", "EQU", "2"),
	stmt("CPA", "EQU", "1"),
}

// Each of the twelve directives (S4D58 7.5) either assembles a known
// amount of data or assembles none, and every other operation takes
// one address unit.
func TestStatementSizes(t *testing.T) {
	for _, tt := range []struct {
		name string
		line string
		size int
	}{
		{"TITLE assembles nothing", stmt("", "TITLE", "'A title'"), 0},
		{"EQU assembles nothing", stmt("N", "EQU", "5"), 0},
		{"LHERE assembles nothing", stmt("L", "LHERE", ","), 0},
		{"PROC assembles nothing", stmt("P", "PROC", ","), 0},
		{"DESCR is one descriptor", stmt("D", "DESCR", "0,0,0"), 1},
		{"SPEC is one specifier", stmt("S", "SPEC", "0,0,0,0,0"), 2},
		{"ARRAY is N descriptors", stmt("A", "ARRAY", "7"), 7},
		{"ARRAY counts in descriptors", stmt("A", "ARRAY", "2*DESCR"), 2},
		{"BUFFER is N characters", stmt("B", "BUFFER", "9"), 9},
		{"STRING is a specifier and its characters", stmt("T", "STRING", "'ABC'"), 5},
		{"FORMAT is characters only", stmt("F", "FORMAT", "'(I5)'"), 4},
		{"an operation is one address unit", stmt("", "MOVD", "DD,DD"), 1},
		{"a branching operation is one address unit", stmt("", "ACOMP", "DD,DD,LL,LL,LL"), 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// The line under test comes first and everything after it
			// assembles nothing, so the line is the whole assembly:
			// it starts at the origin and its size is the end.
			l, _ := lay(t, append([]string{tt.line}, append(narrow,
				stmt("DD", "LHERE", ","),
				stmt("LL", "LHERE", ","))...)...)
			p := l.Placement(0)
			if p.Size != tt.size {
				t.Errorf("%q assembles %d address units, want %d", tt.line, p.Size, tt.size)
			}
			if p.Addr != layout.Origin {
				t.Errorf("%q was placed at %d, want the origin %d", tt.line, p.Addr, layout.Origin)
			}
			if got, want := l.End(), layout.Origin+tt.size; got != want {
				t.Errorf("the assembly ends at %d, want %d", got, want)
			}
		})
	}
}

// CPA is the number of characters per address unit, so a string that
// does not fill its last unit still occupies it (S4D58 6.20).
func TestCharactersPerAddressUnit(t *testing.T) {
	for _, tt := range []struct {
		chars int
		units int
	}{{0, 0}, {1, 1}, {4, 1}, {5, 2}, {8, 2}, {9, 3}} {
		t.Run(fmt.Sprintf("%d characters", tt.chars), func(t *testing.T) {
			l, stmts := lay(t,
				stmt("DESCR", "EQU", "1"),
				stmt("SPEC", "EQU", "2"),
				stmt("CPA", "EQU", "4"),
				stmt("B", "BUFFER", fmt.Sprint(tt.chars)),
			)
			if got := l.Placement(len(stmts) - 1).Size; got != tt.units {
				t.Errorf("BUFFER %d occupies %d address units at CPA=4, want %d", tt.chars, got, tt.units)
			}
		})
	}
}

// A label that is not an equivalence names the address of whatever
// comes after it, so LHERE and PROC both mean "here" (S4D58 6.54, and
// 6.78 note 2: PROC "may be implemented as LHERE").
func TestLabelsAddressWhatFollowsThem(t *testing.T) {
	l, _ := lay(t, append(narrow,
		stmt("HEAD", "DESCR", "0,0,0"),
		stmt("BODY", "ARRAY", "4"),
		stmt("HERE", "LHERE", ","),
		stmt("ENTRY", "PROC", ","),
		stmt("", "MOVD", "HEAD,HEAD"),
		stmt("TAIL", "LHERE", ","),
	)...)

	for _, tt := range []struct {
		name string
		want int
	}{
		{"HEAD", 0},
		{"BODY", 1},
		{"HERE", 5},
		{"ENTRY", 5},
		{"TAIL", 6},
	} {
		v, ok := l.Value(tt.name)
		if !ok {
			t.Errorf("%s has no value", tt.name)
			continue
		}
		if v.N != tt.want || !v.Reloc {
			t.Errorf("%s = %v, want the address %d", tt.name, v, tt.want)
		}
	}
}

// An equivalence is a fact rather than a step, so it may be written
// before the things it depends on. PRMSIZ at line 6336 of the SNOBOL4
// source is used at line 6316.
func TestEquivalencesResolveInAnyOrder(t *testing.T) {
	l, _ := lay(t,
		stmt("SIZE", "EQU", "END-BLOCK-DESCR"),
		stmt("WIDTH", "EQU", "2*DESCR"),
		stmt("DESCR", "EQU", "1"),
		stmt("SPEC", "EQU", "2"),
		stmt("CPA", "EQU", "1"),
		stmt("BLOCK", "DESCR", "BLOCK,0,SIZE"),
		stmt("", "ARRAY", "3"),
		stmt("END", "LHERE", ","),
	)

	for _, tt := range []struct {
		name string
		want layout.Value
	}{
		{"WIDTH", layout.Abs(2)},
		{"SIZE", layout.Abs(3)},
		{"BLOCK", layout.Addr(0)},
		{"END", layout.Addr(4)},
	} {
		if got, _ := l.Value(tt.name); got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// END terminates the assembly (S4D58 6.28), so nothing after it takes
// room.
func TestENDTerminatesTheAssembly(t *testing.T) {
	l, _ := lay(t, append(narrow,
		stmt("A", "DESCR", "0,0,0"),
		stmt("", "END", ""),
		stmt("B", "DESCR", "0,0,0"),
	)...)
	if got := l.End(); got != 1 {
		t.Errorf("the assembly ends at %d, want 1", got)
	}
}

// The relocatable/absolute discipline. An address plus a count is an
// address, the difference of two addresses is a count, and an address
// multiplied or negated is nothing at all.
func TestAddressArithmetic(t *testing.T) {
	for _, tt := range []struct {
		expr string
		want layout.Value
		bad  string // the diagnostic expected instead, if any
	}{
		{expr: "7", want: layout.Abs(7)},
		{expr: "HERE", want: layout.Addr(3)},
		{expr: "N", want: layout.Abs(10)},
		{expr: "HERE+N", want: layout.Addr(13)},
		{expr: "N+HERE", want: layout.Addr(13)},
		{expr: "HERE-N", want: layout.Addr(-7)},
		{expr: "THERE-HERE", want: layout.Abs(1)},
		{expr: "THERE-HERE-DESCR", want: layout.Abs(0)},
		{expr: "N*DESCR", want: layout.Abs(10)},
		{expr: "HERE+DESCR*N", want: layout.Addr(13)},
		{expr: "-N", want: layout.Abs(-10)},
		{expr: "-2*DESCR", want: layout.Abs(-2)},

		{expr: "HERE+THERE", bad: "cannot add the addresses HERE and THERE"},
		{expr: "N-HERE", bad: "cannot subtract the address HERE from the number N"},
		{expr: "HERE*N", bad: "cannot multiply the address HERE"},
		{expr: "N*HERE", bad: "cannot multiply the address HERE"},
		{expr: "-HERE", bad: "cannot negate the address HERE"},
		{expr: "MISSING", bad: "MISSING has no value"},
		{expr: "HERE+MISSING", bad: "MISSING has no value"},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			src := append(append([]string{}, narrow...),
				stmt("N", "EQU", "10"),
				stmt("", "MOVD", "N,N"),
				stmt("", "MOVD", "N,N"),
				stmt("", "MOVD", "N,N"),
				stmt("HERE", "LHERE", ","),
				stmt("", "MOVD", "N,N"),
				stmt("THERE", "LHERE", ","),
				stmt("UNDER", "EQU", tt.expr),
			)
			l, ds := run(t, src...)

			if tt.bad != "" {
				if err := ds.Err(); err == nil {
					t.Fatalf("%s resolved to %v, want the diagnostic %q", tt.expr, mustValue(t, l, "UNDER"), tt.bad)
				} else if !strings.Contains(err.Error(), tt.bad) {
					t.Fatalf("%s reported\n%v\nwant a diagnostic containing %q", tt.expr, err, tt.bad)
				}
				return
			}
			if err := ds.Err(); err != nil {
				t.Fatalf("%s reported diagnostics:\n%v", tt.expr, err)
			}
			if got := mustValue(t, l, "UNDER"); got != tt.want {
				t.Errorf("%s = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

// Diagnostics accumulate rather than stopping the run, and each one
// says where it is and what is wrong with it.
func TestDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  []string
		// bare leaves out the machine parameters, for the cases that
		// are about not having them.
		bare bool
		want string
	}{
		{
			name: "EQU without a label",
			src:  []string{stmt("", "EQU", "1")},
			want: "EQU without a label defines nothing",
		},
		{
			name: "EQU of something that is not an expression",
			src:  []string{stmt("N", "EQU", "'TEXT'")},
			want: "N EQU: expected one expression",
		},
		{
			name: "an array of an address",
			src:  []string{stmt("A", "ARRAY", "A")},
			want: "ARRAY: the count A is an address",
		},
		{
			name: "a buffer of a negative length",
			src:  []string{stmt("B", "BUFFER", "0-1")},
			want: "BUFFER: the count (0-1) is -1",
		},
		{
			name: "a string of something that is not a literal",
			src:  []string{stmt("S", "STRING", "DESCR")},
			want: "STRING: expected a character literal",
		},
		{
			name: "equivalences that depend on each other",
			src:  []string{stmt("A", "EQU", "B"), stmt("B", "EQU", "A")},
			want: "B has no value",
		},
		{
			name: "no DESCR",
			src:  []string{stmt("SPEC", "EQU", "2"), stmt("CPA", "EQU", "1")},
			bare: true,
			want: "DESCR is not defined; COPY PARMS must define it",
		},
		{
			name: "a descriptor width of zero",
			src:  []string{stmt("DESCR", "EQU", "0"), stmt("SPEC", "EQU", "2"), stmt("CPA", "EQU", "1")},
			bare: true,
			want: "DESCR is 0; it must be positive",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var src []string
			if !tt.bare {
				src = append(src, narrow...)
			}
			_, ds := run(t, append(src, tt.src...)...)
			err := ds.Err()
			if err == nil {
				t.Fatalf("no diagnostic, want one containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("reported\n%v\nwant a diagnostic containing %q", err, tt.want)
			}
		})
	}
}

// stmt renders one SIL statement into its columns (S4D58 7.6): the
// label in 1 through 6, the operation in 8 through 13, the operands
// from 16.
func stmt(label, op, operand string) string {
	return strings.TrimRight(fmt.Sprintf("%-6s %-6s  %s", label, op, operand), " ")
}

// lay assembles the lines and requires every stage to be silent.
func lay(t *testing.T, lines ...string) (*layout.Layout, []parser.Statement) {
	t.Helper()
	l, ds := run(t, lines...)
	if err := ds.Err(); err != nil {
		t.Fatalf("layout reported diagnostics:\n%v", err)
	}
	stmts, _ := parse(t, lines)
	return l, stmts
}

// run assembles the lines and returns whatever the layout reported.
func run(t *testing.T, lines ...string) (*layout.Layout, diag.List) {
	t.Helper()
	stmts, _ := parse(t, lines)
	return layout.Run(stmts)
}

func parse(t *testing.T, lines []string) ([]parser.Statement, diag.List) {
	t.Helper()
	src := []byte(strings.Join(lines, "\n") + "\n")
	scanned, ds := scanner.Scan("test.sil", src)
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner reported diagnostics:\n%v", err)
	}
	stmts, ds := parser.Parse(scanned)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}
	return stmts, ds
}

func mustValue(t *testing.T, l *layout.Layout, name string) layout.Value {
	t.Helper()
	v, ok := l.Value(name)
	if !ok {
		t.Fatalf("%s has no value", name)
	}
	return v
}
