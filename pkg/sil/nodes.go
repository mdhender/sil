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

// The tree-node operations of S4D58 7.5 and the two that build
// patterns.
//
// # Where a field is left blank
//
// These entries name some fields of an altered descriptor and leave
// others blank, and the difference matters here more than anywhere
// else: ADDSIB alters only the value field of the descriptor at
// A3+CODE, and MAKNOD alters only the address fields of two of the
// four descriptors it writes. A blank is not a zero. 6.100 shows what
// the document does when it means zero -- SETAV's figure reads
// "V 0 0", with the zeros written -- so a field the figure does not
// name is a field the operation does not touch.
//
// # The offsets are the program's
//
// 6.4 note 2, 6.5 note 2 and 6.47 note 3 all say that FATHER, LSON,
// RSIB and CODE "are symbols defined in the source program", so the
// machine reads them out of the assembly rather than choosing them.
// The SNOBOL4 source sets them at lines 233 to 236, as DESCR, 2*DESCR,
// 3*DESCR and 4*DESCR.

// nodeLayout is the four offsets a code node is laid out by.
type nodeLayout struct{ father, lson, rsib, code int }

func (s *VM) nodeLayout(of string) (nodeLayout, error) {
	var n nodeLayout
	for _, f := range []struct {
		name string
		at   *int
	}{
		{"FATHER", &n.father},
		{"LSON", &n.lson},
		{"RSIB", &n.rsib},
		{"CODE", &n.code},
	} {
		v, ok := s.Symbols[f.name]
		if !ok {
			return n, s.fault("%s: %s is not defined, and a code node is laid out by it", of, f.name)
		}
		*f.at = v
	}
	return n, nil
}

// node reads a descriptor of a code node, checking that it is in core
// first so that a walk off the end of a tree reports where it was
// rather than stopping the machine somewhere else.
func (s *VM) node(of string, at int) (Cell, error) {
	if !s.inCore(at) {
		return Cell{}, s.fault("%s: %d is outside core", of, at)
	}
	return descriptorAt(s.Core, at), nil
}

// putNode writes a whole descriptor of a code node.
func (s *VM) putNode(of string, at int, c Cell) error {
	if !s.inCore(at) {
		return s.fault("%s: %d is outside core", of, at)
	}
	s.Core[at] = Cell{Kind: Data, A: c.A, F: c.F, V: c.V}
	return nil
}

// bumpCode adds one to the value field of the descriptor at the CODE
// offset of a node, which is the one thing all three tree operations
// do and the one field of it they touch.
func (s *VM) bumpCode(of string, at int) error {
	if !s.inCore(at) {
		return s.fault("%s: %d is outside core", of, at)
	}
	s.Core[at].V++
	return nil
}

// ADDSIB (add sibling to tree node) is used to add a tree node as a
// sibling to another node.
//
// Data Input:
//
//	DESCR1    A1
//	DESCR2    A2,F2,V2
//	A1+FATHER A3,F3,V3
//	A1+RSIB   A4,F4,V4
//	A3+CODE   I
//
// Data Altered:
//
//	A2+RSIB   A4,F4,V4
//	A2+FATHER A3,F3,V3
//	A1+RSIB   A2,F2,V2
//	A3+CODE   I+1
//
// Programming Notes:
//  1. ADDSIB is only used by compilation procedures.
//  2. FATHER, RSIB, and CODE are symbols defined in the source
//     program.
//  3. See also ADDSON and INSERT.
//
// A1+RSIB is both read and written, so everything is read first --
// the same care 6.47 note 1 asks for by name.
//
// S4D58.PDF: 6.4
func (s *VM) ADDSIB(descr1, descr2 int) error {
	n, err := s.nodeLayout("ADDSIB")
	if err != nil {
		return err
	}
	a1, new2 := s.Core[descr1].A, descriptorAt(s.Core, descr2)
	a2 := new2.A

	father, err := s.node("ADDSIB", a1+n.father)
	if err != nil {
		return err
	}
	rsib, err := s.node("ADDSIB", a1+n.rsib)
	if err != nil {
		return err
	}

	if err := s.putNode("ADDSIB", a2+n.rsib, rsib); err != nil {
		return err
	}
	if err := s.putNode("ADDSIB", a2+n.father, father); err != nil {
		return err
	}
	if err := s.putNode("ADDSIB", a1+n.rsib, new2); err != nil {
		return err
	}
	return s.bumpCode("ADDSIB", father.A+n.code)
}

// ADDSON (add son to tree node) is used to add a tree node as a son to
// another node.
//
// Data Input:
//
//	DESCR1  A1,F1,V1
//	DESCR2  A2,F2,V2
//	A1+LSON A3,F3,V3
//	A1+CODE I
//
// Data Altered:
//
//	A2+FATHER A1,F1,V1
//	A2+RSIB   A3,F3,V3
//	A1+LSON   A2,F2,V2
//	A1+CODE   I+1
//
// Programming Notes:
//  1. ADDSON is only used by compilation procedures.
//  2. FATHER, LSON, RSIB, and CODE are symbols defined in the source
//     program.
//  3. See also ADDSIB and INSERT.
//
// The new node takes the place of the left son and the old left son
// becomes its right sibling, so A1+LSON is read before it is written.
//
// S4D58.PDF: 6.5
func (s *VM) ADDSON(descr1, descr2 int) error {
	n, err := s.nodeLayout("ADDSON")
	if err != nil {
		return err
	}
	up, new2 := descriptorAt(s.Core, descr1), descriptorAt(s.Core, descr2)
	a1, a2 := up.A, new2.A

	lson, err := s.node("ADDSON", a1+n.lson)
	if err != nil {
		return err
	}

	if err := s.putNode("ADDSON", a2+n.father, up); err != nil {
		return err
	}
	if err := s.putNode("ADDSON", a2+n.rsib, lson); err != nil {
		return err
	}
	if err := s.putNode("ADDSON", a1+n.lson, new2); err != nil {
		return err
	}
	return s.bumpCode("ADDSON", a1+n.code)
}

// INSERT (insert node in tree) is used to insert a tree node above
// another node.
//
// Data Input:
//
//	DESCR1    A1,F1,V1
//	DESCR2    A2,F2,V2
//	A1+FATHER A3,F3,V3
//	A3+LSON   A4,F4,V4
//	A2+CODE   I
//
// Data Altered:
//
//	A1+FATHER A2,F2,V2
//	A4+RSIB   A2,F2,V2
//	A2+FATHER A3,F3,V3
//	A2+LSON   A1,F1,V1
//	A2+CODE   I+1
//
// Programming Notes:
//  1. Since the fields of the descriptor at A1+FATHER are used in the
//     data to be altered, care should be taken not to modify this
//     descriptor until its former values have been used.
//  2. INSERT is only used by compilation procedures.
//  3. FATHER, LSON, RSIB, and CODE are symbols defined in the source
//     program.
//  4. See also ADDSIB and ADDSON.
//
// S4D58.PDF: 6.47
func (s *VM) INSERT(descr1, descr2 int) error {
	n, err := s.nodeLayout("INSERT")
	if err != nil {
		return err
	}
	below, new2 := descriptorAt(s.Core, descr1), descriptorAt(s.Core, descr2)
	a1, a2 := below.A, new2.A

	// Note 1: A1+FATHER is altered, and A3 comes out of it.
	father, err := s.node("INSERT", a1+n.father)
	if err != nil {
		return err
	}
	lson, err := s.node("INSERT", father.A+n.lson)
	if err != nil {
		return err
	}

	if err := s.putNode("INSERT", a1+n.father, new2); err != nil {
		return err
	}
	if err := s.putNode("INSERT", lson.A+n.rsib, new2); err != nil {
		return err
	}
	if err := s.putNode("INSERT", a2+n.father, father); err != nil {
		return err
	}
	if err := s.putNode("INSERT", a2+n.lson, below); err != nil {
		return err
	}
	return s.bumpCode("INSERT", a2+n.code)
}

// MAKNOD (make pattern node) is used to make a node for a pattern.
// DESCR6 may be omitted. If it is, one less descriptor is modified,
// but the two forms are otherwise the same.
//
// Data Input:
//
//	DESCR2 A2,F2,V2
//	DESCR3 A3
//	DESCR4 A4
//	DESCR5 A5,F5,V5
//
// Additional Data Input if DESCR6 is Given:
//
//	DESCR6 A6,F6,V6
//
// Data Altered:
//
//	DESCR1 A2,F2,V2
//	A2+D   A5,F5,V5
//	A2+2D  A4
//	A2+3D  A3
//
// Additional Data Altered if DESCR6 is Given:
//
//	A2+4D A6,F6,V6
//
// Programming Notes:
//  1. As indicated, there are two forms of MAKNOD. If DESCR6 is given,
//     an additional descriptor is modified, but otherwise the two
//     forms are the same.
//  2. DESCR1 must be changed last, since DESCR6 may be the same
//     descriptor as DESCR1.
//  3. MAKNOD is used only for constructing patterns.
//
// A2+2D and A2+3D take an address field only. The figure names no
// other field of either, and 6.100 shows that this document writes a
// zero when it means one; the value fields there are the pattern's own
// links, which CPYPAT reads back as V8 and V9.
//
// An omitted DESCR6 arrives as address zero, which is how the
// assembler renders every omitted descriptor operand -- see 6.87 note
// 3, where RCALL's omitted DESCR does the same thing.
//
// S4D58.PDF: 6.62
func (s *VM) MAKNOD(descr1, descr2, descr3, descr4, descr5, descr6 int) error {
	node := descriptorAt(s.Core, descr2)
	a2, d := node.A, s.Descr

	last := 3 * d
	if descr6 != 0 {
		last = 4 * d
	}
	if !s.inCore(a2+d) || !s.inCore(a2+last) {
		return s.fault("MAKNOD: the node at %d does not fit in core", a2)
	}

	s.Core[a2+d] = descriptorAt(s.Core, descr5)
	s.Core[a2+2*d].A = s.Core[descr4].A
	s.Core[a2+3*d].A = s.Core[descr3].A
	if descr6 != 0 {
		s.Core[a2+4*d] = descriptorAt(s.Core, descr6)
	}
	// Note 2: DESCR1 last, because DESCR6 may be the same descriptor.
	s.Core[descr1] = node
	return nil
}

// CPYPAT (copy pattern) is used to copy a pattern. First set
//
//	R1 = A1
//	R2 = A2
//	R3 = A6
//
// where R1, R2, and R3 are temporary locations. Sections of the
// pattern are copied for successive values of R1 and R2. After copying
// each section, set
//
//	R3 = R3-(1+V7)*D
//
// Then set
//
//	R1 = R1+(1+V7)*D
//	R2 = R2+(1+V7)*D
//
// If R3 > 0, continue, copying the next section. Otherwise the
// operation is complete. The final value of R1 is inserted in the
// address field of DESCR1.
//
// The functions F1 and F2 are defined as follows:
//
//	F1(X) = 0 if X = 0
//	F1(X) = X+A4 otherwise
//
//	F2(X) = A5 if X = 0
//	F2(X) = X+A4 otherwise
//
// Initial Data Input:
//
//	DESCR1 A1
//	DESCR2 A2
//	DESCR3 A3
//	DESCR4 A4
//	DESCR5 A5
//	DESCR6 A6
//
// Data Input for Successive Values of R2:
//
//	R2+D  A7,F7,V7
//	R2+2D A8,0,V8
//	R2+3D A9,0,V9
//
// Data Altered for Successive Values of R1:
//
//	R1+D  A7,F7,V7
//	R1+2D F1(A8),0,F2(V8)
//	R1+3D A9+A3,0,V9+A3
//
// Additional Data Input for Successive Values of R2 if V7 = 3:
//
//	R2+4D A10,F10,V10
//
// Additional Data Altered for Successive Values of R1 if V7 = 3:
//
//	R1+4D A10,F10,V10
//
// Data Altered when Copying is Complete:
//
//	DESCR1 R1
//
// # A transposition in the figure headings
//
// The document guards the input of R2+4D with "if V7 = 3" and the
// output of R1+4D with "if V3 = 7". Only one of those can be right,
// since the two figures describe the two halves of one copy, and the
// source says which: V7 is the value field of what MAKNOD wrote at
// A2+D from its DESCR5, and the value field of that operand is the
// node's size. The three-descriptor nodes are built from NMECL, which
// line 5521 gives a value of 2; the four-descriptor ones from ATOPCL
// and CHRCL, which lines 5501 and 5504 give a value of 3, and those
// are exactly the MAKNOD calls that pass a DESCR6. So the condition is
// V7 = 3 on both sides, and the section stride (1+V7)*D -- 3*D for a
// three-descriptor node and 4*D for a four-descriptor one -- says the
// same thing a second way. V3 is never read at all.
//
// The loop is a do-while: 6.21 tests R3 after copying a section, so
// one section is always copied however small A6 is.
//
// S4D58.PDF: 6.21
func (s *VM) CPYPAT(descr1, descr2, descr3, descr4, descr5, descr6 int) error {
	d := s.Descr
	a3, a4, a5 := s.Core[descr3].A, s.Core[descr4].A, s.Core[descr5].A
	r1, r2, r3 := s.Core[descr1].A, s.Core[descr2].A, s.Core[descr6].A

	f1 := func(x int) int {
		if x == 0 {
			return 0
		}
		return x + a4
	}
	f2 := func(x int) int {
		if x == 0 {
			return a5
		}
		return x + a4
	}

	for {
		from, err := s.node("CPYPAT", r2+d)
		if err != nil {
			return err
		}
		second, err := s.node("CPYPAT", r2+2*d)
		if err != nil {
			return err
		}
		third, err := s.node("CPYPAT", r2+3*d)
		if err != nil {
			return err
		}

		if err := s.putNode("CPYPAT", r1+d, from); err != nil {
			return err
		}
		if err := s.putNode("CPYPAT", r1+2*d, Cell{A: f1(second.A), V: f2(second.V)}); err != nil {
			return err
		}
		if err := s.putNode("CPYPAT", r1+3*d, Cell{A: third.A + a3, V: third.V + a3}); err != nil {
			return err
		}
		if from.V == 3 {
			fourth, err := s.node("CPYPAT", r2+4*d)
			if err != nil {
				return err
			}
			if err := s.putNode("CPYPAT", r1+4*d, fourth); err != nil {
				return err
			}
		}

		stride := (1 + from.V) * d
		if stride <= 0 {
			return s.fault("CPYPAT: a section of %d units would not advance", stride)
		}
		r3 -= stride
		r1 += stride
		r2 += stride
		if r3 <= 0 {
			break
		}
	}

	s.Core[descr1].A = r1
	return nil
}
