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
	case op.ACOMP:
		s.ACOMP(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.ACOMPC:
		s.ACOMPC(c.Ops[0], c.Ops[1], c.Ops[2], c.Ops[3], c.Ops[4])
	case op.BRANCH:
		s.BRANCH(c.Ops[0])
	case op.ENDEX:
		s.ENDEX(c.Ops[0])
	case op.INIT:
		s.INIT()
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
	if to, ok := s.Symbols[Interrupt]; ok {
		s.PC = to
		return nil
	}
	return s.fault("%s, and %s is not defined", fmt.Sprintf(format, args...), Interrupt)
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
