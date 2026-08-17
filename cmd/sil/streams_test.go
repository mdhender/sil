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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// read gives back what a run wrote to a file, and says so plainly when
// there is no file to read.
func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return string(b)
}

// The defaults are the behaviour that was here before -out and -system
// were, and the rest of the tests in this package depend on them.
func TestTheDefaultDestinationsAreUnchanged(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	if _, err := run([]string{write(t, "hello.sno", hello)}, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if got, want := out.String(), "HELLO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
	if !strings.Contains(errs.String(), "SNOBOL4 (VERSION 3.11") {
		t.Errorf("the banner did not go to standard error:\n%s", errs.String())
	}
}

// Either stream goes to a file, and what went to the file did not also
// go to the terminal.
func TestOutAndSystemGoToFiles(t *testing.T) {
	needsEngine(t)

	dir := t.TempDir()
	printed := filepath.Join(dir, "printed.txt")
	reported := filepath.Join(dir, "reported.txt")

	var out, errs bytes.Buffer
	args := []string{"-out", printed, "-system", reported, write(t, "hello.sno", hello)}
	if _, err := run(args, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}

	if got, want := read(t, printed), "HELLO\n"; got != want {
		t.Errorf("the printing file has %q, want %q", got, want)
	}
	if !strings.Contains(read(t, reported), "NO ERRORS DETECTED IN SOURCE PROGRAM") {
		t.Errorf("the system's file does not say the program compiled:\n%s", read(t, reported))
	}
	if out.Len() != 0 || errs.Len() != 0 {
		t.Errorf("something reached the terminal as well:\nout %q\nerr %q", out.String(), errs.String())
	}
}

// A comma-separated destination is a tee: the stream goes to every
// place named.
func TestADestinationCanBeSeveralPlaces(t *testing.T) {
	needsEngine(t)

	path := filepath.Join(t.TempDir(), "tee.txt")

	var out, errs bytes.Buffer
	args := []string{"-out", stdName + "," + path, write(t, "hello.sno", hello)}
	if _, err := run(args, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if got, want := out.String(), "HELLO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
	if got, want := read(t, path), "HELLO\n"; got != want {
		t.Errorf("the file has %q, want %q", got, want)
	}
}

// The two streams can be crossed over, which is the shortest statement
// of what these flags do: neither stream is tied to the descriptor it
// usually goes down.
func TestTheStreamsCanBeSwapped(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	args := []string{"-out", errName, "-system", stdName, write(t, "hello.sno", hello)}
	if _, err := run(args, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if got, want := errs.String(), "HELLO\n"; got != want {
		t.Errorf("standard error is %q, want the program's printing %q", got, want)
	}
	if !strings.Contains(out.String(), "SNOBOL4 (VERSION 3.11") {
		t.Errorf("the banner did not go to standard output:\n%s", out.String())
	}
}

// One file named by both flags is one open file, written in the order
// the machine wrote it. Two opens would truncate each other and
// interleave whatever survived.
func TestBothStreamsCanShareOneFile(t *testing.T) {
	needsEngine(t)

	path := filepath.Join(t.TempDir(), "log.txt")

	var out, errs bytes.Buffer
	args := []string{"-out", path, "-system", path, write(t, "hello.sno", hello)}
	if _, err := run(args, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}

	// The order the original printed in: the banner, then what the
	// program printed, then the statistics.
	text := read(t, path)
	banner := strings.Index(text, "SNOBOL4 (VERSION 3.11")
	program := strings.Index(text, "HELLO")
	stats := strings.Index(text, "STATISTICS SUMMARY")
	if banner < 0 || program < 0 || stats < 0 {
		t.Fatalf("the file is missing one of the three:\n%s", text)
	}
	if !(banner < program && program < stats) {
		t.Errorf("the file is out of order: banner %d, printing %d, statistics %d\n%s",
			banner, program, stats, text)
	}
}

// none throws a stream away, which is how the banner and the
// statistics are silenced.
func TestNoneDiscardsAStream(t *testing.T) {
	needsEngine(t)

	var out, errs bytes.Buffer
	args := []string{"-system", offName, write(t, "hello.sno", hello)}
	if _, err := run(args, &out, &errs); err != nil {
		t.Fatalf("%v\n%s", err, errs.String())
	}
	if got, want := out.String(), "HELLO\n"; got != want {
		t.Errorf("standard output is %q, want %q", got, want)
	}
	if errs.Len() != 0 {
		t.Errorf("the system's output survived -system %s:\n%s", offName, errs.String())
	}
}

// -merge is what it always was, and is now shorthand.
func TestMergeIsShorthandForSystemStdout(t *testing.T) {
	needsEngine(t)

	var merged, mergedErr bytes.Buffer
	if _, err := run([]string{"-merge", write(t, "a.sno", hello)}, &merged, &mergedErr); err != nil {
		t.Fatalf("%v\n%s", err, mergedErr.String())
	}
	var spelled, spelledErr bytes.Buffer
	args := []string{"-system", stdName, write(t, "b.sno", hello)}
	if _, err := run(args, &spelled, &spelledErr); err != nil {
		t.Fatalf("%v\n%s", err, spelledErr.String())
	}
	if merged.String() != spelled.String() {
		t.Errorf("-merge and -system %s differ:\n%q\n%q", stdName, merged.String(), spelled.String())
	}

	// Saying both is a contradiction rather than a precedence to
	// remember.
	var out, errs bytes.Buffer
	both := []string{"-merge", "-system", errName, write(t, "c.sno", hello)}
	if _, err := run(both, &out, &errs); err == nil {
		t.Error("-merge and -system together were accepted")
	}
}

// The parsing, without a machine behind it. These do not need the
// engine and so do not skip.
func TestDestinations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()

	t.Run("what a destination may say", func(t *testing.T) {
		for _, spec := range []string{
			stdName, errName, offName,
			filepath.Join(dir, "one.txt"),
			stdName + "," + filepath.Join(dir, "two.txt"),
			errName + "," + stdName,
			// Spaces round the commas are a person typing, not a
			// mistake.
			stdName + " , " + errName,
			// A place repeated is one place, not two copies.
			stdName + "," + stdName,
		} {
			o := newOpened(&stdout, &stderr)
			w, err := o.resolve("out", spec)
			o.closeAll()
			if err != nil {
				t.Errorf("-out %q: %v", spec, err)
			} else if w == nil {
				t.Errorf("-out %q: no writer and no error", spec)
			}
		}
	})

	t.Run("what it may not", func(t *testing.T) {
		for _, spec := range []string{
			"",
			"   ",
			",",
			stdName + ",",
			"," + stdName,
			// none is the whole destination or none of it: discarding
			// half a stream is not a thing to mean.
			offName + "," + stdName,
			stdName + "," + offName,
			// A directory is not a file, whatever the name looks
			// like.
			dir,
		} {
			o := newOpened(&stdout, &stderr)
			_, err := o.resolve("out", spec)
			o.closeAll()
			if err == nil {
				t.Errorf("-out %q was accepted", spec)
			}
		}
	})

	// Writing everything twice is the one thing a repeated place must
	// not do.
	t.Run("a place repeated is written once", func(t *testing.T) {
		var to bytes.Buffer
		o := newOpened(&to, &stderr)
		w, err := o.resolve("out", stdName+","+stdName)
		if err != nil {
			t.Fatal(err)
		}
		defer o.closeAll()
		if _, err := w.Write([]byte("once\n")); err != nil {
			t.Fatal(err)
		}
		if got, want := to.String(), "once\n"; got != want {
			t.Errorf("wrote %q, want %q", got, want)
		}
	})

	// A file named by both streams is opened once. Two opens would
	// leave two truncating writers on one name.
	t.Run("one name is one open file", func(t *testing.T) {
		path := filepath.Join(dir, "shared.txt")
		o := newOpened(&stdout, &stderr)
		first, err := o.resolve("out", path)
		if err != nil {
			t.Fatal(err)
		}
		second, err := o.resolve("system", path)
		if err != nil {
			t.Fatal(err)
		}
		if first != second {
			t.Error("the same name gave two different writers")
		}
		if len(o.order) != 1 {
			t.Errorf("opened %d files for one name", len(o.order))
		}
		if _, err := first.Write([]byte("one\n")); err != nil {
			t.Fatal(err)
		}
		if _, err := second.Write([]byte("two\n")); err != nil {
			t.Fatal(err)
		}
		o.closeAll()
		if got, want := read(t, path), "one\ntwo\n"; got != want {
			t.Errorf("the file has %q, want %q", got, want)
		}
	})

	// A destination that cannot be opened stops the run before the
	// machine starts, and does not leave the other stream's file open.
	t.Run("a file that will not open", func(t *testing.T) {
		bad := filepath.Join(dir, "no-such-directory", "x.txt")
		good := filepath.Join(dir, "good.txt")
		_, _, closeAll, err := streams(good, bad, &stdout, &stderr)
		if err == nil {
			closeAll()
			t.Fatal("a file under a directory that is not there was accepted")
		}
		if !strings.Contains(err.Error(), "-system") {
			t.Errorf("the error does not say which flag: %v", err)
		}
	})
}
