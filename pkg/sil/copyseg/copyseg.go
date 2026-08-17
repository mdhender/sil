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

// Package copyseg supplies the machine-dependent segments that COPY
// pulls into an assembly.
//
// S4D58 6.20 names three: MDATA, MLINK and PARMS. Between them they
// define every symbol the SNOBOL4 source uses but never defines, which
// is why COPY is the one directive the assembler cannot leave for
// later -- until these segments are in, nothing has a value.
//
// The segments are SIL source text, and COPY splices that text into
// the line stream ahead of the parser. 6.20 note 1 allows other
// implementations ("COPY may, for example, simply expand into the data
// required"), but text keeps the machine-dependent choices readable in
// the same language as the program they serve, and keeps them subject
// to the same scanner, parser and symbol table as everything else.
//
// Expansion is textual and non-recursive: a segment may not COPY
// another. Nothing in the historical source or in these segments does.
package copyseg

import (
	"embed"
	"sort"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

//go:embed *.sil
var files embed.FS

// segments maps a COPY operand to the file holding that segment.
var segments = map[string]string{
	"MDATA": "mdata.sil",
	"MLINK": "mlink.sil",
	"PARMS": "parms.sil",
}

// Opcode is the operation whose operand names a segment (S4D58 6.20).
const Opcode = "COPY"

// Names returns the segment names COPY understands, sorted.
func Names() []string {
	out := make([]string, 0, len(segments))
	for name := range segments {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Source returns the SIL text of the named segment.
func Source(name string) (file string, src []byte, ok bool) {
	file, ok = segments[name]
	if !ok {
		return "", nil, false
	}
	src, err := files.ReadFile(file)
	if err != nil {
		// The segment is embedded, so a read failure means the map and
		// the directory have gone out of step.
		panic("copyseg: " + file + ": " + err.Error())
	}
	return file, src, true
}

// Expand replaces every COPY statement with the lines of the segment
// it names, and returns the resulting line stream.
//
// This is the assembler's first piece of per-operation knowledge, and
// deliberately its only one until the instruction table arrives: COPY
// has to be understood before symbol resolution because it is what
// supplies the symbols. A COPY whose operand is not a segment name is
// reported and left in place, so later stages still see the whole
// file.
//
// Lines from a segment keep the segment's own file name and line
// numbers, so a diagnostic inside PARMS cites PARMS rather than the
// line of the SNOBOL4 source that copied it.
func Expand(lines []scanner.Line, ds *diag.List) []scanner.Line {
	return ExpandWith(lines, Source, ds)
}

// Resolver returns the SIL text of a named COPY segment.
type Resolver func(name string) (file string, src []byte, ok bool)

// ExpandWith is Expand against segments other than the ones this
// package embeds.
//
// It exists so that the choice of DESCR, SPEC and CPA can be varied
// and the assembly rerun. Those three numbers are the most expensive
// thing to get wrong -- everything downstream inherits them -- and an
// assembly that only ever closes for one set of them has not been
// shown to be independent of it.
func ExpandWith(lines []scanner.Line, resolve Resolver, ds *diag.List) []scanner.Line {
	out := make([]scanner.Line, 0, len(lines))
	for _, l := range lines {
		if l.Comment || l.Op != Opcode {
			out = append(out, l)
			continue
		}
		name := segmentName(l.Operand)
		file, src, ok := resolve(name)
		if !ok {
			ds.Addf(l.File, l.Num, 16, "COPY %s: not a machine-dependent segment; expected one of %v", name, Names())
			out = append(out, l)
			continue
		}
		seg, segDiags := scanner.Scan(file, src)
		*ds = append(*ds, segDiags...)
		out = append(out, seg...)
	}
	return out
}

// segmentName trims the operand field down to the name. COPY's operand
// is a single identifier, so anything else is passed through unchanged
// and reported by the caller.
func segmentName(operand string) string {
	for i := 0; i < len(operand); i++ {
		if c := operand[i]; !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') {
			return operand[:i]
		}
	}
	return operand
}
