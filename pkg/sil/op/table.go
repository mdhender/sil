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

// The operations, in the alphabetical order of S4D58 6. The ordinal of
// an operation in that order is the number of the section that defines
// it, which is what Doc records and what a test checks.
const (
	// ACOMP is the first operation of S4D58 6, and Invalid takes
	// the zero Kind so that a missing entry is never a valid one.
	ACOMP Kind = iota + 1
	ACOMPC
	ADDLG
	ADDSIB
	ADDSON
	ADJUST
	ADREAL
	AEQL
	AEQLC
	AEQLIC
	APDSP
	ARRAY
	BKSIZE
	BKSPCE
	BRANCH
	BRANIC
	BUFFER
	CHKVAL
	CLERTB
	COPY
	CPYPAT
	DATE
	DECRA
	DEQL
	DESCR
	DIVIDE
	DVREAL
	END
	ENDEX
	ENFILE
	EQU
	EXPINT
	EXREAL
	FORMAT
	FSHRTN
	GETAC
	GETBAL
	GETD
	GETDC
	GETLG
	GETLTH
	GETSIZ
	GETSPC
	INCRA
	INCRV
	INIT
	INSERT
	INTRL
	INTSPC
	ISTACK
	LCOMP
	LEQLC
	LEXCMP
	LHERE
	LINK
	LINKOR
	LOAD
	LOCAPT
	LOCAPV
	LOCSP
	LVALUE
	MAKNOD
	MNREAL
	MNSINT
	MOVA
	MOVBLK
	MOVD
	MOVDIC
	MOVV
	MPREAL
	MSTIME
	MULT
	MULTC
	ORDVST
	OUTPUT
	PLUGTB
	POP
	PROC
	PSTACK
	PUSH
	PUTAC
	PUTD
	PUTDC
	PUTLG
	PUTSPC
	PUTVC
	RCALL
	RCOMP
	REALST
	REMSP
	RESETF
	REWIND
	RLINT
	RPLACE
	RRTURN
	RSETFI
	SBREAL
	SELBRA
	SETAC
	SETAV
	SETF
	SETFI
	SETLC
	SETSIZ
	SETSP
	SETVA
	SETVC
	SHORTN
	SPCINT
	SPEC
	SPOP
	SPREAL
	SPUSH
	STPRNT
	STREAD
	STREAM
	STRING
	SUBSP
	SUBTRT
	SUM
	TESTF
	TESTFI
	TITLE
	TOP
	TRIMSP
	UNLOAD
	VARID
	VCMPIC
	VEQL
	VEQLC
	ZERBLK
)

// table is indexed by Kind. Entry i describes the operation named by
// the constant whose value is i, so the enum and the table cannot get
// out of step: adding a constant without an entry leaves a hole that
// the table test finds.
var table = [...]Entry{
	// ACOMP -- address comparison
	ACOMP: {Mnemonic: "ACOMP", Cat: CatCompare, Doc: "S4D58 6.1", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// ACOMPC -- address comparison with constant
	ACOMPC: {Mnemonic: "ACOMPC", Cat: CatCompare, Doc: "S4D58 6.2", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// ADDLG -- add to specifier length
	ADDLG: {Mnemonic: "ADDLG", Cat: CatSpec, Doc: "S4D58 6.3", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// ADDSIB -- add sibling to tree node
	ADDSIB: {Mnemonic: "ADDSIB", Cat: CatTree, Doc: "S4D58 6.4", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// ADDSON -- add son to tree node
	ADDSON: {Mnemonic: "ADDSON", Cat: CatTree, Doc: "S4D58 6.5", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// ADJUST -- compute adjusted address
	ADJUST: {Mnemonic: "ADJUST", Cat: CatAddress, Doc: "S4D58 6.6", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
	}},
	// ADREAL -- add real numbers
	ADREAL: {Mnemonic: "ADREAL", Cat: CatReal, Doc: "S4D58 6.7", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// AEQL -- addresses equal test
	AEQL: {Mnemonic: "AEQL", Cat: CatCompare, Doc: "S4D58 6.8", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// AEQLC -- address equal to constant test
	AEQLC: {Mnemonic: "AEQLC", Cat: CatCompare, Doc: "S4D58 6.9", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// AEQLIC -- address equal to constant indirect test
	AEQLIC: {Mnemonic: "AEQLIC", Cat: CatCompare, Doc: "S4D58 6.10", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N1"},
		{Slot: SlotConst, Name: "N2"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// APDSP -- append specifier
	APDSP: {Mnemonic: "APDSP", Cat: CatSpec, Doc: "S4D58 6.11", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
	}},
	// ARRAY -- assemble array of descriptors
	ARRAY: {Mnemonic: "ARRAY", Cat: CatData, Doc: "S4D58 6.12", Size: SizeArray, Directive: true, Operands: []Operand{
		{Slot: SlotConst, Name: "N"},
	}},
	// BKSIZE -- get block size
	BKSIZE: {Mnemonic: "BKSIZE", Cat: CatAddress, Doc: "S4D58 6.13", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// BKSPCE -- backspace record
	BKSPCE: {Mnemonic: "BKSPCE", Cat: CatIO, Doc: "S4D58 6.14", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// BRANCH -- branch to program location
	BRANCH: {Mnemonic: "BRANCH", Cat: CatBranch, Doc: "S4D58 6.15", Terminates: true, Operands: []Operand{
		{Slot: SlotBranch, Name: "LOC"},
		{Slot: SlotProc, Name: "PROC", Optional: true},
	}},
	// BRANIC -- branch indirect with offset constant
	BRANIC: {Mnemonic: "BRANIC", Cat: CatBranch, Doc: "S4D58 6.16", Terminates: true, Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// BUFFER -- assemble buffer of blank characters
	BUFFER: {Mnemonic: "BUFFER", Cat: CatData, Doc: "S4D58 6.17", Size: SizeBuffer, Directive: true, Operands: []Operand{
		{Slot: SlotConst, Name: "N"},
	}},
	// CHKVAL -- check value
	CHKVAL: {Mnemonic: "CHKVAL", Cat: CatCompare, Doc: "S4D58 6.18", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// CLERTB -- clear syntax table
	CLERTB: {Mnemonic: "CLERTB", Cat: CatSyntax, Doc: "S4D58 6.19", Operands: []Operand{
		{Slot: SlotTable, Name: "TABLE"},
		{Slot: SlotKey, Name: "KEY"},
	}},
	// COPY -- copy file into assembly
	COPY: {Mnemonic: "COPY", Cat: CatAssembly, Doc: "S4D58 6.20", Size: SizeNone, Directive: true, Operands: []Operand{
		{Slot: SlotSegment, Name: "FILE"},
	}},
	// CPYPAT -- copy pattern
	CPYPAT: {Mnemonic: "CPYPAT", Cat: CatPattern, Doc: "S4D58 6.21", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotDescr, Name: "DESCR4"},
		{Slot: SlotDescr, Name: "DESCR5"},
		{Slot: SlotDescr, Name: "DESCR6"},
	}},
	// DATE -- get date
	DATE: {Mnemonic: "DATE", Cat: CatSystem, Doc: "S4D58 6.22", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// DECRA -- decrement address
	DECRA: {Mnemonic: "DECRA", Cat: CatAddress, Doc: "S4D58 6.23", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// DEQL -- descriptor equal test
	DEQL: {Mnemonic: "DEQL", Cat: CatCompare, Doc: "S4D58 6.24", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// DESCR -- assemble descriptor
	DESCR: {Mnemonic: "DESCR", Cat: CatData, Doc: "S4D58 6.25", Size: SizeDescr, Directive: true, Operands: []Operand{
		{Slot: SlotAddr, Name: "A", Optional: true},
		{Slot: SlotFlag, Name: "F", Optional: true},
		{Slot: SlotConst, Name: "V", Optional: true},
	}},
	// DIVIDE -- divide integers
	DIVIDE: {Mnemonic: "DIVIDE", Cat: CatInteger, Doc: "S4D58 6.26", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// DVREAL -- divide real numbers
	DVREAL: {Mnemonic: "DVREAL", Cat: CatReal, Doc: "S4D58 6.27", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// END -- end assembly
	END: {Mnemonic: "END", Cat: CatAssembly, Doc: "S4D58 6.28", Size: SizeNone, Directive: true, Operands: []Operand{}},
	// ENDEX -- end execution of SNOBOL4 run
	ENDEX: {Mnemonic: "ENDEX", Cat: CatSystem, Doc: "S4D58 6.29", Terminates: true, Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// ENFILE -- write end of file
	ENFILE: {Mnemonic: "ENFILE", Cat: CatIO, Doc: "S4D58 6.30", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// EQU -- symbol equivalence
	EQU: {Mnemonic: "EQU", Cat: CatAssembly, Doc: "S4D58 6.31", Size: SizeNone, Directive: true, Operands: []Operand{
		{Slot: SlotExpr, Name: "N"},
	}},
	// EXPINT -- exponentiate integers
	EXPINT: {Mnemonic: "EXPINT", Cat: CatInteger, Doc: "S4D58 6.32", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// EXREAL -- exponentiate real numbers
	EXREAL: {Mnemonic: "EXREAL", Cat: CatReal, Doc: "S4D58 6.33", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// FORMAT -- assemble format string
	FORMAT: {Mnemonic: "FORMAT", Cat: CatData, Doc: "S4D58 6.34", Size: SizeChars, Directive: true, Operands: []Operand{
		{Slot: SlotLiteral, Name: "C1...CL"},
	}},
	// FSHRTN -- foreshorten specifier
	FSHRTN: {Mnemonic: "FSHRTN", Cat: CatSpec, Doc: "S4D58 6.35", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotConst, Name: "N"},
	}},
	// GETAC -- get address with offset constant
	GETAC: {Mnemonic: "GETAC", Cat: CatAddress, Doc: "S4D58 6.36", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotConst, Name: "N"},
	}},
	// GETBAL -- get parenthesis balanced string
	GETBAL: {Mnemonic: "GETBAL", Cat: CatSpec, Doc: "S4D58 6.37", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// GETD -- get descriptor
	GETD: {Mnemonic: "GETD", Cat: CatDescriptor, Doc: "S4D58 6.38", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
	}},
	// GETDC -- get descriptor with offset constant
	GETDC: {Mnemonic: "GETDC", Cat: CatDescriptor, Doc: "S4D58 6.39", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotConst, Name: "N"},
	}},
	// GETLG -- get length of specifier
	GETLG: {Mnemonic: "GETLG", Cat: CatAddress, Doc: "S4D58 6.40", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// GETLTH -- get length for string structure
	GETLTH: {Mnemonic: "GETLTH", Cat: CatAddress, Doc: "S4D58 6.41", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// GETSIZ -- get size
	GETSIZ: {Mnemonic: "GETSIZ", Cat: CatAddress, Doc: "S4D58 6.42", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// GETSPC -- get specifier with constant offset
	GETSPC: {Mnemonic: "GETSPC", Cat: CatMoveSpec, Doc: "S4D58 6.43", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// INCRA -- increment address
	INCRA: {Mnemonic: "INCRA", Cat: CatAddress, Doc: "S4D58 6.44", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// INCRV -- increment value field
	INCRV: {Mnemonic: "INCRV", Cat: CatValue, Doc: "S4D58 6.45", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// INIT -- initialize SNOBOL4 run
	INIT: {Mnemonic: "INIT", Cat: CatSystem, Doc: "S4D58 6.46", Operands: []Operand{}},
	// INSERT -- insert node in tree
	INSERT: {Mnemonic: "INSERT", Cat: CatTree, Doc: "S4D58 6.47", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// INTRL -- convert integer to real number
	INTRL: {Mnemonic: "INTRL", Cat: CatReal, Doc: "S4D58 6.48", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// INTSPC -- convert integer to specifier
	INTSPC: {Mnemonic: "INTSPC", Cat: CatSpec, Doc: "S4D58 6.49", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// ISTACK -- initialize stack
	ISTACK: {Mnemonic: "ISTACK", Cat: CatStack, Doc: "S4D58 6.50", Operands: []Operand{}},
	// LCOMP -- length comparison
	LCOMP: {Mnemonic: "LCOMP", Cat: CatCompare, Doc: "S4D58 6.51", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// LEQLC -- length equal to constant test
	LEQLC: {Mnemonic: "LEQLC", Cat: CatCompare, Doc: "S4D58 6.52", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// LEXCMP -- lexical comparison of strings
	LEXCMP: {Mnemonic: "LEXCMP", Cat: CatCompare, Doc: "S4D58 6.53", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// LHERE -- location here
	LHERE: {Mnemonic: "LHERE", Cat: CatAssembly, Doc: "S4D58 6.54", Size: SizeNone, Directive: true, Operands: []Operand{}},
	// LINK -- link to external function
	LINK: {Mnemonic: "LINK", Cat: CatSystem, Doc: "S4D58 6.55", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotDescr, Name: "DESCR4"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// LINKOR -- link ‘‘‘or’’’ fields of pattern nodes
	LINKOR: {Mnemonic: "LINKOR", Cat: CatMisc, Doc: "S4D58 6.56", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// LOAD -- load external function
	LOAD: {Mnemonic: "LOAD", Cat: CatSystem, Doc: "S4D58 6.57", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// LOCAPT -- locate attribute pair by type
	LOCAPT: {Mnemonic: "LOCAPT", Cat: CatMisc, Doc: "S4D58 6.58", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// LOCAPV -- locate attribute pair by value
	LOCAPV: {Mnemonic: "LOCAPV", Cat: CatMisc, Doc: "S4D58 6.59", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// LOCSP -- locate specifier to string
	LOCSP: {Mnemonic: "LOCSP", Cat: CatSpec, Doc: "S4D58 6.60", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// LVALUE -- get least length value
	LVALUE: {Mnemonic: "LVALUE", Cat: CatMisc, Doc: "S4D58 6.61", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// MAKNOD -- make pattern node
	// 6.62: "DESCR6 may be omitted. If it is, one less descriptor is
	// modified, but the two forms are otherwise the same."
	MAKNOD: {Mnemonic: "MAKNOD", Cat: CatPattern, Doc: "S4D58 6.62", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotDescr, Name: "DESCR4"},
		{Slot: SlotDescr, Name: "DESCR5"},
		{Slot: SlotDescr, Name: "DESCR6", Optional: true},
	}},
	// MNREAL -- minus real number
	MNREAL: {Mnemonic: "MNREAL", Cat: CatReal, Doc: "S4D58 6.63", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// MNSINT -- minus integer
	MNSINT: {Mnemonic: "MNSINT", Cat: CatInteger, Doc: "S4D58 6.64", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// MOVA -- move address
	MOVA: {Mnemonic: "MOVA", Cat: CatAddress, Doc: "S4D58 6.65", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// MOVBLK -- move block of descriptors
	MOVBLK: {Mnemonic: "MOVBLK", Cat: CatDescriptor, Doc: "S4D58 6.66", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
	}},
	// MOVD -- move descriptor
	MOVD: {Mnemonic: "MOVD", Cat: CatDescriptor, Doc: "S4D58 6.67", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// MOVDIC -- move descriptor indirect with constant offset
	MOVDIC: {Mnemonic: "MOVDIC", Cat: CatDescriptor, Doc: "S4D58 6.68", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotConst, Name: "N1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotConst, Name: "N2"},
	}},
	// MOVV -- move value field
	MOVV: {Mnemonic: "MOVV", Cat: CatValue, Doc: "S4D58 6.69", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// MPREAL -- multiply real numbers
	MPREAL: {Mnemonic: "MPREAL", Cat: CatReal, Doc: "S4D58 6.70", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// MSTIME -- get millisecond time
	MSTIME: {Mnemonic: "MSTIME", Cat: CatSystem, Doc: "S4D58 6.71", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// MULT -- multiply integers
	MULT: {Mnemonic: "MULT", Cat: CatInteger, Doc: "S4D58 6.72", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// MULTC -- multiply address by constant
	MULTC: {Mnemonic: "MULTC", Cat: CatInteger, Doc: "S4D58 6.73", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotConst, Name: "N"},
	}},
	// ORDVST -- order variable storage
	ORDVST: {Mnemonic: "ORDVST", Cat: CatMisc, Doc: "S4D58 6.74", Operands: []Operand{}},
	// OUTPUT -- output record
	// 6.75 does not say the list may be omitted, but N = 0 is what an
	// omitted list means, and the source writes it that way fifteen
	// times, always with a format of pure literals.
	OUTPUT: {Mnemonic: "OUTPUT", Cat: CatIO, Doc: "S4D58 6.75", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFormat, Name: "FORMAT"},
		{Slot: SlotList, Elem: SlotDescr, Name: "DESCRn", Optional: true},
	}},
	// PLUGTB -- plug syntax table
	PLUGTB: {Mnemonic: "PLUGTB", Cat: CatSyntax, Doc: "S4D58 6.76", Operands: []Operand{
		{Slot: SlotTable, Name: "TABLE"},
		{Slot: SlotKey, Name: "KEY"},
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// POP -- pop descriptors from stack
	POP: {Mnemonic: "POP", Cat: CatStack, Doc: "S4D58 6.77", Operands: []Operand{
		{Slot: SlotList, Elem: SlotDescr, Name: "DESCRn"},
	}},
	// PROC -- procedure entry
	PROC: {Mnemonic: "PROC", Cat: CatStack, Doc: "S4D58 6.78", Size: SizeNone, Directive: true, Operands: []Operand{
		{Slot: SlotProc, Name: "LOC2", Optional: true},
	}},
	// PSTACK -- post stack position
	PSTACK: {Mnemonic: "PSTACK", Cat: CatStack, Doc: "S4D58 6.79", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// PUSH -- push descriptors onto stack
	PUSH: {Mnemonic: "PUSH", Cat: CatStack, Doc: "S4D58 6.80", Operands: []Operand{
		{Slot: SlotList, Elem: SlotDescr, Name: "DESCRn"},
	}},
	// PUTAC -- put address with offset constant
	PUTAC: {Mnemonic: "PUTAC", Cat: CatAddress, Doc: "S4D58 6.81", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// PUTD -- put descriptor
	PUTD: {Mnemonic: "PUTD", Cat: CatDescriptor, Doc: "S4D58 6.82", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
	}},
	// PUTDC -- put descriptor with constant offset
	PUTDC: {Mnemonic: "PUTDC", Cat: CatDescriptor, Doc: "S4D58 6.83", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// PUTLG -- put specifier length
	PUTLG: {Mnemonic: "PUTLG", Cat: CatSpec, Doc: "S4D58 6.84", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// PUTSPC -- put specifier with offset constant
	PUTSPC: {Mnemonic: "PUTSPC", Cat: CatMoveSpec, Doc: "S4D58 6.85", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// PUTVC -- put value field with offset constant
	PUTVC: {Mnemonic: "PUTVC", Cat: CatValue, Doc: "S4D58 6.86", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// RCALL -- recursive call
	RCALL: {Mnemonic: "RCALL", Cat: CatStack, Doc: "S4D58 6.87", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR", Optional: true},
		{Slot: SlotProc, Name: "PROC"},
		{Slot: SlotList, Elem: SlotDescr, Name: "DESCRn", Optional: true},
		{Slot: SlotList, Elem: SlotBranch, Name: "LOCm", Optional: true},
	}},
	// RCOMP -- real comparison
	RCOMP: {Mnemonic: "RCOMP", Cat: CatCompare, Doc: "S4D58 6.88", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// REALST -- convert real number to string
	REALST: {Mnemonic: "REALST", Cat: CatReal, Doc: "S4D58 6.89", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// REMSP -- specify remaining string
	REMSP: {Mnemonic: "REMSP", Cat: CatSpec, Doc: "S4D58 6.90", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotSpec, Name: "SPEC3"},
	}},
	// RESETF -- reset flag
	RESETF: {Mnemonic: "RESETF", Cat: CatFlag, Doc: "S4D58 6.91", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFlag, Name: "FLAG"},
	}},
	// REWIND -- rewind file
	REWIND: {Mnemonic: "REWIND", Cat: CatIO, Doc: "S4D58 6.92", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
	}},
	// RLINT -- convert real number to integer
	RLINT: {Mnemonic: "RLINT", Cat: CatReal, Doc: "S4D58 6.93", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// RPLACE -- replace characters
	RPLACE: {Mnemonic: "RPLACE", Cat: CatMisc, Doc: "S4D58 6.94", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotSpec, Name: "SPEC3"},
	}},
	// RRTURN -- recursive return
	// 6.95 note 2: DESCR may be omitted, in which case no value is
	// returned.
	RRTURN: {Mnemonic: "RRTURN", Cat: CatStack, Doc: "S4D58 6.95", Terminates: true, Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR", Optional: true},
		{Slot: SlotConst, Name: "N"},
	}},
	// RSETFI -- reset flag indirect
	RSETFI: {Mnemonic: "RSETFI", Cat: CatFlag, Doc: "S4D58 6.96", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFlag, Name: "FLAG"},
	}},
	// SBREAL -- subtract real numbers
	SBREAL: {Mnemonic: "SBREAL", Cat: CatReal, Doc: "S4D58 6.97", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// SELBRA -- select branch point
	SELBRA: {Mnemonic: "SELBRA", Cat: CatBranch, Doc: "S4D58 6.98", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotList, Elem: SlotBranch, Name: "LOCn", Optional: true},
	}},
	// SETAC -- set address to constant
	SETAC: {Mnemonic: "SETAC", Cat: CatAddress, Doc: "S4D58 6.99", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// SETAV -- set address from value field
	SETAV: {Mnemonic: "SETAV", Cat: CatAddress, Doc: "S4D58 6.100", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// SETF -- set flag
	SETF: {Mnemonic: "SETF", Cat: CatFlag, Doc: "S4D58 6.101", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFlag, Name: "FLAG"},
	}},
	// SETFI -- set flag indirect
	SETFI: {Mnemonic: "SETFI", Cat: CatFlag, Doc: "S4D58 6.102", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFlag, Name: "FLAG"},
	}},
	// SETLC -- set length of specifier to constant
	SETLC: {Mnemonic: "SETLC", Cat: CatSpec, Doc: "S4D58 6.103", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotConst, Name: "N"},
	}},
	// SETSIZ -- set size
	SETSIZ: {Mnemonic: "SETSIZ", Cat: CatValue, Doc: "S4D58 6.104", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// SETSP -- set specifier
	SETSP: {Mnemonic: "SETSP", Cat: CatMoveSpec, Doc: "S4D58 6.105", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
	}},
	// SETVA -- set value field from address
	SETVA: {Mnemonic: "SETVA", Cat: CatValue, Doc: "S4D58 6.106", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
	// SETVC -- set value to constant
	SETVC: {Mnemonic: "SETVC", Cat: CatValue, Doc: "S4D58 6.107", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
	}},
	// SHORTN -- shorten specifier
	SHORTN: {Mnemonic: "SHORTN", Cat: CatSpec, Doc: "S4D58 6.108", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotConst, Name: "N"},
	}},
	// SPCINT -- convert specifier to integer
	SPCINT: {Mnemonic: "SPCINT", Cat: CatMisc, Doc: "S4D58 6.109", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// SPEC -- assemble specifier
	SPEC: {Mnemonic: "SPEC", Cat: CatData, Doc: "S4D58 6.110", Size: SizeSpec, Directive: true, Operands: []Operand{
		{Slot: SlotAddr, Name: "A", Optional: true},
		{Slot: SlotFlag, Name: "F", Optional: true},
		{Slot: SlotConst, Name: "V", Optional: true},
		{Slot: SlotConst, Name: "O", Optional: true},
		{Slot: SlotConst, Name: "L", Optional: true},
	}},
	// SPOP -- pop specifier from stack
	SPOP: {Mnemonic: "SPOP", Cat: CatStack, Doc: "S4D58 6.111", Operands: []Operand{
		{Slot: SlotList, Elem: SlotSpec, Name: "SPECn"},
	}},
	// SPREAL -- convert specified string to real number
	SPREAL: {Mnemonic: "SPREAL", Cat: CatReal, Doc: "S4D58 6.112", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// SPUSH -- push specifiers onto stack
	SPUSH: {Mnemonic: "SPUSH", Cat: CatStack, Doc: "S4D58 6.113", Operands: []Operand{
		{Slot: SlotList, Elem: SlotSpec, Name: "SPECn"},
	}},
	// STPRNT -- string print
	STPRNT: {Mnemonic: "STPRNT", Cat: CatIO, Doc: "S4D58 6.114", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// STREAD -- string read
	STREAD: {Mnemonic: "STREAD", Cat: CatIO, Doc: "S4D58 6.115", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotBranch, Name: "EOF", Optional: true},
		{Slot: SlotBranch, Name: "ERROR", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// STREAM -- stream for token
	STREAM: {Mnemonic: "STREAM", Cat: CatSpec, Doc: "S4D58 6.116", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotTable, Name: "TABLE"},
		{Slot: SlotBranch, Name: "ERROR", Optional: true},
		{Slot: SlotBranch, Name: "RUNOUT", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// STRING -- assemble specified string
	STRING: {Mnemonic: "STRING", Cat: CatData, Doc: "S4D58 6.117", Size: SizeString, Directive: true, Operands: []Operand{
		{Slot: SlotLiteral, Name: "C1...CL"},
	}},
	// SUBSP -- substring specification
	SUBSP: {Mnemonic: "SUBSP", Cat: CatSpec, Doc: "S4D58 6.118", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
		{Slot: SlotSpec, Name: "SPEC3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// SUBTRT -- subtract addresses
	SUBTRT: {Mnemonic: "SUBTRT", Cat: CatInteger, Doc: "S4D58 6.119", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// SUM -- sum addresses
	SUM: {Mnemonic: "SUM", Cat: CatInteger, Doc: "S4D58 6.120", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// TESTF -- test flag
	TESTF: {Mnemonic: "TESTF", Cat: CatCompare, Doc: "S4D58 6.121", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFlag, Name: "FLAG"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// TESTFI -- test flag indirect
	TESTFI: {Mnemonic: "TESTFI", Cat: CatCompare, Doc: "S4D58 6.122", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotFlag, Name: "FLAG"},
		{Slot: SlotBranch, Name: "FLOC", Optional: true},
		{Slot: SlotBranch, Name: "SLOC", Optional: true},
	}},
	// TITLE -- title assembly listing
	TITLE: {Mnemonic: "TITLE", Cat: CatAssembly, Doc: "S4D58 6.123", Size: SizeNone, Directive: true, Operands: []Operand{
		{Slot: SlotLiteral, Name: "C1...CN"},
	}},
	// TOP -- get to top of block
	TOP: {Mnemonic: "TOP", Cat: CatMisc, Doc: "S4D58 6.124", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotDescr, Name: "DESCR3"},
	}},
	// TRIMSP -- trim blanks from specifier
	TRIMSP: {Mnemonic: "TRIMSP", Cat: CatSpec, Doc: "S4D58 6.125", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC1"},
		{Slot: SlotSpec, Name: "SPEC2"},
	}},
	// UNLOAD -- unload external function
	UNLOAD: {Mnemonic: "UNLOAD", Cat: CatSystem, Doc: "S4D58 6.126", Operands: []Operand{
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// VARID -- compute variable identification numbers
	VARID: {Mnemonic: "VARID", Cat: CatMisc, Doc: "S4D58 6.127", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotSpec, Name: "SPEC"},
	}},
	// VCMPIC -- value field compare indirect with offset constant
	VCMPIC: {Mnemonic: "VCMPIC", Cat: CatCompare, Doc: "S4D58 6.128", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "GTLOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
		{Slot: SlotBranch, Name: "LTLOC", Optional: true},
	}},
	// VEQL -- value fields equal test
	VEQL: {Mnemonic: "VEQL", Cat: CatCompare, Doc: "S4D58 6.129", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// VEQLC -- value field equal to constant test
	VEQLC: {Mnemonic: "VEQLC", Cat: CatCompare, Doc: "S4D58 6.130", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR"},
		{Slot: SlotConst, Name: "N"},
		{Slot: SlotBranch, Name: "NELOC", Optional: true},
		{Slot: SlotBranch, Name: "EQLOC", Optional: true},
	}},
	// ZERBLK -- zero block
	ZERBLK: {Mnemonic: "ZERBLK", Cat: CatDescriptor, Doc: "S4D58 6.131", Operands: []Operand{
		{Slot: SlotDescr, Name: "DESCR1"},
		{Slot: SlotDescr, Name: "DESCR2"},
	}},
}
