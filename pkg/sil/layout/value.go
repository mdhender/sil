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

package layout

import (
	"fmt"

	"github.com/mdhender/sil/pkg/sil/parser"
)

// Value is the assembly-time value of an expression, together with
// whether it is an address or a plain number.
//
// The distinction matters because SIL expressions mix the two freely
// and only some combinations mean anything. LNKFLD EQU 3*DESCR is a
// count of address units; OBLIST EQU OBSTRT-LNKFLD is the address you
// reach by stepping back that many; BUFEXT EQU DTEND-ANYSP is the
// distance between two addresses and is a count again. Tracking Reloc
// is what lets the assembler tell a typing mistake from a legitimate
// piece of address arithmetic.
type Value struct {
	N     int
	Reloc bool // an address in core rather than a plain number
}

// Abs returns an absolute value.
func Abs(n int) Value { return Value{N: n} }

// Addr returns a relocatable value.
func Addr(n int) Value { return Value{N: n, Reloc: true} }

func (v Value) String() string {
	if v.Reloc {
		return fmt.Sprintf("%d(reloc)", v.N)
	}
	return fmt.Sprintf("%d", v.N)
}

// evalErr is why an expression could not be evaluated.
//
// Unresolved is separated from every other failure because the two are
// handled differently: an unresolved symbol may simply not have been
// reached yet, and the resolver retries, while a malformed expression
// will never become well formed and is reported once at the end.
type evalErr struct {
	col        int
	msg        string
	unresolved bool
}

func (e *evalErr) Error() string { return e.msg }

func unresolved(name string, col int) *evalErr {
	return &evalErr{col: col, msg: fmt.Sprintf("%s has no value", name), unresolved: true}
}

func malformed(col int, format string, args ...any) *evalErr {
	return &evalErr{col: col, msg: fmt.Sprintf(format, args...)}
}

// eval computes the value of an expression under the discipline of
// S4D58 7.6's arithmetic.
//
// The document does not spell the rules out, but they are forced by
// what the operands mean. An address plus a count is an address; the
// difference of two addresses is a count; an address times anything is
// not a location. Three of the four identities this stage is measured
// against depend on getting each of these right, and OBEND at line
// 5475 depends on all of them at once:
//
//	OBEND  DESCR   OBLIST+DESCR*OBOFF,0,0
//
// where DESCR*OBOFF is a count of address units and OBLIST is an
// address, so the sum is an address. Had `*` bound loosest, OBLIST
// would have been multiplied by 254 and this rule would have caught it
// as `reloc * anything`.
func (l *Layout) eval(e parser.Expr) (Value, *evalErr) {
	switch e := e.(type) {
	case *parser.Number:
		return Abs(e.Value), nil

	case *parser.Symbol:
		v, ok := l.values[e.Name]
		if !ok {
			return Value{}, unresolved(e.Name, e.Col)
		}
		return v, nil

	case *parser.Unary:
		x, err := l.eval(e.X)
		if err != nil {
			return Value{}, err
		}
		if x.Reloc {
			return Value{}, malformed(e.Col, "cannot negate the address %s", e.X)
		}
		return Abs(-x.N), nil

	case *parser.Binary:
		x, err := l.eval(e.X)
		if err != nil {
			return Value{}, err
		}
		y, err := l.eval(e.Y)
		if err != nil {
			return Value{}, err
		}
		switch e.Op {
		case '*':
			if x.Reloc || y.Reloc {
				return Value{}, malformed(e.Col, "cannot multiply the address %s", relocOperand(e, x))
			}
			return Abs(x.N * y.N), nil
		case '+':
			if x.Reloc && y.Reloc {
				return Value{}, malformed(e.Col, "cannot add the addresses %s and %s", e.X, e.Y)
			}
			return Value{N: x.N + y.N, Reloc: x.Reloc || y.Reloc}, nil
		case '-':
			if !x.Reloc && y.Reloc {
				return Value{}, malformed(e.Col, "cannot subtract the address %s from the number %s", e.Y, e.X)
			}
			// address - address is the distance between them, and so a
			// count; address - count is an address.
			return Value{N: x.N - y.N, Reloc: x.Reloc && !y.Reloc}, nil
		}
		return Value{}, malformed(e.Col, "unknown operator %q", string(e.Op))
	}
	return Value{}, malformed(e.Column(), "unknown expression %s", e)
}

// relocOperand names whichever side of a product is an address, for
// the diagnostic.
func relocOperand(e *parser.Binary, x Value) parser.Expr {
	if x.Reloc {
		return e.X
	}
	return e.Y
}
