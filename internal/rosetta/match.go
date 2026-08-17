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

package rosetta

import (
	"fmt"
	"sort"
	"strings"
)

// Match reports every way the output fails what the task requires, and
// nil when it satisfies all of it. All of them are reported rather
// than the first, because one run of a SNOBOL4 program on an emulated
// machine is slow enough that a test should get everything it can from
// the one it did.
//
// A task with no expectation at all matches everything, which is not a
// useful thing for it to do; Validate is what catches that, and it
// runs whether or not the corpus is in the checkout.
func Match(t Task, got string) []string {
	var bad []string
	hay, fold := got, func(s string) string { return s }
	if t.Fold {
		hay = strings.ToUpper(got)
		fold = strings.ToUpper
	}

	if t.Want != "" && got != t.Want {
		bad = append(bad, fmt.Sprintf("the output is %q, and the task's is %q", got, t.Want))
	}
	for _, s := range t.Contains {
		if !strings.Contains(hay, fold(s)) {
			bad = append(bad, fmt.Sprintf("the output does not have %q in it", s))
		}
	}
	for _, s := range t.Absent {
		if strings.Contains(hay, fold(s)) {
			bad = append(bad, fmt.Sprintf("the output has %q in it, and the task's answer does not", s))
		}
	}
	// A map ranges in no order, and a test that reports its failures
	// in a different order every run is a test nobody can diff.
	for _, s := range sorted(t.Counts) {
		if n, want := strings.Count(hay, fold(s)), t.Counts[s]; n != want {
			bad = append(bad, fmt.Sprintf("the output has %q %d times, and the task's has it %d times", s, n, want))
		}
	}
	return bad
}

// Validate reports what is wrong with the manifest itself: an entry
// that claims nothing, two entries that would fetch to the same file,
// an Unsupported entry that does not say what is missing. These are
// mistakes in this repository rather than in the machine, so the test
// that calls it does not skip when the corpus is absent.
func Validate(tasks []Task) []string {
	var bad []string
	seen := map[string]string{}
	for _, t := range tasks {
		switch {
		case t.Name == "":
			bad = append(bad, fmt.Sprintf("%s: no task name, and the fetcher has nothing to ask for", t.File))
		case t.File == "":
			bad = append(bad, fmt.Sprintf("%q: no file name", t.Name))
		}
		if prev, ok := seen[t.File]; ok {
			bad = append(bad, fmt.Sprintf("%q and %q both fetch to %s", prev, t.Name, t.File))
		}
		seen[t.File] = t.Name

		if t.Want == "" && len(t.Contains) == 0 && len(t.Absent) == 0 && len(t.Counts) == 0 {
			bad = append(bad, fmt.Sprintf("%q: no expectation, so it would pass on any output at all", t.Name))
		}
		if t.Status == Unsupported && t.Reason == "" {
			bad = append(bad, fmt.Sprintf("%q: unsupported without saying what 3.11 lacks", t.Name))
		}
		if t.Note == "" {
			bad = append(bad, fmt.Sprintf("%q: no note saying where the expectation came from", t.Name))
		}
		// Pinning is the whole defence against a wiki page moving
		// under the expectation, so an entry is pinned in both halves
		// or in neither, and a half-pinned one is a fetch that was
		// never reviewed.
		if (t.Oldid == 0) != (t.SHA256 == "") {
			bad = append(bad, fmt.Sprintf("%q: pinned to revision %d but hashed %q; pin both or neither",
				t.Name, t.Oldid, t.SHA256))
		}
	}
	return bad
}

// Pinned reports whether every task names the revision and the program
// its expectation was written against. Until they all do, a green run
// is a fact about today's wiki.
func Pinned(tasks []Task) []string {
	var loose []string
	for _, t := range tasks {
		if t.Oldid == 0 || t.SHA256 == "" {
			loose = append(loose, t.Name)
		}
	}
	return loose
}

func sorted(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
