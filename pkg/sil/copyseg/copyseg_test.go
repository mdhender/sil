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

package copyseg_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/copyseg"
	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
	"github.com/mdhender/sil/pkg/sil/symtab"
)

// The other side of the implementer's contract.
//
// symtab's corpus test asserts the 37 names the SNOBOL4 source uses
// and never defines. Three of those are the segment names themselves,
// which COPY consumes, so 34 names have to come from here. The twelve
// extra syntax tables are the ones Appendix A defines but the source
// never mentions, reachable only through a GOTO(TABLE) action inside
// another table.
//
// Keeping the two lists in separate packages and asserting both is
// what makes the contract a contract: adding a name here that nothing
// asks for shows up as an extra, and dropping one shows up in the
// layout corpus test as an undefined symbol.
var wantDefined = []string{
	// PARMS: sizes, flags and unit numbers (S4D58 6.20).
	"ALPHSZ", "CPA", "DESCR", "SIZLIM", "SPEC",
	"FNC", "MARK", "PTR", "STTL", "TTL",
	"UNITI", "UNITO", "UNITP",
	// PARMS: syntax table action codes (S4D58 4.2).
	"CONTIN", "ERROR", "STOP", "STOPSH",
	// MDATA: character strings (S4D58 6.20).
	"ALPHA", "AMPST", "COLSTR", "QTSTR",
	// MDATA: the thirteen syntax tables the source names.
	"BIOPTB", "CARDTB", "ELEMTB", "EOSTB", "FRWDTB", "GOTOTB", "IBLKTB",
	"LBLTB", "LBLXTB", "NUMBTB", "SNABTB", "UNOPTB", "VARATB",
	// MDATA: the twelve Appendix A tables reached only by GOTO(TABLE).
	"DQLITB", "FLITB", "GOTFTB", "GOTSTB", "INTGTB", "NBLKTB",
	"NUMCTB", "SQLITB", "STARTB", "TBLKTB", "VARBTB", "VARTB",
}

func TestSegmentsDefineTheContract(t *testing.T) {
	var lines []scanner.Line
	for _, name := range copyseg.Names() {
		lines = append(lines, scanSegment(t, name)...)
	}

	stmts, ds := parser.Parse(lines)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}
	tab, ds := symtab.Collect("segments", stmts)
	if err := ds.Err(); err != nil {
		t.Fatalf("symbol collection reported diagnostics:\n%v", err)
	}

	want := slices.Sorted(slices.Values(wantDefined))
	if got := tab.Defined(); !slices.Equal(got, want) {
		t.Errorf("the segments do not define the expected contract")
		for _, extra := range missing(got, want) {
			t.Errorf("  unexpected definition of %s", extra)
		}
		for _, absent := range missing(want, got) {
			t.Errorf("  %s is not defined", absent)
		}
	}

	// A segment may only reference names its own segments define. The
	// syntax tables are ALPHSZ entries wide and the character strings
	// are sized in characters, so ALPHSZ is the one name MDATA borrows
	// from PARMS.
	if undef := tab.Undefined(); len(undef) != 0 {
		t.Errorf("the segments reference %d names nothing defines: %v", len(undef), undef)
	}
}

// A segment is SIL source, so it has to obey the column format like
// any other (S4D58 7.6). Rejoin catches a mis-typed column that the
// field extraction would otherwise absorb.
func TestSegmentsAreWellFormedSILSource(t *testing.T) {
	for _, name := range copyseg.Names() {
		t.Run(name, func(t *testing.T) {
			file, _, _ := copyseg.Source(name)
			lines := scanSegment(t, name)
			for _, l := range lines {
				if got := l.Rejoin(); got != l.Text {
					t.Errorf("%s:%d: rejoins as\n\t%q\nwant\t%q", file, l.Num, got, l.Text)
				}
			}
			t.Logf("%s: %d lines", file, len(lines))
		})
	}
}

// MLINK assembles nothing. Nothing in this machine is reached by an
// external linkage, so the segment is comment only (S4D58 6.20).
func TestMLINKIsEmpty(t *testing.T) {
	for _, l := range scanSegment(t, "MLINK") {
		if !l.Comment {
			t.Errorf("MLINK line %d is a statement: %q", l.Num, l.Text)
		}
	}
}

func TestExpand(t *testing.T) {
	src := strings.Join([]string{
		stmt("", "TITLE", "'Before'"),
		stmt("", "COPY", "MLINK\t\t   Linkage segment"),
		stmt("", "COPY", "PARMS"),
		stmt("BEGIN", "INIT", ","),
	}, "\n") + "\n"

	lines, ds := scanner.Scan("caller.sil", []byte(src))
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner reported diagnostics:\n%v", err)
	}

	var xd diag.List
	out := copyseg.ExpandWith(lines, copyseg.Source, &xd)
	if err := xd.Err(); err != nil {
		t.Fatalf("expansion reported diagnostics:\n%v", err)
	}

	for _, l := range out {
		if l.Op == copyseg.Opcode {
			t.Errorf("caller.sil:%d: a COPY survived expansion: %q", l.Num, l.Text)
		}
	}

	// The caller's own lines are still there, in order, and the
	// segment's lines cite the segment rather than the caller.
	if first, last := out[0], out[len(out)-1]; first.Op != "TITLE" || last.Op != "INIT" {
		t.Errorf("expansion returned %q ... %q, want TITLE ... INIT", first.Op, last.Op)
	}
	var files []string
	for _, l := range out {
		if len(files) == 0 || files[len(files)-1] != l.File {
			files = append(files, l.File)
		}
	}
	want := []string{"caller.sil", "mlink.sil", "parms.sil", "caller.sil"}
	if !slices.Equal(files, want) {
		t.Errorf("expanded lines come from %v, want %v", files, want)
	}
}

func TestExpandReportsAnUnknownSegment(t *testing.T) {
	lines, ds := scanner.Scan("caller.sil", []byte(stmt("", "COPY", "MPARMS")+"\n"))
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner reported diagnostics:\n%v", err)
	}

	var xd diag.List
	out := copyseg.ExpandWith(lines, copyseg.Source, &xd)
	err := xd.Err()
	if err == nil {
		t.Fatal("no diagnostic for COPY MPARMS")
	}
	if want := "COPY MPARMS: not a machine-dependent segment"; !strings.Contains(err.Error(), want) {
		t.Errorf("reported %v, want a diagnostic containing %q", err, want)
	}
	// The line stays, so later stages still see the whole file.
	if len(out) != 1 || out[0].Op != copyseg.Opcode {
		t.Errorf("the unexpanded COPY was dropped")
	}
}

func scanSegment(t *testing.T, name string) []scanner.Line {
	t.Helper()
	file, src, ok := copyseg.Source(name)
	if !ok {
		t.Fatalf("%s is not a segment; Names() returns %v", name, copyseg.Names())
	}
	lines, ds := scanner.Scan(file, src)
	if err := ds.Err(); err != nil {
		t.Fatalf("%s reported diagnostics:\n%v", file, err)
	}
	return lines
}

// stmt renders one SIL statement into its columns (S4D58 7.6).
func stmt(label, op, operand string) string {
	return strings.TrimRight(fmt.Sprintf("%-6s %-6s  %s", label, op, operand), " ")
}

// missing returns the members of a that are not in b.
func missing(a, b []string) []string {
	var out []string
	for _, s := range a {
		if !slices.Contains(b, s) {
			out = append(out, s)
		}
	}
	return out
}
