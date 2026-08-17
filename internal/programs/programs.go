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

// Package programs carries a set of SNOBOL4 programs written for this
// repository, with the output each one is supposed to produce.
//
// # Why these exist beside internal/rosetta
//
// The RosettaCode corpus answers "does this machine run SNOBOL4 that
// somebody else wrote". It cannot answer anything else, because the
// programs are not ours to keep: they are fetched into a gitignored
// directory, a checkout may not have them, and a test that skips is
// not a test that passed.
//
// These are ours. They are committed, they are in the embed, and they
// run on every checkout on every machine. That makes them the ones to
// hang the other three jobs on:
//
//   - golden output, so a refactor that changes what the machine
//     prints has to say so;
//   - benchmarks, so a refactor that changes what the machine costs
//     has to say so;
//   - coverage of the source language, chosen on purpose rather than
//     by what RosettaCode happens to have in upper case.
//
// # They are clean-room in the ordinary sense, not the legal one
//
// Every one was written here, from the task rather than from anyone
// else's solution, and version 3.11 is old enough that most of what is
// on RosettaCode would not compile anyway. But the author of these had
// read several of RosettaCode's SNOBOL4 entries earlier in the same
// sitting, so this is not a clean room in the sense a lawyer means. It
// is unencumbered because it was written here and is licensed with the
// rest of the repository -- not because anybody was kept in the dark.
//
// Where a task has an obvious shape the resemblance is unavoidable and
// uninteresting: ackermann.sno is the three-line definition, and there
// is no second way to write it. Where there was a choice, these went
// the other way on purpose -- roman.sno is an iterative table walk
// rather than a recursive REPLACE, sieve.sno crosses off in an array
// rather than in a string.
//
// # The golden output is written by hand, not recorded
//
// The .out beside each .sno is what the program is supposed to print,
// worked out from what the program says rather than captured from a
// run. Recording a run and calling it correct is how a wrong answer
// becomes the standard; the -update flag on the test exists for the
// case where the change was intended, and it is meant to be used after
// reading the diff and not instead of reading it.
//
// # Programs that are not supposed to work
//
// Some of these are invalid on purpose, because what the SNOBOL4
// system does with a bad program is most of what it does and none of
// it was covered. An erroneous or missing break character, an
// undefined function, arithmetic past the largest integer this machine
// has, a goto to a label that is not there, a statement limit reached,
// an interpreter stack run out, and a deck with no END card are seven
// different paths and they now have goldens.
//
// Two streams come back from a run and both are kept. The .out is what
// reached the printer -- the program's own output, the compilation
// listing, and the text of any error message. The .err is what the
// system reported about itself: "ERROR 24 IN STATEMENT 2 AT LEVEL 0".
// They are separate files because they are separate streams, and which
// message goes down which is a fact about the runner worth pinning.
//
// # Directives
//
// A comment card of the form "*  KEY: value" is read by this package.
// A directive's text runs on over the comment cards that follow it,
// until a blank comment card or the copyright card.
//
//	PENDING:   this machine gets the program wrong, and here is why.
//	           The test asserts it is still wrong, so that fixing the
//	           cause says so rather than leaving a stale note behind.
//	           See charcode.sno, waiting on ALPHA.
//	ERRORS:    the program is invalid on purpose, and here is how.
//	           Such a program carries a -LIST card after its header,
//	           because the listing is where the compiler names the
//	           card it could not read.
//	RUNAWAY:   the program does not stop, and here is why. The test
//	           expects a *sil.Bound and not a fault. See noend.sno.
//	STACK:     assemble with STSIZE set to this many descriptors,
//	           which is what cmd/sil's -stack does. See ackermann3.sno.
package programs

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed sno
var files embed.FS

// Dir is the module-relative directory the programs live in, which the
// test needs when -update writes a golden file back.
const Dir = "internal/programs/sno"

// A Program is one SNOBOL4 program and what it should do.
type Program struct {
	// Name is the file name without its extension: "fizzbuzz".
	Name string

	// Doc is the one-line description from the program's first
	// comment card, without the file name in front of it.
	Doc string

	// Pending, when set, is why this machine gets the program wrong.
	Pending string

	// Errors, when set, is why the program is invalid on purpose. Such
	// a program is compiled with the listing on; see Card.
	Errors string

	// Runaway, when set, is why the program does not stop. The run is
	// expected to end in a *sil.Bound.
	Runaway string

	// Stack, when not zero, is what STSIZE should be for this program,
	// in descriptors.
	Stack int

	// Source is the program, as a deck of cards.
	Source []byte

	// Want is what should reach the printer: the program's own output,
	// the compilation listing when there is one, and the text of any
	// error message. HasWant tells an empty golden from a missing one.
	Want    []byte
	HasWant bool

	// Report is what the system should say about itself, cut down to
	// the diagnostics; see Diagnostics.
	Report    []byte
	HasReport bool
}

// Card is the control card put in front of every program: -UNLIST
// turns the compilation listing off, which otherwise shares the
// printer with the program's own output and lands in the text being
// compared.
//
// It is a separate card rather than something pasted onto the front of
// the program, so that what reaches the compiler is the program as it
// is on disk. A program that wants the listing after all -- an invalid
// one does, because the listing is where the compiler names the card
// it could not read -- turns it back on with a -LIST card of its own,
// after its header, so that its golden holds statements and not prose.
const Card = "-UNLIST\n"

// All returns every program, in file-name order so that a test's
// output is the same from one run to the next.
func All() []Program {
	entries, err := files.ReadDir("sno")
	if err != nil {
		panic(err) // the embed is compiled in; it cannot be missing
	}
	var out []Program
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".sno" {
			continue
		}
		out = append(out, read(strings.TrimSuffix(e.Name(), ".sno")))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get returns one program by name.
func Get(name string) (Program, bool) {
	if _, err := files.ReadFile("sno/" + name + ".sno"); err != nil {
		return Program{}, false
	}
	return read(name), true
}

func read(name string) Program {
	p := Program{Name: name}
	p.Source, _ = files.ReadFile("sno/" + name + ".sno")

	var err error
	// A program may arrive before its goldens do; the test says so
	// rather than the loader panicking, and an empty golden is a real
	// answer that has to be told from a missing one.
	p.Want, err = files.ReadFile("sno/" + name + ".out")
	p.HasWant = err == nil
	p.Report, err = files.ReadFile("sno/" + name + ".err")
	p.HasReport = err == nil

	d := directives(p.Source)
	p.Doc = d["DOC"]
	p.Pending = d["PENDING"]
	p.Errors = d["ERRORS"]
	p.Runaway = d["RUNAWAY"]
	if n, err := strconv.Atoi(strings.TrimSpace(d["STACK"])); err == nil {
		p.Stack = n
	}
	return p
}

// directives reads what this package needs out of a program's own
// comment cards, so that none of it has to be repeated in a Go file
// that would then drift from the program it describes.
//
// The first comment card is "*  name.sno -- what it does.", and DOC is
// what follows the dashes. Every other card of the form "*  KEY: text"
// opens a directive whose text runs on over the comment cards that
// follow, until a blank comment card or the copyright card. Anything
// else is prose about the program and is ignored.
func directives(src []byte) map[string]string {
	out := map[string]string{}
	key := ""
	for _, line := range strings.Split(string(src), "\n") {
		if !strings.HasPrefix(line, "*") {
			break
		}
		text := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if k, rest, ok := directive(text); ok {
			key = k
			out[key] = strings.TrimSpace(rest)
			continue
		}
		if text == "" || strings.HasPrefix(text, "Part of ") {
			key = ""
			continue
		}
		if key != "" {
			out[key] = strings.TrimSpace(out[key] + " " + text)
			continue
		}
		if _, after, ok := strings.Cut(text, " -- "); ok && out["DOC"] == "" {
			out["DOC"] = after
		}
	}
	return out
}

// directive splits "KEY: text". A key is upper case and nothing else,
// which is what keeps an ordinary sentence with a colon in it from
// being read as one.
func directive(text string) (key, rest string, ok bool) {
	key, rest, ok = strings.Cut(text, ":")
	if !ok || key == "" {
		return "", "", false
	}
	for i := 0; i < len(key); i++ {
		if c := key[i]; c < 'A' || c > 'Z' {
			return "", "", false
		}
	}
	return key, rest, true
}

// Columns is how wide the statement field of a card is. Anything past
// it is a sequence number and does not reach the compiler, so a long
// line is silently truncated -- usually in the middle of a literal,
// which then reads as an error somewhere else entirely.
const Columns = 72

// TooWide reports the lines of a program that run past the statement
// field, as "line 14: 75 columns".
func TooWide(p Program) []string {
	var bad []string
	for i, line := range strings.Split(string(p.Source), "\n") {
		if n := len(strings.TrimRight(line, "\r")); n > Columns {
			bad = append(bad, fmt.Sprintf("line %d: %d columns", i+1, n))
		}
	}
	return bad
}

// Normalize prepares a stream for comparison against a golden file:
// trailing blanks come off every line, and the whole ends in one
// newline or in nothing.
//
// The blanks are not noise, they are carriage control doing its job --
// the compilation listing goes out under a format that pads every line
// to the width of the printer -- but a golden file whose meaning lives
// in ninety columns of trailing space is one that the next editor to
// touch it will quietly break. What the padding does is pkg/fortran's
// business and is tested there. Trailing blanks on a line are
// therefore not asserted here.
func Normalize(b []byte) []byte {
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	out := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if out == "" {
		return nil
	}
	return []byte(out + "\n")
}

// statistics is the line the system's own report ends at.
const statistics = "SNOBOL4 STATISTICS SUMMARY-"

// Diagnostics cuts the system's stream down to what it said about the
// program, dropping the banner in front and the statistics behind.
//
// Both of the dropped parts are covered elsewhere and neither can be
// compared here. The banner is fixed text and cmd/sil asserts it once.
// The statistics carry four timings in milliseconds, and a golden file
// cannot hold a number that depends on how busy the machine was.
func Diagnostics(system []byte) []byte {
	lines := strings.Split(string(system), "\n")

	// The banner is the first four lines, under a carriage control
	// that starts a page. Cutting to the second page start is what
	// finds the end of it wherever it moves to.
	from := 0
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, "\f") {
			from = i
			break
		}
	}
	to := len(lines)
	for i := from; i < len(lines); i++ {
		if strings.Contains(lines[i], statistics) {
			to = i
			break
		}
	}
	kept := strings.Join(lines[from:to], "\n")
	// The form feeds are carriage control, one per record, and they
	// say nothing here that the line breaks do not.
	return Normalize([]byte(strings.ReplaceAll(kept, "\f", "")))
}
