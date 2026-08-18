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
	"errors"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil/asm"
	"github.com/mdhender/sil/pkg/sil/copyseg"
	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/layout"
	"github.com/mdhender/sil/pkg/sil/op"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

// What it costs to load the engine: assembling the 6580 lines of the
// SNOBOL4 implementation into a machine ready to run.
//
// AGENTS.md wants a benchmark before anyone reasons about speed, and
// this is that benchmark. Nothing here has been optimized and nothing
// here should be until this says it is worth it.
func BenchmarkAssembleTheEngine(b *testing.B) {
	name, src := engine(b)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		vm, ds := asm.Assemble(name, src, asm.Options{})
		if vm == nil || len(ds) > 0 {
			b.Fatalf("%d diagnostics", len(ds))
		}
	}
}

// The same work broken up, so that a number that moves says which
// stage moved it. Emission and the syntax tables are what is left over
// from the total above; they have no entry of their own because emit
// is not exported and there is no reason to export it for this.
func BenchmarkStages(b *testing.B) {
	name, src := engine(b)

	lines, ds := scanner.Scan(name, src)
	if len(ds) > 0 {
		b.Fatal(ds.Err())
	}
	expanded := copyseg.ExpandWith(lines, copyseg.Source, &ds)
	if len(ds) > 0 {
		b.Fatal(ds.Err())
	}
	stmts, ds := parser.Parse(expanded)
	if len(ds) > 0 {
		b.Fatal(ds.Err())
	}

	for _, stage := range []struct {
		name string
		run  func()
	}{
		{"scan", func() { scanner.Scan(name, src) }},
		{"copy", func() {
			var ds diag.List
			copyseg.ExpandWith(lines, copyseg.Source, &ds)
		}},
		{"parse", func() { parser.Parse(expanded) }},
		{"check", func() { op.Check(stmts) }},
		{"layout", func() { layout.Run(stmts) }},
	} {
		b.Run(stage.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				stage.run()
			}
		})
	}
}

// engine is the historical source, or a skip. It takes a testing.TB
// because both the benchmarks and TestOperationCoverage need it, and a
// skip is not a pass either way.
func engine(tb testing.TB) (string, []byte) {
	tb.Helper()
	name, src, err := corpus.Load()
	if errors.Is(err, corpus.ErrAbsent) {
		tb.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		tb.Fatal(err)
	}
	return name, src
}
