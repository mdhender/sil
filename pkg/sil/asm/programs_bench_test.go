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
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/programs"
	"github.com/mdhender/sil/pkg/sil/asm"
)

// What it costs to run a SNOBOL4 program, one benchmark per program in
// internal/programs.
//
// BenchmarkAssembleTheEngine measures getting the machine ready.
// This measures the machine working, and each iteration is the whole
// of it -- assemble, compile the program, execute it -- because that
// is what `sil x.sno` does and there is no image format to load from
// instead. Assembly is around a third of hello.sno and a smaller share
// of everything else, so a change that moves one program and not the
// others is a change in the machine rather than in the front end.
//
// The number worth watching is cycles/op, which counts SIL
// instructions executed and does not depend on the machine this is
// running on. ns/op does, and moves with the weather.
//
//	go test ./pkg/sil/asm -bench Programs -benchmem
//	go test ./pkg/sil/asm -bench Programs/sieve -count 10
//
// AGENTS.md wants a benchmark before anyone reasons about speed.
// Nothing here has been optimized and nothing should be until these
// say where it would be worth it.
func BenchmarkPrograms(b *testing.B) {
	name, src := engine(b)

	for _, p := range programs.All() {
		b.Run(p.Name, func(b *testing.B) {
			// A program that does not stop has nothing to measure but
			// the instruction bound, and measuring that takes as long
			// as the bound is.
			if p.Runaway != "" {
				b.Skipf("does not stop: %s", p.Runaway)
			}

			deck := programs.Card + string(p.Source)
			opts := asm.Options{}
			if p.Stack != 0 {
				// Without this an invalid stack size is what gets
				// measured: ackermann3.sno runs the stack out at m=3
				// and still reports a clean compilation, so nothing
				// downstream would notice.
				opts.Equates = map[string]int{"STSIZE": p.Stack}
			}
			b.ReportAllocs()

			var cycles, core int
			for b.Loop() {
				h := &snobolHost{input: strings.NewReader(deck)}
				opts.Host = h
				vm, ds := asm.Assemble(name, src, opts)
				if err := ds.Err(); err != nil {
					b.Fatalf("%d diagnostics: %v", len(ds), err)
				}
				vm.MaxCycles = 100_000_000
				if err := vm.Run(); err != nil {
					b.Fatalf("%v\n%s", err, h.system.String())
				}
				// A run that went wrong somewhere is not the run being
				// measured. The invalid programs are supposed to end
				// in a diagnostic, so they are held to their golden
				// instead of to a clean compilation, and the pending
				// one is held to neither.
				if p.Pending == "" {
					got := programs.Normalize(h.printed.Bytes())
					if want := programs.Normalize(p.Want); !bytes.Equal(got, want) {
						b.Fatalf("this is not the run %s.out describes:\ngot  %q\nwant %q",
							p.Name, got, want)
					}
				}
				cycles, core = vm.Cycles, len(vm.Core)
			}
			b.ReportMetric(float64(cycles), "cycles/op")
			b.ReportMetric(float64(core), "core")
		})
	}
}
