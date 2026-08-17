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

import "testing"

// A code node is five descriptors: the node itself and the four the
// source names at lines 233 to 236. The tests use the source's own
// offsets, because 6.4 note 2 says the program chooses them.
const (
	father = 1
	lson   = 2
	rsib   = 3
	code   = 4

	np1 = 1 // three descriptors that point at nodes
	np2 = 2
	np3 = 3
	ndA = 16 // four nodes
	ndB = 24
	ndC = 32
	ndD = 40
	src = 56 // a pattern to copy, and where to copy it
	dst = 80

	nodeCore = 128
)

func nodeMachine() *VM {
	s := machine()
	s.Core = make([]Cell, nodeCore)
	s.Symbols["FATHER"] = father
	s.Symbols["LSON"] = lson
	s.Symbols["RSIB"] = rsib
	s.Symbols["CODE"] = code
	return s
}

// full reads a whole descriptor, which is what these operations move.
func (s *VM) full(a int) [3]int { return [3]int{s.Core[a].A, s.Core[a].F, s.Core[a].V} }

func checkNodes(t *testing.T, s *VM, want map[int][3]int) {
	t.Helper()
	for at, w := range want {
		if got := s.full(at); got != w {
			t.Errorf("%d is %v, want %v", at, got, w)
		}
	}
}

// ADDSON (6.5) makes the new node the left son and the old left son
// its right sibling.
func TestADDSON(t *testing.T) {
	s := nodeMachine()
	s.set(np1, ndA, 1, 2)
	s.set(np2, ndB, 3, 4)
	s.set(ndA+lson, ndC, 5, 6)
	s.set(ndA+code, 90, 91, 7)

	if err := s.ADDSON(np1, np2); err != nil {
		t.Fatalf("ADDSON: %v", err)
	}
	checkNodes(t, s, map[int][3]int{
		ndB + father: {ndA, 1, 2},
		ndB + rsib:   {ndC, 5, 6},
		ndA + lson:   {ndB, 3, 4},
		// Only the value field of the CODE descriptor is named in
		// the figure, so only it changes.
		ndA + code: {90, 91, 8},
	})
}

// ADDSIB (6.4) links the new node in after DESCR1, taking DESCR1's
// father and its old right sibling.
func TestADDSIB(t *testing.T) {
	s := nodeMachine()
	s.set(np1, ndA, 0, 0)
	s.set(np2, ndB, 3, 4)
	s.set(ndA+father, ndC, 5, 6)
	s.set(ndA+rsib, ndD, 7, 8)
	s.set(ndC+code, 90, 91, 7)

	if err := s.ADDSIB(np1, np2); err != nil {
		t.Fatalf("ADDSIB: %v", err)
	}
	checkNodes(t, s, map[int][3]int{
		ndB + rsib:   {ndD, 7, 8},
		ndB + father: {ndC, 5, 6},
		ndA + rsib:   {ndB, 3, 4},
		ndC + code:   {90, 91, 8},
	})
}

// INSERT (6.47) puts the new node above DESCR1: it takes DESCR1's
// father, DESCR1 becomes its left son, and the father's left son
// points back at it.
func TestINSERT(t *testing.T) {
	s := nodeMachine()
	s.set(np1, ndA, 1, 2)
	s.set(np2, ndB, 3, 4)
	s.set(ndA+father, ndC, 5, 6)
	s.set(ndC+lson, ndD, 7, 8)
	s.set(ndB+code, 90, 91, 7)

	if err := s.INSERT(np1, np2); err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	checkNodes(t, s, map[int][3]int{
		ndA + father: {ndB, 3, 4},
		ndD + rsib:   {ndB, 3, 4},
		ndB + father: {ndC, 5, 6},
		ndB + lson:   {ndA, 1, 2},
		ndB + code:   {90, 91, 8},
	})
}

// The four offsets are the program's, so an assembly that does not
// define them cannot run a tree operation (6.4 note 2).
func TestTreeOperationsNeedTheProgramsOffsets(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"ADDSIB": func(s *VM) error { return s.ADDSIB(np1, np2) },
		"ADDSON": func(s *VM) error { return s.ADDSON(np1, np2) },
		"INSERT": func(s *VM) error { return s.INSERT(np1, np2) },
	} {
		s := nodeMachine()
		delete(s.Symbols, "RSIB")
		if err := run(s); err == nil {
			t.Errorf("%s: no error with RSIB undefined", name)
		}
	}
}

// A node the program points outside core is a fault, not a panic.
func TestTreeOperationsCheckTheirAddresses(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"ADDSIB": func(s *VM) error { return s.ADDSIB(np1, np2) },
		"ADDSON": func(s *VM) error { return s.ADDSON(np1, np2) },
		"INSERT": func(s *VM) error { return s.INSERT(np1, np2) },
		"MAKNOD": func(s *VM) error { return s.MAKNOD(np3, np1, np3, np3, np3, 0) },
		"CPYPAT": func(s *VM) error { return s.CPYPAT(np1, np1, np3, np3, np3, np3) },
	} {
		s := nodeMachine()
		s.set(np1, nodeCore+9, 0, 0)
		s.set(np2, nodeCore+9, 0, 0)
		if err := run(s); err == nil {
			t.Errorf("%s: no error reaching outside core", name)
		}
	}
}

// MAKNOD (6.62) fills a pattern node. A2+2D and A2+3D take an address
// field and nothing else.
func TestMAKNOD(t *testing.T) {
	s := nodeMachine()
	s.set(np2, ndA, 3, 4)
	s.set(np3, 111, 0, 0)
	s.set(ndB, 222, 0, 0)   // DESCR4
	s.set(ndB+1, 333, 5, 6) // DESCR5
	s.set(ndB+2, 444, 7, 8) // DESCR6
	s.set(ndA+2, 1, 9, 10)  // what A2+2D was carrying
	s.set(ndA+3, 2, 11, 12) // and A2+3D

	if err := s.MAKNOD(np1, np2, np3, ndB, ndB+1, ndB+2); err != nil {
		t.Fatalf("MAKNOD: %v", err)
	}
	checkNodes(t, s, map[int][3]int{
		np1:     {ndA, 3, 4},
		ndA + 1: {333, 5, 6},
		ndA + 2: {222, 9, 10},
		ndA + 3: {111, 11, 12},
		ndA + 4: {444, 7, 8},
	})
}

// Note 1: with DESCR6 omitted, one less descriptor is modified.
func TestMAKNODWithoutDESCR6(t *testing.T) {
	s := nodeMachine()
	s.set(np2, ndA, 3, 4)
	s.set(np3, 111, 0, 0)
	s.set(ndB, 222, 0, 0)
	s.set(ndB+1, 333, 5, 6)
	s.set(ndA+4, 99, 98, 97)

	if err := s.MAKNOD(np1, np2, np3, ndB, ndB+1, 0); err != nil {
		t.Fatalf("MAKNOD: %v", err)
	}
	if got, want := s.full(ndA+4), [3]int{99, 98, 97}; got != want {
		t.Errorf("A2+4D is %v, want %v; it is not altered", got, want)
	}
}

// Note 2: DESCR1 must be changed last, since DESCR6 may be the same
// descriptor. The node gets what DESCR1 held before, and DESCR1 gets
// the node.
func TestMAKNODWithDESCR6TheSameAsDESCR1(t *testing.T) {
	s := nodeMachine()
	s.set(np1, 55, 56, 57)
	s.set(np2, ndA, 3, 4)
	s.set(np3, 111, 0, 0)
	s.set(ndB, 222, 0, 0)
	s.set(ndB+1, 333, 5, 6)

	if err := s.MAKNOD(np1, np2, np3, ndB, ndB+1, np1); err != nil {
		t.Fatalf("MAKNOD: %v", err)
	}
	checkNodes(t, s, map[int][3]int{
		ndA + 4: {55, 56, 57},
		np1:     {ndA, 3, 4},
	})
}

// CPYPAT (6.21) copies a pattern section by section, relocating the
// links as it goes. This one has two sections, the first a
// three-descriptor node and the second a four-descriptor one, which
// is the case the document's two figure headings disagree about.
func TestCPYPAT(t *testing.T) {
	const (
		a3 = 100 // added to both fields of the third descriptor
		a4 = 10  // added by F1 and F2 to a nonzero field
		a5 = 77  // what F2 gives for a zero field
	)
	s := nodeMachine()
	s.set(np1, dst, 0, 0)
	s.set(np2, src, 0, 0)
	s.set(ndA, a3, 0, 0) // DESCR3
	s.set(ndB, a4, 0, 0) // DESCR4
	s.set(ndC, a5, 0, 0) // DESCR5
	s.set(ndD, 6, 0, 0)  // DESCR6: three descriptors, then one more

	// The first section: a node of size 2, so three descriptors.
	s.set(src+1, 5, 6, 2)
	s.set(src+2, 3, 0, 0)
	s.set(src+3, 4, 0, 5)
	// The second: a node of size 3, so four, including R2+4D.
	s.set(src+4, 8, 9, 3)
	s.set(src+5, 0, 0, 7)
	s.set(src+6, 0, 0, 0)
	s.set(src+7, 12, 13, 14)

	if err := s.CPYPAT(np1, np2, ndA, ndB, ndC, ndD); err != nil {
		t.Fatalf("CPYPAT: %v", err)
	}
	checkNodes(t, s, map[int][3]int{
		dst + 1: {5, 6, 2},       // copied whole
		dst + 2: {3 + a4, 0, a5}, // F1(3), F2(0)
		dst + 3: {4 + a3, 0, 5 + a3},
		dst + 4: {8, 9, 3},
		dst + 5: {0, 0, 7 + a4}, // F1(0), F2(7)
		dst + 6: {a3, 0, a3},
		dst + 7: {12, 13, 14}, // only because V7 is 3
	})
	// Three descriptors then four, and the final R1 goes back into
	// the address field of DESCR1.
	if got, want := s.Core[np1].A, dst+7; got != want {
		t.Errorf("DESCR1 addresses %d, want %d", got, want)
	}
}

// 6.21 tests R3 after copying a section, so one section is always
// copied however small A6 is.
func TestCPYPATAlwaysCopiesOneSection(t *testing.T) {
	s := nodeMachine()
	s.set(np1, dst, 0, 0)
	s.set(np2, src, 0, 0)
	s.set(ndD, 0, 0, 0) // DESCR6 is zero
	s.set(src+1, 5, 6, 2)

	if err := s.CPYPAT(np1, np2, ndA, ndB, ndC, ndD); err != nil {
		t.Fatalf("CPYPAT: %v", err)
	}
	if got, want := s.full(dst+1), [3]int{5, 6, 2}; got != want {
		t.Errorf("R1+D is %v, want %v", got, want)
	}
	if got, want := s.Core[np1].A, dst+3; got != want {
		t.Errorf("DESCR1 addresses %d, want %d", got, want)
	}
}
