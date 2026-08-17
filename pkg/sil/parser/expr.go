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

package parser

import (
	"fmt"
	"sort"
)

// Expr is an assembly-time expression appearing in an operand.
//
// S4D58 7.6 allows "arithmetic expressions containing identifiers and
// constants". The historical source uses only three operators, `*`,
// `+` and `-`, with no parentheses and no division.
type Expr interface {
	fmt.Stringer

	// Column reports the one-based source column the expression
	// starts at.
	Column() int

	exprNode()
}

// Symbol is a reference to a name defined elsewhere in the assembly,
// or supplied by a COPY segment.
type Symbol struct {
	Name string
	Col  int
}

// Number is an unsigned integer constant.
type Number struct {
	Value int
	Col   int
}

// Unary is a negated expression. Only `-` occurs; it appears twice in
// the historical source, both times as `GETAC TVAL,PDLPTR,-2*DESCR`.
type Unary struct {
	Op  byte
	X   Expr
	Col int
}

// Binary is `*`, `+` or `-` applied to two expressions.
//
// `*` binds tighter than `+` and `-`. Exactly one statement in the
// historical source depends on that -- OBEND at line 5475 --  and it
// depends on it completely: OBLIST is a relocatable address and OBOFF
// is 254, so evaluating left to right would multiply an address by
// 254 instead of offsetting it.
type Binary struct {
	Op   byte
	X, Y Expr
	Col  int
}

func (e *Symbol) Column() int { return e.Col }
func (e *Number) Column() int { return e.Col }
func (e *Unary) Column() int  { return e.Col }
func (e *Binary) Column() int { return e.Col }

func (*Symbol) exprNode() {}
func (*Number) exprNode() {}
func (*Unary) exprNode()  {}
func (*Binary) exprNode() {}

func (e *Symbol) String() string { return e.Name }
func (e *Number) String() string { return fmt.Sprintf("%d", e.Value) }
func (e *Unary) String() string  { return fmt.Sprintf("%c%s", e.Op, e.X) }

// String parenthesises fully, so that a test of an expression's shape
// is a test of its precedence.
func (e *Binary) String() string {
	return fmt.Sprintf("(%s%c%s)", e.X, e.Op, e.Y)
}

// Symbols returns the names referenced by e, sorted and deduplicated.
func Symbols(e Expr) []string {
	seen := map[string]bool{}
	collect(e, seen)
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func collect(e Expr, seen map[string]bool) {
	switch e := e.(type) {
	case *Symbol:
		seen[e.Name] = true
	case *Unary:
		collect(e.X, seen)
	case *Binary:
		collect(e.X, seen)
		collect(e.Y, seen)
	}
}
