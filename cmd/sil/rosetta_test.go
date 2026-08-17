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
	"strconv"
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/rosetta"
	"github.com/mdhender/sil/pkg/sil"
)

// The fourth testing layer, on programs nobody here wrote.
//
// Every other test in this repository was written against the machine
// by somebody who knew what the machine does. These were written by
// strangers, for other SNOBOL4s, to solve problems posed without any
// reference to this implementation, and the expectation each is held
// to comes from the task's description rather than from the program --
// internal/rosetta says why at length. What they are good for is the
// thing a hand-written test cannot do: turning up the part of SNOBOL4
// that nobody here thought to exercise.
//
// The programs are not in this repository. Each subtest skips on its
// own when its program was not fetched, and a skip is not a pass.
func TestRosettaCode(t *testing.T) {
	needsEngine(t)

	// -UNLIST is a control card, and it goes in front of the program
	// as its own card rather than being pasted onto it: the runner
	// reads the files named on its command line as one deck, so the
	// program reaches the compiler exactly as RosettaCode has it.
	// Without it the compilation listing shares STPRNT with the
	// program's own printing and lands in the output being asserted.
	card := write(t, "unlist.card", "-UNLIST\n")

	var ran, skipped int
	for _, task := range rosetta.Tasks {
		t.Run(strings.TrimSuffix(task.File, ".sno"), func(t *testing.T) {
			path, _, err := rosetta.Load(task)
			if errors.Is(err, rosetta.ErrAbsent) {
				skipped++
				t.Skipf("%s: %s", path, rosetta.SkipMessage)
			}
			if err != nil {
				t.Fatal(err)
			}
			ran++
			t.Logf("%s\n%s", rosetta.Page(task), task.Note)

			var out, errs bytes.Buffer
			max := strconv.Itoa(rosetta.Cycles(task))
			_, err = run([]string{"-max", max, card, path}, &out, &errs)

			// The instruction bound and a fault are both errors and
			// they are not the same news. A *sil.Bound says the
			// program will not stop; anything else says the machine
			// broke, and no entry in the manifest is allowed to
			// expect that.
			var stopped *sil.Bound
			ended := rosetta.Diagnosed
			switch {
			case errors.As(err, &stopped):
				ended = rosetta.Runaway
			case err != nil:
				t.Fatalf("the machine faulted: %v\n%s", err, errs.String())
			}

			// The system says whether it compiled what it was given.
			// The negative is what is tested for, because the message
			// that says it did contains the message that says it did
			// not.
			compiled := strings.Contains(errs.String(), "NO ERRORS DETECTED IN SOURCE PROGRAM")
			bad := rosetta.Match(task, out.String())

			if task.Status == rosetta.Unsupported {
				// The claim is not that the program fails. It is that
				// the machine reaches the end the manifest names, so
				// that a change turning a diagnosed program into a
				// runaway, or the other way about, is caught rather
				// than absorbed.
				if ended != task.Ends {
					t.Errorf("this ended as %s and the manifest says %s.\n"+
						"Reason given: %s\n\n%v\n%s",
						ending(ended), ending(task.Ends), task.Reason, err, errs.String())
				}
				if compiled && len(bad) == 0 {
					t.Errorf("this ran, and satisfied the task.\n"+
						"The manifest says 3.11 cannot: %s\nMove it to Runs.", task.Reason)
				}
				t.Logf("unsupported, %s, as the manifest says: %s", ending(ended), task.Reason)
				return
			}

			if stopped != nil {
				// A machine fault has already stopped this subtest, so
				// the only way here is a program that will not stop,
				// and the entry belongs at Unsupported with Ends set
				// to Runaway.
				t.Fatalf("this did not stop: %v\n%s", stopped, errs.String())
			}
			if !compiled {
				t.Fatalf("the system did not compile the program.\n"+
					"If that is 3.11 lacking something a later SNOBOL4 has, the entry\n"+
					"belongs at Unsupported with the reason.\n\n%s", errs.String())
			}
			if len(bad) != 0 {
				t.Errorf("%d ways the output is not the task's answer:\n  %s\n\n"+
					"What it printed:\n%s\nWhat the task requires: %s",
					len(bad), strings.Join(bad, "\n  "), indent(out.String()), task.Note)
			}
		})
	}

	// A run where every task skipped looks exactly like a run where
	// every task passed unless somebody says so.
	if ran == 0 {
		t.Logf("none of the %d tasks ran: %s", len(rosetta.Tasks), rosetta.SkipMessage)
	} else if skipped != 0 {
		t.Logf("%d of %d tasks ran; %d were not fetched", ran, len(rosetta.Tasks), skipped)
	}
}

// ending names one for a failure message. There are two of them and
// the distinction is the whole of what an Unsupported entry asserts,
// so it is worth saying in words rather than as a number.
func ending(e rosetta.Ending) string {
	if e == rosetta.Runaway {
		return "a runaway, stopped by the instruction bound"
	}
	return "diagnosed, and terminated normally"
}

// indent sets a program's output off from the test's own words, and
// bounds it: FizzBuzz prints a hundred lines and 99 bottles of beer
// prints three hundred, and neither is worth reading in full to find
// out that one substring was missing.
func indent(s string) string {
	const most = 40
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	cut := ""
	if len(lines) > most {
		cut = "\n\t... and " + strconv.Itoa(len(lines)-most) + " more lines"
		lines = lines[:most]
	}
	return "\t" + strings.Join(lines, "\n\t") + cut
}
