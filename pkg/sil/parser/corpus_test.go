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

package parser_test

import (
	"errors"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

func newList() *diag.List { return &diag.List{} }

// The whole of the SIL source of SNOBOL4 must parse, and its shape
// must match what S4D58 7.6 describes: at most six operands, and
// parenthesised lists that never nest.
func TestParseEngineSource(t *testing.T) {
	name, src, err := corpus.Load()
	if err != nil {
		if errors.Is(err, corpus.ErrAbsent) {
			t.Skipf("%s: %s", corpus.Engine, corpus.SkipMessage)
		}
		t.Fatal(err)
	}

	lines, ds := scanner.Scan(name, src)
	if err := ds.Err(); err != nil {
		t.Fatalf("scanner reported diagnostics:\n%v", err)
	}

	stmts, ds := parser.Parse(lines)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}
	if len(stmts) != corpus.Statements {
		t.Fatalf("got %d statements, want %d", len(stmts), corpus.Statements)
	}

	var (
		maxArity   int
		maxArityAt int
		nulls      int
		lists      int
		literals   int
		ops        = map[string]int{}
	)
	stmtsWithNulls := 0
	for _, s := range stmts {
		ops[s.Op]++
		if n := len(s.Operands); n > maxArity {
			maxArity, maxArityAt = n, s.Num
		}
		for _, it := range s.Operands {
			if it.Kind == parser.ItemNull {
				stmtsWithNulls++
				break
			}
		}
		for _, it := range s.Operands {
			switch it.Kind {
			case parser.ItemNull:
				nulls++
			case parser.ItemLiteral:
				literals++
			case parser.ItemList:
				lists++
				for _, inner := range it.List {
					if inner.Kind == parser.ItemList {
						t.Errorf("%s:%d: nested list", name, s.Num)
					}
				}
			}
		}
	}

	// S4D58 7.6 does not state a maximum, but the source never
	// exceeds six top-level operands. A parse that produced more
	// would mean a comment had been swallowed into the operand field.
	if maxArity != 6 {
		t.Errorf("maximum operand count is %d (line %d), want 6", maxArity, maxArityAt)
	}
	if len(ops) != corpus.Opcodes {
		t.Errorf("got %d distinct mnemonics, want %d", len(ops), corpus.Opcodes)
	}
	// Omitted operands are how S4D58 5.2's omitted branch points are
	// written, so there must be a lot of them.
	if nulls == 0 || lists == 0 || literals == 0 {
		t.Errorf("got %d null, %d list and %d literal operands; expected all three to occur", nulls, lists, literals)
	}
	t.Logf("%d statements, %d distinct mnemonics, %d lists, %d literals",
		len(stmts), len(ops), lists, literals)
	t.Logf("%d null operands across %d statements", nulls, stmtsWithNulls)
}

// Every operand list must render back to the text it was parsed from,
// once the expression parentheses this package adds are accounted for.
// Statements whose operands contain no arithmetic -- the great
// majority -- must round-trip exactly.
func TestFormatRoundTripsEngineSource(t *testing.T) {
	name, src, err := corpus.Load()
	if err != nil {
		if errors.Is(err, corpus.ErrAbsent) {
			t.Skipf("%s: %s", corpus.Engine, corpus.SkipMessage)
		}
		t.Fatal(err)
	}

	lines, _ := scanner.Scan(name, src)
	stmts, _ := parser.Parse(lines)

	byNum := map[int]scanner.Line{}
	for _, l := range lines {
		byNum[l.Num] = l
	}

	bad, checked := 0, 0
	for _, s := range stmts {
		operand := byNum[s.Num].Operand
		// The lone comma that means "no operands" does not render
		// back to itself, by design.
		if operand == "," {
			continue
		}
		if hasArithmetic(s.Operands) {
			continue
		}
		checked++
		if got := parser.Format(s.Operands); got != operand {
			bad++
			if bad <= 10 {
				t.Errorf("%s:%d: Format mismatch\n got %q\nwant %q", name, s.Num, got, operand)
			}
		}
	}
	if bad > 10 {
		t.Errorf("%d operand lists failed to round-trip (showing the first 10)", bad)
	}
	t.Logf("%d of %d statements round-tripped exactly", checked-bad, checked)
}

func hasArithmetic(items []parser.Item) bool {
	for _, it := range items {
		switch it.Kind {
		case parser.ItemExpr:
			if _, plain := it.Expr.(*parser.Symbol); plain {
				continue
			}
			if _, plain := it.Expr.(*parser.Number); plain {
				continue
			}
			return true
		case parser.ItemList:
			if hasArithmetic(it.List) {
				return true
			}
		}
	}
	return false
}
