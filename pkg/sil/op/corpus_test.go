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

package op_test

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/pkg/sil/op"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

// The milestone: every statement of the SNOBOL4 source fits the table.
func TestEngineTypeChecks(t *testing.T) {
	stmts := engine(t)
	if got := len(stmts); got != corpus.Statements {
		t.Fatalf("got %d statements, want %d", got, corpus.Statements)
	}
	if err := op.Check(stmts).Err(); err != nil {
		t.Errorf("the source does not fit the table:\n%v", err)
	}
}

// Every mnemonic in the source is in the table, and every entry in the
// table is used by the source. S4D58 7.4's occurrence table says the
// SNOBOL4 program uses all 131.
func TestEngineUsesEveryOperation(t *testing.T) {
	stmts := engine(t)

	count := map[op.Kind]int{}
	for _, s := range stmts {
		k := op.Lookup(s.Op)
		if k == op.Invalid {
			t.Errorf("%s:%d: %s is not in the table", s.File, s.Num, s.Op)
			continue
		}
		count[k]++
	}
	for _, k := range op.Kinds() {
		if count[k] == 0 {
			t.Errorf("%s is in the table but not in the source", k)
		}
	}
	if got := len(count); got != corpus.Opcodes {
		t.Errorf("the source uses %d operations, want %d", got, corpus.Opcodes)
	}
}

// The free oracle for the branch-point slots.
//
// The source marks a break in the flow of control with a comment line
// holding "*_" and nothing else. It is not in S4D58 -- 7.6 describes
// only comments and program text -- but the deck is consistent about
// it: the 516 markers each follow a statement that control cannot pass
// through, and nothing else does.
//
// So every one of them is a claim about the statement above it, and
// the claim is checkable against the table: either the operation
// always transfers, or every branch point it has is filled in. Getting
// a branch slot wrong -- missing one, or calling something a branch
// that is not -- breaks this over 516 sites and 30 distinct
// operations.
//
// It is a necessary condition rather than a sufficient one. 6.87 note
// 6 and 6.98 note 2 both let an RCALL or a SELBRA fall through even
// with every location supplied, so those 55 markers rest on what the
// procedure being called actually returns, which is a property of the
// program.
func TestFlowBreakMarkersFollowStatementsThatCannotFallThrough(t *testing.T) {
	name, src, lines := engineLines(t)

	prev := -1
	markers, checked := 0, map[string]int{}
	for i, l := range lines {
		if !l.Comment {
			prev = i
			continue
		}
		if !strings.HasPrefix(l.Text, flowBreak) {
			continue
		}
		markers++
		if prev < 0 {
			t.Fatalf("%s:%d: %s with no statement before it", name, l.Num, flowBreak)
		}
		p := lines[prev]
		stmt := parser.ParseOne(p, nil)
		e := op.Get(op.Lookup(p.Op))
		checked[p.Op]++
		if e.Terminates {
			continue
		}
		if missing := unfilledBranches(e, stmt); missing != "" {
			t.Errorf("%s:%d: %s before %s leaves %s unfilled\n\t%s", name, p.Num, p.Op, flowBreak, missing, p.Text)
		}
	}

	if markers != flowBreaks {
		t.Errorf("found %d %s markers, want %d", markers, flowBreak, flowBreaks)
	}
	t.Logf("%d markers over %d distinct operations; %d bytes of source", markers, len(checked), len(src))
}

// The oracle that actually holds the slot classification together.
//
// A name in the SNOBOL4 source is used for one thing. Over the whole
// 4832 statements, no symbol appears both as a descriptor operand and
// as a specifier operand, or as a branch point and a constant, or in
// any other two of these roles -- 1131 names, each in exactly one.
//
// That makes the table falsifiable in both directions, which the flow
// break markers above are not. Calling something a branch point when
// it is a constant puts a program label in a constant slot; missing a
// branch point puts one in whatever slot it was called instead. Either
// way a name lands in two roles and this fires. Both mistakes were
// tried against this test, and both showed up: 43 collisions and 23
// respectively.
//
// Four slots are left out, because for them the overlap is the
// documented meaning rather than a mistake:
//
//   - SlotAddr, the address field of a DESCR, holds anything with an
//     address -- function entry points, other descriptors, program
//     labels, plain numbers;
//   - SlotExpr, the operand of EQU, is whatever it is being equated to;
//   - SlotProc names a procedure, and a procedure's entry point is
//     also branched to, so 6.15's BRANCH LOC,PROC has the same label
//     in both roles by construction;
//   - SlotList resolves to the kind of its elements and is counted as
//     that.
func TestNoSymbolIsUsedInTwoRoles(t *testing.T) {
	roles := map[string]map[op.Slot]int{}
	for _, s := range engine(t) {
		e := op.Get(op.Lookup(s.Op))
		for i, o := range e.Operands {
			if i >= len(s.Operands) {
				break
			}
			slot, elems := o.Slot, []parser.Item{s.Operands[i]}
			if o.Slot == op.SlotList {
				slot = o.Elem
				if s.Operands[i].Kind == parser.ItemList {
					elems = s.Operands[i].List
				}
			}
			if !oneRole(slot) {
				continue
			}
			for _, el := range elems {
				sym, ok := el.Expr.(*parser.Symbol)
				if el.Kind != parser.ItemExpr || !ok {
					continue
				}
				if roles[sym.Name] == nil {
					roles[sym.Name] = map[op.Slot]int{}
				}
				roles[sym.Name][slot]++
			}
		}
	}

	clashes := 0
	for _, name := range sortedKeys(roles) {
		if len(roles[name]) < 2 {
			continue
		}
		clashes++
		var used []string
		for slot, n := range roles[name] {
			used = append(used, fmt.Sprintf("%s %d times", slot, n))
		}
		sort.Strings(used)
		t.Errorf("%s is used as %s", name, strings.Join(used, " and as "))
	}
	if clashes > 0 {
		t.Errorf("%d of %d symbols are used in more than one role", clashes, len(roles))
	}
	t.Logf("%d symbols, each in exactly one role", len(roles))
}

// oneRole reports whether a slot is one of the roles a name may not
// share. See TestNoSymbolIsUsedInTwoRoles for why the others are out.
func oneRole(s op.Slot) bool {
	switch s {
	case op.SlotDescr, op.SlotSpec, op.SlotBranch, op.SlotConst,
		op.SlotFlag, op.SlotTable, op.SlotKey, op.SlotFormat:
		return true
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// flowBreak is the comment the source uses to mark a point control
// cannot reach by falling through. Thirteen of the 516 carry a
// trailing comment of their own, so it is a prefix rather than the
// whole line; no other comment in the source begins with it.
const flowBreak = "*_"

// flowBreaks is how many of them there are.
const flowBreaks = 516

// unfilledBranches names the branch points of a statement that were
// left out, or "" when they were all supplied.
func unfilledBranches(e op.Entry, s parser.Statement) string {
	var out []string
	for i, o := range e.Operands {
		var it parser.Item
		if i < len(s.Operands) {
			it = s.Operands[i]
		}
		switch {
		case o.Slot == op.SlotBranch:
			if i >= len(s.Operands) || it.Kind == parser.ItemNull {
				out = append(out, o.Name)
			}
		case o.Slot == op.SlotList && o.Elem == op.SlotBranch:
			if i >= len(s.Operands) || it.Kind == parser.ItemNull {
				out = append(out, o.Name)
				continue
			}
			for j, el := range it.List {
				if el.Kind == parser.ItemNull {
					out = append(out, fmt.Sprintf("%s[%d]", o.Name, j+1))
				}
			}
		}
	}
	return strings.Join(out, ", ")
}

func engine(t *testing.T) []parser.Statement {
	t.Helper()
	_, _, lines := engineLines(t)
	stmts, ds := parser.Parse(lines)
	if err := ds.Err(); err != nil {
		t.Fatalf("parser reported diagnostics:\n%v", err)
	}
	return stmts
}

func engineLines(t *testing.T) (string, []byte, []scanner.Line) {
	t.Helper()
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
	return name, src, lines
}
