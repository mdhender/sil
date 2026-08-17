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

// Package parser turns scanned SIL lines into statements with parsed
// operand lists.
//
// The grammar is the whole of S4D58 7.6's variable field:
//
//	operands := ε | ',' | item (',' item)*
//	item     := ε | expr | '(' [item] (',' [item])* ')' | literal
//	expr     := addend (('+'|'-') addend)*
//	addend   := factor ('*' factor)*
//	factor   := ['-'] (symbol | integer)
//
// Two shapes are easy to get wrong. A lone comma in column 16 means
// *no operands at all*, not two null ones -- 7.6 says "If there are no
// operands, there is a comma in column 16 and a blank in column 17",
// which is why PROC, INIT, ISTACK and LHERE are all written `OP ,`.
// And a null item elsewhere is a real, positional operand: an omitted
// branch point, meaning "fall through" (5.2).
//
// The parser knows nothing about what any operation means. It does not
// know which operands are branch points, and it does not resolve any
// name.
package parser

import (
	"strings"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/scanner"
)

// operandColumn is the one-based column the variable field starts at
// (S4D58 7.6).
const operandColumn = 16

// ItemKind distinguishes the four shapes an operand may take.
type ItemKind int

const (
	// ItemNull is an omitted operand. For a branch point this means
	// "fall through to the next operation" (S4D58 5.2).
	ItemNull ItemKind = iota
	// ItemExpr is a symbol, an integer, or arithmetic on them.
	ItemExpr
	// ItemList is a parenthesised list. Lists do not nest (S4D58 7.6).
	ItemList
	// ItemLiteral is a character literal in single quotes.
	ItemLiteral
)

func (k ItemKind) String() string {
	switch k {
	case ItemNull:
		return "null"
	case ItemExpr:
		return "expression"
	case ItemList:
		return "list"
	case ItemLiteral:
		return "literal"
	}
	return "unknown"
}

// Item is one operand.
type Item struct {
	Kind    ItemKind
	Expr    Expr   // ItemExpr
	List    []Item // ItemList
	Literal string // ItemLiteral, quotes removed
	Col     int    // one-based source column
}

// Statement is one SIL statement with its operands parsed.
type Statement struct {
	File     string
	Num      int    // one-based source line
	Text     string // the raw line, kept for listings and traces
	Label    string
	Op       string
	Operands []Item
}

// Parse parses the operand field of every statement, skipping comment
// lines, and accumulates diagnostics rather than stopping at the first
// bad statement.
func Parse(lines []scanner.Line) ([]Statement, diag.List) {
	var (
		stmts []Statement
		ds    diag.List
	)
	for _, l := range lines {
		if l.Comment || l.Op == "" {
			continue
		}
		stmts = append(stmts, ParseOne(l, &ds))
	}
	return stmts, ds
}

// ParseOne parses the operand field of one scanned line. Pass a nil
// list to parse without collecting diagnostics.
func ParseOne(l scanner.Line, ds *diag.List) Statement {
	if ds == nil {
		ds = &diag.List{}
	}
	return Statement{
		File:     l.File,
		Num:      l.Num,
		Text:     l.Text,
		Label:    l.Label,
		Op:       l.Op,
		Operands: ParseOperands(l.File, l.Num, l.Operand, ds),
	}
}

// ParseOperands parses one variable field.
func ParseOperands(file string, line int, s string, ds *diag.List) []Item {
	// An empty field, or the lone comma that stands for one.
	if s == "" || s == "," {
		return nil
	}
	p := &parser{file: file, line: line, s: s, ds: ds}
	items := p.items(0)
	if !p.eof() {
		p.errorf(p.i, "unexpected %q in the operand field", string(p.s[p.i]))
	}
	return items
}

type parser struct {
	file string
	line int
	s    string
	i    int
	ds   *diag.List
}

func (p *parser) eof() bool { return p.i >= len(p.s) }

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.s[p.i]
}

// col converts an offset in the variable field to a source column.
func (p *parser) col(off int) int { return operandColumn + off }

func (p *parser) errorf(off int, format string, args ...any) {
	p.ds.Addf(p.file, p.line, p.col(off), format, args...)
}

// items parses a comma-separated operand list. At depth 0 it runs to
// the end of the field; inside parentheses it stops at the closing
// one.
func (p *parser) items(depth int) []Item {
	var items []Item
	for {
		items = append(items, p.item(depth))
		if p.peek() == ',' {
			p.i++
			continue
		}
		return items
	}
}

func (p *parser) item(depth int) Item {
	start := p.i
	switch c := p.peek(); {
	case c == 0, c == ',':
		return Item{Kind: ItemNull, Col: p.col(start)}
	case depth > 0 && c == ')':
		return Item{Kind: ItemNull, Col: p.col(start)}
	case c == '\'':
		return p.literal()
	case c == '(':
		return p.list(depth)
	default:
		return Item{Kind: ItemExpr, Expr: p.expr(), Col: p.col(start)}
	}
}

func (p *parser) literal() Item {
	start := p.i
	p.i++ // the opening quote
	for !p.eof() && p.s[p.i] != '\'' {
		p.i++
	}
	if p.eof() {
		// The scanner reports this too, but a literal that reaches
		// here unterminated would silently swallow the rest of the
		// field.
		p.errorf(start, "unterminated character literal")
		return Item{Kind: ItemLiteral, Literal: p.s[start+1:], Col: p.col(start)}
	}
	text := p.s[start+1 : p.i]
	p.i++ // the closing quote
	return Item{Kind: ItemLiteral, Literal: text, Col: p.col(start)}
}

func (p *parser) list(depth int) Item {
	start := p.i
	if depth > 0 {
		p.errorf(start, "nested parentheses: lists do not nest")
	}
	p.i++ // the opening paren
	items := p.items(depth + 1)
	if p.peek() != ')' {
		p.errorf(start, "unclosed parenthesis")
	} else {
		p.i++
	}
	return Item{Kind: ItemList, List: items, Col: p.col(start)}
}

// expr parses addends separated by `+` and `-`, which bind loosest.
func (p *parser) expr() Expr {
	x := p.addend()
	for {
		op := p.peek()
		if op != '+' && op != '-' {
			return x
		}
		col := p.col(p.i)
		p.i++
		x = &Binary{Op: op, X: x, Y: p.addend(), Col: col}
	}
}

// addend parses factors separated by `*`, which binds tightest.
func (p *parser) addend() Expr {
	x := p.factor()
	for p.peek() == '*' {
		col := p.col(p.i)
		p.i++
		x = &Binary{Op: '*', X: x, Y: p.factor(), Col: col}
	}
	return x
}

func (p *parser) factor() Expr {
	start := p.i
	if p.peek() == '-' {
		p.i++
		return &Unary{Op: '-', X: p.factor(), Col: p.col(start)}
	}
	switch c := p.peek(); {
	case c >= '0' && c <= '9':
		n := 0
		for c := p.peek(); c >= '0' && c <= '9'; c = p.peek() {
			n = n*10 + int(c-'0')
			p.i++
		}
		return &Number{Value: n, Col: p.col(start)}
	case c >= 'A' && c <= 'Z':
		for c := p.peek(); (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'); c = p.peek() {
			p.i++
		}
		return &Symbol{Name: p.s[start:p.i], Col: p.col(start)}
	default:
		if c == 0 {
			p.errorf(start, "expected a symbol or an integer, found end of operand")
		} else {
			p.errorf(start, "expected a symbol or an integer, found %q", string(c))
			p.i++ // consume it, so the parser makes progress
		}
		return &Number{Value: 0, Col: p.col(start)}
	}
}

// String renders an operand list back to the source form, which makes
// a parse testable against the text it came from.
func Format(items []Item) string {
	var sb strings.Builder
	for i, it := range items {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(it.String())
	}
	return sb.String()
}

func (it Item) String() string {
	switch it.Kind {
	case ItemNull:
		return ""
	case ItemExpr:
		return it.Expr.String()
	case ItemLiteral:
		return "'" + it.Literal + "'"
	case ItemList:
		return "(" + Format(it.List) + ")"
	}
	return "?"
}
