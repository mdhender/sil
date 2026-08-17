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

// Package symtab records where names are defined and where they are
// used.
//
// It closes the reference graph of an assembly without knowing what
// any operation means. SIL's columnar format is what makes that
// possible: a name in the label field is a definition and a name in
// the variable field is a reference, and no per-operation knowledge is
// needed to tell them apart. Character literals contribute nothing,
// because the parser has already separated them from expressions.
//
// The point of resolving names this early is that the answer is
// checkable against the historical source before a single instruction
// has semantics. What comes back undefined is exactly the set of
// symbols the machine-dependent COPY segments have to supply.
//
// Symbol values are not this package's business yet. A value needs a
// location counter, which needs the size of every statement, which
// needs the size of a descriptor to have been chosen.
package symtab

import (
	"sort"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/parser"
)

// Ref is one use of a name.
type Ref struct {
	Line int
	Col  int
}

// Symbol is one name, with the place it was defined and every place
// it was used.
type Symbol struct {
	Name    string
	Defined bool
	DefLine int
	Refs    []Ref
}

// Table holds every name seen in an assembly.
type Table struct {
	file string
	syms map[string]*Symbol
}

// New returns an empty table for the named source file.
func New(file string) *Table {
	return &Table{file: file, syms: make(map[string]*Symbol)}
}

func (t *Table) sym(name string) *Symbol {
	s, ok := t.syms[name]
	if !ok {
		s = &Symbol{Name: name}
		t.syms[name] = s
	}
	return s
}

// Define records a definition of name at the given line. Redefinition
// is reported against the line that defined it first.
func (t *Table) Define(name string, line int, ds *diag.List) {
	s := t.sym(name)
	if s.Defined {
		ds.Addf(t.file, line, 1, "%s is already defined at line %d", name, s.DefLine)
		return
	}
	s.Defined, s.DefLine = true, line
}

// Reference records a use of name.
func (t *Table) Reference(name string, line, col int) {
	s := t.sym(name)
	s.Refs = append(s.Refs, Ref{Line: line, Col: col})
}

// Lookup returns the named symbol, or nil.
func (t *Table) Lookup(name string) *Symbol { return t.syms[name] }

// Len reports how many distinct names the table holds, defined or not.
func (t *Table) Len() int { return len(t.syms) }

// Defined returns the names that have a definition, sorted.
func (t *Table) Defined() []string {
	return t.names(func(s *Symbol) bool { return s.Defined })
}

// Undefined returns the names that are used but never defined,
// sorted.
//
// For the SIL source of SNOBOL4 this is the implementer's contract:
// the list is what COPY PARMS, COPY MLINK and COPY MDATA must supply.
func (t *Table) Undefined() []string {
	return t.names(func(s *Symbol) bool { return !s.Defined && len(s.Refs) > 0 })
}

// Unreferenced returns defined names that nothing uses, sorted. An
// entry point is the usual reason for one.
func (t *Table) Unreferenced() []string {
	return t.names(func(s *Symbol) bool { return s.Defined && len(s.Refs) == 0 })
}

func (t *Table) names(keep func(*Symbol) bool) []string {
	var out []string
	for name, s := range t.syms {
		if keep(s) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// ReportUndefined adds one diagnostic per undefined name, at its first
// use, rather than one per use. A name used forty times is one
// problem, not forty.
func (t *Table) ReportUndefined(ds *diag.List) {
	for _, name := range t.Undefined() {
		s := t.syms[name]
		first := s.Refs[0]
		if n := len(s.Refs); n > 1 {
			ds.Addf(t.file, first.Line, first.Col, "undefined symbol %s, used here and %d more times", name, n-1)
		} else {
			ds.Addf(t.file, first.Line, first.Col, "undefined symbol %s", name)
		}
	}
}

// Collect walks the statements, recording a definition for every
// label field and a reference for every symbol in every operand
// expression.
//
// It applies no per-operation knowledge at all. In particular COPY's
// operand -- MLINK, PARMS, MDATA -- is treated as an ordinary symbol
// reference and so appears among the undefined names, rather than
// being recognised as the name of a source segment. Teaching the
// collector that one fact would move three names out of Undefined and
// is the only per-operation knowledge this stage could have; it is
// left out so the stage has none.
func Collect(file string, stmts []parser.Statement) (*Table, diag.List) {
	var ds diag.List
	t := New(file)
	for _, s := range stmts {
		if s.Label != "" {
			t.Define(s.Label, s.Num, &ds)
		}
		for _, it := range s.Operands {
			collectItem(t, s.Num, it)
		}
	}
	return t, ds
}

func collectItem(t *Table, line int, it parser.Item) {
	switch it.Kind {
	case parser.ItemExpr:
		collectExpr(t, line, it.Expr)
	case parser.ItemList:
		for _, inner := range it.List {
			collectItem(t, line, inner)
		}
	}
}

func collectExpr(t *Table, line int, e parser.Expr) {
	switch e := e.(type) {
	case *parser.Symbol:
		t.Reference(e.Name, line, e.Col)
	case *parser.Unary:
		collectExpr(t, line, e.X)
	case *parser.Binary:
		collectExpr(t, line, e.X)
		collectExpr(t, line, e.Y)
	}
}
