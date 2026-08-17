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

// The specifier group works on more operands at once than the
// descriptor group does, so it gets its own addresses: three
// specifiers, two scratch descriptors, and three character regions far
// enough apart that an append cannot reach the next one.
const (
	sp1 = 1 // three specifiers, two cells each
	sp2 = 3
	sp3 = 5
	sd1 = 7 // two scratch descriptors
	sd2 = 8
	sc1 = 16 // three character regions
	sc2 = 32
	sc3 = 48

	specStack = 60 // a system stack, for SPUSH and SPOP
	specCore  = 96
	specFail  = 80 // FLOC and SLOC, outside anything else
	specOK    = 81
)

func specMachine() *VM {
	s := machine()
	s.Core = make([]Cell, specCore)
	return s
}

// fullSpec writes all five fields of a specifier.
func (s *VM) fullSpec(at, addr, flag, value, offset, length int) {
	s.Core[at] = Cell{Kind: Data, A: addr, F: flag, V: value}
	s.Core[at+1] = Cell{Kind: Data, A: offset, V: length}
}

// readSpec reads all five, in the order the document draws them.
func (s *VM) readSpec(at int) [5]int {
	a, f, v, o, l := s.Specifier(at)
	return [5]int{a, f, v, o, l}
}

// The operations that alter one field of a specifier and leave the
// other four alone. Each case starts from the same specifier so that
// "leaves the others alone" is checked by the same assertion that
// checks the change.
func TestSpecifierFields(t *testing.T) {
	const (
		addr   = sc1
		flag   = 3
		value  = 9
		offset = 2
		length = 5
	)
	for _, tt := range []struct {
		name string
		run  func(*VM)
		want [5]int
	}{
		// 6.3: L becomes L+I, I from the address field of DESCR.
		{"ADDLG", func(s *VM) { s.ADDLG(sp1, sd1) }, [5]int{addr, flag, value, offset, length + 4}},
		// 6.35: O becomes O+N and L becomes L-N.
		{"FSHRTN", func(s *VM) { s.FSHRTN(sp1, 3) }, [5]int{addr, flag, value, offset + 3, length - 3}},
		// 6.84: L becomes I.
		{"PUTLG", func(s *VM) { s.PUTLG(sp1, sd1) }, [5]int{addr, flag, value, offset, 4}},
		// 6.103: L becomes N.
		{"SETLC", func(s *VM) { s.SETLC(sp1, 0) }, [5]int{addr, flag, value, offset, 0}},
		// 6.108: L becomes L-N.
		{"SHORTN", func(s *VM) { s.SHORTN(sp1, 2) }, [5]int{addr, flag, value, offset, length - 2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := specMachine()
			s.fullSpec(sp1, addr, flag, value, offset, length)
			s.set(sd1, 4, 0, 0)
			s.PC = 0
			tt.run(s)
			if got := s.readSpec(sp1); got != tt.want {
				t.Errorf("SPEC is %v, want %v", got, tt.want)
			}
			if s.PC != 0 {
				t.Errorf("PC is %d; %s does not branch", s.PC, tt.name)
			}
		})
	}
}

// SETSP copies all five fields (6.105); REMSP builds a remainder
// (6.90); GETSPC and PUTSPC move a specifier through an address field
// (6.43, 6.85).
func TestSpecifierMoves(t *testing.T) {
	t.Run("SETSP", func(t *testing.T) {
		s := specMachine()
		s.fullSpec(sp1, 0, 0, 0, 0, 0)
		s.fullSpec(sp2, sc1, 3, 9, 2, 5)
		s.SETSP(sp1, sp2)
		if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 2, 5}; got != want {
			t.Errorf("SPEC1 is %v, want %v", got, want)
		}
	})

	// 6.90: SPEC1 gets A2,F2,V2,O2+L3,L2-L3.
	t.Run("REMSP", func(t *testing.T) {
		s := specMachine()
		s.fullSpec(sp2, sc1, 3, 9, 2, 5)
		s.fullSpec(sp3, 0, 0, 0, 0, 2)
		s.REMSP(sp1, sp2, sp3)
		if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 4, 3}; got != want {
			t.Errorf("SPEC1 is %v, want %v", got, want)
		}
	})

	// 6.90 note 1: SPEC1 and SPEC3 may be the same, so L3 has to be
	// read before SPEC1 is written.
	t.Run("REMSP into SPEC3", func(t *testing.T) {
		s := specMachine()
		s.fullSpec(sp2, sc1, 3, 9, 2, 5)
		s.fullSpec(sp3, 0, 0, 0, 0, 2)
		s.REMSP(sp3, sp2, sp3)
		if got, want := s.readSpec(sp3), [5]int{sc1, 3, 9, 4, 3}; got != want {
			t.Errorf("SPEC1 is %v, want %v", got, want)
		}
	})

	// 6.43 and 6.85 are inverses through the same address.
	t.Run("PUTSPC then GETSPC", func(t *testing.T) {
		s := specMachine()
		s.fullSpec(sp2, sc1, 3, 9, 2, 5)
		s.set(sd1, sc3, 0, 0)
		if err := s.PUTSPC(sd1, 4, sp2); err != nil {
			t.Fatalf("PUTSPC: %v", err)
		}
		if got, want := s.readSpec(sc3+4), [5]int{sc1, 3, 9, 2, 5}; got != want {
			t.Errorf("the specifier at %d is %v, want %v", sc3+4, got, want)
		}
		if err := s.GETSPC(sp1, sd1, 4); err != nil {
			t.Fatalf("GETSPC: %v", err)
		}
		if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 2, 5}; got != want {
			t.Errorf("SPEC is %v, want %v", got, want)
		}
	})
}

// The specifier operations that reach an address the program computed
// can be pointed outside core.
func TestSpecifiersCheckTheirAddresses(t *testing.T) {
	for name, run := range map[string]func(*VM) error{
		"APDSP":  func(s *VM) error { return s.APDSP(sp1, sp2) },
		"GETBAL": func(s *VM) error { return s.GETBAL(sp1, sd1, specFail, specOK) },
		"GETSPC": func(s *VM) error { return s.GETSPC(sp1, sd1, 0) },
		"LOCSP":  func(s *VM) error { return s.LOCSP(sp1, sd1) },
		"PUTSPC": func(s *VM) error { return s.PUTSPC(sd1, 0, sp1) },
		"TRIMSP": func(s *VM) error { return s.TRIMSP(sp1, sp2) },
	} {
		s := specMachine()
		s.set(sd1, specCore+9, 0, 0)
		s.fullSpec(sp1, specCore+9, 0, 0, 0, 4)
		s.fullSpec(sp2, specCore+9, 0, 0, 0, 4)
		if err := run(s); err == nil {
			t.Errorf("%s: no error reaching outside core", name)
		}
	}
}

// APDSP (6.11) appends the characters of SPEC2 after those of SPEC1
// and adds the lengths.
func TestAPDSP(t *testing.T) {
	for _, tt := range []struct {
		name    string
		length1 string
		text2   string
		want    string
	}{
		{"append", "AB", "CDE", "ABCDE"},
		// Note 1: if L1 = 0, C21 is placed at A1+O1.
		{"onto nothing", "", "CDE", "CDE"},
		{"nothing", "AB", "", "AB"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := specMachine()
			s.chars(sc1, tt.length1)
			s.fullSpec(sp1, sc1, 0, 0, 0, len(tt.length1))
			s.chars(sc2, tt.text2)
			s.fullSpec(sp2, sc2, 0, 0, 0, len(tt.text2))
			if err := s.APDSP(sp1, sp2); err != nil {
				t.Fatalf("APDSP: %v", err)
			}
			if got := string(s.Text(sp1)); got != tt.want {
				t.Errorf("SPEC1 specifies %q, want %q", got, tt.want)
			}
			if got, want := s.readSpec(sp1), [5]int{sc1, 0, 0, 0, len(tt.want)}; got != want {
				t.Errorf("SPEC1 is %v, want %v", got, want)
			}
		})
	}
}

// APDSP reads the source before writing the destination, so appending
// a string to itself doubles it rather than looping.
func TestAPDSPOntoItself(t *testing.T) {
	s := specMachine()
	s.chars(sc1, "AB")
	s.fullSpec(sp1, sc1, 0, 0, 0, 2)
	if err := s.APDSP(sp1, sp1); err != nil {
		t.Fatalf("APDSP: %v", err)
	}
	if got := string(s.Text(sp1)); got != "ABAB" {
		t.Errorf("SPEC1 specifies %q, want %q", got, "ABAB")
	}
}

// SUBSP (6.118) branches, and leaves SPEC1 alone on the failing arm.
func TestSUBSP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		l2, l3 int
		wantPC int
		want   [5]int
	}{
		{"shorter", 3, 5, specOK, [5]int{sc1, 3, 9, 2, 3}},
		{"equal", 5, 5, specOK, [5]int{sc1, 3, 9, 2, 5}},
		{"longer", 6, 5, specFail, [5]int{0, 0, 0, 0, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := specMachine()
			s.fullSpec(sp1, 0, 0, 0, 0, 0)
			s.fullSpec(sp2, 0, 0, 0, 0, tt.l2)
			s.fullSpec(sp3, sc1, 3, 9, 2, tt.l3)
			s.SUBSP(sp1, sp2, sp3, specFail, specOK)
			if s.PC != tt.wantPC {
				t.Errorf("PC is %d, want %d", s.PC, tt.wantPC)
			}
			if got := s.readSpec(sp1); got != tt.want {
				t.Errorf("SPEC1 is %v, want %v", got, tt.want)
			}
		})
	}

	// As in REMSP, SPEC1 and SPEC3 may be the same one.
	t.Run("into SPEC3", func(t *testing.T) {
		s := specMachine()
		s.fullSpec(sp2, 0, 0, 0, 0, 3)
		s.fullSpec(sp3, sc1, 3, 9, 2, 5)
		s.SUBSP(sp3, sp2, sp3, specFail, specOK)
		if got, want := s.readSpec(sp3), [5]int{sc1, 3, 9, 2, 3}; got != want {
			t.Errorf("SPEC1 is %v, want %v", got, want)
		}
	})
}

// TRIMSP (6.125) drops a trailing run of blanks from the length.
func TestTRIMSP(t *testing.T) {
	for _, tt := range []struct {
		name string
		text string
		want int
	}{
		{"trailing blanks", "AB   ", 2},
		// Note 1: if CL is not blank, J = L.
		{"no trailing blank", "AB C", 4},
		{"all blanks", "    ", 0},
		// Note 2: if L = 0, TRIMSP is equivalent to SETSP.
		{"null string", "", 0},
		{"blanks inside only", "A  B", 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := specMachine()
			s.chars(sc1, tt.text)
			s.fullSpec(sp2, sc1, 3, 9, 0, len(tt.text))
			s.fullSpec(sp1, 0, 0, 0, 0, 0)
			if err := s.TRIMSP(sp1, sp2); err != nil {
				t.Fatalf("TRIMSP: %v", err)
			}
			if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 0, tt.want}; got != want {
				t.Errorf("SPEC1 is %v, want %v", got, want)
			}
			// SPEC2 is data input only.
			if got, want := s.readSpec(sp2), [5]int{sc1, 3, 9, 0, len(tt.text)}; got != want {
				t.Errorf("SPEC2 is %v, want %v; it is not altered", got, want)
			}
		})
	}
}

// TRIMSP counts from the end of the specified string, not the end of
// the characters, so an offset and a short length are respected.
func TestTRIMSPReadsThroughTheOffset(t *testing.T) {
	s := specMachine()
	s.chars(sc1, "xxAB  xx")
	s.fullSpec(sp2, sc1, 0, 0, 2, 4)
	if err := s.TRIMSP(sp1, sp2); err != nil {
		t.Fatalf("TRIMSP: %v", err)
	}
	if got, want := s.readSpec(sp1), [5]int{sc1, 0, 0, 2, 2}; got != want {
		t.Errorf("SPEC1 is %v, want %v", got, want)
	}
}

// LOCSP (6.60) points a specifier at the characters of a string
// structure, whose title holds the length and whose characters begin
// four descriptors in.
func TestLOCSP(t *testing.T) {
	s := specMachine()
	const title = sc2
	s.set(title, 0, 0, 3)                 // the title: value field is the length
	s.chars(title+4*s.Descr*s.CPA, "ABC") // the characters, past four descriptors
	s.set(sd1, title, 5, 7)
	if err := s.LOCSP(sp1, sd1); err != nil {
		t.Fatalf("LOCSP: %v", err)
	}
	if got, want := s.readSpec(sp1), [5]int{title, 5, 7, 4, 3}; got != want {
		t.Errorf("SPEC is %v, want %v", got, want)
	}
	if got := string(s.Text(sp1)); got != "ABC" {
		t.Errorf("SPEC specifies %q, want %q", got, "ABC")
	}
}

// Note 1: a zero address field is the null string, and only the length
// of SPEC is altered.
func TestLOCSPNullString(t *testing.T) {
	s := specMachine()
	s.fullSpec(sp1, sc1, 3, 9, 2, 5)
	s.set(sd1, 0, 0, 0)
	if err := s.LOCSP(sp1, sd1); err != nil {
		t.Fatalf("LOCSP: %v", err)
	}
	if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 2, 0}; got != want {
		t.Errorf("SPEC is %v, want %v; only the length changes", got, want)
	}
}

// INTSPC (6.49) converts an integer to a normalized string and
// specifies it.
func TestINTSPC(t *testing.T) {
	for _, tt := range []struct {
		i    int
		want string
	}{
		{0, "0"},
		{7, "7"},
		{-7, "-7"},
		{1234567, "1234567"},
		{-1234567, "-1234567"},
	} {
		s := specMachine()
		s.set(sd1, tt.i, 0, 0)
		s.INTSPC(sp1, sd1)
		if got := string(s.Text(sp1)); got != tt.want {
			t.Errorf("INTSPC of %d specifies %q, want %q", tt.i, got, tt.want)
		}
		addr, flag, value, _, length := s.Specifier(sp1)
		if flag != 0 || value != 0 || length != len(tt.want) {
			t.Errorf("INTSPC of %d gives flag %d, value %d, length %d; want 0, 0, %d",
				tt.i, flag, value, length, len(tt.want))
		}
		if addr < specCore {
			t.Errorf("INTSPC of %d specifies %d, which is inside the assembled image", tt.i, addr)
		}
	}
}

// Note 2: the buffer is local to INTSPC and a later use overwrites it.
// One buffer, not one per call.
func TestINTSPCReusesItsBuffer(t *testing.T) {
	s := specMachine()
	s.set(sd1, 12, 0, 0)
	s.INTSPC(sp1, sd1)
	first, _, _, _, _ := s.Specifier(sp1)

	s.set(sd1, 34, 0, 0)
	s.INTSPC(sp2, sd1)
	second, _, _, _, _ := s.Specifier(sp2)

	if first != second {
		t.Errorf("the second INTSPC used %d, not the %d the first did", second, first)
	}
	if got := string(s.Text(sp1)); got != "34" {
		t.Errorf("the first specifier now specifies %q, want %q", got, "34")
	}
}

// GETBAL (6.37) extends a specifier over the shortest balanced
// substring that follows it.
func TestGETBAL(t *testing.T) {
	for _, tt := range []struct {
		name   string
		text   string // the whole character region
		l      int    // how much of it SPEC already specifies
		n      int    // how much of the rest to examine
		wantPC int
		wantL  int
	}{
		// "If CL+1 is not a parenthesis, then J = 1."
		{"a plain character", "ABC", 0, 3, specOK, 1},
		{"after what is already specified", "ABC", 2, 1, specOK, 3},
		// "the least integer such that CL+1...CL+J is balanced"
		{"an empty pair", "()X", 0, 3, specOK, 2},
		{"a balanced group", "(AB)X", 0, 5, specOK, 4},
		{"nested groups", "((A)(B))X", 0, 9, specOK, 8},
		{"the shortest, not the longest", "(A)(B)", 0, 6, specOK, 3},
		// "If CL+1 is a right parenthesis ... transfer is to FLOC"
		{"a right parenthesis", ")AB", 0, 3, specFail, 0},
		// "or if no such balanced string exists"
		{"never closed", "(AB", 0, 3, specFail, 0},
		{"closed past the window", "(AB)", 0, 3, specFail, 0},
		{"an empty window", "(AB)", 0, 0, specFail, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := specMachine()
			s.chars(sc1, tt.text)
			s.fullSpec(sp1, sc1, 3, 9, 0, tt.l)
			s.set(sd1, tt.n, 0, 0)
			if err := s.GETBAL(sp1, sd1, specFail, specOK); err != nil {
				t.Fatalf("GETBAL: %v", err)
			}
			if s.PC != tt.wantPC {
				t.Errorf("PC is %d, want %d", s.PC, tt.wantPC)
			}
			wantL := tt.l
			if tt.wantPC == specOK {
				wantL = tt.wantL
			}
			if got, want := s.readSpec(sp1), [5]int{sc1, 3, 9, 0, wantL}; got != want {
				t.Errorf("SPEC is %v, want %v", got, want)
			}
		})
	}
}
