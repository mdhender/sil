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

// Package diag carries assembly diagnostics.
//
// Every stage accumulates into a List and keeps going rather than
// returning on the first problem. The SIL source of SNOBOL4 is 6580
// lines; a stage that stopped at the first bad line would report one
// problem per run.
package diag

import (
	"fmt"
	"strings"
)

// Diagnostic is one problem found in one place in the source.
//
// Col is one-based and may be zero when the problem belongs to the
// whole line rather than to a column of it.
type Diagnostic struct {
	File string
	Line int
	Col  int
	Msg  string
}

// String renders a diagnostic the way AGENTS.md asks for:
//
//	snobol.sil:143: undefined symbol FOO
//
// with the column included when there is one.
func (d Diagnostic) String() string {
	if d.Col > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", d.File, d.Line, d.Col, d.Msg)
	}
	return fmt.Sprintf("%s:%d: %s", d.File, d.Line, d.Msg)
}

// List accumulates diagnostics in the order they were found.
type List []Diagnostic

// Addf appends a diagnostic. Pass col 0 when the problem has no column.
func (l *List) Addf(file string, line, col int, format string, args ...any) {
	*l = append(*l, Diagnostic{File: file, Line: line, Col: col, Msg: fmt.Sprintf(format, args...)})
}

// Err reports every accumulated diagnostic as a single error, or nil
// when the list is empty.
func (l List) Err() error {
	if len(l) == 0 {
		return nil
	}
	var sb strings.Builder
	for i, d := range l {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(d.String())
	}
	return fmt.Errorf("%s", sb.String())
}
