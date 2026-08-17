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

package main

import (
	"bufio"
	"fmt"
	"io"
	"time"
)

// host is what the machine asks the outside world for (S4D58 2.1).
//
// One input stream and two output streams, and it ignores the unit
// reference number on all of them. 2.1 gives the SNOBOL4 system three
// units -- input, print output and punch output -- and a command-line
// runner has one place for each direction, so a unit is not a thing
// this host distinguishes. A host that wanted files per unit would
// implement the same interface.
type host struct {
	out    io.Writer // what the program printed
	system io.Writer // what the SNOBOL4 system printed about itself
	in     io.Reader

	deck    *bufio.Reader
	started time.Time
}

// Print writes what a program printed, and what the compiler listed.
// The format is dropped: 6.114 note 1 calls it a FORTRAN IV format in
// undigested form, and this host has no interpreter for one. The
// characters are a whole line either way, so they go out as one.
func (h *host) Print(unit int, format, s []byte) (int, error) {
	if err := line(h.out, s); err != nil {
		return 0, err
	}
	// 6.114 note 3: the condition the output routine signals is not
	// used.
	return 0, nil
}

// Output writes what the SNOBOL4 system says about itself. Until there
// is a FORTRAN IV interpreter this is the format and the numbers it
// would have been given, which is legible rather than typeset.
func (h *host) Output(unit int, format []byte, values []int) error {
	out := append([]byte{}, format...)
	for _, v := range values {
		out = append(out, ' ')
		out = fmt.Appendf(out, "%d", v)
	}
	return line(h.system, out)
}

// Read hands over one line of the deck, truncated to what STREAD asked
// for. A record shorter than that is padded with blanks by the machine
// (6.115 note 1), so a short line is a short card and not an error.
func (h *host) Read(unit, n int) ([]byte, bool, error) {
	if h.deck == nil {
		h.deck = bufio.NewReader(h.in)
	}
	text, err := h.deck.ReadBytes('\n')
	if len(text) == 0 && err == io.EOF {
		return nil, true, nil
	}
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	text = trimNewline(text)
	if len(text) > n {
		text = text[:n]
	}
	return text, false, nil
}

// A stream of lines cannot be positioned, and 6.14, 6.30 and 6.92 have
// no failure branch, so failing here would stop the machine over
// something the SNOBOL4 functions BACKSPACE, ENDFILE and REWIND are
// entitled to ask for and not entitled to be told about.
func (h *host) Backspace(unit int) error { return nil }
func (h *host) EndFile(unit int) error   { return nil }
func (h *host) Rewind(unit int) error    { return nil }

// Time is milliseconds since the run started. 6.71 note 1: "The origin
// with respect to which the time is obtained is not important. The
// SNOBOL4 system deals only with differences in times."
func (h *host) Time() int {
	if h.started.IsZero() {
		h.started = time.Now()
	}
	return int(time.Since(h.started).Milliseconds())
}

// Date is the current date. 6.22 note 1 says the representation does
// not matter and gives 04/01/81 as one of four acceptable ones.
func (h *host) Date() []byte {
	return []byte(time.Now().Format("01/02/06"))
}

func line(w io.Writer, s []byte) error {
	_, err := w.Write(append(append(make([]byte, 0, len(s)+1), s...), '\n'))
	return err
}

func trimNewline(s []byte) []byte {
	s, _ = cutSuffix(s, '\n')
	s, _ = cutSuffix(s, '\r')
	return s
}

func cutSuffix(s []byte, c byte) ([]byte, bool) {
	if n := len(s) - 1; n >= 0 && s[n] == c {
		return s[:n], true
	}
	return s, false
}
