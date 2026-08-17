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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/internal/programs"
	"github.com/mdhender/sil/pkg/sil"
)

// update rewrites the golden files from what the machine printed.
//
//	go test ./cmd/sil -run TestPrograms -update
//
// It is for the case where the change was intended. Read the diff
// first: the golden files were written from what each program is
// supposed to do, and recording a run over the top of one is how a
// wrong answer becomes the standard.
var update = flag.Bool("update", false,
	"rewrite internal/programs/sno/*.out from this run")

// bound stops a program that will not stop. None of these needs more
// than a few million instructions; this is loose enough not to be a
// limit and tight enough that a mistake reports rather than hangs.
const bound = "100000000"

// Golden output for the programs in internal/programs, which are ours
// and are in every checkout.
//
// This is the test a refactor is held to. The RosettaCode corpus is
// the one that says whether the machine runs SNOBOL4 nobody here
// wrote; it skips when the programs have not been fetched, and it is
// no use at all for "did I change anything" because a skip looks like
// a pass. These do not skip.
//
// Two streams are compared, because a run produces two and which
// message goes down which is a fact about the runner worth pinning.
// The .out is what reached the printer: the program's own output, the
// compilation listing when one is asked for, and the text of any error
// message. The .err is the system's own report, cut down to the
// diagnostics by programs.Diagnostics.
func TestPrograms(t *testing.T) {
	needsEngine(t)

	for _, p := range programs.All() {
		t.Run(p.Name, func(t *testing.T) {
			if !p.HasWant && !*update {
				t.Fatalf("%s.sno has no %s.out beside it.\n"+
					"Write one -- what the program should print, worked out from what it\n"+
					"says -- or run with -update and then read what it wrote.", p.Name, p.Name)
			}

			out, errs, err := runProgram(t, p)

			// The instruction bound is not a fault, and the two are
			// told apart because they mean opposite things: one says
			// the program will not stop, the other says the machine
			// broke.
			var stopped *sil.Bound
			switch {
			case errors.As(err, &stopped):
				if p.Runaway == "" {
					t.Fatalf("this did not stop, and nothing says it should not.\n"+
						"If that is the point, say so with a RUNAWAY: directive.\n\n%v\n%s",
						stopped, errs.String())
				}
			case err != nil:
				t.Fatalf("the machine faulted: %v\n%s", err, errs.String())
			case p.Runaway != "":
				t.Fatalf("this stopped, and %s.sno says it should not:\nRUNAWAY: %s",
					p.Name, p.Runaway)
			}

			compiled := strings.Contains(errs.String(), "NO ERRORS DETECTED IN SOURCE PROGRAM")
			if p.Errors == "" && p.Runaway == "" && !compiled {
				t.Fatalf("the system did not compile it, and nothing says it should not.\n"+
					"If the program is invalid on purpose, say so with an ERRORS:\n"+
					"directive.\n\n%s", errs.String())
			}

			got := programs.Normalize(out.Bytes())
			report := programs.Diagnostics(errs.Bytes())

			if *update {
				// Not a pending one. Its golden is the answer this
				// machine does not yet give, and recording what it
				// does give instead would throw away the only record
				// of what the program is for.
				if p.Pending != "" {
					t.Logf("not updated: %s.sno is pending, and its golden is what it\n"+
						"should print rather than what it does:\nPENDING: %s", p.Name, p.Pending)
					return
				}
				writeGolden(t, p.Name+".out", got)
				if p.Errors != "" || p.Runaway != "" {
					writeGolden(t, p.Name+".err", report)
				}
				return
			}

			same := bytes.Equal(got, programs.Normalize(p.Want))
			if p.Pending != "" {
				// A pending program is one this machine gets wrong on
				// purpose, with the reason recorded on the program
				// itself. What is asserted is that it is still wrong,
				// so that fixing the cause is announced here instead
				// of leaving a stale note behind.
				if same {
					t.Errorf("this now prints what it should, and the program still says\n"+
						"PENDING: %s\n\nClear that line from %s.sno.", p.Pending, p.Name)
				} else {
					t.Logf("pending, as %s.sno says: %s\n\n%s", p.Name, p.Pending,
						diff(p.Want, got))
				}
				return
			}
			if !same {
				t.Errorf("what reached the printer is not what %s.out has.\n%s\n\n%s",
					p.Name, p.Doc, diff(p.Want, got))
			}
			if p.HasReport && !bytes.Equal(report, programs.Normalize(p.Report)) {
				t.Errorf("what the system reported is not what %s.err has.\n%s\n\n%s",
					p.Name, p.Doc, diff(p.Report, report))
			}
		})
	}
}

// runProgram runs one program the way cmd/sil would, with the control
// card and the stack size the program itself asks for.
func runProgram(t *testing.T, p programs.Program) (out, errs bytes.Buffer, err error) {
	t.Helper()

	args := []string{"-max", bound}
	if p.Stack != 0 {
		args = append(args, "-stack", strconv.Itoa(p.Stack))
	}
	args = append(args,
		write(t, "control.card", programs.Card),
		write(t, p.Name+".sno", string(p.Source)))

	_, err = run(args, &out, &errs)
	return out, errs, err
}

// Every program is a deck of cards, and the statement field of a card
// stops at column 72. Anything past it is silently dropped, usually
// in the middle of a literal, and what the compiler then reports is an
// error somewhere else entirely. This is cheaper than finding that out
// twice.
func TestProgramsFitOnACard(t *testing.T) {
	for _, p := range programs.All() {
		if bad := programs.TooWide(p); len(bad) != 0 {
			t.Errorf("%s.sno runs past column %d:\n  %s",
				p.Name, programs.Columns, strings.Join(bad, "\n  "))
		}
	}
}

// writeGolden puts a golden file back where the embed reads it from,
// which is the working tree rather than the embedded copy.
func writeGolden(t *testing.T, name string, out []byte) {
	t.Helper()
	root, err := corpus.Root()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, programs.Dir, name)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s -- read the diff before committing it", path)
}

// diff shows the first line that differs with a little either side of
// it, because bottles.sno prints three hundred lines and the useful
// part of a mismatch is one of them.
func diff(want, got []byte) string {
	w := strings.Split(strings.TrimRight(string(want), "\n"), "\n")
	g := strings.Split(strings.TrimRight(string(got), "\n"), "\n")

	at := -1
	for i := 0; i < len(w) || i < len(g); i++ {
		if i >= len(w) || i >= len(g) || w[i] != g[i] {
			at = i
			break
		}
	}
	if at < 0 {
		return "the lines are the same; the difference is in the trailing newlines"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "line %d of %d is the first to differ (%d lines printed)\n",
		at+1, len(w), len(g))
	for i := max(0, at-2); i < min(max(len(w), len(g)), at+3); i++ {
		mark := "  "
		if i == at {
			mark = "> "
		}
		fmt.Fprintf(&b, "%swant %s\n%s got  %s\n", mark, lineAt(w, i), mark, lineAt(g, i))
	}
	return b.String()
}

// lineAt quotes one line, or says there is not one.
func lineAt(lines []string, i int) string {
	if i >= len(lines) {
		return "(no line)"
	}
	return strconv.Quote(lines[i])
}
