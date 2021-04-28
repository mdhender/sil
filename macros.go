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

package main

// ACOMP (address comparison) is used to compare the address fields
// of two descriptors. The comparison is arithmetic with A1 and A2
// being considered as signed integers.
//   If A1 > A2, transfer is to GTLOC.
//   If A1 = A2, transfer is to EQLOC.
//   If A1 < A2, transfer is to LTLOC.
//
// Data Input:
//   DESCR1 A1
//   DESCR2 A2
//
// Programming Notes:
//  1. A1 and A2 may be relocatable addresses.
//  2. See also LCOMP, ACOMPC, AEQL, AEQLC, and AEQLIC.
//
// S4D58.PDF: 6.1
func (s *am) ACOMP(descr1, descr2 descriptor, gtloc, eqloc, ltloc address) {
	if descr1.address > descr2.address {
		s.pc = gtloc // go to GTLOC
	} else if descr1.address == descr2.address {
		s.pc = eqloc // go to EQLOC
	} else { // descr1.address < descr2.address
		s.pc = ltloc // go to LTLOC
	}
}

// ACOMPC (address comparison with constant) is used to compare the
// address field of a descriptor to a constant. The comparison is
// arithmetic with A being considered as a signed integer.
//   If A > N, transfer is to GTLOC.
//   If A = N, transfer is to EQLOC.
//   If A < N, transfer is to LTLOC.
//
// Data Input:
//   DESCR A
//
// Programming Notes:
//  1. A may be a relocatable address.
//  2. N is never negative.
//  3. N is often 0.
//  4. See also ACOMP, AEQL, AEQLC, and AEQLIC.
//
// S4D58.PDF: 6.2
func (s *am) ACOMPC(descr descriptor, n int, gtloc, eqloc, ltloc address) {
	if descr.address > n {
		s.pc = gtloc // go to GTLOC
	} else if descr.address == n {
		s.pc = eqloc // go to EQLOC
	} else { // descr.address < n
		s.pc = ltloc // go to LTLOC
	}
}
