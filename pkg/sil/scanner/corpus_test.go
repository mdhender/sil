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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/scanner"
)

// engineSource is the historical Macro SNOBOL4 implementation. It is
// not redistributable, so it is gitignored and may be absent from a
// checkout; the corpus tests skip on the file itself rather than on an
// environment variable or a build tag, so that the skip expires by
// itself when the file arrives. A skip and a pass are different
// things -- check which one you are looking at.
const engineSource = "engines/sil-v3.11.sil"

// Counts measured over the whole of sil-v3.11.sil. They are the exit
// criterion for the scanner: the scanner reproduces the numbers or it
// has misread the columns somewhere in 6580 lines.
const (
	wantLines    = 6580
	wantComments = 1748
	wantStmts    = 4832
	wantLabels   = 1624
)

// Labels are one to six characters, a letter followed by letters or
// digits. S4D58 7.6 says two to six, but the data type codes at the
// head of the source (S B P A T I R C N K E L) are one character
// each, so the document is wrong and the source wins.
var labelRe = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,5}$`)

func TestScanEngineSource(t *testing.T) {
	name, src := loadEngine(t)

	lines, ds := scanner.Scan(name, src)
	if err := ds.Err(); err != nil {
		t.Fatalf("%s: scanner reported diagnostics:\n%v", name, err)
	}
	if len(lines) != wantLines {
		t.Fatalf("got %d lines, want %d", len(lines), wantLines)
	}

	var comments, stmts int
	labels := make(map[string]int) // label -> first line that defined it
	for _, l := range lines {
		if l.Comment {
			comments++
			continue
		}
		stmts++

		if l.Op == "" {
			t.Errorf("%s:%d: statement with no operation field", name, l.Num)
		}
		if l.Labeled() {
			if !labelRe.MatchString(l.Label) {
				t.Errorf("%s:%d: label %q does not match %s", name, l.Num, l.Label, labelRe)
			}
			if first, dup := labels[l.Label]; dup {
				t.Errorf("%s:%d: label %q already defined on line %d", name, l.Num, l.Label, first)
			} else {
				labels[l.Label] = l.Num
			}
		}
		// Rejoin pads the operation field out to column 15 and then
		// trims, which is exact only because no statement in this
		// source carries trailing blanks. Assert that rather than
		// assume it.
		if trimmed := strings.TrimRight(l.Text, " \t"); trimmed != l.Text {
			t.Errorf("%s:%d: statement has trailing whitespace: %q", name, l.Num, l.Text)
		}
	}

	if comments != wantComments {
		t.Errorf("got %d comment lines, want %d", comments, wantComments)
	}
	if stmts != wantStmts {
		t.Errorf("got %d statements, want %d", stmts, wantStmts)
	}
	if len(labels) != wantLabels {
		t.Errorf("got %d distinct labels, want %d", len(labels), wantLabels)
	}
}

// Every line, comment or statement, must rebuild from its located
// fields. This is what proves the field boundaries are right: a
// scanner that swallowed a column or misplaced the comment would
// still produce plausible-looking fields, but it could not put the
// line back together.
func TestRejoinEngineSource(t *testing.T) {
	name, src := loadEngine(t)

	lines, ds := scanner.Scan(name, src)
	if err := ds.Err(); err != nil {
		t.Fatalf("%s: scanner reported diagnostics:\n%v", name, err)
	}

	bad := 0
	for _, l := range lines {
		if got := l.Rejoin(); got != l.Text {
			bad++
			if bad <= 10 {
				t.Errorf("%s:%d: Rejoin mismatch\n got %q\nwant %q", name, l.Num, got, l.Text)
			}
		}
	}
	if bad > 10 {
		t.Errorf("%s: %d lines failed to rejoin (showing the first 10)", name, bad)
	}
}

// loadEngine returns the path and contents of the historical source,
// skipping the test when it is not in this checkout.
func loadEngine(t *testing.T) (string, []byte) {
	t.Helper()
	name := filepath.Join(moduleRoot(t), engineSource)
	src, err := os.ReadFile(name)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("%s: not present in this checkout; see references/MANIFEST.md for where to obtain it", engineSource)
		}
		t.Fatalf("%s: %v", name, err)
	}
	return name, src
}

// moduleRoot walks up from the package directory to the directory
// holding go.mod, because tests run with the package directory as
// their working directory.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}
