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
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil"
	"github.com/mdhender/sil/pkg/sil/asm"
	"github.com/mdhender/sil/pkg/sil/op"
	"github.com/mdhender/sil/pkg/sil/syntab"
)

// M8: the historical source assembles clean.
//
// Zero diagnostics over 6580 lines, the entry point is the address of
// BEGIN, and a listing is byte-stable across runs. The first two are
// the milestone's exit criteria; the third is what makes a listing
// worth diffing when something later goes wrong.
func TestTheHistoricalSourceAssembles(t *testing.T) {
	name, src, err := corpus.Load()
	if errors.Is(err, corpus.ErrAbsent) {
		t.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		t.Fatal(err)
	}

	vm, ds := asm.Assemble(name, src, asm.Options{})
	if err := ds.Err(); err != nil {
		t.Fatalf("%d diagnostics:\n%v", len(ds), err)
	}

	// 6.46 makes INIT the first instruction executed, and the source
	// puts it at BEGIN, line 303. Everything above it is TITLE, COPY
	// and EQU, which assemble nothing, so BEGIN is the bottom of core.
	at, ok := vm.Symbols["BEGIN"]
	if !ok {
		t.Fatal("BEGIN is not defined")
	}
	if vm.PC != at {
		t.Errorf("the entry point is %d, want %d, the address of BEGIN", vm.PC, at)
	}
	if c := vm.Core[vm.PC]; c.Kind != sil.Instr || c.Op != op.INIT {
		t.Errorf("the entry point holds %s, want INIT", c)
	}

	// Every cell was assembled by something, so a listing cites a
	// source line for all of them.
	for a, c := range vm.Core {
		if c.Src.File == "" {
			t.Fatalf("%d was never assembled: %s", a, c)
		}
	}

	var first, second bytes.Buffer
	if err := asm.Listing(&first, vm); err != nil {
		t.Fatal(err)
	}
	again, ds := asm.Assemble(name, src, asm.Options{})
	if err := ds.Err(); err != nil {
		t.Fatalf("the second assembly reported diagnostics:\n%v", err)
	}
	if err := asm.Listing(&second, again); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("two assemblies of the same source give different listings")
	}

	t.Logf("%s: %d units of core, entry at %d", name, len(vm.Core), vm.PC)
}

// The twenty-five tables of Appendix A reach core through MDATA, which
// declares the storage, and the loader, which fills it (4.2). Every
// one of them has to be there and none may be left as the zeroes ARRAY
// assembled.
func TestTheSyntaxTablesAreLoaded(t *testing.T) {
	name, src, err := corpus.Load()
	if errors.Is(err, corpus.ErrAbsent) {
		t.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		t.Fatal(err)
	}
	vm, ds := asm.Assemble(name, src, asm.Options{})
	if err := ds.Err(); err != nil {
		t.Fatalf("%d diagnostics:\n%v", len(ds), err)
	}

	alphsz := vm.Symbols["ALPHSZ"]
	value := func(n string) (int, bool) { v, ok := vm.Symbols[n]; return v, ok }

	for _, table := range syntab.Names() {
		at, ok := vm.Symbols[table]
		if !ok {
			t.Errorf("%s is not defined; MDATA should declare it", table)
			continue
		}
		want, err := syntab.Build(table, alphsz, value)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		for c, e := range want {
			got := vm.Core[at+c*vm.Descr]
			if got.A != e.Next || got.F != e.Indicator || got.V != e.Put {
				t.Fatalf("%s: the entry for character %d is %d,%d,%d, want %d,%d,%d",
					table, c, got.A, got.F, got.V, e.Next, e.Indicator, e.Put)
			}
		}
	}

	// A table that is all zeroes would pass the loop above only if
	// Appendix A said so, and none of them does: every description has
	// an ELSE with a nonzero indicator or a next table.
	blank := 0
	for _, table := range syntab.Names() {
		at := vm.Symbols[table]
		empty := true
		for c := 0; c < alphsz; c++ {
			if e := vm.Core[at+c*vm.Descr]; e.A != 0 || e.F != 0 || e.V != 0 {
				empty = false
				break
			}
		}
		if empty {
			blank++
			t.Errorf("%s is still the storage ARRAY assembled", table)
		}
	}
	t.Logf("%d tables of %d entries, %d left blank", len(syntab.Names()), alphsz, blank)
}
