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

// Package layout runs the location counter and gives every symbol a
// value.
//
// This is the first stage that has to commit to machine-dependent
// choices. The symbol table can close the reference graph of an
// assembly knowing nothing about the machine, because a name is
// defined or it is not; a value needs to know how much room a
// descriptor takes, which is DESCR, which comes from the PARMS segment
// (S4D58 6.20). Everything downstream -- emission, the image, the
// machine itself -- inherits those choices, so they are checked here
// against relationships the SNOBOL4 source states about its own data.
//
// Which shape a statement has comes from the instruction table;
// turning a shape into a number needs DESCR, SPEC and CPA, and that is
// this package. Every operation is one address unit except RCALL and
// SELBRA, each of which is followed by the vector of branch points
// that S4D58 6.87 and 6.98 print as part of what the macro assembles.
//
// Nothing is emitted. A Layout says where each statement goes and what
// each symbol is worth; it holds no cells.
//
// A Layout is meaningful only when Run's diagnostic list is empty. A
// statement whose size could not be computed is placed as though it
// occupied nothing, so one bad count shifts everything after it.
package layout

import (
	"sort"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/op"
	"github.com/mdhender/sil/pkg/sil/parser"
)

// Origin is the address the location counter starts at.
const Origin = 0

// Placement is where a statement was put and how much room it took, in
// address units.
type Placement struct {
	Addr int
	Size int
}

// Layout holds the result of running the location counter.
type Layout struct {
	values map[string]Value
	place  []Placement
	end    int

	// Machine parameters, read out of PARMS before the location
	// counter can run.
	descr int
	spec  int
	cpa   int
}

// Value returns the assembly-time value of a symbol.
func (l *Layout) Value(name string) (Value, bool) {
	v, ok := l.values[name]
	return v, ok
}

// Addr returns the address of a symbol, and reports whether it has a
// value at all. A symbol that resolved to a plain number returns that
// number, which is what an operand field would use.
func (l *Layout) Addr(name string) (int, bool) {
	v, ok := l.values[name]
	return v.N, ok
}

// Evaluate computes the value of an expression against a finished
// layout. It is how a later stage turns an operand into the number
// that goes in a cell.
func (l *Layout) Evaluate(e parser.Expr) (Value, error) {
	v, err := l.eval(e)
	if err != nil {
		return Value{}, err
	}
	return v, nil
}

// Placement returns where the i'th statement passed to Run was put.
func (l *Layout) Placement(i int) Placement { return l.place[i] }

// End is the address just past the last statement, and so the number
// of address units the assembly occupies.
func (l *Layout) End() int { return l.end }

// Params returns the three sizes the location counter needed: the
// width of a descriptor, the width of a specifier, and the number of
// characters stored per address unit (S4D58 6.20).
func (l *Layout) Params() (descr, spec, cpa int) { return l.descr, l.spec, l.cpa }

// Symbols returns every name that has a value, sorted.
func (l *Layout) Symbols() []string {
	out := make([]string, 0, len(l.values))
	for name := range l.values {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Run places every statement and resolves every symbol.
//
// Resolution is iterative rather than two-pass, because EQU is not
// ordered: PRMSIZ at line 6336 is the difference of two labels defined
// twenty lines earlier, while PRMTBL at line 6316 assembles PRMSIZ
// into a value field twenty lines before it exists. The passes are
//
//  1. resolve every EQU that depends only on other EQUs -- this is
//     where DESCR, SPEC and CPA come from, and where the array bounds
//     the location counter needs come from;
//  2. run the location counter, giving every other label an address;
//  3. resolve the EQUs that were waiting on an address;
//  4. evaluate every operand in the assembly, which is what reports
//     an unresolved name or an expression that abuses an address.
//
// Statements after END are placed but contribute nothing, since END
// terminates the assembly (6.28).
func Run(stmts []parser.Statement) (*Layout, diag.List) {
	var ds diag.List

	l := &Layout{
		values: make(map[string]Value, len(stmts)),
		place:  make([]Placement, len(stmts)),
	}

	equs := l.indexEqus(stmts, &ds)
	l.resolveEqus(stmts, equs)

	if l.machineParams(&ds) {
		l.locate(stmts, &ds)
		l.resolveEqus(stmts, equs)
	}

	l.check(stmts, &ds)
	return l, ds
}

// indexEqus collects the statements that define a symbol by
// equivalence rather than by position.
func (l *Layout) indexEqus(stmts []parser.Statement, ds *diag.List) []int {
	var equs []int
	for i, s := range stmts {
		if op.Lookup(s.Op) != op.EQU {
			continue
		}
		if s.Label == "" {
			ds.Addf(s.File, s.Num, 1, "EQU without a label defines nothing")
			continue
		}
		if len(s.Operands) != 1 || s.Operands[0].Kind != parser.ItemExpr {
			ds.Addf(s.File, s.Num, 16, "%s EQU: expected one expression, found %q", s.Label, parser.Format(s.Operands))
			continue
		}
		equs = append(equs, i)
	}
	return equs
}

// resolveEqus evaluates equivalences until none of the ones left can
// be evaluated. Order does not matter -- an equivalence is a fact, not
// a step -- so repeating until nothing moves resolves any acyclic
// chain regardless of how it is written.
func (l *Layout) resolveEqus(stmts []parser.Statement, equs []int) {
	for {
		progress := false
		for _, i := range equs {
			s := stmts[i]
			if _, done := l.values[s.Label]; done {
				continue
			}
			if v, err := l.eval(s.Operands[0].Expr); err == nil {
				l.values[s.Label] = v
				progress = true
			}
		}
		if !progress {
			return
		}
	}
}

// machineParams reads the three sizes the location counter needs and
// reports whether it can run at all.
func (l *Layout) machineParams(ds *diag.List) bool {
	ok := true
	for _, p := range []struct {
		name string
		into *int
	}{
		{"DESCR", &l.descr},
		{"SPEC", &l.spec},
		{"CPA", &l.cpa},
	} {
		v, found := l.values[p.name]
		switch {
		case !found:
			ds.Addf(copySegment, 0, 0, "%s is not defined; COPY PARMS must define it (S4D58 6.20)", p.name)
			ok = false
		case v.Reloc:
			ds.Addf(copySegment, 0, 0, "%s is an address; it must be a number of address units", p.name)
			ok = false
		case v.N < 1:
			ds.Addf(copySegment, 0, 0, "%s is %d; it must be positive", p.name, v.N)
			ok = false
		default:
			*p.into = v.N
		}
	}
	return ok
}

// copySegment names the source of a diagnostic about a value that no
// line of the assembly is responsible for.
const copySegment = "COPY PARMS"

// locate walks the statements in order, giving every label that is not
// an equivalence the address of the statement it stands in front of,
// and advancing the location counter by the size of that statement.
//
// LHERE and PROC are the pure cases: they assemble nothing and exist
// only to name the address of whatever comes next (6.54, and 6.78 note
// 2, which says PROC "has no functional use and may be implemented as
// LHERE").
func (l *Layout) locate(stmts []parser.Statement, ds *diag.List) {
	lc := Origin
	ended := false
	for i, s := range stmts {
		if s.Label != "" && op.Lookup(s.Op) != op.EQU {
			if _, taken := l.values[s.Label]; !taken {
				l.values[s.Label] = Addr(lc)
			}
		}
		size := 0
		if !ended {
			size = l.sizeOf(s, ds)
		}
		l.place[i] = Placement{Addr: lc, Size: size}
		lc += size
		if op.Lookup(s.Op) == op.END {
			ended = true
		}
	}
	l.end = lc
}

// sizeOf reports how many address units a statement assembles.
//
// The instruction table says which of the eight shapes a statement
// has; this turns the shape into a number, which needs DESCR, SPEC and
// CPA and so cannot live in the table.
func (l *Layout) sizeOf(s parser.Statement, ds *diag.List) int {
	k := op.Lookup(s.Op)
	if k == op.Invalid {
		ds.Addf(s.File, s.Num, 8, "unknown operation %s: cannot tell how much room it takes", s.Op)
		return 0
	}
	switch op.Get(k).Size {
	case op.SizeNone:
		// TITLE, EQU, LHERE, PROC, END and COPY. A COPY that reaches
		// the location counter was not expanded, which the expander
		// has already reported; here it just takes no room.
		return 0

	case op.SizeDescr:
		return l.descr

	case op.SizeSpec:
		return l.spec

	case op.SizeArray:
		return l.count(s, ds) * l.descr

	case op.SizeBuffer:
		return l.chars(l.count(s, ds))

	case op.SizeString:
		// 6.117 note 1: LOC is the location of the specifier, not the
		// string. This assembles the string immediately after it,
		// which the note allows and which keeps a string in one piece.
		return l.spec + l.chars(l.literal(s, ds))

	case op.SizeChars:
		return l.chars(l.literal(s, ds))

	case op.SizeCall:
		// 6.87: the RCALL, then the return slot at LOC that holds the
		// operation returning the value, then one BRANCH per exit.
		return 2 + vectorLen(s, op.Get(k))

	case op.SizeVector:
		// 6.98: the SELBRA, then one branch point per location.
		return 1 + vectorLen(s, op.Get(k))

	default:
		return 1
	}
}

// vectorLen is how many branch points the statement's vector holds. A
// vector of one is written without its parentheses, and an omitted
// vector is a vector of none (6.87 note 4).
func vectorLen(s parser.Statement, e op.Entry) int {
	i, ok := e.Vector()
	if !ok || i >= len(s.Operands) {
		return 0
	}
	switch it := s.Operands[i]; it.Kind {
	case parser.ItemList:
		return len(it.List)
	case parser.ItemNull:
		return 0
	default:
		return 1
	}
}

// count evaluates the operand of ARRAY or BUFFER, which is how many
// elements to reserve.
func (l *Layout) count(s parser.Statement, ds *diag.List) int {
	if len(s.Operands) != 1 || s.Operands[0].Kind != parser.ItemExpr {
		ds.Addf(s.File, s.Num, 16, "%s: expected a count, found %q", s.Op, parser.Format(s.Operands))
		return 0
	}
	v, err := l.eval(s.Operands[0].Expr)
	switch {
	case err != nil:
		ds.Addf(s.File, s.Num, err.col, "%s: %s", s.Op, err.msg)
		return 0
	case v.Reloc:
		ds.Addf(s.File, s.Num, s.Operands[0].Col, "%s: the count %s is an address", s.Op, s.Operands[0].Expr)
		return 0
	case v.N < 0:
		ds.Addf(s.File, s.Num, s.Operands[0].Col, "%s: the count %s is %d", s.Op, s.Operands[0].Expr, v.N)
		return 0
	}
	return v.N
}

// literal reports the length of the character literal STRING or FORMAT
// assembles.
func (l *Layout) literal(s parser.Statement, ds *diag.List) int {
	if len(s.Operands) != 1 || s.Operands[0].Kind != parser.ItemLiteral {
		ds.Addf(s.File, s.Num, 16, "%s: expected a character literal, found %q", s.Op, parser.Format(s.Operands))
		return 0
	}
	return len(s.Operands[0].Literal)
}

// chars converts a count of characters into a count of address units.
// CPA is the number of characters per address unit (S4D58 6.20), so a
// string that does not fill its last unit still occupies it.
func (l *Layout) chars(n int) int { return (n + l.cpa - 1) / l.cpa }

// check evaluates every expression in the assembly and reports what
// does not work.
//
// It runs over every operand rather than only the ones the location
// counter needed, because that is what makes the relocatable-versus-
// absolute discipline a property of the whole source rather than of
// the handful of places a size is computed. An address multiplied,
// negated, or added to another address is reported here whether it
// occurs in an EQU or in the third operand of a GETAC.
func (l *Layout) check(stmts []parser.Statement, ds *diag.List) {
	for _, s := range stmts {
		for _, it := range s.Operands {
			l.checkItem(s, it, ds)
		}
	}
}

func (l *Layout) checkItem(s parser.Statement, it parser.Item, ds *diag.List) {
	switch it.Kind {
	case parser.ItemExpr:
		if _, err := l.eval(it.Expr); err != nil {
			ds.Addf(s.File, s.Num, err.col, "%s", err.msg)
		}
	case parser.ItemList:
		for _, inner := range it.List {
			l.checkItem(s, inner, ds)
		}
	}
}
