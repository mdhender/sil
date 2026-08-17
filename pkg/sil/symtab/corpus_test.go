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

package symtab_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
	"github.com/mdhender/sil/pkg/sil/symtab"
)

// The implementer's contract.
//
// These are the names the SIL source of SNOBOL4 uses but never
// defines, so they are exactly what COPY PARMS, COPY MLINK and COPY
// MDATA have to supply (S4D58 6.20). The list is derived by the
// assembler rather than transcribed from the document, and asserting
// it exactly is what makes the scanner and the parser trustworthy:
// a swallowed comment, a mis-sliced column, or a literal parsed as an
// expression would all show up here as an extra name.
//
// MLINK, PARMS and MDATA are in the list because Collect applies no
// per-operation knowledge -- COPY's operand is treated as an ordinary
// symbol reference. See Collect's documentation.
var wantUndefined = []string{
	// Machine-dependent sizes. DESCR is the central one: it appears
	// in 186 k*DESCR expressions, so nothing resolves until it is
	// chosen.
	"ALPHSZ", "CPA", "DESCR", "SIZLIM", "SPEC",
	// Flag bits (S4D58 3.1.2 -- five flags are used in SNOBOL4).
	"FNC", "MARK", "PTR", "STTL", "TTL",
	// FORTRAN unit reference numbers (S4D58 2.1).
	"UNITI", "UNITO", "UNITP",
	// Character data supplied by the machine-dependent segments.
	"ALPHA", "AMPST", "COLSTR", "QTSTR",
	// Syntax tables (S4D58 4.2, Appendix A). Thirteen are named
	// here; Appendix A defines twenty-five, the rest reachable only
	// through GOTO(TABLE).
	"BIOPTB", "CARDTB", "ELEMTB", "EOSTB", "FRWDTB", "GOTOTB", "IBLKTB",
	"LBLTB", "LBLXTB", "NUMBTB", "SNABTB", "UNOPTB", "VARATB",
	// Syntax table action codes (S4D58 4.2).
	"CONTIN", "ERROR", "STOP", "STOPSH",
	// COPY segment names (S4D58 6.20).
	"MDATA", "MLINK", "PARMS",
}

func TestUndefinedSymbolsAreExactlyTheCopySegments(t *testing.T) {
	tab, name := collectEngine(t)

	// wantUndefined is grouped by what each name is for, which is the
	// documentation; Undefined returns them sorted.
	want := slices.Sorted(slices.Values(wantUndefined))

	got := tab.Undefined()
	if !slices.Equal(got, want) {
		t.Errorf("%s: undefined symbols do not match the expected contract", name)
		for _, extra := range missing(got, want) {
			s := tab.Lookup(extra)
			t.Errorf("  unexpected undefined symbol %s, first used at line %d", extra, s.Refs[0].Line)
		}
		for _, absent := range missing(want, got) {
			t.Errorf("  expected %s to be undefined, but it is defined", absent)
		}
		t.Errorf("  got %d undefined symbols, want %d", len(got), len(want))
	}
}

// Every label in the source is defined exactly once, and every one of
// them but the entry point is used.
func TestDefinedSymbols(t *testing.T) {
	tab, name := collectEngine(t)

	if got := len(tab.Defined()); got != corpus.Labels {
		t.Errorf("got %d defined symbols, want %d", got, corpus.Labels)
	}
	if got := tab.Len(); got != corpus.Labels+len(wantUndefined) {
		t.Errorf("got %d distinct names, want %d", got, corpus.Labels+len(wantUndefined))
	}

	// BEGIN is the entry point (S4D58 6.46: "INIT is the first
	// instruction executed"), and nothing in the source branches to
	// it, so it is expected to be unreferenced. Anything else that is
	// unreferenced is worth knowing about.
	unref := tab.Unreferenced()
	if !slices.Contains(unref, "BEGIN") {
		t.Errorf("%s: BEGIN is referenced; expected the entry point to be unreferenced", name)
	}
	t.Logf("%d defined, %d undefined, %d unreferenced: %v",
		len(tab.Defined()), len(tab.Undefined()), len(unref), unref)
}

// collectEngine runs scanner, parser and symbol collection over the
// historical source and requires the first two to be silent.
func collectEngine(t *testing.T) (*symtab.Table, string) {
	t.Helper()
	name, src, err := corpus.Load()
	if err != nil {
		if errors.Is(err, corpus.ErrAbsent) {
			t.Skipf("%s: %s", corpus.Engine, corpus.SkipMessage)
		}
		t.Fatal(err)
	}

	lines, ds := scanner.Scan(name, src)
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner reported diagnostics:\n%v", err)
	}
	stmts, ds := parser.Parse(lines)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}
	tab, ds := symtab.Collect(name, stmts)
	if err := ds.Err(); err != nil {
		t.Fatalf("symbol collection reported diagnostics:\n%v", err)
	}
	return tab, name
}

// missing returns the elements of a that are not in b.
func missing(a, b []string) []string {
	var out []string
	for _, s := range a {
		if !slices.Contains(b, s) {
			out = append(out, s)
		}
	}
	return out
}
