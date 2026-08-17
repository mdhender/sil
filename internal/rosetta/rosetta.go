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

// Package rosetta locates the SNOBOL4 programs taken from RosettaCode
// and says what each task requires of their output.
//
// # The programs are not in this repository
//
// RosettaCode's contributions are licensed CC BY-SA, which is not the
// licence this repository carries, so testdata/rosetta is gitignored
// the way engines/ and references/ are and scripts/fetch-rosetta puts
// the files there. Tests skip per task on ErrAbsent, keyed on the file
// itself, so a skip expires by itself when the file arrives.
//
// A skip is not a pass. With the corpus absent, nothing here checks
// that the machine runs anything but the programs written into the
// tests by hand.
//
// # What is in this repository is the expectation
//
// Tasks holds what each task's own description fixes about its output,
// written from that description rather than read off the contributor's
// program. That is the point of the arrangement, not a consequence of
// it: the test then asserts the task, and a RosettaCode solution that
// is subtly wrong fails here instead of being blessed by a golden file
// recorded from its own output.
//
// It also means an expectation can be wrong in the other direction --
// too tight for a solution that satisfies the task by printing it
// differently. Where a task's description does not fix the formatting,
// the entry says so and asserts the part that is fixed. See Task.
//
// # Pinning
//
// A wiki page changes. Oldid pins the revision the expectation was
// written against and SHA256 pins the extracted program, so that a red
// test means the machine changed and not that somebody edited a wiki
// page. Both are empty until the first fetch; scripts/fetch-rosetta
// prints the values to paste back into Tasks, and refuses to overwrite
// a pinned file whose contents no longer hash the same.
package rosetta

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdhender/sil/internal/corpus"
)

// Dir is the module-relative directory the fetched programs live in.
const Dir = "testdata/rosetta"

// ErrAbsent reports that a task's program is not in this checkout.
var ErrAbsent = errors.New("the program was not fetched into this checkout")

// SkipMessage tells a reader how to turn a skipped test back on.
const SkipMessage = "not in this checkout; run `go run ./scripts/fetch-rosetta` to fetch it"

// DefaultMax bounds a run that has no bound of its own, in SIL
// instructions. A wrong program loops; without this the test hangs
// until the whole package times out and says nothing about which task
// did it. Around fifteen million instructions go by in a second, so
// this is a few seconds of a machine that is getting nowhere.
const DefaultMax = 100_000_000

// Status says what this machine is expected to make of the program.
type Status int

const (
	// Runs: the program compiles and executes on version 3.11, and
	// its output satisfies the task.
	Runs Status = iota

	// Unsupported: it does not, because it was written for a later
	// SNOBOL4 than this one. Most of RosettaCode's SNOBOL4 targets
	// CSNOBOL4 or SNOBOL4+, which have functions and keywords 1975
	// did not. Reason says which, and Ends says how the system gets
	// rid of it.
	//
	// These entries are not dead weight. What they assert is that the
	// machine reaches a stated end rather than an unexplained one, so
	// that a change which turns a diagnosed program into a fault, or
	// a runaway into silence, is caught. Together they are a checked
	// inventory of what version 3.11 lacks.
	Unsupported
)

// Ending is how the system gets rid of a program it cannot run, and
// it is part of what an Unsupported entry asserts.
type Ending int

const (
	// Diagnosed: the system reports the error itself and terminates
	// normally. This is the ending to want, and the one S4D58
	// describes.
	Diagnosed Ending = iota

	// Runaway: it never terminates, and the instruction bound is what
	// stops it.
	//
	// There is one way into this that is not a defect. Version 3.11
	// is case-sensitive, so a program written in lower case for a
	// later SNOBOL4 has no END card as far as the compiler is
	// concerned; the compiler reads on past the last card, and the
	// card-reading loop in the source reads again on end-of-file --
	//
	//	FORRUR STREAD  INBFSP,UNIT,FORRUR,COMP5
	//
	// where the third operand is STREAD's EOF branch (6.115) and it
	// is this instruction's own label. A deck without an END card
	// hangs the original too. It is faithful, not broken, and it is
	// what most of RosettaCode's SNOBOL4 does here.
	Runaway
)

// A Task is one RosettaCode task, the program taken from its SNOBOL4
// section, and what its description requires of the output.
//
// The four kinds of expectation, in order of how much they claim:
//
//   - Want is the whole of standard output, exactly. Use it only when
//     the task's description leaves no choice about the formatting.
//   - Counts is a substring and how many times it must occur. It says
//     nothing about layout, so it survives a solution that prints one
//     line where another prints three, and it still pins the answer.
//   - Contains is a substring that must appear, and is the usual case:
//     the distinctive part of the answer, with the arrangement left to
//     the program.
//   - Absent is a substring that must not. It is how a task says what
//     a wrong answer looks like -- a composite among the primes -- and
//     it is worth more than another Contains.
//
// Fold compares without case. SNOBOL4 of this period is written in
// upper case and much of RosettaCode's is too, so a task whose answer
// is words rather than digits usually wants it. A task that is *about*
// case must not have it.
//
// Note records where the expectation came from and what about it is
// shaky, for whoever has to change it later.
type Task struct {
	Name  string // the RosettaCode page title, verbatim
	File  string // the basename under Dir
	Block int    // which code block of the SNOBOL4 section, from 0

	Oldid  int    // the pinned revision; 0 is unpinned
	SHA256 string // the pinned program; "" is unpinned

	Status Status
	Reason string // for Unsupported, what 3.11 does not have
	Ends   Ending // for Unsupported, how the system gets rid of it

	Want     string
	Contains []string
	Absent   []string
	Counts   map[string]int
	Fold     bool

	// Max bounds the run in SIL instructions; 0 takes DefaultMax. The
	// whole corpus finishes in under a second at present, because the
	// programs 3.11 will not run stop early and the ones it will run
	// are small. Set it when a task earns a longer leash, and set it
	// low for a runaway, which only has to be shown not to stop.
	Max int

	Note string
}

// Path is where the program belongs in this checkout.
func Path(t Task) (string, error) {
	root, err := corpus.Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, Dir, t.File), nil
}

// Load returns the path and contents of a task's program, wrapping
// ErrAbsent when it was not fetched.
func Load(t Task) (name string, src []byte, err error) {
	name, err = Path(t)
	if err != nil {
		return "", nil, err
	}
	src, err = os.ReadFile(name)
	if os.IsNotExist(err) {
		return name, nil, fmt.Errorf("%s/%s: %w", Dir, t.File, ErrAbsent)
	}
	if err != nil {
		return name, nil, err
	}
	return name, src, nil
}

// Cycles is the instruction bound to run this task under.
func Cycles(t Task) int {
	if t.Max > 0 {
		return t.Max
	}
	return DefaultMax
}

// API is the URL the fetcher asks for a task's wikitext, which is one
// request for both the text and the revision it came from -- prop asks
// for both, and the vertical bar between them is what a query string
// spells %7C. An unpinned task asks for the page and gets whatever is
// current; a pinned one asks for the revision by number and gets the
// same text every time.
func API(t Task) string {
	const base = "https://rosettacode.org/w/api.php?action=parse&prop=wikitext%7Crevid&format=json&formatversion=2"
	if t.Oldid != 0 {
		return fmt.Sprintf("%s&oldid=%d", base, t.Oldid)
	}
	return base + "&page=" + urlQuery(t.Name)
}

// Page is the URL a person reads, and is what a Task's provenance
// amounts to: the task description the expectation was written from.
func Page(t Task) string {
	if t.Oldid != 0 {
		return fmt.Sprintf("https://rosettacode.org/w/index.php?oldid=%d", t.Oldid)
	}
	return "https://rosettacode.org/wiki/" + urlQuery(t.Name)
}

// urlQuery escapes a page title for a query string. MediaWiki takes a
// title with underscores or with spaces; percent-encoding the few
// characters that matter is enough and keeps the URLs readable.
func urlQuery(title string) string {
	const hex = "0123456789ABCDEF"
	var b []byte
	for i := 0; i < len(title); i++ {
		c := title[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~', c == '/', c == '(', c == ')':
			b = append(b, c)
		case c == ' ':
			b = append(b, '_')
		default:
			b = append(b, '%', hex[c>>4], hex[c&0xf])
		}
	}
	return string(b)
}
