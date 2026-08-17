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

package sil

import (
	"fmt"
	"strings"

	"github.com/mdhender/sil/pkg/sil/op"
)

// Kind says what a cell is.
//
// SIL does not distinguish code from data -- both are descriptors, and
// the garbage collector walks the stack as descriptors regardless of
// what was pushed onto it (S4D58 6.87 note 8). The distinction here is
// the machine's, not the language's: it is what lets a corrupted
// branch trap at the cell it lands on rather than executing whatever
// happens to be there.
type Kind uint8

const (
	// Data is a cell assembled by one of the six data-assembling
	// directives of S4D58 7.5: a descriptor, half of a specifier, an
	// element of an array or a buffer, or a character of a string.
	Data Kind = iota
	// Instr is an operation the machine executes.
	Instr
	// Return is the cell at LOC, immediately after an RCALL. 6.87
	// calls it "OP DESCR1" and note 2 says OP is ordinarily the
	// instruction that stores the value RRTURN returns; RRTURN does
	// that itself here, so the cell is data, and its address field is
	// the descriptor the value goes into -- zero when the RCALL asked
	// for no value (note 3).
	Return
)

func (k Kind) String() string {
	switch k {
	case Data:
		return "data"
	case Instr:
		return "instruction"
	case Return:
		return "return"
	}
	return fmt.Sprintf("Kind(%d)", uint8(k))
}

// Cell is one address unit of core.
//
// A cell is a descriptor. S4D58 3.1 gives a descriptor three fields --
// address, flag and value -- and says the size of each is whatever the
// data requires; 3.1.1 adds that "descriptors do not have to address
// individual characters of strings", which is what allows DESCR to be
// one address unit here and a cell and a descriptor to be the same
// thing. See pkg/sil/copyseg/parms.sil for the rest of that choice.
//
// A specifier (3.2) is two adjacent cells, not a fifth and sixth field
// on one: "specifiers are composed of two descriptors ... the address
// field of [the second] is used to represent the offset ... the value
// field of this other descriptor is used for the length." The SNOBOL4
// source moves specifiers half at a time, so this is required rather
// than decorative. Spec, Offset and Length below read the pair.
type Cell struct {
	// Kind is what the machine will do with this cell.
	Kind Kind
	// Op is the operation the cell executes, or, for a cell that is
	// not executed, the directive that assembled it. Keeping it on
	// data cells is what makes a core listing readable.
	Op op.Kind

	// A is the address field (3.1.1). It holds an address, an integer,
	// or, in the second half of a specifier, a character offset.
	A int
	// F is the flag field (3.1.2), a set of bits tested and set
	// individually. The five SNOBOL4 uses -- TTL, MARK, PTR, FNC and
	// STTL -- get their values from PARMS, so this is a plain number
	// and the machine never names a bit.
	F int
	// V is the value field (3.1.3): an unsigned quantity, the encoded
	// data type of a source-language object, the length of a string,
	// or the size of an aggregate. In the second half of a specifier
	// it is the length.
	V int

	// Ch is the character this cell holds, when it holds string data
	// (3.3). One character per cell, which is what CPA = 1 means; a
	// machine that packed characters would need this to be as wide as
	// CPA.
	Ch byte

	// Ops are the operands of an instruction cell, resolved to
	// numbers, in the order the instruction table gives them. A list
	// operand contributes its elements in order, so an operation with
	// a variable-length list -- PUSH, POP, the arguments of an RCALL
	// -- has as many entries as it was written with.
	Ops []int

	// Src is where the cell came from. Core doubles as its own
	// listing, and every trace line can cite a source line.
	Src Src
}

// Src locates a cell in the source that assembled it.
type Src struct {
	File string
	Line int
	Text string
}

func (s Src) String() string {
	if s.File == "" {
		return "<none>"
	}
	return fmt.Sprintf("%s:%d", s.File, s.Line)
}

// String renders a cell the way a core listing would.
func (c Cell) String() string {
	var sb strings.Builder
	switch c.Kind {
	case Instr:
		sb.WriteString(c.Op.String())
		for i, o := range c.Ops {
			if i == 0 {
				sb.WriteByte(' ')
			} else {
				sb.WriteByte(',')
			}
			fmt.Fprintf(&sb, "%d", o)
		}
	case Return:
		fmt.Fprintf(&sb, "RETURN %d", c.A)
	default:
		fmt.Fprintf(&sb, "%d,%d,%d", c.A, c.F, c.V)
		if c.Ch != 0 {
			fmt.Fprintf(&sb, " %q", string(c.Ch))
		}
	}
	return sb.String()
}
