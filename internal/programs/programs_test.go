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

package programs

import (
	"strings"
	"testing"
)

// The directives are read out of each program's own comment cards
// rather than kept in a Go file beside it, so that neither can drift
// from the program it describes. This is that reading.
func TestDirectives(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
		want map[string]string
	}{
		{
			name: "a description on the first card",
			src: "*  fizzbuzz.sno -- one to a hundred.\n" +
				"*  Part of github.com/mdhender/sil. BSD 2-Clause; see LICENSE.\n" +
				"        OUTPUT = 1\nEND\n",
			want: map[string]string{"DOC": "one to a hundred."},
		},
		{
			name: "a directive that runs over several cards",
			src: "*  charcode.sno -- a character to its code.\n" +
				"*  PENDING: ALPHA is never filled in, so &ALPHABET reads\n" +
				"*  as blanks. Clear this when it is.\n" +
				"*  Part of github.com/mdhender/sil. BSD 2-Clause; see LICENSE.\n" +
				"        OUTPUT = 1\nEND\n",
			want: map[string]string{
				"DOC":     "a character to its code.",
				"PENDING": "ALPHA is never filled in, so &ALPHABET reads as blanks. Clear this when it is.",
			},
		},
		{
			name: "the copyright card ends a directive",
			src: "*  x.sno -- a thing.\n" +
				"*  ERRORS: one line only.\n" +
				"*  Part of github.com/mdhender/sil. BSD 2-Clause; see LICENSE.\n" +
				"*\n" +
				"*  This paragraph is prose about the program, not the reason.\n" +
				"        OUTPUT = 1\nEND\n",
			want: map[string]string{"DOC": "a thing.", "ERRORS": "one line only."},
		},
		{
			name: "a blank comment card ends a directive",
			src: "*  x.sno -- a thing.\n" +
				"*  RUNAWAY: it never stops.\n" +
				"*\n" +
				"*  Prose that is not part of the reason.\n" +
				"        OUTPUT = 1\nEND\n",
			want: map[string]string{"DOC": "a thing.", "RUNAWAY": "it never stops."},
		},
		{
			name: "a value that is a number",
			src:  "*  x.sno -- a thing.\n*  STACK: 35000\n        OUTPUT = 1\nEND\n",
			want: map[string]string{"DOC": "a thing.", "STACK": "35000"},
		},
		{
			// A colon in ordinary prose is not a directive, or every
			// sentence in a header would become one.
			name: "prose with a colon in it",
			src: "*  x.sno -- a thing.\n" +
				"*  It works like this: not like that.\n" +
				"        OUTPUT = 1\nEND\n",
			want: map[string]string{"DOC": "a thing."},
		},
		{
			name: "no comment cards at all",
			src:  "        OUTPUT = 1\nEND\n",
			want: map[string]string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := directives([]byte(tt.src))
			if len(got) != len(tt.want) {
				t.Errorf("read %v, want %v", got, tt.want)
				return
			}
			for k, want := range tt.want {
				if got[k] != want {
					t.Errorf("%s is %q, want %q", k, got[k], want)
				}
			}
		})
	}
}

// Normalize is what lets a compilation listing be kept in a golden
// file: it goes out padded to the width of the printer, and no editor
// leaves ninety columns of trailing space alone.
func TestNormalize(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"a   \nb\t\n", "a\nb\n"},
		{"a\n\n\n", "a\n"},
		{"a", "a\n"},
		{"   \n  \n", ""},
		{"", ""},
	} {
		if got := string(Normalize([]byte(tt.in))); got != tt.want {
			t.Errorf("Normalize(%q) is %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Diagnostics keeps what the system said about the program and drops
// the two parts around it: the banner, which is fixed, and the
// statistics, which carry timings a golden file cannot hold.
func TestDiagnostics(t *testing.T) {
	system := "\fSNOBOL4 (VERSION 3.11, MAY 19, 1975)\n" +
		"_______\n\n" +
		"BELL TELEPHONE LABORATORIES, INCORPORATED\n" +
		"\f\n\n" +
		"NO ERRORS DETECTED IN SOURCE PROGRAM\n" +
		"\f\n" +
		"\fERROR 24 IN STATEMENT    2 AT LEVEL  0\n" +
		"\fSNOBOL4 STATISTICS SUMMARY-\n\n" +
		"              0 MS. COMPILATION TIME\n"

	got := string(Diagnostics([]byte(system)))
	want := "\n\nNO ERRORS DETECTED IN SOURCE PROGRAM\n\nERROR 24 IN STATEMENT    2 AT LEVEL  0\n"
	if got != want {
		t.Errorf("Diagnostics is %q,\nwant                %q", got, want)
	}
	if strings.Contains(got, "MS.") {
		t.Error("a timing survived, and a golden file cannot hold one")
	}
	if strings.Contains(got, "BELL TELEPHONE") {
		t.Error("the banner survived")
	}
}

// The programs are embedded, so unlike every other corpus in this
// repository they are here on every checkout and this does not skip.
func TestTheProgramsAreAllHere(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("no programs in the embed")
	}
	for _, p := range all {
		if len(p.Source) == 0 {
			t.Errorf("%s has no source", p.Name)
		}
		if !p.HasWant {
			t.Errorf("%s has no %s.out beside it", p.Name, p.Name)
		}
		// An END card is what stops the compiler reading. A program
		// without one does not stop, which is noend.sno's whole point
		// and nothing else's.
		hasEnd := strings.HasSuffix(string(p.Source), "END\n")
		if hasEnd == (p.Runaway != "") {
			if hasEnd {
				t.Errorf("%s says RUNAWAY and has an END card, so it would stop", p.Name)
			} else {
				t.Errorf("%s has no END card, so the compiler reads past it for ever.\n"+
					"If that is the point, say so with a RUNAWAY: directive.", p.Name)
			}
		}
		// An invalid program is worth nothing without the diagnostic
		// it is supposed to produce.
		if p.Errors != "" && !p.HasReport {
			t.Errorf("%s says ERRORS and has no %s.err beside it", p.Name, p.Name)
		}
		if p.Errors == "" && p.Runaway == "" && p.HasReport {
			t.Errorf("%s has a %s.err and is not supposed to fail", p.Name, p.Name)
		}
		if bad := TooWide(p); len(bad) != 0 {
			t.Errorf("%s runs past column %d:\n  %s", p.Name, Columns, strings.Join(bad, "\n  "))
		}
		if p.Doc == "" {
			t.Errorf("%s has no `*  %s.sno -- what it does.` first card", p.Name, p.Name)
		}
	}

	// A corpus where everything passes and some entries are quietly
	// excused is a corpus that has stopped telling the truth, so what
	// is excused gets said out loud.
	var valid, invalid, pending int
	for _, p := range all {
		switch {
		case p.Pending != "":
			pending++
			t.Logf("%s is pending: %s", p.Name, p.Pending)
		case p.Errors != "":
			invalid++
			t.Logf("%s is invalid on purpose: %s", p.Name, p.Errors)
		case p.Runaway != "":
			invalid++
			t.Logf("%s does not stop: %s", p.Name, p.Runaway)
		default:
			valid++
		}
	}
	t.Logf("%d programs: %d valid, %d invalid on purpose, %d pending",
		len(all), valid, invalid, pending)

	if _, ok := Get("hello"); !ok {
		t.Error("Get did not find hello")
	}
	if _, ok := Get("no-such-program"); ok {
		t.Error("Get found a program that is not there")
	}
}
