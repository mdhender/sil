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

// The syntax tables: CLERTB and PLUGTB, which write them, and STREAM,
// which is driven by them.
//
// # What a table is
//
// One entry per character of the character set, at the position the
// character's internal code gives it (4.2), and an entry is a
// descriptor: 5.1 names its three fields A, T and P -- the next table
// address, the type indicator, and the put field -- in that order, so
// they are the address, flag and value fields.
//
// E, the address width of an entry (5.3), is DESCR on this machine,
// and ALPHSZ entries make a table; both choices are recorded in
// pkg/sil/copyseg/mdata.sil, where the tables are declared as
// ARRAY ALPHSZ.
//
// # The five actions, and why only three need code
//
// 4.2 lists six actions. CONTIN, GOTO(TABLE) and PUT(ADDRESS) are not
// indicators at all: CONTIN and GOTO both say which table processes
// the next character, which is the entry's address field either way --
// 6.19's figure for CLERTB CONTIN sets that field to the table itself,
// so "the current table" and "some other table" are one mechanism --
// and PUT is the value field, which STREAM carries along. Only STOP,
// STOPSH and ERROR are indicators that stop the streaming, and the
// fourth value of the indicator field is CONTIN, meaning "do not
// stop". PARMS chooses all four numbers (4.2), so the machine reads
// them out of the assembly rather than naming them.

// syntax is the four indicator values PARMS chooses and the two other
// program symbols the table operations need.
type syntax struct {
	contin, stop, stopsh, error int
	alphsz                      int
}

func (s *VM) syntax(of string) (syntax, error) {
	var t syntax
	for _, f := range []struct {
		name string
		at   *int
	}{
		{"CONTIN", &t.contin},
		{"STOP", &t.stop},
		{"STOPSH", &t.stopsh},
		{"ERROR", &t.error},
		{"ALPHSZ", &t.alphsz},
	} {
		v, ok := s.Symbols[f.name]
		if !ok {
			return t, s.fault("%s: %s is not defined; PARMS supplies it", of, f.name)
		}
		*f.at = v
	}
	if t.alphsz <= 0 {
		return t, s.fault("%s: ALPHSZ is %d", of, t.alphsz)
	}
	return t, nil
}

// setEntry writes one syntax table entry the way CLERTB and PLUGTB
// both write it: for CONTIN, the address field becomes the table and
// the indicator is cleared; otherwise only the indicator changes. The
// put field is never touched by either.
func (s *VM) setEntry(at, table, key int, t syntax) {
	if key == t.contin {
		s.Core[at].A, s.Core[at].F = table, 0
		return
	}
	s.Core[at].F = key
}

// CLERTB (clear syntax table) is used to set the indicator fields of
// all entries of a syntax table to a constant. KEY may be one of four
// values: CONTIN, ERROR, STOP, STOPSH. The indicator field of each
// entry of TABLE is set to T where T is the indicator that corresponds
// to the value of KEY.
//
// Data Altered for ERROR, STOP, or STOPSH:
//
//	TABLE     T
//	...
//	TABLE+Z*E T
//
// Data Altered for CONTIN:
//
//	TABLE     TABLE,0
//	...
//	TABLE+Z*E TABLE,0
//
// Programming Notes:
//  1. See Section 4.2.
//  2. See also PLUGTB.
//
// Z is "the number of the last character in collating sequence" (5.3),
// and characters are numbered from 0 to Z, so a table is Z+1 entries
// and that is ALPHSZ.
//
// S4D58.PDF: 6.19
func (s *VM) CLERTB(table, key int) error {
	t, err := s.syntax("CLERTB")
	if err != nil {
		return err
	}
	if !s.inCore(table) || !s.inCore(table+(t.alphsz-1)*s.Descr) {
		return s.fault("CLERTB: the table at %d does not fit in core", table)
	}
	for c := 0; c < t.alphsz; c++ {
		s.setEntry(table+c*s.Descr, table, key, t)
	}
	return nil
}

// PLUGTB (plug syntax table) is used to set selected indicator fields
// in the entries of a syntax table to a constant. KEY may be one of
// four values: CONTIN, ERROR, STOP, STOPSH. The indicator fields of
// entries corresponding to C1,...,CL are set to T where T is the
// indicator that corresponds to the value of KEY.
//
// Data Input:
//
//	SPEC A,O,L
//	A+O  C1...CL
//
// Data Altered for ERROR, STOP, or STOPSH:
//
//	TABLE+E*C1 T
//	...
//	TABLE+E*CL T
//
// Data Altered for CONTIN:
//
//	TABLE+E*C1 TABLE,0
//	...
//	TABLE+E*CL TABLE,0
//
// Programming Notes:
//  1. See Section 4.2.
//  2. See also CLERTB.
//
// S4D58.PDF: 6.76
func (s *VM) PLUGTB(table, key, spec int) error {
	t, err := s.syntax("PLUGTB")
	if err != nil {
		return err
	}
	addr, _, _, offset, length := s.Specifier(spec)
	if !s.charsInCore(addr+offset, length) {
		return s.fault("PLUGTB: %d characters at %d are outside core", length, addr+offset)
	}
	for _, c := range s.Chars(addr+offset, length) {
		at := table + int(c)*s.Descr
		if !s.inCore(at) {
			return s.fault("PLUGTB: the entry for character %d is at %d, outside core", c, at)
		}
		s.setEntry(at, table, key, t)
	}
	return nil
}

// STREAM (stream for token) is used to locate a syntactic token at the
// beginning of the string specified by SPEC2. If there is an I
// (1 <= I <= L) such that TI is ERROR, STOP, or STOPSH, and J is the
// least such I, then if TJ is ERROR, transfer is to ERROR, while if TJ
// is STOP or STOPSH, transfer is to SLOC. Otherwise transfer is to
// RUNOUT.
//
// In the figures that follow, J is the least value of I for which TI
// is STOP or STOPSH. P is the last value of P (1 <= I <= J) that is
// nonzero (i.e. for which a PUT is specified in the syntax table
// description for the tables given). If no PUT is specified, P is
// zero.
//
// Data Input:
//
//	SPEC2      A,F,V,O,L
//	A+O        C1...CJ,CJ+1...CL
//	TABLE+E*C1 A2,T1,P1
//	A2+E*C2    A3,T2,P2
//	...
//	AL+E*CL    TL,PL
//
// Data Altered if Termination is STOP:
//
//	STYPE P
//	SPEC1 A,F,V,O,J
//	SPEC2 A,F,V,O+J,L-J
//
// Data Altered if Termination is STOPSH:
//
//	STYPE P
//	SPEC1 A,F,V,O,J-1
//	SPEC2 A,F,V,O+J-1,L-J+1
//
// Data Altered if Termination is ERROR:
//
//	STYPE 0
//	SPEC1 A,F,V,O,L
//
// Data Altered if Termination is RUNOUT:
//
//	STYPE P
//	SPEC1 A,F,V,O,L
//	SPEC2 A,F,V,O,0
//
// Programming Notes:
//  1. Termination with STOP or STOPSH may occur on the last character,
//     CL.
//  2. If L = 0 (i.e. if SPEC2 specifies the null string), RUNOUT
//     occurs. In this case the address field of STYPE should be set to
//     0.
//  3. See Section 4.2.
//
// # A dropped clause
//
// 6.116's own sentence reads "if TJ is ERROR, transfer is to ERRROR,
// while if if TJ is STOPSH, transfer is to SLOC. Otherwise transfer is
// to RUNOUT", which leaves STOP with nowhere to go and makes RUNOUT
// mean two different things. The sentence before it defines J over
// "ERROR, STOP, or STOPSH" and the figures give STOP its own arm, so
// the clause reads "STOP or STOPSH" and RUNOUT is what happens when no
// such J exists -- the string ran out. Note 2 confirms it from the
// edge: the null string, which has no characters to stop on, is
// RUNOUT.
//
// STYPE is a program symbol, the descriptor 4.2's PUT action names.
// The SNOBOL4 source defines it at line 5572 as DESCR 0,FNC,0, and
// only its address field is written here, so the flag survives.
//
// S4D58.PDF: 6.116
func (s *VM) STREAM(spec1, spec2, table, errloc, runout, sloc int) error {
	t, err := s.syntax("STREAM")
	if err != nil {
		return err
	}
	stype, ok := s.Symbols["STYPE"]
	if !ok {
		return s.fault("STREAM: the program symbol STYPE is not defined")
	}

	addr, flag, value, offset, length := s.Specifier(spec2)
	if !s.charsInCore(addr+offset, length) {
		return s.fault("STREAM: %d characters at %d are outside core", length, addr+offset)
	}

	put, at := 0, table
	for i := 0; i < length; i++ {
		entry := at + int(s.Core[addr+offset+i].Ch)*s.Descr
		if !s.inCore(entry) {
			return s.fault("STREAM: the entry at %d is outside core", entry)
		}
		e := s.Core[entry]
		if e.V != 0 {
			put = e.V
		}

		j := i + 1
		switch e.F {
		case t.error:
			s.Core[stype].A = 0
			s.putSpecifier(spec1, addr, flag, value, offset, length)
			s.PC = errloc
			return nil
		case t.stop:
			s.Core[stype].A = put
			s.putSpecifier(spec1, addr, flag, value, offset, j)
			s.putSpecifier(spec2, addr, flag, value, offset+j, length-j)
			s.PC = sloc
			return nil
		case t.stopsh:
			s.Core[stype].A = put
			s.putSpecifier(spec1, addr, flag, value, offset, j-1)
			s.putSpecifier(spec2, addr, flag, value, offset+j-1, length-j+1)
			s.PC = sloc
			return nil
		}
		// CONTIN and GOTO(TABLE) are the same thing: the entry's
		// address field is the table that processes the next
		// character.
		at = e.A
	}

	// No J: the string ran out. Note 2's null string is this case with
	// nothing examined, so P is still zero.
	s.Core[stype].A = put
	s.putSpecifier(spec1, addr, flag, value, offset, length)
	s.putSpecifier(spec2, addr, flag, value, offset, 0)
	s.PC = runout
	return nil
}
