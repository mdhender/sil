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
	"slices"
	"testing"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
	"github.com/mdhender/sil/pkg/sil/symtab"
)

// collect runs the whole front end over a fragment written in real
// SIL column positions.
func collect(t *testing.T, src string) (*symtab.Table, diag.List) {
	t.Helper()
	lines, ds := scanner.Scan("t.sil", []byte(src))
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	stmts, ds := parser.Parse(lines)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser: %v", err)
	}
	return symtab.Collect("t.sil", stmts)
}

func TestCollect(t *testing.T) {
	tab, ds := collect(t,
		//234567890123456789
		"ATTRIB EQU     2*DESCR\n"+
			"START  BRANCH  ATTRIB\n"+
			"       RCALL   ,GC,(ARG1CL),(ALOC2,START)\n")
	if err := ds.Err(); err != nil {
		t.Fatalf("unexpected diagnostics:\n%v", err)
	}

	if got, want := tab.Defined(), []string{"ATTRIB", "START"}; !slices.Equal(got, want) {
		t.Errorf("Defined() = %v, want %v", got, want)
	}
	// GC, ARG1CL, ALOC2 and DESCR are used and never defined. Names
	// inside a parenthesised list are references like any other.
	if got, want := tab.Undefined(), []string{"ALOC2", "ARG1CL", "DESCR", "GC"}; !slices.Equal(got, want) {
		t.Errorf("Undefined() = %v, want %v", got, want)
	}
	// START is referenced once, from inside the return list. Its own
	// line defines it rather than referencing it.
	if s := tab.Lookup("START"); len(s.Refs) != 1 {
		t.Errorf("START has %d references, want 1", len(s.Refs))
	}
	if s := tab.Lookup("START"); s.DefLine != 2 {
		t.Errorf("START defined at line %d, want 2", s.DefLine)
	}
}

// A character literal is data, not a name. If literals leaked into the
// symbol table, STRING and FORMAT operands would fill it with
// nonsense.
func TestLiteralsAreNotReferences(t *testing.T) {
	tab, ds := collect(t,
		"ARRSP  STRING  'ARRAY'\n"+
			"EJECTF FORMAT  '(1H1)'\n")
	if err := ds.Err(); err != nil {
		t.Fatalf("unexpected diagnostics:\n%v", err)
	}
	if got := tab.Undefined(); len(got) != 0 {
		t.Errorf("Undefined() = %v, want none", got)
	}
	if got, want := tab.Len(), 2; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
}

func TestDuplicateDefinition(t *testing.T) {
	_, ds := collect(t,
		"SAME   EQU     1\n"+
			"SAME   EQU     2\n")
	want := "t.sil:2:1: SAME is already defined at line 1"
	if len(ds) != 1 || ds[0].String() != want {
		t.Errorf("got %v, want [%s]", ds, want)
	}
}

// One diagnostic per undefined name, at its first use, rather than one
// per use.
func TestReportUndefined(t *testing.T) {
	tab, _ := collect(t,
		"       BRANCH  NOWHER\n"+
			"       BRANCH  NOWHER\n"+
			"       BRANCH  ONCE\n")
	var ds diag.List
	tab.ReportUndefined(&ds)

	want := []string{
		"t.sil:1:16: undefined symbol NOWHER, used here and 1 more times",
		"t.sil:3:16: undefined symbol ONCE",
	}
	var got []string
	for _, d := range ds {
		got = append(got, d.String())
	}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// A label and an operation may share a name, and the label field is
// the only thing that defines. COPY is both here: a procedure named
// COPY, and the COPY operation whose operand is the segment MLINK.
func TestOperationNameIsNotAReference(t *testing.T) {
	tab, ds := collect(t,
		"COPY   PROC    ,\n"+
			"       COPY    MLINK\n")
	if err := ds.Err(); err != nil {
		t.Fatalf("unexpected diagnostics:\n%v", err)
	}
	if got, want := tab.Defined(), []string{"COPY"}; !slices.Equal(got, want) {
		t.Errorf("Defined() = %v, want %v", got, want)
	}
	// PROC is an operation, so it is not a reference; MLINK is an
	// operand, so it is one.
	if got, want := tab.Undefined(), []string{"MLINK"}; !slices.Equal(got, want) {
		t.Errorf("Undefined() = %v, want %v", got, want)
	}
}
