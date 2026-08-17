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

// Command sil runs SNOBOL4 programs.
//
// It carries the historical Macro SNOBOL4 implementation -- 6580 lines
// of SIL, the machine language S4D58 describes -- assembles it, and
// runs it. The SNOBOL4 program named on the command line is what that
// implementation then reads and compiles, the way it read a deck of
// cards.
//
//	sil hello.sno
//	sil < hello.sno
//	sil -listing core.txt hello.sno
//	sil -stack 35k deep.sno
//
// The exit status is 0 when the run reached ENDEX, however the program
// went, and 1 when something stopped it first: a source that would not
// assemble, a machine fault, a file that would not open. SNOBOL4 has
// no notion of an exit status to report -- ENDEX's operand is the
// keyword &ABEND (6.29 note 2), not a status -- and a program whose
// statements failed is a program that ran.
//
// # Where the output goes
//
// Standard output is what the program printed. Standard error is what
// the SNOBOL4 system printed about itself: its banner, whether the
// program compiled, and the statistics at the end.
//
// They are separated because they leave the machine by different
// operations: a program's output goes through STPRNT (S4D58 6.114) and
// the system's own messages go through OUTPUT (6.75). Both go under a
// FORTRAN IV format, which pkg/fortran reads, so both come out
// typeset; keeping them apart is what makes the program's output the
// only thing on standard output. Use -merge for the single stream the
// original printed, in the order it printed it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mdhender/sil/engines"
	"github.com/mdhender/sil/pkg/sil/asm"
)

func main() {
	status, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sil: %v\n", err)
		os.Exit(1)
	}
	os.Exit(status)
}

// run does the whole of it. Reaching ENDEX is success; everything that
// stops the machine short of one is an error, and main turns that into
// 1.
func run(args []string, stdout, stderr io.Writer) (int, error) {
	fs := flag.NewFlagSet("sil", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		engine  = fs.String("engine", "", "assemble this SIL source instead of the embedded SNOBOL4")
		listing = fs.String("listing", "", "write a core listing to this file")
		trace   = fs.String("trace", "", "write one line per instruction executed to this file")
		max     = fs.Int("max", 0, "stop after this many instructions; 0 does not stop")
		dynamic = fs.Int("dynamic", 0, "descriptors of dynamic storage; 0 takes the default")
		stack   = fs.String("stack", "", "descriptors of interpreter stack, as 35000 or 35k; empty keeps what the SIL source chose")
		merge   = fs.Bool("merge", false, "put the system's own output on standard output, with the program's")
	)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: sil [flags] [program.sno ...]\n\n"+
			"Runs a SNOBOL4 program on the historical Macro SNOBOL4 implementation.\n"+
			"With no files, the program is read from standard input.\n\n"+
			"Standard output is what the program printed. Standard error is what the\n"+
			"SNOBOL4 system printed about itself.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 0, err
	}

	name, src, err := source(*engine)
	if err != nil {
		return 0, err
	}

	program, closeInput, err := input(fs.Args())
	if err != nil {
		return 0, err
	}
	defer closeInput()

	system := io.Writer(stderr)
	if *merge {
		system = stdout
	}
	host := &host{out: stdout, system: system, in: program, printer: -1}

	traced, closeTrace, err := create(*trace)
	if err != nil {
		return 0, err
	}
	defer closeTrace()

	equates, err := equates(*stack)
	if err != nil {
		return 0, err
	}

	vm, ds := asm.Assemble(name, src, asm.Options{Host: host, Trace: traced, Equates: equates})
	if err := ds.Err(); err != nil {
		return 0, fmt.Errorf("%s: %d diagnostics:\n%v", name, len(ds), err)
	}
	vm.MaxCycles, vm.Dynamic = *max, *dynamic

	// Carriage control belongs to the printer, so the host has to know
	// which unit that is. PARMS names it (6.20), and a SIL source that
	// does not copy PARMS has no printer, which leaves every unit
	// taking its records whole.
	if unit, ok := vm.Symbols["UNITO"]; ok {
		host.printer = unit
	}

	if *listing != "" {
		w, closeListing, err := create(*listing)
		if err != nil {
			return 0, err
		}
		err = asm.Listing(w, vm)
		closeListing()
		if err != nil {
			return 0, fmt.Errorf("%s: %w", *listing, err)
		}
	}

	if err := vm.Run(); err != nil {
		return 0, err
	}
	// 6.29: "If I is nonzero, a post-mortem dump of user core should
	// be given", and I is the source-language keyword &ABEND. There is
	// no dump here, which 6.29 note 1 allows -- "&ABEND will not have
	// its specified effect. Nothing else will be affected" -- so the
	// request is reported rather than silently dropped.
	if vm.Status != 0 {
		fmt.Fprintf(stderr, "sil: the program set &ABEND to %d, and this machine has no core dump to give\n",
			vm.Status)
	}
	return 0, nil
}

// equates turns the sizing flags into the EQU overrides the assembler
// takes. There is one so far.
//
// STSIZE is the interpreter stack, and the historical source sets it
// to a thousand descriptors -- enough for about forty levels of a
// recursive defined function, which was a reasonable call for the core
// a 1975 machine had and is a low ceiling now. Raising it here rather
// than in the source keeps engines/sil-v3.11.sil read-only input.
func equates(stack string) (map[string]int, error) {
	if stack == "" {
		return nil, nil
	}
	n, err := size(stack)
	if err != nil {
		return nil, fmt.Errorf("-stack %s: %w", stack, err)
	}
	if n <= 0 {
		return nil, fmt.Errorf("-stack %s: the stack has to hold something", stack)
	}
	return map[string]int{"STSIZE": n}, nil
}

// size reads a count that may carry a k or m suffix, so that a stack
// of thirty-five thousand descriptors can be written the way anybody
// would say it.
func size(s string) (int, error) {
	scale := 1
	switch {
	case strings.HasSuffix(s, "k"), strings.HasSuffix(s, "K"):
		scale, s = 1000, s[:len(s)-1]
	case strings.HasSuffix(s, "m"), strings.HasSuffix(s, "M"):
		scale, s = 1000000, s[:len(s)-1]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("not a count")
	}
	return n * scale, nil
}

// source returns the SIL source to assemble: the file named by
// -engine, or the embedded SNOBOL4 implementation.
func source(engine string) (string, []byte, error) {
	if engine != "" {
		src, err := os.ReadFile(engine)
		if err != nil {
			return "", nil, err
		}
		return engine, src, nil
	}
	name, src, err := engines.Source()
	if errors.Is(err, engines.ErrAbsent) {
		return "", nil, fmt.Errorf(
			"%s was not in the tree when this binary was built.\n"+
				"It is assumed not to be redistributable; see engines/README.md for\n"+
				"where to fetch it, then rebuild. Or pass -engine to assemble another\n"+
				"SIL source", name)
	}
	return name, src, err
}

// input returns the SNOBOL4 program: the named files, read one after
// another as one deck, or standard input when none are named.
func input(names []string) (io.Reader, func(), error) {
	if len(names) == 0 {
		return os.Stdin, func() {}, nil
	}
	var readers []io.Reader
	var open []*os.File
	closeAll := func() {
		for _, f := range open {
			f.Close()
		}
	}
	for _, name := range names {
		f, err := os.Open(name)
		if err != nil {
			closeAll()
			return nil, nil, err
		}
		open = append(open, f)
		readers = append(readers, f)
		// A deck is lines; a file that does not end in one would
		// otherwise run its last line into the next file's first.
		//
		// Only when it does not end in one. A newline added to a file
		// that already had it is a blank card, and a blank card is a
		// statement: it takes a number, and every number after it in
		// the listing and in every error message moves up by one. The
		// deck is the files and nothing else.
		ends, err := endsInNewline(f)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}
		if !ends {
			readers = append(readers, strings.NewReader("\n"))
		}
	}
	return io.MultiReader(readers...), closeAll, nil
}

// endsInNewline reports whether a file's last byte is a newline,
// leaving it positioned where it was. An empty file counts as ending
// in one, having no last line to run into the next file's first.
func endsInNewline(f *os.File) (bool, error) {
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() == 0 {
		return true, nil
	}
	var last [1]byte
	if _, err := f.ReadAt(last[:], info.Size()-1); err != nil {
		return false, err
	}
	return last[0] == '\n', nil
}

// create opens a file for writing, or returns nil for no file, which
// is what the machine and the listing both take to mean "do not".
func create(name string) (io.Writer, func(), error) {
	if name == "" {
		return nil, func() {}, nil
	}
	f, err := os.Create(name)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}
