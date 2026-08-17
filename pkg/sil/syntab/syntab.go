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

// Package syntab is Appendix A of S4D58: the twenty-five syntax table
// descriptions, and the character classes of 4.1 they are written in
// terms of.
//
// 4.2 says a syntax table is a finite state machine's transition
// table, one entry per character of the character set, and that the
// tables "are generated from such descriptions using a (SNOBOL4)
// program in which the character classes and the order of the internal
// character codes are parameters. The use of some kind of automatic
// technique to generate the syntax tables is advisable, both to ensure
// accuracy and because of the large amount of data involved." This is
// that technique: the descriptions are held verbatim, so that they can
// be read against the appendix line by line, and expanded at load time
// against this machine's character codes.
//
// The two parameters 4.2 names are exactly the two things this package
// supplies over the appendix: Classes, which is 4.1's table of
// character classes in ASCII, and the size of the character set, which
// the caller passes in as ALPHSZ.
//
// The values a description refers to -- the four indicators, the PUT
// codes, the addresses of the tables themselves -- all belong to the
// assembly, and Build asks for them by name.
package syntab

import (
	"fmt"
	"strings"
)

// Appendix is S4D58 Appendix A, transcribed. Nothing is added and
// nothing is reordered; TestAppendixMatchesTheDocument checks it
// against the manual.
const Appendix = `
BEGIN BIOPTB
FOR(PLUS) PUT(ADDFN) GOTO(TBLKTB)
FOR(MINUS) PUT(SUBFN) GOTO(TBLKTB)
FOR(DOT) PUT(NAMFN) GOTO(TBLKTB)
FOR(DOLLAR) PUT(DOLFN) GOTO(TBLKTB)
FOR(STAR) PUT(MPYFN) GOTO(STARTB)
FOR(SLASH) PUT(DIVFN) GOTO(TBLKTB)
FOR(AT) PUT(BIATFN) GOTO(TBLKTB)
FOR(POUND) PUT(BIPDFN) GOTO(TBLKTB)
FOR(PERCENT) PUT(BIPRFN) GOTO(TBLKTB)
FOR(RAISE) PUT(EXPFN) GOTO(TBLKTB)
FOR(ORSYM) PUT(ORFN) GOTO(TBLKTB)
FOR(KEYSYM) PUT(BIAMFN) GOTO(TBLKTB)
FOR(NOTSYM) PUT(BINGFN) GOTO(TBLKTB)
FOR(QUESYM) PUT(BIQSFN) GOTO(TBLKTB)
ELSE ERROR
END BIOPTB

BEGIN CARDTB
FOR(CMT) PUT(CMTTYP) STOPSH
FOR(CTL) PUT(CTLTYP) STOPSH
FOR(CNT) PUT(CNTTYP) STOPSH
ELSE PUT(NEWTYP) STOPSH
END CARDTB

BEGIN DQLITB
FOR(DQUOTE) STOP
ELSE CONTIN
END DQLITB

BEGIN ELEMTB
FOR(NUMBER) PUT(ILITYP) GOTO(INTGTB)
FOR(LETTER) PUT(VARTYP) GOTO(VARTB)
FOR(SQUOTE) PUT(QLITYP) GOTO(SQLITB)
FOR(DQUOTE) PUT(QLITYP) GOTO(DQLITB)
FOR(LEFTPAREN) PUT(NSTTYP) STOP
ELSE ERROR
END ELEMTB

BEGIN EOSTB
FOR(EOS) STOP
ELSE CONTIN
END EOSTB

BEGIN FLITB
FOR(NUMBER) CONTIN
FOR(TERMINATOR) STOPSH
ELSE ERROR
END FLITB

BEGIN FRWDTB
FOR(BLANK) CONTIN
FOR(EQUAL) PUT(EQTYP) STOP
FOR(RIGHTPAREN) PUT(RPTYP) STOP
FOR(RIGHTBR) PUT(RBTYP) STOP
FOR(COMMA) PUT(CMATYP) STOP
FOR(COLON) PUT(CLNTYP) STOP
FOR(EOS) PUT(EOSTYP) STOP
ELSE PUT(NBTYP) STOPSH
END FRWDTB

BEGIN GOTFTB
FOR(LEFTPAREN) PUT(FGOTYP) STOP
FOR(LEFTBR) PUT(FTOTYP) STOP
ELSE ERROR
END GOTFTB

BEGIN GOTOTB
FOR(SGOSYM) GOTO(GOTSTB)
FOR(FGOSYM) GOTO(GOTFTB)
FOR(LEFTPAREN) PUT(UGOTYP) STOP
FOR(LEFTBR) PUT(UTOTYP) STOP
ELSE ERROR
END GOTOTB

BEGIN GOTSTB
FOR(LEFTPAREN) PUT(SGOTYP) STOP
FOR(LEFTBR) PUT(STOTYP) STOP
ELSE ERROR
END GOTSTB

BEGIN IBLKTB
FOR(BLANK) GOTO(FRWDTB)
FOR(EOS) PUT(EOSTYP) STOP
ELSE ERROR
END IBLKTB

BEGIN INTGTB
FOR(NUMBER) CONTIN
FOR(TERMINATOR) PUT(ILITYP) STOPSH
FOR(DOT) PUT(FLITYP) GOTO(FLITB)
ELSE ERROR
END INTGTB

BEGIN LBLTB
FOR(ALPHANUMERIC) GOTO(LBLXTB)
FOR(BLANK,EOS) STOPSH
ELSE ERROR
END LBLTB

BEGIN LBLXTB
FOR(BLANK,EOS) STOPSH
ELSE CONTIN
END LBLXTB

BEGIN NBLKTB
FOR(TERMINATOR) ERROR
ELSE STOPSH
END NBLKTB

BEGIN NUMBTB
FOR(NUMBER) GOTO(NUMCTB)
FOR(PLUS,MINUS) GOTO(NUMCTB)
FOR(COMMA) PUT(CMATYP) STOPSH
FOR(COLON) PUT(DIMTYP) STOPSH
ELSE ERROR
END NUMBTB

BEGIN NUMCTB
FOR(NUMBER) CONTIN
FOR(COMMA) PUT(CMATYP) STOPSH
FOR(COLON) PUT(DIMTYP) STOPSH
ELSE ERROR
END NUMCTB

BEGIN SNABTB
FOR(FGOSYM) STOP
FOR(SGOSYM) STOPSH
ELSE ERROR
END SNABTB

BEGIN SQLITB
FOR(SQUOTE) STOP
ELSE CONTIN
END SQLITB

BEGIN STARTB
FOR(BLANK) STOP
FOR(STAR) PUT(EXPFN) GOTO(TBLKTB)
ELSE ERROR
END STARTB

BEGIN TBLKTB
FOR(BLANK) STOP
ELSE ERROR
END TBLKTB

BEGIN UNOPTB
FOR(PLUS) PUT(PLSFN) GOTO(NBLKTB)
FOR(MINUS) PUT(MNSFN) GOTO(NBLKTB)
FOR(DOT) PUT(DOTFN) GOTO(NBLKTB)
FOR(DOLLAR) PUT(INDFN) GOTO(NBLKTB)
FOR(STAR) PUT(STRFN) GOTO(NBLKTB)
FOR(SLASH) PUT(SLHFN) GOTO(NBLKTB)
FOR(PERCENT) PUT(PRFN) GOTO(NBLKTB)
FOR(AT) PUT(ATFN) GOTO(NBLKTB)
FOR(POUND) PUT(PDFN) GOTO(NBLKTB)
FOR(KEYSYM) PUT(KEYFN) GOTO(NBLKTB)
FOR(NOTSYM) PUT(NEGFN) GOTO(NBLKTB)
FOR(ORSYM) PUT(BARFN) GOTO(NBLKTB)
FOR(QUESYM) PUT(QUESFN) GOTO(NBLKTB)
FOR(RAISE) PUT(AROWFN) GOTO(NBLKTB)
ELSE ERROR
END UNOPTB

BEGIN VARATB
FOR(LETTER) GOTO(VARBTB)
FOR(COMMA) PUT(CMATYP) STOPSH
FOR(RIGHTPAREN) PUT(RPTYP) STOPSH
ELSE ERROR
END VARATB

BEGIN VARBTB
FOR(ALPHANUMERIC,BREAK) CONTIN
FOR(LEFTPAREN) PUT(LPTYP) STOPSH
FOR(COMMA) PUT(CMATYP) STOPSH
FOR(RIGHTPAREN) PUT(RPTYP) STOPSH
ELSE ERROR
END VARBTB

BEGIN VARTB
FOR(ALPHANUMERIC,BREAK) CONTIN
FOR(TERMINATOR) PUT(VARTYP) STOPSH
FOR(LEFTPAREN) PUT(FNCTYP) STOP
FOR(LEFTBR) PUT(ARYTYP) STOP
ELSE ERROR
END VARTB
`

// A Rule is one FOR or ELSE line of a description.
type Rule struct {
	// Classes are the character classes the line applies to. An ELSE
	// line has none, and applies to every character no other line
	// claims.
	Classes []string
	// Goto is the table the next character is read with, empty for
	// CONTIN, which 4.2 makes the same table. It is empty for a
	// stopping rule too, where no next table is read.
	Goto string
	// Stop is which of the four indicators 4.2 lists the entry gets.
	Stop Indicator
	// Put is the symbol PUT names, empty when the line has no PUT.
	Put string
}

// Indicator is the type indicator field of an entry, T in 5.1's
// figure. The four names are the ones 6.19 and 6.76 accept as KEY, and
// PARMS chooses the numbers.
type Indicator string

const (
	Contin Indicator = "CONTIN"
	Stop   Indicator = "STOP"
	StopSh Indicator = "STOPSH"
	Error  Indicator = "ERROR"
)

// A Table is one description, in the order the appendix gives its
// lines.
type Table struct {
	Name  string
	Rules []Rule // the FOR lines
	Else  Rule   // the ELSE line
}

// Entry is one expanded syntax table entry: 5.1's A, T and P, with the
// indicator and the put field already resolved to numbers and the next
// table already resolved to an address.
type Entry struct {
	Next      int
	Indicator int
	Put       int
}

// Tables is the appendix, parsed. It is built once, and a malformed
// constant is a programming error rather than a run-time one.
var Tables = mustParse(Appendix)

// Names returns the table names, in the order the appendix gives them.
func Names() []string {
	out := make([]string, len(Tables))
	for i, t := range Tables {
		out[i] = t.Name
	}
	return out
}

// Lookup returns the table of a given name.
func Lookup(name string) (Table, bool) {
	for _, t := range Tables {
		if t.Name == name {
			return t, true
		}
	}
	return Table{}, false
}

// Build expands one table into alphsz entries, in character code
// order.
//
// value resolves every name the description uses against the assembly:
// the four indicators and the PUT codes, which are program symbols,
// and the table names, which are addresses. CONTIN resolves the
// table's own name, since 4.2 makes "the current table" an address
// like any other.
//
// A stopping entry gets a next-table address of zero. STREAM does not
// read the address field of an entry it stops on, and neither CLERTB
// nor PLUGTB writes one for STOP, STOPSH or ERROR, so there is nothing
// for it to be.
func Build(name string, alphsz int, value func(string) (int, bool)) ([]Entry, error) {
	t, ok := Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%s is not a table Appendix A describes", name)
	}

	number := func(n string) (int, error) {
		v, ok := value(n)
		if !ok {
			return 0, fmt.Errorf("%s: %s is not defined", name, n)
		}
		return v, nil
	}

	entry := func(r Rule) (Entry, error) {
		var e Entry
		ind, err := number(string(r.Stop))
		if err != nil {
			return e, err
		}
		e.Indicator = ind
		if r.Stop == Contin {
			to := r.Goto
			if to == "" {
				to = name
			}
			if e.Next, err = number(to); err != nil {
				return e, err
			}
		}
		if r.Put != "" {
			if e.Put, err = number(r.Put); err != nil {
				return e, err
			}
		}
		return e, nil
	}

	fallback, err := entry(t.Else)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, alphsz)
	for i := range out {
		out[i] = fallback
	}

	claimed := map[byte]string{}
	for _, r := range t.Rules {
		e, err := entry(r)
		if err != nil {
			return nil, err
		}
		for _, class := range r.Classes {
			chars, ok := Classes[class]
			if !ok {
				return nil, fmt.Errorf("%s: %s is not a character class of 4.1", name, class)
			}
			for _, c := range chars {
				// Within one table the FOR lines are disjoint, so the
				// appendix never says what an overlap would mean. If
				// one ever appears it is a transcription error, not a
				// precedence rule to guess at.
				if was, dup := claimed[c]; dup {
					return nil, fmt.Errorf("%s: character %d is claimed by both %s and %s",
						name, c, was, class)
				}
				claimed[c] = class
				if int(c) >= alphsz {
					return nil, fmt.Errorf("%s: %s names character %d, outside the %d of the character set",
						name, class, c, alphsz)
				}
				out[c] = e
			}
		}
	}
	return out, nil
}

func mustParse(text string) []Table {
	tables, err := parse(text)
	if err != nil {
		panic("syntab: " + err.Error())
	}
	return tables
}

// parse reads the description language of 4.2: BEGIN name, a run of
// FOR lines, an ELSE line, END name.
func parse(text string) ([]Table, error) {
	var out []Table
	var cur *Table
	var ended bool

	for n, line := range strings.Split(text, "\n") {
		where := func(format string, args ...any) error {
			return fmt.Errorf("line %d: %s: %s", n+1, line, fmt.Sprintf(format, args...))
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch head := fields[0]; {
		case head == "BEGIN":
			if cur != nil {
				return nil, where("%s is not ended", cur.Name)
			}
			if len(fields) != 2 {
				return nil, where("BEGIN takes one name")
			}
			out = append(out, Table{Name: fields[1]})
			cur, ended = &out[len(out)-1], false

		case head == "END":
			if cur == nil {
				return nil, where("END outside a table")
			}
			if len(fields) != 2 || fields[1] != cur.Name {
				return nil, where("END does not name %s", cur.Name)
			}
			if !ended {
				return nil, where("%s has no ELSE", cur.Name)
			}
			cur = nil

		case cur == nil:
			return nil, where("outside a table")

		case ended:
			return nil, where("follows the ELSE of %s", cur.Name)

		case strings.HasPrefix(head, "FOR("):
			classes, err := inside(head, "FOR")
			if err != nil {
				return nil, where("%v", err)
			}
			r, err := rule(fields[1:])
			if err != nil {
				return nil, where("%v", err)
			}
			r.Classes = strings.Split(classes, ",")
			cur.Rules = append(cur.Rules, r)

		case head == "ELSE":
			r, err := rule(fields[1:])
			if err != nil {
				return nil, where("%v", err)
			}
			cur.Else, ended = r, true

		default:
			return nil, where("unknown line")
		}
	}
	if cur != nil {
		return nil, fmt.Errorf("%s is not ended", cur.Name)
	}
	return out, nil
}

// rule reads what follows FOR(...) or ELSE: an optional PUT and
// exactly one of the five actions 4.2 lists.
func rule(fields []string) (Rule, error) {
	var r Rule
	var acted bool
	for _, f := range fields {
		switch {
		case strings.HasPrefix(f, "PUT("):
			put, err := inside(f, "PUT")
			if err != nil {
				return r, err
			}
			if r.Put != "" {
				return r, fmt.Errorf("two PUTs")
			}
			r.Put = put

		case strings.HasPrefix(f, "GOTO("):
			to, err := inside(f, "GOTO")
			if err != nil {
				return r, err
			}
			r.Goto, r.Stop, acted = to, Contin, true

		case f == string(Contin) || f == string(Stop) || f == string(StopSh) || f == string(Error):
			r.Stop, acted = Indicator(f), true

		default:
			return r, fmt.Errorf("unknown action %s", f)
		}
	}
	if !acted {
		return r, fmt.Errorf("no action")
	}
	return r, nil
}

// inside returns the argument of NAME(argument).
func inside(field, name string) (string, error) {
	arg, ok := strings.CutPrefix(field, name+"(")
	if !ok {
		return "", fmt.Errorf("%s is not %s(...)", field, name)
	}
	arg, ok = strings.CutSuffix(arg, ")")
	if !ok {
		return "", fmt.Errorf("%s is not closed", field)
	}
	if arg == "" {
		return "", fmt.Errorf("%s is empty", field)
	}
	return arg, nil
}
