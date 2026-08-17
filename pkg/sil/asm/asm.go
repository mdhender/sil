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

// Package asm turns SIL source into a machine ready to run.
//
// It is the whole front end in one call: scan the columns, expand the
// COPY segments, parse the operand fields, check every statement
// against the instruction table, run the location counter, and fill
// core. Each of those is a package of its own and each accumulates its
// own diagnostics; asm runs them in order and stops at the first stage
// that reported anything, because a stage's output is not worth
// reading once an earlier one has failed.
//
// There is no image format. Assemble returns a populated machine, and
// a test drives it directly. S4D58 has nothing to say about object
// files and neither does the development workflow yet.
package asm

import (
	"fmt"
	"io"

	"github.com/mdhender/sil/pkg/sil"
	"github.com/mdhender/sil/pkg/sil/copyseg"
	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/layout"
	"github.com/mdhender/sil/pkg/sil/op"
	"github.com/mdhender/sil/pkg/sil/parser"
	"github.com/mdhender/sil/pkg/sil/scanner"
	"github.com/mdhender/sil/pkg/sil/syntab"
)

// Options control what the assembler does besides assembling. The zero
// value is silent.
type Options struct {
	// Trace is handed to the machine, which writes one line per
	// instruction to it. Program output never goes here.
	Trace io.Writer
	// Entry is the label execution starts at. The SNOBOL4 source calls
	// it BEGIN, and 6.46 makes INIT the first instruction executed.
	Entry string
	// Host is what the machine's input and output operations talk to.
	Host sil.Host
	// Segments resolves COPY. Nil uses the machine-dependent segments
	// this implementation ships.
	Segments copyseg.Resolver
}

const defaultEntry = "BEGIN"

// Assemble assembles SIL source and returns a machine positioned at
// the entry point.
//
// The machine is meaningful only when the diagnostic list is empty.
func Assemble(file string, src []byte, opts Options) (*sil.VM, diag.List) {
	lines, ds := scanner.Scan(file, src)
	if len(ds) > 0 {
		return nil, ds
	}

	segments := opts.Segments
	if segments == nil {
		segments = copyseg.Source
	}
	lines = copyseg.ExpandWith(lines, segments, &ds)
	if len(ds) > 0 {
		return nil, ds
	}

	stmts, ds := parser.Parse(lines)
	if len(ds) > 0 {
		return nil, ds
	}

	if ds := op.Check(stmts); len(ds) > 0 {
		return nil, ds
	}

	lay, ds := layout.Run(stmts)
	if len(ds) > 0 {
		return nil, ds
	}

	return emit(stmts, lay, opts)
}

// emitter carries the state of a pass over the statements.
type emitter struct {
	vm    *sil.VM
	lay   *layout.Layout
	stmts []parser.Statement
	ds    diag.List

	// procs holds the address of every PROC entry point, so that the
	// procedure operand of BRANCH and RCALL can be checked before it
	// is discarded (6.15, 6.78 note 1).
	procs map[int]bool

	// formats holds how many characters every FORMAT assembled.
	// 6.75's figure gives OUTPUT's format as characters at a location
	// and never says how many, because a FORTRAN IV routine reads to
	// the closing parenthesis; this machine does not read formats, so
	// the count travels with the address. It is the one thing about a
	// FORMAT operand that only the assembler knows.
	formats map[int]int
}

func emit(stmts []parser.Statement, lay *layout.Layout, opts Options) (*sil.VM, diag.List) {
	descr, spec, cpa := lay.Params()

	e := &emitter{
		vm: &sil.VM{
			Core:    make([]sil.Cell, lay.End()),
			Symbols: symbols(lay),
			Descr:   descr,
			Spec:    spec,
			CPA:     cpa,
			Host:    opts.Host,
			Trace:   opts.Trace,
		},
		lay:     lay,
		stmts:   stmts,
		procs:   map[int]bool{},
		formats: map[int]int{},
	}

	// A cell holds one character, so a machine that packs them would
	// need a wider cell. See sil.VM.Chars.
	if cpa != 1 {
		e.ds.Addf("COPY PARMS", 0, 0, "CPA is %d; this machine stores one character per address unit", cpa)
		return nil, e.ds
	}

	for i, s := range stmts {
		p := lay.Placement(i)
		switch op.Lookup(s.Op) {
		case op.PROC:
			e.procs[p.Addr] = true
		case op.FORMAT:
			e.formats[p.Addr] = p.Size
		}
	}
	for i, s := range stmts {
		e.statement(s, lay.Placement(i))
	}
	e.syntaxTables()

	entry := opts.Entry
	if entry == "" {
		entry = defaultEntry
	}
	at, ok := lay.Addr(entry)
	if !ok {
		e.ds.Addf(fileOf(stmts), 0, 0, "the entry point %s is not defined", entry)
	}
	e.vm.PC = at

	if len(e.ds) > 0 {
		return nil, e.ds
	}
	return e.vm, nil
}

// symbols flattens the layout into what the machine needs at run time.
func symbols(lay *layout.Layout) map[string]int {
	out := make(map[string]int, len(lay.Symbols()))
	for _, name := range lay.Symbols() {
		out[name], _ = lay.Addr(name)
	}
	return out
}

func fileOf(stmts []parser.Statement) string {
	if len(stmts) == 0 {
		return ""
	}
	return stmts[0].File
}

// statement assembles one statement into the cells the location
// counter set aside for it.
func (e *emitter) statement(s parser.Statement, p layout.Placement) {
	k := op.Lookup(s.Op)
	entry := op.Get(k)
	if p.Size == 0 {
		return
	}

	src := sil.Src{File: s.File, Line: s.Num, Text: s.Text}
	// An omitted branch point is the address of the operation after
	// this one (5.2), which is the whole reason the machine never
	// sees a null.
	next := p.Addr + p.Size

	switch entry.Size {
	case op.SizeDescr:
		e.put(p.Addr, sil.Cell{
			Kind: sil.Data, Op: k, Src: src,
			A: e.field(s, 0), F: e.field(s, 1), V: e.field(s, 2),
		})

	case op.SizeSpec:
		e.specifier(p.Addr, k, src, e.field(s, 0), e.field(s, 1), e.field(s, 2), e.field(s, 3), e.field(s, 4))

	case op.SizeArray:
		for a := p.Addr; a < next; a++ {
			e.put(a, sil.Cell{Kind: sil.Data, Op: k, Src: src})
		}

	case op.SizeBuffer:
		// 6.17 note 1: every character must be blank, not zero, when
		// execution begins.
		for a := p.Addr; a < next; a++ {
			e.put(a, sil.Cell{Kind: sil.Data, Op: k, Src: src, Ch: ' '})
		}

	case op.SizeString:
		// 6.117 note 1: LOC is the specifier; the string follows it.
		text := e.literal(s)
		at := p.Addr + e.vm.Spec
		e.specifier(p.Addr, k, src, at, 0, 0, 0, len(text))
		e.characters(at, k, src, text)

	case op.SizeChars:
		e.characters(p.Addr, k, src, e.literal(s))

	case op.SizeCall:
		e.call(s, p, src)

	case op.SizeVector:
		// The second operand is N, the number of locations in the
		// vector, which the machine cannot recover from the cells: a
		// BRANCH belonging to a SELBRA is indistinguishable from any
		// other. 6.98 note 3 asks for a check that I is in 1..N+1, so
		// N is assembled alongside DESCR.
		e.put(p.Addr, sil.Cell{
			Kind: sil.Instr, Op: k, Src: src,
			Ops: []int{e.field(s, 0), next - p.Addr - 1},
		})
		e.vector(s, entry, p.Addr+1, next, src)

	default:
		e.put(p.Addr, sil.Cell{Kind: sil.Instr, Op: k, Src: src, Ops: e.operands(s, entry, next)})
	}
}

// call assembles an RCALL: the operation, the return point that holds
// the descriptor the value comes back in, and one cell per exit
// (6.87).
func (e *emitter) call(s parser.Statement, p layout.Placement, src sil.Src) {
	k := op.Lookup(s.Op)
	entry := op.Get(k)

	proc := e.field(s, 1)
	e.procedure(s, 1, proc)

	ops := []int{e.field(s, 0), proc}
	ops = append(ops, e.list(s, 2)...)
	e.put(p.Addr, sil.Cell{Kind: sil.Instr, Op: k, Src: src, Ops: ops})

	// LOC, the return point. Note 3: an omitted DESCR leaves it zero
	// and RRTURN stores nothing.
	e.put(p.Addr+1, sil.Cell{Kind: sil.Return, Op: k, Src: src, A: ops[0]})

	e.vector(s, entry, p.Addr+2, p.Addr+p.Size, src)
}

// vector assembles the branch points that follow an RCALL or a SELBRA.
//
// They are BRANCH instructions. 6.87 prints them that way -- "LOC OP
// DESCR1 / BRANCH LOC1 / ... / BRANCH LOCM" -- and that is what makes
// the return dispatch nothing but arithmetic: RRTURN sets the program
// counter to LOC+N and the machine executes whatever it lands on,
// which is a branch for N <= M and the operation after the call for N
// = M+1. SELBRA is the same machinery with its own index (6.98), so
// this is written once and called from both.
//
// An omitted branch point is resolved here to the operation after the
// whole thing (6.87 note 5, 6.98 note 1).
func (e *emitter) vector(s parser.Statement, entry op.Entry, at, next int, src sil.Src) {
	i, ok := entry.Vector()
	if !ok {
		return
	}
	for j, target := range e.branches(s, i, next) {
		e.put(at+j, sil.Cell{Kind: sil.Instr, Op: op.BRANCH, Src: src, Ops: []int{target}})
	}
}

// specifier writes the two cells of a specifier (3.2).
func (e *emitter) specifier(at int, k op.Kind, src sil.Src, a, f, v, o, l int) {
	e.put(at, sil.Cell{Kind: sil.Data, Op: k, Src: src, A: a, F: f, V: v})
	e.put(at+e.vm.Descr, sil.Cell{Kind: sil.Data, Op: k, Src: src, A: o, V: l})
}

// characters writes one character per cell.
func (e *emitter) characters(at int, k op.Kind, src sil.Src, text string) {
	for i := 0; i < len(text); i++ {
		e.put(at+i, sil.Cell{Kind: sil.Data, Op: k, Src: src, Ch: text[i]})
	}
}

// operands resolves an operation's operands in table order, flattening
// a list into the elements it holds.
func (e *emitter) operands(s parser.Statement, entry op.Entry, next int) []int {
	var out []int
	for i, o := range entry.Operands {
		switch {
		case o.Slot == op.SlotProc:
			// 6.15: the procedure a branch names is checked and then
			// dropped. Nothing in this machine is based.
			e.procedure(s, i, e.field(s, i))
		case o.Slot == op.SlotList && o.Elem == op.SlotBranch:
			out = append(out, e.branches(s, i, next)...)
		case o.Slot == op.SlotList:
			out = append(out, e.list(s, i)...)
		case o.Slot == op.SlotBranch:
			out = append(out, e.branch(s, i, next))
		case o.Slot == op.SlotFormat:
			// The address and then the length; see emitter.formats.
			at := e.field(s, i)
			out = append(out, at, e.formats[at])
		default:
			out = append(out, e.field(s, i))
		}
	}
	return out
}

// field resolves the i'th operand, which is zero when it was omitted.
func (e *emitter) field(s parser.Statement, i int) int {
	if i >= len(s.Operands) || s.Operands[i].Kind != parser.ItemExpr {
		return 0
	}
	v, err := e.lay.Evaluate(s.Operands[i].Expr)
	if err != nil {
		e.ds.Addf(s.File, s.Num, s.Operands[i].Col, "%v", err)
		return 0
	}
	return v.N
}

// branch resolves a branch point, with an omitted one becoming the
// address of the operation that follows (5.2).
func (e *emitter) branch(s parser.Statement, i, next int) int {
	if i >= len(s.Operands) || s.Operands[i].Kind == parser.ItemNull {
		return next
	}
	return e.field(s, i)
}

// branches resolves the elements of a list operand of branch points.
func (e *emitter) branches(s parser.Statement, i, next int) []int {
	var out []int
	for j, it := range elements(s, i) {
		if it.Kind == parser.ItemNull {
			out = append(out, next)
			continue
		}
		out = append(out, e.element(s, i, j, it))
	}
	return out
}

// list resolves the elements of a list operand.
func (e *emitter) list(s parser.Statement, i int) []int {
	var out []int
	for j, it := range elements(s, i) {
		out = append(out, e.element(s, i, j, it))
	}
	return out
}

func (e *emitter) element(s parser.Statement, i, j int, it parser.Item) int {
	if it.Kind != parser.ItemExpr {
		e.ds.Addf(s.File, s.Num, it.Col, "%s: element %d of operand %d is not an expression", s.Op, j+1, i+1)
		return 0
	}
	v, err := e.lay.Evaluate(it.Expr)
	if err != nil {
		e.ds.Addf(s.File, s.Num, it.Col, "%v", err)
		return 0
	}
	return v.N
}

// elements returns the items of a list operand. A list of one is
// written without its parentheses, and an omitted list is empty.
func elements(s parser.Statement, i int) []parser.Item {
	if i >= len(s.Operands) {
		return nil
	}
	switch it := s.Operands[i]; it.Kind {
	case parser.ItemList:
		return it.List
	case parser.ItemNull:
		return nil
	default:
		return []parser.Item{it}
	}
}

// literal returns the characters of a STRING or FORMAT operand.
func (e *emitter) literal(s parser.Statement) string {
	if len(s.Operands) == 0 || s.Operands[0].Kind != parser.ItemLiteral {
		return ""
	}
	return s.Operands[0].Literal
}

// procedure checks that an operand names a procedure entry point.
func (e *emitter) procedure(s parser.Statement, i, at int) {
	if i >= len(s.Operands) || s.Operands[i].Kind == parser.ItemNull {
		return
	}
	if !e.procs[at] {
		e.ds.Addf(s.File, s.Num, s.Operands[i].Col,
			"%s: %s is at %d, which is not a PROC entry point (6.78 note 1)",
			s.Op, s.Operands[i].Expr, at)
	}
}

// put writes a cell, reporting an overlap rather than letting one
// statement quietly overwrite another.
func (e *emitter) put(at int, c sil.Cell) {
	if at < 0 || at >= len(e.vm.Core) {
		e.ds.Addf(c.Src.File, c.Src.Line, 0, "assembles at %d, outside the %d units of core", at, len(e.vm.Core))
		return
	}
	if old := e.vm.Core[at]; old.Src.File != "" {
		e.ds.Addf(c.Src.File, c.Src.Line, 0, "assembles at %d, which %s already holds", at, old.Src)
		return
	}
	e.vm.Core[at] = c
}

// syntaxTables fills in the contents of every syntax table the
// assembly declares.
//
// 6.20 note 1 says COPY "may simply expand into the data required",
// and MDATA declares the twenty-five tables of Appendix A as
// ARRAY ALPHSZ -- the right amount of storage, with nothing in it. The
// contents cannot be written as SIL text, because a table entry is
// generated data rather than something a person types: 4.2 says so
// itself, recommending "some kind of automatic technique ... both to
// ensure accuracy and because of the large amount of data involved".
// So the expansion happens in two steps, and this is the second.
//
// It runs over the tables Appendix A describes, not over the ones the
// assembly happens to define, and skips any the assembly does not
// declare: a test program that wants one table should not have to
// carry twenty-four others. What it will not do is leave a declared
// table half filled -- a name a description needs and the assembly
// does not define is a diagnostic.
//
// Only the three fields of an entry are written. The cells keep the
// source line the ARRAY assembled them from, so a core listing still
// says where the storage came from.
func (e *emitter) syntaxTables() {
	alphsz, ok := e.lay.Addr("ALPHSZ")
	if !ok {
		return // no PARMS, so no character set and no tables
	}
	value := func(name string) (int, bool) { return e.lay.Addr(name) }

	for _, name := range syntab.Names() {
		at, ok := e.lay.Addr(name)
		if !ok {
			continue
		}
		entries, err := syntab.Build(name, alphsz, value)
		if err != nil {
			e.ds.Addf("Appendix A", 0, 0, "%v", err)
			continue
		}
		for i, entry := range entries {
			a := at + i*e.vm.Descr
			if a < 0 || a >= len(e.vm.Core) {
				e.ds.Addf("Appendix A", 0, 0,
					"%s needs %d entries from %d, which does not fit in the %d units of core",
					name, len(entries), at, len(e.vm.Core))
				break
			}
			c := &e.vm.Core[a]
			c.A, c.F, c.V = entry.Next, entry.Indicator, entry.Put
		}
	}
}

// Listing renders core the way an assembly listing would, one line per
// cell, each citing the source line that assembled it.
func Listing(w io.Writer, vm *sil.VM) error {
	for a, c := range vm.Core {
		if _, err := fmt.Fprintf(w, "%6d %-11s %-44s %s\n", a, c.Kind, c, c.Src.Text); err != nil {
			return err
		}
	}
	return nil
}
