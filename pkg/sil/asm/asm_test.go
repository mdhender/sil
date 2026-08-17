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
	"fmt"
	"strings"
	"testing"

	"github.com/mdhender/sil/pkg/sil/asm"
)

// stmt renders one SIL statement into its columns (S4D58 7.6).
func stmt(label, op, operand string) string {
	return strings.TrimRight(fmt.Sprintf("%-6s %-6s  %s", label, op, operand), " ")
}

// A minimal assembly: the machine parameters, an entry point and a
// stack.
func program(lines ...string) []byte {
	src := []string{stmt("", "COPY", "PARMS")}
	src = append(src, lines...)
	src = append(src,
		stmt("STACK", "DESCR", "0,0,0"),
		stmt("", "ARRAY", "7"),
		stmt("", "END", ""))
	return []byte(strings.Join(src, "\n") + "\n")
}

// The whole front end runs and each stage reports in its own terms.
func TestDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  []byte
		want string
	}{
		{
			name: "a bad column, from the scanner",
			src:  []byte("BEGIN INIT ,\n"),
			want: `column 7 must be blank`,
		},
		{
			name: "an unknown COPY segment, from the expander",
			src:  []byte(stmt("", "COPY", "MPARMS") + "\n"),
			want: "not a machine-dependent segment",
		},
		{
			name: "an unknown operation, from the table",
			src:  program(stmt("BEGIN", "MOVEIT", "STACK,STACK")),
			want: "unknown operation MOVEIT",
		},
		{
			name: "an operand of the wrong shape, from the table",
			src:  program(stmt("BEGIN", "MOVD", "STACK,'A'")),
			want: "MOVD: DESCR2 must be an expression",
		},
		{
			name: "an undefined name, from the location counter",
			src:  program(stmt("BEGIN", "MOVD", "STACK,NOSUCH")),
			want: "NOSUCH has no value",
		},
		{
			name: "an address multiplied, from the location counter",
			src:  program(stmt("BEGIN", "SETAC", "STACK,STACK*2")),
			want: "cannot multiply the address STACK",
		},
		{
			name: "a branch to something that is not a procedure",
			src: program(
				stmt("BEGIN", "BRANCH", "BEGIN,STACK"),
			),
			want: "not a PROC entry point",
		},
		{
			name: "an entry point that does not exist",
			src:  program(stmt("START", "INIT", ",")),
			want: "the entry point BEGIN is not defined",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			vm, ds := asm.Assemble("test.sil", tt.src, asm.Options{})
			err := ds.Err()
			if err == nil {
				t.Fatalf("assembled with no diagnostic, want one containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("reported\n%v\nwant a diagnostic containing %q", err, tt.want)
			}
			if vm != nil {
				t.Error("a machine came back from a failed assembly")
			}
		})
	}
}

// A procedure named by BRANCH or RCALL has to be one (6.15, 6.78 note
// 1). It is then discarded: nothing in this machine is based, so the
// operand does not reach core.
func TestProcedureOperandIsCheckedAndDiscarded(t *testing.T) {
	vm := assemble(t, program(
		stmt("BEGIN", "BRANCH", "PROCA,PROCA"),
		stmt("PROCA", "PROC", ","),
		stmt("", "ENDEX", "STACK"),
	), asm.Options{})

	c := vm.Core[vm.PC]
	if len(c.Ops) != 1 {
		t.Errorf("BRANCH assembled %d operands, want 1: %s", len(c.Ops), c)
	}
	if c.Ops[0] != vm.Symbols["PROCA"] {
		t.Errorf("BRANCH goes to %d, want PROCA at %d", c.Ops[0], vm.Symbols["PROCA"])
	}
}

// The entry point is where execution starts, and Options names it.
func TestEntryPoint(t *testing.T) {
	src := program(
		stmt("START", "INIT", ","),
		stmt("", "ENDEX", "STACK"),
	)
	vm := assemble(t, src, asm.Options{Entry: "START"})
	if want := vm.Symbols["START"]; vm.PC != want {
		t.Errorf("PC is %d, want START at %d", vm.PC, want)
	}
}

// Options.Equates changes the value of a named EQU, which is how a
// program that needs more room than the historical source allowed for
// gets it without editing a file that is meant to be read-only input.
func TestEquatesOverrideAnEQU(t *testing.T) {
	const src = "" +
		"       COPY    PARMS\n" +
		"ROOM   EQU     1000\n" +
		"BIG    EQU     ROOM*2\n" +
		"BEGIN  ENDEX   ZERO\n" +
		"ZERO   DESCR   0,0,0\n" +
		"       END\n"

	t.Run("the source's own value", func(t *testing.T) {
		vm, ds := asm.Assemble("t.sil", []byte(src), asm.Options{})
		if err := ds.Err(); err != nil {
			t.Fatal(err)
		}
		if got := vm.Symbols["ROOM"]; got != 1000 {
			t.Errorf("ROOM is %d, want 1000", got)
		}
		if got := vm.Symbols["BIG"]; got != 2000 {
			t.Errorf("BIG is %d, want 2000", got)
		}
	})

	t.Run("overridden", func(t *testing.T) {
		vm, ds := asm.Assemble("t.sil", []byte(src), asm.Options{
			Equates: map[string]int{"ROOM": 35000},
		})
		if err := ds.Err(); err != nil {
			t.Fatal(err)
		}
		if got := vm.Symbols["ROOM"]; got != 35000 {
			t.Errorf("ROOM is %d, want 35000", got)
		}
		// Everything computed from it moves with it, because the
		// override happens before layout resolves anything.
		if got := vm.Symbols["BIG"]; got != 70000 {
			t.Errorf("BIG is %d, want 70000", got)
		}
	})

	// A caller that misspells the name should be told. Silently doing
	// nothing is the one behaviour that would leave somebody
	// wondering why a bigger stack did not help.
	t.Run("a name that is not an EQU", func(t *testing.T) {
		_, ds := asm.Assemble("t.sil", []byte(src), asm.Options{
			Equates: map[string]int{"STSIZE": 35000},
		})
		if ds.Err() == nil {
			t.Fatal("no diagnostic for a name that is not in the source")
		}
		if !strings.Contains(ds.Err().Error(), "STSIZE is not an EQU") {
			t.Errorf("reported %v", ds.Err())
		}
	})

	// A label that is not an EQU is not an equate either, however
	// much it looks like a name.
	t.Run("a label that is not an EQU", func(t *testing.T) {
		_, ds := asm.Assemble("t.sil", []byte(src), asm.Options{
			Equates: map[string]int{"BEGIN": 1},
		})
		if ds.Err() == nil || !strings.Contains(ds.Err().Error(), "BEGIN is not an EQU") {
			t.Fatalf("reported %v", ds.Err())
		}
	})
}
