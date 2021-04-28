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

type address = int

// am is the abstract machine that SIL targets
type am struct {
	pc address // program counter
}

// Character strings are represented in packed format, as many
// characters per descriptor as possible. Storage of character
// strings in SNOBOL4 dynamic storage is always in storage units
// that are multiples of descriptors
//
// S4D58.PDF: 3.3
type characterString struct {
	data []byte
}

// descriptor is used to represent all pointers, integers, and real
// numbers. A descriptor may be thought of as the basic "word" of
// SNOBOL4. Descriptors consist of three fixed-length fields:
//   address
//   flag
//   value
// The size and position of these fields is determined from the data
// they must represent and the way that they are used in the various
// operations. The following paragraphs describe some specific
// requirements.
//
//
// On the IBM System/360, a descriptor is two words (eight bytes).
// The first word is the address field. The second word consists of
// one byte for the flag field and three bytes for the value field.
// The three bytes (24 bits) for the value field permits
// representation of data objects as large as 2^24-1 bytes. On the
// other hand, two bytes would limit objects to 2^16-1 bytes. Since
// on the IBM System/360 there are eight bytes per descriptor,
// 2^16-1 bytes would limit objects to 8191 descriptors, which would
// be too restrictive. For machines with fewer address units per
// descriptor, the value field need not be as large
//
// S4D58.PDF: 3.1
type descriptor struct {
	// The address field of a descriptor must be large enough to
	// address any descriptor, specifier, or program instruction
	// within the SNOBOL4 system.
	//   (Descriptors do not have to address individual characters
	//    of strings. See Section 3.2.)
	// The address field must also be large enough to contain any
	// integer or real number (including sign) that is to be used
	// in a SNOBOL4 program. The address field is the most
	// frequently used field of a descriptor and is used frequently
	// for addressing and integer arithmetic and it should be
	// positioned so that these operations can be performed efficiently
	//
	// S4D58.PDF: 3.1.1
	address address
	// The flag field is used to represent the states of a number of
	// disjoint conditions and is treated as a set of bits that are
	// individually tested, turned on, and turned off.
	// Five flag bits used in SNOBOL4.
	//
	// S4D58.PDF: 3.1.2
	flag int
	// The value field is used to represent a number of internal
	// quantities that are represented as unsigned integers
	// (magnitudes). These quantities the encoded representation
	// of source-language data types, the length of strings, and
	// the size (in address units) of various data aggregates.
	// The value field need not be as large as the address field,
	// but it must be large enough to represent the size of the
	// largest data aggregate that can be formed.
	//
	// S4D58.PDF: 3.1.3
	value int
}

// Specifiers are used to refer to character strings. Almost all
// operations performed on character strings are handled through
// operations on specifiers. All specifiers are the same size and
// have five fields:
//   address
//   flag
//   value
//   offset
//   length
// Specifiers and descriptors may be stored in the same area
// indiscriminately, and are indistinguishable to many processes in
// the SNOBOL4 system. As a result, specifiers are composed of two
// descriptors. One descriptor is used in the standard way to
// provide the address, flag, and value fields. The other descriptor
// is used in a nonstandard way. Its address field is used to
// represent the offset of an individual character from the address
// given in the specifier’s address field. The value field of this
// other descriptor is used for the length.
//
// S4D58.PDF: 3.2
type specifier struct {
	address address
	flag    int
	value   int
	offset  int
	length  int
}

// Syntax tables are necessarily somewhat machine dependent.
// Consequently, implementation of these tables is done individually
// for each machine. A description of the table requirements is given
// in section 4.
//
// S4D58.PDF: 3.4
type syntaxTableEntry struct{}
