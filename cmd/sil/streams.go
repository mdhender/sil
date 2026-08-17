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

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// A run has two streams of text coming out of it, and where each one
// goes is what -out and -system say.
//
// They are two streams because they leave the machine by two
// operations: what the program printed goes through STPRNT (S4D58
// 6.114) and what the SNOBOL4 system printed about itself goes through
// OUTPUT (6.75). The default sends the first to standard output and
// the second to standard error, so that a pipeline gets the program's
// printing and nothing else.
//
// # A destination
//
// A destination is a comma-separated list of places, and the text goes
// to every one of them. A place is:
//
//	stdout    standard output
//	stderr    standard error
//	none      nowhere; the text is discarded
//	anything  else is a file name, created or truncated
//
// So -out stdout,run.txt tees the program's printing to standard
// output and to a file, and -out run.txt -system run.txt puts both
// streams in one file in the order the machine wrote them, which is
// the single stream the original printed.
//
// A file named more than once, by either flag, is opened once. Two
// writers on one file would truncate each other's work and interleave
// what survived.
const destinationHelp = "stdout, stderr, none, or a file; comma-separated to send it to several"

// stdName and errName are the two reserved words. A file of either
// name is reached by ./stdout, the way any other shell-ambiguous name
// is.
const (
	stdName  = "stdout"
	errName  = "stderr"
	offName  = "none"
	defOut   = stdName
	defSyste = errName
)

// opened keeps the files a run has opened, so that a name given twice
// is one file and everything is closed once at the end.
type opened struct {
	stdout, stderr io.Writer
	files          map[string]*os.File
	order          []*os.File
}

func newOpened(stdout, stderr io.Writer) *opened {
	return &opened{stdout: stdout, stderr: stderr, files: map[string]*os.File{}}
}

// closeAll closes every file the run opened, in the order they were
// opened. The standard streams are not ours to close.
func (o *opened) closeAll() {
	for _, f := range o.order {
		f.Close()
	}
}

// resolve turns one destination into the writer a stream goes to.
func (o *opened) resolve(flag, spec string) (io.Writer, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, fmt.Errorf("-%s: no destination; %s", flag, destinationHelp)
	}

	var ws []io.Writer
	seen := map[string]bool{}
	for _, place := range strings.Split(spec, ",") {
		place = strings.TrimSpace(place)
		if place == "" {
			return nil, fmt.Errorf("-%s %s: an empty destination between the commas", flag, spec)
		}
		// A place repeated inside one destination is one place. Two
		// copies of stdout would print everything twice.
		if seen[place] {
			continue
		}
		seen[place] = true

		switch place {
		case stdName:
			ws = append(ws, o.stdout)
		case errName:
			ws = append(ws, o.stderr)
		case offName:
			// Nothing, and nothing else either: discarding some of a
			// stream and keeping the rest is not a thing to mean.
			if len(strings.Split(spec, ",")) > 1 {
				return nil, fmt.Errorf("-%s %s: %s goes with nothing else", flag, spec, offName)
			}
			return io.Discard, nil
		default:
			f, err := o.file(place)
			if err != nil {
				return nil, fmt.Errorf("-%s %s: %w", flag, spec, err)
			}
			ws = append(ws, f)
		}
	}
	if len(ws) == 1 {
		return ws[0], nil
	}
	return io.MultiWriter(ws...), nil
}

// file opens a name, or hands back the file already open under it.
func (o *opened) file(name string) (*os.File, error) {
	if f, ok := o.files[name]; ok {
		return f, nil
	}
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	o.files[name] = f
	o.order = append(o.order, f)
	return f, nil
}

// streams resolves both destinations together, so that a file named by
// each of them is one file and the two streams land in it in the order
// the machine wrote them.
func streams(outSpec, systemSpec string, stdout, stderr io.Writer) (out, system io.Writer, closeAll func(), err error) {
	o := newOpened(stdout, stderr)
	if out, err = o.resolve("out", outSpec); err != nil {
		o.closeAll()
		return nil, nil, nil, err
	}
	if system, err = o.resolve("system", systemSpec); err != nil {
		o.closeAll()
		return nil, nil, nil, err
	}
	return out, system, o.closeAll, nil
}
