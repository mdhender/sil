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

package asm_test

import (
	"bytes"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/programs"
	"github.com/mdhender/sil/pkg/sil"
	"github.com/mdhender/sil/pkg/sil/asm"
	"github.com/mdhender/sil/pkg/sil/op"
)

// Which of the 119 operations the historical implementation actually
// executes when it runs a SNOBOL4 program.
//
// # Why this is a different claim from the unit tests
//
// Every operation has unit tests and at least one hand-written SIL
// program behind it, and that pair says the operation does what 6.x
// describes when this repository drives it. It does not say the
// historical source ever reaches it, or that it reaches it with
// operands of the shape the SNOBOL4 system actually builds. Those are
// the interesting failures -- an operation that is right against the
// document and wrong against its only caller passes every other test
// here.
//
// So this measures the other direction: assemble the unmodified
// engine, run SNOBOL4 through it, and record what the machine
// executed. An operation counted here has been driven by the 1981
// implementation rather than by us.
//
// # How it measures
//
// The machine's own trace, one line per instruction, with the opcode
// in the second field. Names are kept only when op.Lookup knows them,
// so a comment or a source line that happens to tokenize oddly cannot
// invent an operation.
//
// # Why the probes are not held to golden output
//
// internal/programs is where a program with an answer belongs, and
// those are compared against a .out. The probes below exist to reach
// operations no such program does, and two of them cannot be goldens:
// DATE returns today, and the file-positioning group returns nothing
// at all. What is asserted about a probe is that it runs without the
// machine faulting -- which, for these operations, is the claim.
func TestOperationCoverage(t *testing.T) {
	name, src := engine(t)

	executed := make(map[string]bool)
	for _, p := range programs.All() {
		t.Run(p.Name, func(t *testing.T) {
			record(t, executed, name, src, programs.Card+string(p.Source), p.Stack, p.Runaway != "")
		})
	}
	for _, p := range probes {
		t.Run(p.name, func(t *testing.T) {
			record(t, executed, name, src, p.source, 0, false)
		})
	}

	// Every executable operation is either executed or excused, and
	// the excuses are the literal set below. A new program that
	// reaches one of them, or an implementation that stops taking
	// 7.1's alternative, fails here and says so.
	var missing []string
	for _, k := range op.Kinds() {
		e := op.Get(k)
		if e.Directive || executed[e.Mnemonic] {
			continue
		}
		if _, ok := unreached[e.Mnemonic]; !ok {
			missing = append(missing, e.Mnemonic)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Errorf("no SNOBOL4 program here executes %s.\n"+
			"Either write one that does -- a probe below, or better, a program in\n"+
			"internal/programs with a golden -- or add it to unreached with the\n"+
			"reason it cannot be reached.", strings.Join(missing, ", "))
	}

	for mnemonic, why := range unreached {
		if executed[mnemonic] {
			t.Errorf("%s is listed as unreachable, and something executed it:\n"+
				"unreached says %q.\n"+
				"That reason is now wrong. Delete the entry.", mnemonic, why)
		}
	}

	// The count is the claim, so it is reported rather than left to be
	// recomputed by hand. op.Count is all 131 mnemonics; the twelve
	// directives assemble rather than execute and are not in it.
	var executable int
	for _, k := range op.Kinds() {
		if !op.Get(k).Directive {
			executable++
		}
	}
	t.Logf("%d of %d executable operations were executed by the historical "+
		"implementation; %d excused, see unreached", len(executed), executable, len(unreached))
}

// unreached is every executable operation that no SNOBOL4 program here
// drives, and why not. Four of the five are operations this machine
// implements by the alternative the document offers rather than as
// written, so there is nothing to reach; the fifth is a real gap.
var unreached = map[string]string{
	"LOAD": "7.1's alternative: this machine has no external functions, so LOAD " +
		"branches to UNDF (6.57 note 2) and no SNOBOL4 program can get past it",
	"LINK": "7.1's alternative: nothing reaches a LINK without a LOAD having " +
		"succeeded, and none can (6.55 note 2)",
	"UNLOAD": "7.1's alternative, which 6.126 note 2 requires rather than permits: " +
		"no operation",
	"ORDVST": "6.74 note 1's documented alternative -- no operation -- so the one " +
		"call site (sil-v3.11.sil:5118) does nothing. It runs only under &DUMP",
	"INCRV": "a real gap rather than an excuse. One call site " +
		"(sil-v3.11.sil:3543, 'Increment data type code') and nothing written " +
		"here drives the path that reaches it",
}

// The probes reach operations the corpus in internal/programs does
// not. Each is the smallest SNOBOL4 that gets there, and the comment
// says which operations it is for, so an operation that stops being
// reached can be traced back to the line that was supposed to reach
// it.
var probes = []struct {
	name   string
	source string
}{
	{
		// ADREAL SBREAL MPREAL DVREAL EXREAL MNREAL, RCOMP (the last
		// comparison), SPREAL and REALST (the two conversions), and
		// DATE. 7.1 marks the whole real-number group optional; it is
		// implemented, so it is exercised.
		name: "real-arithmetic",
		source: `        OUTPUT = 1.5 + 2.25
        OUTPUT = 3.5 - 1.25
        OUTPUT = 2.0 * 3.5
        OUTPUT = 7.0 / 2.0
        OUTPUT = 2.0 ** 3.0
        OUTPUT = -4.5
        OUTPUT = LT(1.5,2.5) 'LESS'
        OUTPUT = 'X' DATE()
END
`,
	},
	{
		// EXPINT and MNSINT, the two integer operations no program in
		// internal/programs reaches, and INSERT.
		//
		// INSERT is the compiler's, not the interpreter's: 6.48's one
		// call site (sil-v3.11.sil:1293) is in the expression parser,
		// which inserts a node above when an operator arrives whose
		// precedence is lower than the subtree already built. One
		// level of climbing is done by ADDSIB, so reaching INSERT
		// takes two -- which is why the alternation below has a
		// three-element concatenation on the right of the bar and not
		// the two-element one that would read as the obvious test.
		name: "integers-and-precedence",
		source: `        OUTPUT = 2 ** 10
        OUTPUT = -5
        P = 'A' 'B' 'C' | 'D' 'E' 'F'
        'ZDEFZ' P . W                  :F(NO)
        OUTPUT = W
NO      OUTPUT = 'DONE'
END
`,
	},
	{
		// GETBAL through the BAL pattern, LINKOR through alternation,
		// RLINT through CONVERT, and ENFILE, REWIND and BKSPCE, which
		// are the three file-positioning operations. Nothing in
		// internal/programs alternates or positions a file.
		name: "patterns-and-positioning",
		source: `        X = '(A(B)C)'
        X BAL . Y                      :F(N1)
        OUTPUT = Y
N1      'ABC' ('X' | 'B') . Z          :F(N2)
        OUTPUT = Z
N2      OUTPUT = CONVERT(3.7,'INTEGER')
        ENDFILE(6)
        REWIND(6)
        BACKSPACE(6)
END
`,
	},
}

// record runs one deck on a fresh machine and adds every operation the
// machine executed to seen.
func record(t *testing.T, seen map[string]bool, name string, src []byte, deck string, stack int, runaway bool) {
	t.Helper()

	w := &opWriter{seen: seen}
	h := &snobolHost{input: strings.NewReader(deck)}
	opts := asm.Options{Trace: w, Host: h}
	if stack != 0 {
		opts.Equates = map[string]int{"STSIZE": stack}
	}

	vm, ds := asm.Assemble(name, src, opts)
	if err := ds.Err(); err != nil {
		t.Fatalf("%d diagnostics: %v", len(ds), err)
	}

	// A program that does not stop needs a bound, and it does not need
	// the real one: everything it executes, it executes early, and
	// tracing a hundred million instructions to learn nothing new is
	// the whole cost of this test.
	vm.MaxCycles = 20_000_000
	if runaway {
		vm.MaxCycles = 200_000
	}

	err := vm.Run()
	var stopped *sil.Bound
	switch {
	case errors.As(err, &stopped):
		if !runaway {
			t.Fatalf("this did not stop, and nothing says it should not: %v\n%s",
				stopped, h.system.String())
		}
	case err != nil:
		t.Fatalf("the machine faulted: %v\n%s", err, h.system.String())
	}
	w.flush()
}

// opWriter reads the machine's trace as it is written and keeps the
// opcodes out of it. The trace is one line per instruction with the
// opcode in the second field (see VM.Step); a name op.Lookup does not
// know is not an operation and is dropped, so nothing in a source
// line's text can be counted as one.
type opWriter struct {
	seen map[string]bool
	rest []byte
}

func (w *opWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.rest = append(w.rest, p...)
	for {
		i := bytes.IndexByte(w.rest, '\n')
		if i < 0 {
			return n, nil
		}
		w.line(w.rest[:i])
		w.rest = w.rest[i+1:]
	}
}

// flush counts a last line that had no newline after it.
func (w *opWriter) flush() {
	w.line(w.rest)
	w.rest = nil
}

func (w *opWriter) line(b []byte) {
	f := strings.Fields(string(b))
	if len(f) < 2 {
		return
	}
	if op.Lookup(f[1]) != op.Invalid {
		w.seen[f[1]] = true
	}
}
