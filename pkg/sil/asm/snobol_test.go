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
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/fortran"
	"github.com/mdhender/sil/pkg/sil/asm"
)

// M9 and M10: the historical SNOBOL4 implementation runs, and it runs
// a SNOBOL4 program.
//
// M9's exit criterion is that every halt is a documented ENDEX or a
// named unimplemented opcode. It is an ENDEX, at status zero, which is
// 6.29's normal termination: the system compiles the program, executes
// it, prints its statistics and stops.
//
// M10's is the program of PLAN.md, and what it prints reaches the host
// through STPRNT, which is the path the source-language OUTPUT
// variable takes. The system's own banner and statistics go through
// OUTPUT (6.75) instead, under FORTRAN IV formats that nothing here
// interprets -- risk 9, still open -- so the two streams are kept
// apart and only the program's is asserted exactly.
func TestRunsASNOBOL4Program(t *testing.T) {
	name, src, err := corpus.Load()
	if errors.Is(err, corpus.ErrAbsent) {
		t.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		t.Fatal(err)
	}

	// -UNLIST is a control card (line 1394), and it turns the
	// compilation listing off. Without it the listing shares the
	// STPRNT path with the program's own output, which is correct and
	// unhelpful; with it, what reaches Print is exactly what the
	// program printed. Getting there at all exercises CARDTB, the
	// syntax table that tells a control card from a statement.
	const program = "-UNLIST\n" +
		"        X = 'HELLO'\n" +
		"        OUTPUT = X\n" +
		"END\n"

	// A clock that advances a second on every MSTIME, so that the run
	// has a duration and the statistics have a real number in them.
	h := &snobolHost{input: strings.NewReader(program), tick: 1000}
	var trace bytes.Buffer
	vm, ds := asm.Assemble(name, src, asm.Options{Host: h, Trace: &trace})
	if err := ds.Err(); err != nil {
		t.Fatalf("%d diagnostics:\n%v", len(ds), err)
	}
	vm.MaxCycles = 1000000

	if err := vm.Run(); err != nil {
		t.Fatalf("%v\n%s", err, tail(trace.String(), 30))
	}
	if !vm.Halted {
		t.Fatal("the machine stopped without halting")
	}
	// 6.29: I is the value ENDEX reports, and the system passes zero
	// for a run with no errors.
	if vm.Status != 0 {
		t.Errorf("ENDEX reported %d, want 0\n%s", vm.Status, tail(trace.String(), 30))
	}

	if got, want := h.printed.String(), "HELLO\n"; got != want {
		t.Errorf("the program printed %q, want %q", got, want)
	}
	// The system reported no compilation errors and one write, which
	// is the same fact seen from its own statistics. Both are lines
	// now rather than format strings, so the columns are asserted too.
	for _, want := range []string{
		"NO ERRORS DETECTED IN SOURCE PROGRAM",
		"              1 WRITES PERFORMED",
		"           1000 MS. COMPILATION TIME",
		"              2 STATEMENTS EXECUTED,       0 FAILED",
		// The one real number the system prints, and the whole of
		// that path: DVREAL divides a thousand milliseconds by two
		// statements, keeps the answer as IEEE bits in an address
		// field (3.1.1), OUTPUT hands the bits over as they stand,
		// and F15.2 is what decides they are a real number and not an
		// integer.
		"         500.00 MS. AVERAGE PER STATEMENT EXECUTED",
	} {
		if !strings.Contains(h.system.String(), want) {
			t.Errorf("the system did not report %q\n%s", want, h.system.String())
		}
	}
	t.Logf("%d instructions, %d units of core\n%s", vm.Cycles, len(vm.Core), h.system.String())
}

// A wider set of SNOBOL4 programs, each one chosen for the part of the
// machine it reaches that the HELLO program does not: the arithmetic
// group, the specifier group, the syntax tables through SPAN, RPLACE,
// the real-number group, and a defined function, which is the call
// model under the SNOBOL4 compiler rather than under a test program.
//
// These go beyond M10's exit criterion. They are here because each is
// a few lines and each is a different subsystem, and because a machine
// that runs one SNOBOL4 program and not another is more interesting
// than one that runs none.
func TestRunsMoreSNOBOL4Programs(t *testing.T) {
	name, src, err := corpus.Load()
	if errors.Is(err, corpus.ErrAbsent) {
		t.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name    string
		what    string
		program []string
		want    string
	}{
		{
			name: "arithmetic", what: "SUM, MULT and the compiler's precedence",
			program: []string{"        OUTPUT = 2 + 3 * 4"},
			want:    "14\n",
		},
		{
			name: "concatenation", what: "APDSP and the string structures",
			program: []string{"        OUTPUT = 'AB' 'CD'"},
			want:    "ABCD\n",
		},
		{
			name: "size", what: "GETLG and the value field of a title",
			program: []string{"        OUTPUT = SIZE('HELLO')"},
			want:    "5\n",
		},
		{
			name: "real numbers", what: "SPREAL, ADREAL and REALST",
			program: []string{"        OUTPUT = 1.5 + 2.25"},
			want:    "3.75\n",
		},
		{
			name: "pattern matching", what: "the pattern nodes and the goto field",
			program: []string{
				"        X = 'HELLO'",
				"        X 'ELL'                :S(YES)F(NO)",
				"YES     OUTPUT = 'MATCHED'     :(END)",
				"NO      OUTPUT = 'NO MATCH'",
			},
			want: "MATCHED\n",
		},
		{
			name: "SPAN", what: "CLERTB, PLUGTB and STREAM over SNABTB",
			program: []string{
				"        'ABC123DEF' SPAN('0123456789') . N",
				"        OUTPUT = N",
			},
			want: "123\n",
		},
		{
			name: "REPLACE", what: "RPLACE",
			program: []string{"        OUTPUT = REPLACE('ABC','AB','XY')"},
			want:    "XYC\n",
		},
		{
			name: "a loop", what: "the interpreter's goto field, round a loop",
			program: []string{
				"        I = 0",
				"L       I = I + 1",
				"        OUTPUT = I",
				"        LT(I,3)                :S(L)",
			},
			want: "1\n2\n3\n",
		},
		{
			name: "a defined function", what: "RCALL and RRTURN under the compiler",
			program: []string{
				"        DEFINE('F(X)')         :(M)",
				"F       F = X X                :(RETURN)",
				"M       OUTPUT = F('AB')",
			},
			want: "ABAB\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			text := "-UNLIST\n" + strings.Join(tt.program, "\n") + "\nEND\n"
			h := &snobolHost{input: strings.NewReader(text)}
			var trace bytes.Buffer
			vm, ds := asm.Assemble(name, src, asm.Options{Host: h, Trace: &trace})
			if err := ds.Err(); err != nil {
				t.Fatal(err)
			}
			vm.MaxCycles = 2000000

			if err := vm.Run(); err != nil {
				t.Fatalf("%s: %v\n%s", tt.what, err, tail(trace.String(), 30))
			}
			if vm.Status != 0 {
				t.Errorf("ENDEX reported %d, want 0\n%s", vm.Status, h.system.String())
			}
			if !strings.Contains(h.system.String(), "NO ERRORS DETECTED IN SOURCE PROGRAM") {
				t.Fatalf("the program did not compile:\n%s", h.system.String())
			}
			if got := h.printed.String(); got != tt.want {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
			t.Logf("%s: %d instructions", tt.what, vm.Cycles)
		})
	}
}

// snobolHost keeps the program's output and the system's apart:
// STPRNT is what the source-language OUTPUT variable reaches, and
// OUTPUT is what the system prints about itself.
//
// Both go under a FORTRAN IV format, which pkg/fortran reads. That is
// what makes these tests assert the lines S4D58's system was
// documented to print rather than the format strings it printed them
// with, and it is the only place the machine's real numbers meet the
// interpreter: the average below is computed by DVREAL, kept as bits
// in an address field, and rendered by F15.2.
type snobolHost struct {
	input   *strings.Reader
	printed bytes.Buffer
	system  bytes.Buffer

	// tick is what the clock advances by on every MSTIME, so that a
	// run has a measurable duration without depending on how fast the
	// machine this is running on happens to be.
	tick  int
	clock int
}

// unitO is the print unit PARMS chooses (6.20), and the one carriage
// control applies to.
const unitO = 6

func (h *snobolHost) Print(unit int, format, s []byte) (int, error) {
	records, err := fortran.Chars(format, s)
	if err != nil {
		return 0, err
	}
	return 0, h.write(&h.printed, unit, records)
}

func (h *snobolHost) Output(unit int, format []byte, values []int) error {
	records, err := fortran.Numbers(format, values)
	if err != nil {
		return err
	}
	return h.write(&h.system, unit, records)
}

func (h *snobolHost) write(w *bytes.Buffer, unit int, records []fortran.Record) error {
	lines := fortran.Lines(records)
	if unit != unitO {
		lines = nil
		for _, r := range records {
			lines = append(lines, string(r))
		}
	}
	for _, text := range lines {
		w.WriteString(text)
		w.WriteByte('\n')
	}
	return nil
}

func (h *snobolHost) Read(unit, n int) ([]byte, bool, error) {
	var line []byte
	for {
		c, err := h.input.ReadByte()
		if err != nil {
			if len(line) == 0 {
				return nil, true, nil
			}
			break
		}
		if c == '\n' {
			break
		}
		line = append(line, c)
	}
	if len(line) > n {
		line = line[:n]
	}
	return line, false, nil
}

func (h *snobolHost) Backspace(unit int) error { return nil }
func (h *snobolHost) EndFile(unit int) error   { return nil }
func (h *snobolHost) Rewind(unit int) error    { return nil }
func (h *snobolHost) Time() int {
	h.clock += h.tick
	return h.clock
}
func (h *snobolHost) Date() []byte { return nil }

// tail returns the last n lines, which is what a trace is worth
// reading when something stops.
func tail(trace string, n int) string {
	lines := strings.Split(strings.TrimRight(trace, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
