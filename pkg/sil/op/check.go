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

package op

import (
	"fmt"

	"github.com/mdhender/sil/pkg/sil/diag"
	"github.com/mdhender/sil/pkg/sil/parser"
)

// Check reports every statement whose operand list does not fit the
// table.
//
// What it checks is shape: how many operands there are, and whether
// each is the kind of thing -- an expression, a character literal, a
// list -- that the slot takes. It does not check what a name refers
// to. The symbol table and the location counter have already made
// every name resolve; whether the descriptor slot of a MOVD actually
// names a DESCR statement is a question about the program, not about
// its syntax, and nothing in S4D58 says an operand has to be written
// as a label of the right directive.
//
// It reports one diagnostic per bad operand rather than stopping at
// the statement, and keeps going to the end of the assembly.
func Check(stmts []parser.Statement) diag.List {
	var ds diag.List
	for _, s := range stmts {
		CheckStatement(s, &ds)
	}
	return ds
}

// CheckStatement checks one statement against the table.
func CheckStatement(s parser.Statement, ds *diag.List) {
	k := Lookup(s.Op)
	if k == Invalid {
		ds.Addf(s.File, s.Num, 8, "unknown operation %s", s.Op)
		return
	}
	e := Get(k)

	if n := len(s.Operands); n > e.MaxArgs() {
		ds.Addf(s.File, s.Num, operandColumn(s, e.MaxArgs()),
			"%s takes %s, found %d", e.Mnemonic, plural(e.MaxArgs(), "operand"), n)
		return
	}

	for i, o := range e.Operands {
		var it parser.Item
		present := i < len(s.Operands)
		if present {
			it = s.Operands[i]
		}
		switch {
		case !present || it.Kind == parser.ItemNull:
			if !o.Optional {
				ds.Addf(s.File, s.Num, operandColumn(s, i), "%s: %s is required", e.Mnemonic, o.Name)
			}
		case o.Slot == SlotList:
			checkList(s, e, o, it, ds)
		case o.Slot == SlotLiteral:
			if it.Kind != parser.ItemLiteral {
				ds.Addf(s.File, s.Num, it.Col, "%s: %s must be a character literal, found %s", e.Mnemonic, o.Name, it.Kind)
			}
		default:
			if it.Kind != parser.ItemExpr {
				ds.Addf(s.File, s.Num, it.Col, "%s: %s must be an expression, found %s", e.Mnemonic, o.Name, it.Kind)
			}
		}
	}
}

// checkList checks a list operand.
//
// A list of one is written without its parentheses throughout the
// SNOBOL4 source -- POP XCL, RCALL ,GCM,(GCBLK),GCBA2 -- so a bare
// item in a list slot is a list of one. 7.6 describes lists as items
// in parentheses without saying the parentheses are required when
// there is nothing to separate, and 1,266 statements are written the
// short way.
func checkList(s parser.Statement, e Entry, o Operand, it parser.Item, ds *diag.List) {
	elems := it.List
	if it.Kind != parser.ItemList {
		elems = []parser.Item{it}
	}
	for _, el := range elems {
		switch {
		case el.Kind == parser.ItemNull:
			// A null element is an omitted branch point (5.2). In a
			// list of anything else it is a missing argument.
			if o.Elem != SlotBranch {
				ds.Addf(s.File, s.Num, el.Col, "%s: %s has an empty element", e.Mnemonic, o.Name)
			}
		case el.Kind != parser.ItemExpr:
			ds.Addf(s.File, s.Num, el.Col, "%s: every element of %s must be an expression, found %s", e.Mnemonic, o.Name, el.Kind)
		}
	}
}

// operandColumn is where the i'th operand starts, or the end of the
// list when there is no i'th operand.
func operandColumn(s parser.Statement, i int) int {
	if i < len(s.Operands) {
		return s.Operands[i].Col
	}
	if n := len(s.Operands); n > 0 {
		return s.Operands[n-1].Col
	}
	return 16
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
