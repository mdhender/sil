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

// Package op is the instruction table: the 131 operations of S4D58 6,
// what each one takes, and what each one assembles.
//
// There is one table. The name, the section it is documented in, the
// operand signature, the classification of 7.5, how much room it takes
// and whether control can continue past it all live in the same entry,
// and everything that needs any of those reads it from here -- Lookup,
// String, the shape checker, the location counter, and later the
// emitter, the machine and the disassembler.
//
// One table, not several parallel ones. The sibling ML/I project keeps
// an opcode enum, a stringer and a lookup map in three hand-maintained
// files and its own notes call that a mistake; three lists of 131
// things drift.
//
// # What is here and what is not
//
// The signatures are transcribed from the box at the head of each
// entry in 6, which is where the document states an operation's
// operands. Reading an operation's prose and its programming notes is
// what implementing it requires, and happens then. Nothing in this
// package knows what any operation does.
//
// Slot records what an operand denotes. Only the distinctions that
// affect the shape of the operand field -- expression, character
// literal, parenthesised list -- are checked; the rest is what the
// operand is called in the document, which is what a diagnostic should
// say and what the emitter will read.
package op

import "fmt"

// Kind identifies one operation. It is the index of that operation's
// entry in the table, so Kind and table position cannot drift apart.
type Kind uint8

// Invalid is the zero Kind. It is not an operation.
const Invalid Kind = 0

// Slot says what an operand denotes.
type Slot uint8

const (
	SlotNone    Slot = iota
	SlotDescr        // the address of a descriptor (S4D58 3.1)
	SlotSpec         // the address of a specifier (3.2)
	SlotBranch       // a program location; omissible, meaning fall through (5.2)
	SlotProc         // a procedure entry point (6.78)
	SlotConst        // an assembly-time integer
	SlotExpr         // an assembly-time expression, address or integer
	SlotAddr         // an address assembled into a field
	SlotFlag         // a flag, or a sum of flags (3.1.2)
	SlotTable        // a syntax table (4.2)
	SlotKey          // a syntax table action: CONTIN, STOP, STOPSH or ERROR
	SlotFormat       // a location assembled by FORMAT (6.34)
	SlotList         // a list of operands, all of one kind (7.6d)
	SlotLiteral      // a character literal (7.6e)
	SlotSegment      // the name of a COPY segment (6.20)
)

var slotNames = [...]string{
	SlotNone:    "none",
	SlotDescr:   "descriptor",
	SlotSpec:    "specifier",
	SlotBranch:  "branch point",
	SlotProc:    "procedure",
	SlotConst:   "constant",
	SlotExpr:    "expression",
	SlotAddr:    "address",
	SlotFlag:    "flag",
	SlotTable:   "syntax table",
	SlotKey:     "syntax table action",
	SlotFormat:  "format",
	SlotList:    "list",
	SlotLiteral: "character literal",
	SlotSegment: "segment name",
}

func (s Slot) String() string {
	if int(s) < len(slotNames) {
		return slotNames[s]
	}
	return fmt.Sprintf("Slot(%d)", uint8(s))
}

// Operand is one position in an operation's operand list.
type Operand struct {
	Slot Slot
	Elem Slot   // the kind of the elements, when Slot is SlotList
	Name string // the name S4D58 6 gives it, used verbatim in diagnostics
	// Optional marks an operand that may be written as a null, or
	// dropped off the end of the list. Every branch point is optional
	// by 5.2; the rest are optional because their own section says so.
	Optional bool
}

// Size says how much room an operation assembles. Only the six
// data-assembling directives of 7.5 assemble anything that is not one
// address unit.
type Size uint8

const (
	SizeUnit   Size = iota // one address unit
	SizeNone               // nothing
	SizeDescr              // one descriptor
	SizeSpec               // one specifier
	SizeArray              // N descriptors, N being the operand (6.12)
	SizeBuffer             // N characters (6.17)
	SizeString             // a specifier, then the characters of the literal (6.117)
	SizeChars              // the characters of the literal (6.34)
)

// Category is an operation's classification in S4D58 7.5.
//
// Six operations are listed in two groups there; each carries the
// first group that lists it, in the document's order.
type Category uint8

const (
	CatAssembly   Category = iota // assembly control
	CatData                       // assemble data
	CatBranch                     // branch
	CatCompare                    // comparison
	CatStack                      // recursive procedures and stack management
	CatDescriptor                 // move and set descriptors
	CatAddress                    // modify address fields
	CatValue                      // modify value fields
	CatFlag                       // modify flag fields
	CatInteger                    // integer arithmetic on address fields
	CatReal                       // real numbers
	CatMoveSpec                   // move specifiers
	CatSpec                       // operate on specifiers
	CatSyntax                     // operate on syntax tables
	CatPattern                    // construct pattern nodes
	CatTree                       // operate on tree nodes
	CatIO                         // input and output
	CatSystem                     // depend on operating system facilities
	CatMisc                       // miscellaneous
)

var catNames = [...]string{
	CatAssembly:   "assembly control",
	CatData:       "assemble data",
	CatBranch:     "branch",
	CatCompare:    "comparison",
	CatStack:      "recursive procedures and stack management",
	CatDescriptor: "move and set descriptors",
	CatAddress:    "modify address fields",
	CatValue:      "modify value fields",
	CatFlag:       "modify flag fields",
	CatInteger:    "integer arithmetic on address fields",
	CatReal:       "real numbers",
	CatMoveSpec:   "move specifiers",
	CatSpec:       "operate on specifiers",
	CatSyntax:     "operate on syntax tables",
	CatPattern:    "construct pattern nodes",
	CatTree:       "operate on tree nodes",
	CatIO:         "input and output",
	CatSystem:     "depend on operating system facilities",
	CatMisc:       "miscellaneous",
}

func (c Category) String() string {
	if int(c) < len(catNames) {
		return catNames[c]
	}
	return fmt.Sprintf("Category(%d)", uint8(c))
}

// Entry is everything the table records about one operation.
type Entry struct {
	Mnemonic string
	Cat      Category
	Doc      string // the section of S4D58 that defines it
	Size     Size
	// Directive marks the twelve operations that are not executed:
	// the five assembly control macros and the six that assemble data
	// (7.5), and PROC, which 6.78 note 2 says may be implemented as
	// LHERE.
	Directive bool
	// Terminates marks an operation that always transfers control, so
	// nothing can reach the operation written after it by falling
	// through. RCALL and SELBRA are not among them: 6.87 note 6 and
	// 6.98 note 2 both let control return to the operation following.
	Terminates bool
	Operands   []Operand
}

// MaxArgs is how many operands the operation accepts.
func (e Entry) MaxArgs() int { return len(e.Operands) }

// MinArgs is how many operands must be written for every required one
// to be there. It is a count of positions rather than of required
// operands: RCALL and RRTURN each have an optional operand in front of
// a required one (6.87 note 3, 6.95 note 2), so a shorter list would
// drop the required one off the end.
func (e Entry) MinArgs() int {
	n := 0
	for i, o := range e.Operands {
		if !o.Optional {
			n = i + 1
		}
	}
	return n
}

// Branches reports whether any operand of the operation is a branch
// point, including the elements of a list of them.
func (e Entry) Branches() bool {
	for _, o := range e.Operands {
		if o.Slot == SlotBranch || (o.Slot == SlotList && o.Elem == SlotBranch) {
			return true
		}
	}
	return false
}

// Get returns the table entry for a Kind.
func Get(k Kind) Entry {
	if int(k) >= len(table) {
		return Entry{}
	}
	return table[k]
}

// String returns the mnemonic, or a placeholder for Invalid.
func (k Kind) String() string {
	if e := Get(k); e.Mnemonic != "" {
		return e.Mnemonic
	}
	return fmt.Sprintf("Kind(%d)", uint8(k))
}

// Count is how many operations the table holds: the 119 executable
// macros and the 12 directives of S4D58 7.5.
const Count = len(table) - 1

var byMnemonic = func() map[string]Kind {
	m := make(map[string]Kind, len(table))
	for i, e := range table {
		if e.Mnemonic != "" {
			m[e.Mnemonic] = Kind(i)
		}
	}
	return m
}()

// Lookup returns the Kind of a mnemonic, or Invalid.
func Lookup(mnemonic string) Kind { return byMnemonic[mnemonic] }

// Kinds returns every operation, in table order, which is the
// alphabetical order of S4D58 6.
func Kinds() []Kind {
	out := make([]Kind, 0, Count)
	for i := 1; i < len(table); i++ {
		out = append(out, Kind(i))
	}
	return out
}
