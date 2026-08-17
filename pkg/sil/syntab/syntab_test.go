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

package syntab_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil/syntab"
)

// The twenty-five tables of Appendix A, in the order the appendix
// gives them, which is alphabetical.
var wantNames = []string{
	"BIOPTB", "CARDTB", "DQLITB", "ELEMTB", "EOSTB", "FLITB", "FRWDTB",
	"GOTFTB", "GOTOTB", "GOTSTB", "IBLKTB", "INTGTB", "LBLTB", "LBLXTB",
	"NBLKTB", "NUMBTB", "NUMCTB", "SNABTB", "SQLITB", "STARTB", "TBLKTB",
	"UNOPTB", "VARATB", "VARBTB", "VARTB",
}

func TestTheAppendixParses(t *testing.T) {
	if got := syntab.Names(); !slices.Equal(got, wantNames) {
		t.Errorf("the tables are\n\t%v\nwant\n\t%v", got, wantNames)
	}
	for _, table := range syntab.Tables {
		if len(table.Rules) == 0 {
			t.Errorf("%s has no FOR lines", table.Name)
		}
		if table.Else.Stop == "" {
			t.Errorf("%s has no ELSE", table.Name)
		}
	}
	// 4.2's list of the thirteen the SNOBOL4 source names and the
	// twelve reached only through GOTO(TABLE) is the same set, so
	// every GOTO names a table that is here.
	for _, table := range syntab.Tables {
		for _, r := range append(slices.Clone(table.Rules), table.Else) {
			if r.Goto == "" {
				continue
			}
			if _, ok := syntab.Lookup(r.Goto); !ok {
				t.Errorf("%s: GOTO(%s) names no table", table.Name, r.Goto)
			}
		}
	}
}

// Every class a description names is one 4.1 defines, and every class
// 4.1 defines is named by some description. The second half is what
// catches a class transcribed under the wrong name.
func TestTheClassesAndTheDescriptionsAgree(t *testing.T) {
	used := map[string]bool{}
	for _, table := range syntab.Tables {
		for _, r := range table.Rules {
			for _, c := range r.Classes {
				if _, ok := syntab.Classes[c]; !ok {
					t.Errorf("%s: %s is not a class of 4.1", table.Name, c)
				}
				used[c] = true
			}
		}
	}
	for name := range syntab.Classes {
		if !used[name] {
			t.Errorf("4.1's class %s is named by no description", name)
		}
	}
	if len(syntab.Classes) != 35 {
		t.Errorf("4.1 defines %d classes, want 35", len(syntab.Classes))
	}
}

// stub resolves every name to a distinct number, so that Build can be
// exercised without an assembly.
func stub() (func(string) (int, bool), map[string]int) {
	values := map[string]int{"CONTIN": 0, "STOP": 1, "STOPSH": 2, "ERROR": 3}
	next := 100
	return func(name string) (int, bool) {
		if v, ok := values[name]; ok {
			return v, true
		}
		next++
		values[name] = next
		return next, true
	}, values
}

// No table names two classes that share a character. Build refuses one
// that does, so building all twenty-five is the check.
func TestNoTableClaimsACharacterTwice(t *testing.T) {
	value, _ := stub()
	for _, name := range syntab.Names() {
		if _, err := syntab.Build(name, 256, value); err != nil {
			t.Errorf("%v", err)
		}
	}
}

// INTGTB in full, because it is the one table with all four kinds of
// line: a CONTIN, a STOPSH with a PUT, a GOTO with a PUT, and an ELSE.
//
//	BEGIN INTGTB
//	FOR(NUMBER) CONTIN
//	FOR(TERMINATOR) PUT(ILITYP) STOPSH
//	FOR(DOT) PUT(FLITYP) GOTO(FLITB)
//	ELSE ERROR
//	END INTGTB
func TestBuildExpandsATable(t *testing.T) {
	value, values := stub()
	entries, err := syntab.Build("INTGTB", 256, value)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 256 {
		t.Fatalf("%d entries, want 256", len(entries))
	}

	for _, tt := range []struct {
		char byte
		want syntab.Entry
	}{
		// CONTIN is the table's own address, and no PUT.
		{'7', syntab.Entry{Next: values["INTGTB"], Indicator: 0}},
		// TERMINATOR, with a PUT and no next table.
		{';', syntab.Entry{Indicator: 2, Put: values["ILITYP"]}},
		{' ', syntab.Entry{Indicator: 2, Put: values["ILITYP"]}},
		{'\t', syntab.Entry{Indicator: 2, Put: values["ILITYP"]}},
		// DOT, which goes somewhere else.
		{'.', syntab.Entry{Next: values["FLITB"], Indicator: 0, Put: values["FLITYP"]}},
		// Everything else is the ELSE.
		{'A', syntab.Entry{Indicator: 3}},
		{0, syntab.Entry{Indicator: 3}},
		{255, syntab.Entry{Indicator: 3}},
	} {
		if got := entries[tt.char]; got != tt.want {
			t.Errorf("the entry for %d is %+v, want %+v", tt.char, got, tt.want)
		}
	}
}

// A name the assembly does not define is a diagnostic, not a zero.
func TestBuildNeedsItsSymbols(t *testing.T) {
	value := func(name string) (int, bool) { return 0, name != "ILITYP" }
	if _, err := syntab.Build("INTGTB", 256, value); err == nil {
		t.Error("no error with ILITYP undefined")
	}
	if _, err := syntab.Build("NOSUCH", 256, value); err == nil {
		t.Error("no error for a table Appendix A does not describe")
	}
}

// The transcription, read against the manual. The extraction has page
// numbers and rules interleaved with the descriptions, so what is
// compared is the lines that carry the description language -- which
// is every line of the appendix that says anything.
func TestAppendixMatchesTheDocument(t *testing.T) {
	name, src, err := corpus.LoadManual()
	if errors.Is(err, corpus.ErrAbsent) {
		t.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		t.Fatal(err)
	}

	// 4.2 prints IBLKTB and VARBTB as examples before the appendix,
	// and 6.28 documents a directive called END, so the extraction is
	// cut to the appendix itself.
	document := appendixOnly(descriptionLines(string(src)))
	if len(document) == 0 {
		t.Fatalf("%s: no syntax table descriptions in it", name)
	}
	transcribed := descriptionLines(syntab.Appendix)

	if len(document) != len(transcribed) {
		t.Fatalf("the manual has %d description lines and the transcription has %d",
			len(document), len(transcribed))
	}
	for i := range document {
		if document[i] != transcribed[i] {
			t.Errorf("line %d of the appendix is\n\t%q\ntranscribed as\n\t%q",
				i+1, document[i], transcribed[i])
		}
	}
	t.Logf("%s: %d description lines", name, len(document))
}

// appendixOnly cuts a run of description lines down to Appendix A,
// which begins at the first table and ends at the last.
func appendixOnly(lines []string) []string {
	from := slices.Index(lines, "BEGIN "+wantNames[0])
	to := slices.Index(lines, "END "+wantNames[len(wantNames)-1])
	if from < 0 || to < from {
		return nil
	}
	return lines[from : to+1]
}

// descriptionLines returns the BEGIN, FOR, ELSE and END lines of a
// text, with their spacing normalized. Anything else -- a page number,
// a rule, the appendix title -- says nothing about a table.
func descriptionLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch head := fields[0]; {
		case head == "BEGIN", head == "END", head == "ELSE",
			strings.HasPrefix(head, "FOR("):
			out = append(out, strings.Join(fields, " "))
		}
	}
	return out
}
