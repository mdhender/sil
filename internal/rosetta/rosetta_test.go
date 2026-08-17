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
	"strings"
	"testing"
)

// The matcher is what decides whether a run of the corpus passed, and
// it is the one part of this package that does not need the corpus.
// These run whether or not anything was fetched.
func TestMatch(t *testing.T) {
	for _, tt := range []struct {
		name string
		task Task
		got  string
		bad  int
	}{
		{
			name: "Want is the whole output",
			task: Task{Want: "HELLO\n"},
			got:  "HELLO\n",
		},
		{
			name: "and nothing less",
			task: Task{Want: "HELLO\n"},
			got:  "HELLO",
			bad:  1,
		},
		{
			name: "Contains ignores the arrangement",
			task: Task{Contains: []string{"2", "3", "5"}},
			got:  "2 3 5\n",
		},
		{
			name: "Contains reports each one it did not find",
			task: Task{Contains: []string{"2", "3", "5"}},
			got:  "2\n",
			bad:  2,
		},
		{
			name: "Absent is how a task says what a wrong answer looks like",
			task: Task{Contains: []string{"89"}, Absent: []string{"91"}},
			got:  "89 97\n",
		},
		{
			name: "and it fires when the wrong answer turns up",
			task: Task{Contains: []string{"89"}, Absent: []string{"91"}},
			got:  "89 91 97\n",
			bad:  1,
		},
		{
			name: "Counts pins the answer without pinning the layout",
			task: Task{Counts: map[string]int{"Fizz": 2, "Buzz": 1}},
			got:  "1 2 Fizz 4 Buzz Fizz 7\n",
		},
		{
			name: "Counts is exact, over and under",
			task: Task{Counts: map[string]int{"Fizz": 2}},
			got:  "Fizz Fizz Fizz\n",
			bad:  1,
		},
		{
			name: "Fold takes a program that prints in upper case",
			task: Task{Contains: []string{"hahahahaha"}, Fold: true},
			got:  "HAHAHAHAHA\n",
		},
		{
			name: "and without it the case is the assertion",
			task: Task{Contains: []string{"alphabeta"}},
			got:  "ALPHABETA\n",
			bad:  1,
		},
		{
			name: "Fold reaches Counts and Absent too",
			task: Task{Counts: map[string]int{"Fizz": 1}, Absent: []string{"Buzz"}, Fold: true},
			got:  "FIZZ\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bad := Match(tt.task, tt.got)
			if len(bad) != tt.bad {
				t.Errorf("Match reported %d failures, want %d:\n%s",
					len(bad), tt.bad, strings.Join(bad, "\n"))
			}
		})
	}
}

// Match reports everything it found, not the first thing, because a
// run of the corpus is slow enough that a test should get everything
// it can out of the one it did.
func TestMatchReportsEveryFailure(t *testing.T) {
	task := Task{
		Contains: []string{"one", "two"},
		Absent:   []string{"three"},
		Counts:   map[string]int{"four": 1},
	}
	if bad := Match(task, "three\n"); len(bad) != 4 {
		t.Errorf("Match reported %d failures, want 4:\n%s", len(bad), strings.Join(bad, "\n"))
	}
}

// The failures come out in the same order every time, so that two runs
// of a task can be diffed. Counts lives in a map, which is where that
// would otherwise go wrong.
func TestMatchIsOrdered(t *testing.T) {
	task := Task{Counts: map[string]int{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1}}
	first := strings.Join(Match(task, ""), "\n")
	for range 20 {
		if got := strings.Join(Match(task, ""), "\n"); got != first {
			t.Fatalf("Match reported its failures in another order:\n%s\n\n%s", first, got)
		}
	}
}

// The manifest is this repository's own work, so a mistake in it is a
// mistake here and not in the corpus. This does not skip.
func TestTheManifestIsWellFormed(t *testing.T) {
	if bad := Validate(Tasks); len(bad) != 0 {
		t.Errorf("%d problems in the manifest:\n%s", len(bad), strings.Join(bad, "\n"))
	}
}

// Validate earns its keep by catching the entry that asserts nothing,
// which would otherwise pass on any output the machine produced.
func TestValidateCatchesAnEmptyExpectation(t *testing.T) {
	bad := Validate([]Task{{Name: "Nothing", File: "nothing.sno", Note: "n/a"}})
	if len(bad) != 1 {
		t.Fatalf("Validate reported %d problems, want 1:\n%s", len(bad), strings.Join(bad, "\n"))
	}
	if !strings.Contains(bad[0], "no expectation") {
		t.Errorf("Validate said %q", bad[0])
	}
}

// Unpinned tasks are not an error -- an entry has to be written before
// it can be fetched, and pinned after it is read -- but they are worth
// saying out loud, because until a task is pinned a green run is a
// fact about today's wiki page.
func TestWhichTasksArePinned(t *testing.T) {
	if loose := Pinned(Tasks); len(loose) != 0 {
		t.Logf("%d of %d tasks are not pinned to a revision, so what they assert\n"+
			"is whatever RosettaCode serves today:\n  %s",
			len(loose), len(Tasks), strings.Join(loose, "\n  "))
	}
}

// A task title becomes a URL the same way MediaWiki writes one, and
// getting this wrong means the fetcher asks for a page that is not
// there and the corpus quietly stays empty.
func TestURLs(t *testing.T) {
	task := Task{Name: "99 bottles of beer"}
	if got, want := Page(task), "https://rosettacode.org/wiki/99_bottles_of_beer"; got != want {
		t.Errorf("Page is %s, want %s", got, want)
	}
	if got := API(task); !strings.HasSuffix(got, "&page=99_bottles_of_beer") {
		t.Errorf("API is %s", got)
	}
	// Pinned, both URLs name the revision instead of the page, which
	// is the whole of what pinning buys.
	task.Oldid = 347712
	if got, want := Page(task), "https://rosettacode.org/w/index.php?oldid=347712"; got != want {
		t.Errorf("Page is %s, want %s", got, want)
	}
	if got := API(task); !strings.Contains(got, "oldid=347712") || strings.Contains(got, "page=") {
		t.Errorf("API is %s", got)
	}
}
