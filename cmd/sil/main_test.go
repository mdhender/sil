// Command sil runs SNOBOL4 programs.
//
// It carries the historical Macro SNOBOL4 implementation -- 6580 lines
// of SIL, the machine language S4D58 describes -- assembles it, and
// runs it. The SNOBOL4 program named on the command line is what that
// implementation then reads and compiles, the way it read a deck of
// cards.
//
//	sil hello.sno
//	sil < hello.sno
//	sil -listing core.txt hello.sno
//
// The exit status is 0 when the run reached ENDEX, however the program
// went, and 1 when something stopped it first: a source that would not
// assemble, a machine fault, a file that would not open. SNOBOL4 has
// no notion of an exit status to report -- ENDEX's operand is the
// keyword &ABEND (6.29 note 2), not a status -- and a program whose
// statements failed is a program that ran.
//
// # Where the output goes
//
// Standard output is what the program printed. Standard error is what
// the SNOBOL4 system printed about itself: its banner, whether the
// program compiled, and the statistics at the end.
//
// They are separated because they leave the machine by different

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
