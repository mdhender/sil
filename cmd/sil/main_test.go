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

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdhender/sil/engines"
)

// The program used throughout. -UNLIST turns the compilation listing
// off, which otherwise shares the STPRNT path and so standard output
// with the program's own printing.
const hello = "-UNLIST\n" +
	"        X = 'HELLO'\n" +
	"        OUTPUT = X\n" +
	"END\n"

// needsEngine skips when the SIL source was not in the tree when this
// binary was built. See engines/README.md.
func needsEngine(t *testing.T) {
	t.Helper()
	if _, _, err := engines.Source(); errors.Is(err, engines.ErrAbsent) {
		t.Skip("the SIL source of SNOBOL4 is not embedded in this build")
	}
}

// write puts a file in the test's own directory and returns its path.
func write(t *testing.T, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The whole of it: a SNOBOL4 program named on the command line runs,
// and standard output is what it printed and nothing else.
func TestRunsAProgram(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	status, err := run([]string{write(t, "hello.sno", hello)}, &out, &errs)
	if err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if status != 0 {
		t.Errorf("exit status %d, want 0", status)
	}
	if got, want := out.String(), "HELLO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
	// The system's own output goes the other way, and says the
	// program compiled.
	if !strings.Contains(errs.String(), "NO ERRORS DETECTED IN SOURCE PROGRAM") {
		t.Errorf("standard error does not say the program compiled:\n%s", errs.String())
	}
	if strings.Contains(out.String(), "SNOBOL4") {
		t.Errorf("the banner reached standard output:\n%s", out.String())
	}
}

// The system's own output is typeset, not printed as the format
// strings it was given. This is the whole of pkg/fortran seen from the
// outside: the banner is Hollerith text under carriage control, the
// statistics are Iw fields in their columns.
func TestTheSystemsOutputIsTypeset(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	if _, err := run([]string{write(t, "hello.sno", hello)}, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	for _, want := range []string{
		// 37H1SNOBOL4 ... : the 1 is a new page and is not printed.
		"\fSNOBOL4 (VERSION 3.11, MAY 19, 1975)",
		// The underline the banner overprints.
		"\n_______\n",
		// 1H0,I15,...: a blank line, then the count in fifteen
		// columns, then the Hollerith text.
		"\n              1 WRITES PERFORMED\n",
		"\n              2 STATEMENTS EXECUTED,       0 FAILED\n",
	} {
		if !strings.Contains(errs.String(), want) {
			t.Errorf("standard error does not have %q:\n%s", want, errs.String())
		}
	}
	// A format string reaching the output would mean it was not read.
	if strings.Contains(errs.String(), "1H0") || strings.Contains(errs.String(), "I15") {
		t.Errorf("a format string reached the output:\n%s", errs.String())
	}
}

// PUNCH goes to unit 7 under (80A1), where there is no carriage
// control and the first column is data like any other. Getting that
// wrong eats the first character of every punched record.
func TestThePunchHasNoCarriageControl(t *testing.T) {
	needsEngine(t)

	program := "-UNLIST\n" +
		"        PUNCH = '1ONE'\n" +
		"        OUTPUT = '1TWO'\n" +
		"END\n"

	var out, errs bytes.Buffer
	if _, err := run([]string{write(t, "punch.sno", program)}, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	// The punched record keeps its 1; the printed one goes under
	// (1X,132A1), whose leading blank is the carriage control, so its
	// 1 is text as well.
	if got, want := out.String(), "1ONE\n1TWO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
}

// With no files the program comes from standard input, which is what a
// pipeline gives it.
func TestReadsTheProgramFromStandardInput(t *testing.T) {
	needsEngine(t)

	in, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := in.WriteString(hello); err != nil {
		t.Fatal(err)
	}
	if _, err := in.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	saved := os.Stdin
	os.Stdin = in
	defer func() { os.Stdin = saved }()

	var out, errs bytes.Buffer
	if _, err := run(nil, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if got, want := out.String(), "HELLO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
}

// Several files are one deck, read one after another, and a file that
// does not end in a newline does not run into the next one.
func TestSeveralFilesAreOneDeck(t *testing.T) {
	needsEngine(t)

	first := write(t, "first.sno", "-UNLIST\n        OUTPUT = 'ONE'")
	second := write(t, "second.sno", "        OUTPUT = 'TWO'\nEND\n")

	var out, errs bytes.Buffer
	if _, err := run([]string{first, second}, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if got, want := out.String(), "ONE\nTWO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
}

// -merge puts the system's output on standard output, in the order the
// original printed it: the banner, then what the program printed, then
// the statistics.
func TestMergePutsBothOnStandardOutput(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	if _, err := run([]string{"-merge", write(t, "hello.sno", hello)}, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	text := out.String()
	banner := strings.Index(text, "SNOBOL4 (VERSION 3.11")
	printed := strings.Index(text, "HELLO")
	stats := strings.Index(text, "STATISTICS SUMMARY")
	if banner < 0 || printed < 0 || stats < 0 {
		t.Fatalf("standard output is missing one of the three:\n%s", text)
	}
	if !(banner < printed && printed < stats) {
		t.Errorf("out of order: banner at %d, HELLO at %d, statistics at %d", banner, printed, stats)
	}
	if errs.Len() != 0 {
		t.Errorf("standard error is not empty:\n%s", errs.String())
	}
}

// -listing and -trace are the two ways of looking at a run: one is the
// core the assembler laid out, the other is what the machine did to
// it.
func TestListingAndTrace(t *testing.T) {
	needsEngine(t)

	dir := t.TempDir()
	listing := filepath.Join(dir, "core.txt")
	trace := filepath.Join(dir, "trace.txt")

	var out, errs bytes.Buffer
	_, err := run([]string{
		"-listing", listing, "-trace", trace, write(t, "hello.sno", hello),
	}, &out, &errs)
	if err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}

	for _, tt := range []struct {
		path string
		want string
	}{
		// 6.46 makes INIT the first instruction executed, and the
		// source puts it at BEGIN, the bottom of core.
		{listing, "INIT"},
		{trace, "INIT"},
	} {
		text, err := os.ReadFile(tt.path)
		if err != nil {
			t.Fatal(err)
		}
		if len(text) == 0 {
			t.Errorf("%s is empty", tt.path)
			continue
		}
		if !strings.Contains(string(text), tt.want) {
			t.Errorf("%s does not mention %s", tt.path, tt.want)
		}
	}
}

// -max stops a program that will not stop by itself, and says so
// rather than hanging.
func TestMaxStopsARunawayProgram(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	_, err := run([]string{"-max", "1000", write(t, "hello.sno", hello)}, &out, &errs)
	if err == nil {
		t.Fatal("no error from a run cut short")
	}
	if !strings.Contains(err.Error(), "without halting") {
		t.Errorf("reported %v", err)
	}
}

// -engine assembles another SIL source instead of the embedded one,
// which is also how a plain SIL program is run.
func TestEngineNamesAnotherSource(t *testing.T) {
	var out, errs bytes.Buffer
	if _, err := run([]string{"-engine", filepath.Join(t.TempDir(), "nope.sil")}, &out, &errs); err == nil {
		t.Error("no error for a source that is not there")
	}

	// A SIL source that will not assemble is reported with its
	// diagnostics rather than run.
	bad := write(t, "bad.sil", "BEGIN  NOSUCH  ,\n       END\n")
	if _, err := run([]string{"-engine", bad}, &out, &errs); err == nil {
		t.Error("no error for a source that does not assemble")
	} else if !strings.Contains(err.Error(), "diagnostics") {
		t.Errorf("reported %v", err)
	}
}

// A file that is not there is reported before anything is assembled.
func TestAMissingProgramIsReported(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	if _, err := run([]string{filepath.Join(t.TempDir(), "nope.sno")}, &out, &errs); err == nil {
		t.Error("no error for a program that is not there")
	}
}

// -h is not a failure.
func TestHelpIsNotAnError(t *testing.T) {
	var out, errs bytes.Buffer
	status, err := run([]string{"-h"}, &out, &errs)
	if err != nil || status != 0 {
		t.Errorf("status %d, err %v, want 0 and nil", status, err)
	}
	if !strings.Contains(errs.String(), "usage: sil") {
		t.Errorf("no usage:\n%s", errs.String())
	}
}
