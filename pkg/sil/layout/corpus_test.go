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
	"errors"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil/copyseg"
	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/layout"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
	"github.com/mdhender/sil/pkg/sil/symtab"
)

// TestEngineLayoutCloses is the milestone: with the three COPY
// segments in, the SNOBOL4 source has no undefined symbol left and
// every expression in it evaluates.
func TestEngineLayoutCloses(t *testing.T) {
	stmts, l := layEngine(t)

	tab, ds := symtab.Collect(corpus.Engine, stmts)
	if err := ds.Err(); err != nil {
		t.Fatalf("symbol collection reported diagnostics:\n%v", err)
	}
	if undef := tab.Undefined(); len(undef) != 0 {
		t.Errorf("got %d undefined symbols, want none: %v", len(undef), undef)
	}

	descr, spec, cpa := l.Params()
	t.Logf("DESCR=%d SPEC=%d CPA=%d, %d statements in %d address units, %d symbols",
		descr, spec, cpa, len(stmts), l.End(), len(l.Symbols()))
}

// The four identities of PLAN.md's M3. Each is stated in the SNOBOL4
// source as an EQU or an assembled address, and each is checked here
// against something computed from the shape of the source rather than
// from the expression that defines it. Asserting PRMSIZ == 18 would
// only prove that arithmetic works; counting the descriptors between
// PRMTBL and PRMTRM and finding that PRMSIZ is that many DESCRs proves
// that DESCR-sized things were laid down DESCR apart.
//
// None of the four mentions a number that would change if DESCR, SPEC
// or CPA were chosen differently, which is the point: the same test
// has to pass for the CPA=4 run PLAN.md schedules later, and it does
// -- see TestIdentitiesHoldForOtherMachineParameters below.
//
// What they do not cover is SPEC. The width of a specifier reaches the
// location counter only through STRING, and nothing in the SNOBOL4
// source relates two addresses across a run of strings except BUFEXT,
// which measures that run with the same sizes it is being checked
// against. Assembling STRING as a descriptor plus its characters
// instead of a specifier plus its characters passes every test here.
// The first thing that can catch it is a program that runs: an APDSP
// or an STPRNT reading the specifier a STRING laid down, which is M5.

// PRMSIZ EQU PRMTRM-PRMTBL-DESCR  (line 6336)
//
// PRMTBL is the title of the block of basic-block pointers and PRMTRM
// is the LHERE after the last of them, so PRMSIZ is the size of the
// block not counting its title.
func TestPRMSIZIsTheBasicBlockList(t *testing.T) {
	stmts, l := layEngine(t)
	descr, _, _ := l.Params()

	entries := countBetween(t, stmts, "PRMTBL", "PRMTRM", "DESCR")
	if entries == 0 {
		t.Fatal("found no descriptors between PRMTBL and PRMTRM")
	}
	// The title is the first of them and PRMSIZ excludes it.
	want := (entries - 1) * descr
	if got := value(t, l, "PRMSIZ"); got != want {
		t.Errorf("PRMSIZ = %d, want %d: %d descriptors from PRMTBL to PRMTRM, less the title, times DESCR=%d",
			got, want, entries, descr)
	}
	// PRMDX carries the same size into the address field of a
	// descriptor the garbage collector reads (line 5792).
	if got := stmtAddrField(t, stmts, l, "PRMDX"); got != want {
		t.Errorf("PRMDX addresses %d, want PRMSIZ = %d", got, want)
	}
}

// OBLIST EQU OBSTRT-LNKFLD  (line 6343), LNKFLD EQU 3*DESCR (line 231)
//
// OBLIST is a fake string structure whose link field is the first bin,
// so it has to sit LNKFLD below OBSTRT, and the three descriptors of
// OBLOCK's "pseudo heading" (line 6341) are exactly the room that
// takes. If ARRAY 3 assembled anything other than 3*DESCR, OBLIST
// would fall outside the block and the collector would walk off it.
func TestOBLISTIsTheBlocksPseudoLink(t *testing.T) {
	_, l := layEngine(t)
	descr, _, _ := l.Params()

	obstrt, oblock, oblist := value(t, l, "OBSTRT"), value(t, l, "OBLOCK"), value(t, l, "OBLIST")

	if got, want := obstrt-oblist, value(t, l, "LNKFLD"); got != want {
		t.Errorf("OBSTRT-OBLIST = %d, want LNKFLD = %d", got, want)
	}
	if got, want := obstrt-oblist, 3*descr; got != want {
		t.Errorf("OBSTRT-OBLIST = %d, want 3*DESCR = %d", got, want)
	}
	if got, want := oblist, oblock+descr; got != want {
		t.Errorf("OBLIST = %d, want OBLOCK+DESCR = %d: the pseudo link must land on the first cell after the block title", got, want)
	}
}

// OBEND DESCR OBLIST+DESCR*OBOFF,0,0  (line 5475)
//
// OBEND is the sentinel of the bin walk at lines 675-681:
//
//	SETAC   BKPTR,OBLIST-DESCR
//	GCBA1  ACOMP   BKPTR,OBEND,GCLAD
//	INCRA   BKPTR,DESCR
//
// so the walk runs BKPTR from OBLIST-DESCR up to and including OBEND,
// stepping by DESCR, and reads the bin at BKPTR+LNKFLD each time round.
// Counting those steps has to give exactly OBSIZ bins, which is the
// one statement of this identity that does not restate its own
// definition. S4D58 6.74 gives the same fact from the other end: the
// last bin is at OBSTRT+(OBSIZ-1)*D.
func TestOBENDStopsTheBinWalkAfterOBSIZBins(t *testing.T) {
	stmts, l := layEngine(t)
	descr, _, _ := l.Params()

	obend := stmtAddrField(t, stmts, l, "OBEND")
	first := value(t, l, "OBLIST") - descr

	bins := (obend-first)/descr + 1
	if got, want := bins, value(t, l, "OBSIZ"); got != want {
		t.Errorf("the bin walk covers %d bins, want OBSIZ = %d", got, want)
	}
	if got, want := obend+descr+value(t, l, "LNKFLD"), value(t, l, "OBSTRT")+(value(t, l, "OBSIZ")-1)*descr; got != want {
		t.Errorf("the last bin the walk reads is at %d, want OBSTRT+(OBSIZ-1)*DESCR = %d (S4D58 6.74)", got, want)
	}
}

// BUFLEN EQU BUFEXT*CPA  (line 6312), BUFEXT EQU DTEND-ANYSP (6311)
//
// The initialization data from ANYSP to DTEND is reused as a scratch
// buffer once the system is running, and BUFLEN is how many characters
// fit in it -- the length every APDSP into it is checked against
// (ACOMPC TCL,BUFLEN,FXOVR at line 4011 and five other places). So
// BUFLEN has to be the character capacity of exactly the statements in
// that span, which is the sum of their sizes rather than the
// difference of two symbols.
func TestBUFLENIsTheInitializationDataRegion(t *testing.T) {
	stmts, l := layEngine(t)
	_, _, cpa := l.Params()

	units := sizeBetween(t, stmts, l, "ANYSP", "DTEND")
	if units == 0 {
		t.Fatal("found no statements between ANYSP and DTEND")
	}
	if got := value(t, l, "BUFEXT"); got != units {
		t.Errorf("BUFEXT = %d, want %d address units assembled from ANYSP to DTEND", got, units)
	}
	if got, want := value(t, l, "BUFLEN"), units*cpa; got != want {
		t.Errorf("BUFLEN = %d, want %d characters", got, want)
	}
	t.Logf("BUFLEN = %d characters over %d address units", value(t, l, "BUFLEN"), units)
}

// The relocatable/absolute discipline is enforced by layout.Run over
// every operand of every statement, so a clean run of the whole source
// is the check. This test says what the clean run proves.
func TestAddressArithmeticIsWellFormedThroughout(t *testing.T) {
	stmts, l := layEngine(t)

	// A count is a number and an address is an address, and the source
	// says so consistently: the offsets are absolute even though they
	// are written in terms of DESCR, and the labels are relocatable
	// even when they name data.
	for _, name := range []string{"DESCR", "SPEC", "CPA", "LNKFLD", "OBOFF", "OBSIZ", "PRMSIZ", "BUFEXT", "BUFLEN"} {
		if v, ok := l.Value(name); !ok || v.Reloc {
			t.Errorf("%s = %v, want a number", name, v)
		}
	}
	for _, name := range []string{"BEGIN", "PRMTBL", "PRMTRM", "OBLOCK", "OBSTRT", "OBLIST", "STACK", "ANYSP", "DTEND"} {
		if v, ok := l.Value(name); !ok || !v.Reloc {
			t.Errorf("%s = %v, want an address", name, v)
		}
	}

	// Every statement was placed, in order, with no gap.
	next := layout.Origin
	for i := range stmts {
		p := l.Placement(i)
		if p.Addr != next {
			t.Fatalf("%s:%d: placed at %d, want %d", stmts[i].File, stmts[i].Num, p.Addr, next)
		}
		next = p.Addr + p.Size
	}
	if next != l.End() {
		t.Errorf("the last statement ends at %d, want End() = %d", next, l.End())
	}
}

// Nothing above may depend on DESCR being 1, SPEC being 2 or CPA
// being 1. This reruns the whole front end on a machine where a
// descriptor is two address units wide, a specifier four, and four
// characters pack into one unit, and repeats every identity.
//
// PLAN.md schedules this as the check that nothing silently assumed
// CPA=1. It is cheap here and it is the only way to tell an identity
// that holds from one that happens to hold.
func TestIdentitiesHoldForOtherMachineParameters(t *testing.T) {
	const wideParms = `*      A machine with wide descriptors and packed characters.
DESCR  EQU     2
SPEC   EQU     4
CPA    EQU     4
ALPHSZ EQU     256
SIZLIM EQU     16777215
TTL    EQU     1
MARK   EQU     2
PTR    EQU     4
FNC    EQU     8
STTL   EQU     16
UNITI  EQU     5
UNITO  EQU     6
UNITP  EQU     7
CONTIN EQU     0
STOP   EQU     1
STOPSH EQU     2
ERROR  EQU     3
`
	resolve := func(name string) (string, []byte, bool) {
		if name == "PARMS" {
			return "wide PARMS", []byte(wideParms), true
		}
		return copyseg.Source(name)
	}

	stmts, l := layEngineWith(t, resolve)
	descr, spec, cpa := l.Params()
	if descr != 2 || spec != 4 || cpa != 4 {
		t.Fatalf("DESCR=%d SPEC=%d CPA=%d, want 2, 4 and 4", descr, spec, cpa)
	}
	t.Logf("DESCR=%d SPEC=%d CPA=%d, %d address units (%d with the assembled parameters)",
		descr, spec, cpa, l.End(), 16506)

	entries := countBetween(t, stmts, "PRMTBL", "PRMTRM", "DESCR")
	if got, want := value(t, l, "PRMSIZ"), (entries-1)*descr; got != want {
		t.Errorf("PRMSIZ = %d, want %d", got, want)
	}
	if got, want := value(t, l, "OBSTRT")-value(t, l, "OBLIST"), 3*descr; got != want {
		t.Errorf("OBSTRT-OBLIST = %d, want 3*DESCR = %d", got, want)
	}
	if got, want := value(t, l, "OBLIST"), value(t, l, "OBLOCK")+descr; got != want {
		t.Errorf("OBLIST = %d, want OBLOCK+DESCR = %d", got, want)
	}
	obend := stmtAddrField(t, stmts, l, "OBEND")
	bins := (obend-(value(t, l, "OBLIST")-descr))/descr + 1
	if got, want := bins, value(t, l, "OBSIZ"); got != want {
		t.Errorf("the bin walk covers %d bins, want OBSIZ = %d", got, want)
	}
	units := sizeBetween(t, stmts, l, "ANYSP", "DTEND")
	if got := value(t, l, "BUFEXT"); got != units {
		t.Errorf("BUFEXT = %d, want %d address units", got, units)
	}
	if got, want := value(t, l, "BUFLEN"), units*cpa; got != want {
		t.Errorf("BUFLEN = %d, want %d characters", got, want)
	}
}

// value returns a symbol's value, failing the test if it has none.
func value(t *testing.T, l *layout.Layout, name string) int {
	t.Helper()
	n, ok := l.Addr(name)
	if !ok {
		t.Fatalf("%s has no value", name)
	}
	return n
}

// countBetween counts the statements with the given operation from the
// one labelled from up to but not including the one labelled to.
func countBetween(t *testing.T, stmts []parser.Statement, from, to, op string) int {
	t.Helper()
	n := 0
	for _, s := range span(t, stmts, from, to) {
		if s.Op == op {
			n++
		}
	}
	return n
}

// sizeBetween adds up the address units assembled from the statement
// labelled from up to but not including the one labelled to.
func sizeBetween(t *testing.T, stmts []parser.Statement, l *layout.Layout, from, to string) int {
	t.Helper()
	lo, hi := bounds(t, stmts, from, to)
	units := 0
	for i := lo; i < hi; i++ {
		units += l.Placement(i).Size
	}
	return units
}

func span(t *testing.T, stmts []parser.Statement, from, to string) []parser.Statement {
	t.Helper()
	lo, hi := bounds(t, stmts, from, to)
	return stmts[lo:hi]
}

func bounds(t *testing.T, stmts []parser.Statement, from, to string) (int, int) {
	t.Helper()
	lo, hi := -1, -1
	for i, s := range stmts {
		switch s.Label {
		case from:
			lo = i
		case to:
			hi = i
		}
	}
	if lo < 0 {
		t.Fatalf("no statement is labelled %s", from)
	}
	if hi < 0 {
		t.Fatalf("no statement is labelled %s", to)
	}
	if hi < lo {
		t.Fatalf("%s at statement %d comes after %s at %d", to, hi, from, lo)
	}
	return lo, hi
}

// stmtAddrField returns the value of the address field of the
// descriptor assembled by the statement with the given label.
func stmtAddrField(t *testing.T, stmts []parser.Statement, l *layout.Layout, label string) int {
	t.Helper()
	for _, s := range stmts {
		if s.Label != label {
			continue
		}
		if s.Op != "DESCR" || len(s.Operands) == 0 || s.Operands[0].Kind != parser.ItemExpr {
			t.Fatalf("%s:%d: %s is not a descriptor with an address field", s.File, s.Num, label)
		}
		v, err := l.Evaluate(s.Operands[0].Expr)
		if err != nil {
			t.Fatalf("%s:%d: %s: %v", s.File, s.Num, label, err)
		}
		return v.N
	}
	t.Fatalf("no statement is labelled %s", label)
	return 0
}

// layEngine runs the whole front end over the historical source: scan,
// expand the COPY segments, parse, then place and resolve. Every stage
// must be silent.
func layEngine(t *testing.T) ([]parser.Statement, *layout.Layout) {
	t.Helper()
	return layEngineWith(t, copyseg.Source)
}

func layEngineWith(t *testing.T, resolve copyseg.Resolver) ([]parser.Statement, *layout.Layout) {
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

	var xd diag.List
	lines = copyseg.ExpandWith(lines, resolve, &xd)
	if err := xd.Err(); err != nil {
		t.Fatalf("COPY expansion reported diagnostics:\n%v", err)
	}

	stmts, ds := parser.Parse(lines)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}

	l, ds := layout.Run(stmts)
	if err := ds.Err(); err != nil {
		t.Fatalf("layout reported diagnostics:\n%v", err)
	}
	return stmts, l
}
