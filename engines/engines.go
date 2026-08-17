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

// Package engines carries the SIL source of the historical Macro
// SNOBOL4 implementation into the binary.
//
// It is one file and one function. The interesting part is what the
// embed pattern has to work around: the source is assumed not to be
// redistributable, so it is not in the repository, and go:embed has no
// way to say that a file may be absent.
package engines

import (
	"embed"
	"errors"
	"fmt"
)

// Engine is the name of the SIL source of Macro SNOBOL4 version 3.11.
// It is the same file internal/corpus reads from disk for the tests;
// see engines/README.md for where to obtain it.
const Engine = "sil-v3.11.sil"

// files is this directory, as the build found it.
//
// The pattern is a bare * for the reason README.md gives: naming
// sil-v3.11.sil would stop the tree compiling without it, and naming
// only README.md would never pick it up, so the pattern takes what is
// here and Source looks the file up by name. Everything else the
// pattern sweeps in -- this file, .gitignore, the README -- is a
// couple of kilobytes that nothing reads.
//
//go:embed *
var files embed.FS

// ErrAbsent reports that the source was not in the tree when this
// binary was built.
var ErrAbsent = errors.New("engine source not embedded in this build")

// Source returns the name and contents of the SIL source of SNOBOL4.
func Source() (name string, src []byte, err error) {
	src, err = files.ReadFile(Engine)
	if err != nil {
		return Engine, nil, fmt.Errorf("%s: %w", Engine, ErrAbsent)
	}
	return Engine, src, nil
}
