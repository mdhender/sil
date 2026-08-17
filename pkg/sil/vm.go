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

// Package sil is the SIL machine.
//
// Core is a slice of cells, one per address unit, and a cell is a
// descriptor (S4D58 3.1). The machine's own state is small: a program
// counter and the two stack pointers CSTACK and OSTACK, which 6.20
// allows to be registers rather than storage locations. Everything
// else -- the stack itself, every descriptor the program touches --
// lives in core, because the SNOBOL4 garbage collector walks it.
//
// The machine does not parse SIL source. It is handed a populated Core
// and a symbol table by the assembler in pkg/sil/asm.
package sil

import (
	"fmt"
	"io"

	"github.com/mdhender/sil/pkg/sil/op"
)

// VM is the abstract machine that SIL targets.
type VM struct {
	// Core is storage, one cell per address unit.
	Core []Cell

	// PC is the program counter.
	PC int

	// CStack and OStack are the current and old stack pointers
	// (S4D58 6.20). Only their address fields are ever used -- by
	// ISTACK, PUSH, POP, RCALL, RRTURN and PSTACK -- so they are plain
	// addresses here rather than descriptors. 6.20 allows either.
	CStack int
	OStack int

	// Symbols is what the assembly resolved every name to. ISTACK
	// needs STACK (6.50 note 1); the interrupt path needs INTR10
	// (7.3).
	Symbols map[string]int

	// Descr, Spec and CPA are the machine parameters PARMS chose. The
	// operations step the stack by Descr and walk print blocks by
	// Descr, so they have to be here and not only in the assembler.
	Descr int
	Spec  int
	CPA   int

	// Host is the outside world.
	Host Host

	// Dynamic is how many descriptors of dynamic storage INIT
	// allocates (6.46). Zero takes DefaultDynamic.
	Dynamic int

	// Halted is set by ENDEX, and Status is the value it read (6.29).
	Halted bool
	Status int

	// Cycles counts instructions executed. MaxCycles stops a runaway
	// program; zero means no limit. It is a field rather than a
	// constant so a test can set it low and a real run can set it
	// high.
	Cycles    int
	MaxCycles int

	// Trace, when set, receives one line per instruction. It is a
	// separate stream from anything the program prints.
	Trace io.Writer

	// The buffers S4D58 calls "local to" one operation: 6.49 note 2
	// for INTSPC, 6.89 note 3 for REALST and 6.22 note 2 for DATE.
	// Each is taken from past the end of the assembled image the first
	// time its operation runs; see VM.buffer.
	intBuf  scratch
	realBuf scratch
	dateBuf scratch
}

// scratch is a buffer that belongs to the machine rather than to the
// program: nothing in the SNOBOL4 source names one, and each entry
// that uses one says its contents survive only until the next use of
// that same operation.
//
// It is taken from past the end of the assembled image, so an image is
// exactly what the assembler laid out and a core listing is
// unaffected. Core is indexed by address and never by pointer, so
// growing it is invisible to everything already in it.
type scratch struct{ at, n int }

// buffer returns the address of a scratch buffer wide enough for n
// characters, taking or retaking it from the end of core.
func (s *VM) buffer(b *scratch, n int) int {
	if b.at == 0 || b.n < n {
		b.at, b.n = len(s.Core), n
		s.Core = append(s.Core, make([]Cell, n)...)
	}
	return b.at
}

// Fault is a failure of the machine rather than of the program it is
// running: a branch into data, an address outside core, an operation
// with no implementation.
//
// Conditions the SNOBOL4 system expects to handle itself -- stack
// underflow is the example S4D58 7.3 gives -- are not faults. They
// transfer to INTR10, which is a label in the program. See interrupt.
type Fault struct {
	PC  int
	Op  op.Kind
	Src Src
	Msg string
}

func (f *Fault) Error() string {
	where := f.Src.String()
	if f.Op != op.Invalid {
		return fmt.Sprintf("%s: %d: %s: %s", where, f.PC, f.Op, f.Msg)
	}
	return fmt.Sprintf("%s: %d: %s", where, f.PC, f.Msg)
}

// Interrupt is the label S4D58 7.3 says to transfer to when the
// implementation detects a condition that should not occur: "Transfer
// to the label INTR10 upon recognition of such an error causes the
// SNOBOL4 run to terminate with the message ERROR IN SNOBOL4 SYSTEM."
const Interrupt = "INTR10"

// Overflow is the label 6.80 note 1 sends a stack overflow to, "which
// will result in an appropriate error termination". Underflow goes to
// Interrupt instead (6.77 note 1); the two are different because one
// is a program that ran out of room and the other is a bug in the
// implementation of the macro language.
const Overflow = "OVER"

// Run executes until the program halts, faults, or runs out of cycles.
func (s *VM) Run() error {
	for !s.Halted {
		if err := s.Step(); err != nil {
			return err
		}
	}
	return nil
}

// Step executes one instruction.
//
// The program counter is advanced before the operation runs, so an
// operation that does not branch needs to do nothing about it and one
// that does simply assigns. Every branch point reaches the machine
// already resolved -- the assembler turns an omitted one into the
// address of the next operation (5.2) -- so there is no "fall through"
// case to write.
func (s *VM) Step() error {
	if s.Halted {
		return nil
	}
	if s.MaxCycles > 0 && s.Cycles >= s.MaxCycles {
		return s.fault("ran for %d cycles without halting", s.Cycles)
	}
	if s.PC < 0 || s.PC >= len(s.Core) {
		return s.fault("program counter outside core, which is %d units", len(s.Core))
	}

	c := s.Core[s.PC]
	if c.Kind != Instr {
		return s.fault("%s cell reached as an instruction: %s", c.Kind, c)
	}
	if s.Trace != nil {
		fmt.Fprintf(s.Trace, "%6d %-6s %-40s %s\n", s.PC, c.Op, c, c.Src.Text)
	}

	s.Cycles++
	s.PC++
	return s.execute(c)
}

// execute dispatches one instruction cell to the operation that
// implements it. Operands arrive already resolved, in the order the
// instruction table gives them.
func (s *VM) execute(c Cell) error {
	switch c.Op {
	case op.AEQL:
		s.AEQL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.AEQLC:
		s.AEQLC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.AEQLIC:
		return s.AEQLIC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.CHKVAL:
		s.CHKVAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4], c.Ops[5])
	case op.DEQL:
		s.DEQL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.LCOMP:
		s.LCOMP(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.LEQLC:
		s.LEQLC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.LEXCMP:
		s.LEXCMP(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.TESTF:
		s.TESTF(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.TESTFI:
		return s.TESTFI(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.VCMPIC:
		return s.VCMPIC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4], c.Ops[5])
	case op.VEQL:
		s.VEQL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.VEQLC:
		s.VEQLC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])

	case op.RESETF:
		s.RESETF(c.Ops[0], c.Ops[1])
	case op.RSETFI:
		return s.RSETFI(c.Ops[0], c.Ops[1])
	case op.SETF:
		s.SETF(c.Ops[0], c.Ops[1])
	case op.SETFI:
		return s.SETFI(c.Ops[0], c.Ops[1])

	case op.INCRV:
		s.INCRV(c.Ops[0], c.Ops[1])
	case op.MOVV:
		s.MOVV(c.Ops[0], c.Ops[1])
	case op.PUTVC:
		return s.PUTVC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.SETSIZ:
		return s.SETSIZ(c.Ops[0], c.Ops[1])
	case op.SETVA:
		s.SETVA(c.Ops[0], c.Ops[1])
	case op.SETVC:
		s.SETVC(c.Ops[0], c.Ops[1])

	case op.ADJUST:
		return s.ADJUST(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.BKSIZE:
		return s.BKSIZE(c.Ops[0], c.Ops[1])
	case op.DECRA:
		s.DECRA(c.Ops[0], c.Ops[1])
	case op.GETAC:
		return s.GETAC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.GETD:
		return s.GETD(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.GETDC:
		return s.GETDC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.GETLG:
		s.GETLG(c.Ops[0], c.Ops[1])
	case op.GETLTH:
		s.GETLTH(c.Ops[0], c.Ops[1])
	case op.GETSIZ:
		return s.GETSIZ(c.Ops[0], c.Ops[1])
	case op.INCRA:
		s.INCRA(c.Ops[0], c.Ops[1])
	case op.MOVA:
		s.MOVA(c.Ops[0], c.Ops[1])
	case op.MOVBLK:
		return s.MOVBLK(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.MOVDIC:
		return s.MOVDIC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.PUSH:
		return s.PUSH(c.Ops)
	case op.PUTAC:
		return s.PUTAC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.PUTD:
		return s.PUTD(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.PUTDC:
		return s.PUTDC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.SETAV:
		s.SETAV(c.Ops[0], c.Ops[1])
	case op.ZERBLK:
		return s.ZERBLK(c.Ops[0], c.Ops[1])

	case op.CLERTB:
		return s.CLERTB(c.Ops[0], c.Ops[1])
	case op.PLUGTB:
		return s.PLUGTB(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.STREAM:
		return s.STREAM(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4], c.Ops[5])

	case op.OUTPUT:
		return s.OUTPUT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3:])
	case op.STREAD:
		return s.STREAD(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.BKSPCE:
		return s.BKSPCE(c.Ops[0])
	case op.ENFILE:
		return s.ENFILE(c.Ops[0])
	case op.REWIND:
		return s.REWIND(c.Ops[0])

	case op.MSTIME:
		s.MSTIME(c.Ops[0])
	case op.DATE:
		s.DATE(c.Ops[0])
	case op.LOAD:
		return s.LOAD(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.LINK:
		return s.LINK(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4], c.Ops[5])
	case op.UNLOAD:
		s.UNLOAD(c.Ops[0])

	case op.LINKOR:
		return s.LINKOR(c.Ops[0], c.Ops[1])
	case op.LVALUE:
		return s.LVALUE(c.Ops[0], c.Ops[1])
	case op.LOCAPT:
		return s.LOCAPT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.LOCAPV:
		return s.LOCAPV(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.ORDVST:
		s.ORDVST()
	case op.RPLACE:
		return s.RPLACE(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.SPCINT:
		return s.SPCINT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.TOP:
		return s.TOP(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.VARID:
		return s.VARID(c.Ops[0], c.Ops[1])

	case op.PSTACK:
		s.PSTACK(c.Ops[0])
	case op.SPUSH:
		return s.SPUSH(c.Ops)
	case op.SPOP:
		return s.SPOP(c.Ops)
	case op.BRANIC:
		return s.BRANIC(c.Ops[0], c.Ops[1])
	case op.SELBRA:
		return s.SELBRA(c.Ops[0], c.Ops[1])

	case op.ADREAL:
		s.ADREAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.SBREAL:
		s.SBREAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.MPREAL:
		s.MPREAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.DVREAL:
		s.DVREAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.EXREAL:
		s.EXREAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.MNREAL:
		s.MNREAL(c.Ops[0], c.Ops[1])
	case op.RCOMP:
		s.RCOMP(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.INTRL:
		return s.INTRL(c.Ops[0], c.Ops[1])
	case op.RLINT:
		return s.RLINT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.REALST:
		s.REALST(c.Ops[0], c.Ops[1])
	case op.SPREAL:
		return s.SPREAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])

	case op.ADDSIB:
		return s.ADDSIB(c.Ops[0], c.Ops[1])
	case op.ADDSON:
		return s.ADDSON(c.Ops[0], c.Ops[1])
	case op.INSERT:
		return s.INSERT(c.Ops[0], c.Ops[1])
	case op.CPYPAT:
		return s.CPYPAT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4], c.Ops[5])
	case op.MAKNOD:
		return s.MAKNOD(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4], c.Ops[5])

	case op.ADDLG:
		s.ADDLG(c.Ops[0], c.Ops[1])
	case op.APDSP:
		return s.APDSP(c.Ops[0], c.Ops[1])
	case op.FSHRTN:
		s.FSHRTN(c.Ops[0], c.Ops[1])
	case op.GETBAL:
		return s.GETBAL(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	case op.GETSPC:
		return s.GETSPC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.INTSPC:
		s.INTSPC(c.Ops[0], c.Ops[1])
	case op.LOCSP:
		return s.LOCSP(c.Ops[0], c.Ops[1])
	case op.PUTLG:
		s.PUTLG(c.Ops[0], c.Ops[1])
	case op.PUTSPC:
		return s.PUTSPC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.REMSP:
		s.REMSP(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.SETLC:
		s.SETLC(c.Ops[0], c.Ops[1])
	case op.SETSP:
		s.SETSP(c.Ops[0], c.Ops[1])
	case op.SHORTN:
		s.SHORTN(c.Ops[0], c.Ops[1])
	case op.SUBSP:
		s.SUBSP(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.TRIMSP:
		return s.TRIMSP(c.Ops[0], c.Ops[1])

	case op.ACOMP:
		s.ACOMP(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.ACOMPC:
		s.ACOMPC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.BRANCH:
		s.BRANCH(c.Ops[0])
	case op.ENDEX:
		s.ENDEX(c.Ops[0])
	case op.INIT:
		return s.INIT()
	case op.ISTACK:
		return s.ISTACK()
	case op.MOVD:
		s.MOVD(c.Ops[0], c.Ops[1])
	case op.POP:
		return s.POP(c.Ops)
	case op.RCALL:
		return s.RCALL(c.Ops[0], c.Ops[1], c.Ops[2:])
	case op.RRTURN:
		return s.RRTURN(c.Ops[0], c.Ops[1])
	case op.SETAC:
		s.SETAC(c.Ops[0], c.Ops[1])
	case op.STPRNT:
		return s.STPRNT(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.SUM:
		s.SUM(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.SUBTRT:
		s.SUBTRT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.MULT:
		s.MULT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.MULTC:
		s.MULTC(c.Ops[0], c.Ops[1], c.Ops[2])
	case op.DIVIDE:
		s.DIVIDE(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.EXPINT:
		s.EXPINT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.MNSINT:
		s.MNSINT(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3])
	default:
		return s.fault("%s is not implemented", c.Op)
	}
	return nil
}

// fault reports a failure of the machine, citing the instruction that
// was executing.
func (s *VM) fault(format string, args ...any) error {
	f := &Fault{PC: s.PC, Msg: fmt.Sprintf(format, args...)}
	if s.PC >= 0 && s.PC < len(s.Core) {
		f.Op = s.Core[s.PC].Op
		f.Src = s.Core[s.PC].Src
	}
	return f
}

// interrupt handles a condition the SNOBOL4 system is written to
// recover from. S4D58 7.3 and 6.77 note 1 both say to transfer to
// INTR10 rather than to stop, so that is what happens when the program
// defines it. A program that does not -- a test program, or the
// vertical slice -- gets a fault instead, because there is nowhere to
// go.
func (s *VM) interrupt(format string, args ...any) error {
	return s.transfer(Interrupt, format, args...)
}

// overflow handles a stack overflow, which 6.80 note 1 sends to OVER
// rather than to INTR10.
func (s *VM) overflow(format string, args ...any) error {
	return s.transfer(Overflow, format, args...)
}

func (s *VM) transfer(label, format string, args ...any) error {
	if to, ok := s.Symbols[label]; ok {
		s.PC = to
		return nil
	}
	return s.fault("%s, and %s is not defined", fmt.Sprintf(format, args...), label)
}

// inCore reports whether an address is a cell of core.
func (s *VM) inCore(a int) bool { return a >= 0 && a < len(s.Core) }

// Specifier reads the two cells of a specifier (S4D58 3.2): the
// address, flag and value fields from the first, the offset and length
// from the address and value fields of the second.
func (s *VM) Specifier(a int) (addr, flag, value, offset, length int) {
	first, second := s.Core[a], s.Core[a+s.Descr]
	return first.A, first.F, first.V, second.A, second.V
}

// Chars reads n characters starting at an address.
//
// One character per cell, so this machine requires CPA = 1; the
// assembler refuses anything else. Packing characters would make a
// character address different from a cell address, which is the whole
// reason CPA exists (6.20).
func (s *VM) Chars(a, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = s.Core[a+i].Ch
	}
	return out
}

// Text returns the characters a specifier specifies: L of them, at the
// specifier's address plus its offset (3.2).
func (s *VM) Text(spec int) []byte {
	addr, _, _, offset, length := s.Specifier(spec)
	return s.Chars(addr+offset, length)
}
