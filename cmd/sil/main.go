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
// operations and only one of them is finished. A program's output goes
// through STPRNT (S4D58 6.114), which hands over the characters of a
// string; the system's own messages go through OUTPUT (6.75), which
// hands over a FORTRAN IV format and a list of numbers, and nothing
// here interprets a FORTRAN format yet. So the system's stream is
// legible rather than typeset, and keeping it off standard output
// keeps it out of the program's. Use -merge for the single stream the
// original printed, in the order it printed it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
	host := &host{out: stdout, system: system, in: program}

	traced, closeTrace, err := create(*trace)
	if err != nil {
		return 0, err
	}
	defer closeTrace()

	vm, ds := asm.Assemble(name, src, asm.Options{Host: host, Trace: traced})
	if err := ds.Err(); err != nil {
		return 0, fmt.Errorf("%s: %d diagnostics:\n%v", name, len(ds), err)
	}
	vm.MaxCycles, vm.Dynamic = *max, *dynamic

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
		readers = append(readers, strings.NewReader("\n"))
	}
	return io.MultiReader(readers...), closeAll, nil
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
