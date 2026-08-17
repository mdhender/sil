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

// Package corpus locates the historical SIL source for tests.
//
// The Macro SNOBOL4 implementation is not redistributable, so
// engines/ is gitignored and a checkout may not have it. Tests that
// need it skip on ErrAbsent, keyed on the file itself rather than on
// an environment variable or a build tag, so that the skip expires by
// itself when the file arrives.
//
// A skip and a pass are different things. When a change touches
// scanning, parsing or assembly, check that these tests ran.
package corpus

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Engine is the module-relative path of the SIL source of SNOBOL4.
const Engine = "engines/sil-v3.11.sil"

// Counts measured over the whole of that file. They are the exit
// criteria the early stages are held to: reproduce the numbers or the
// stage has misread something in 6580 lines.
const (
	Lines      = 6580 // total source lines
	Comments   = 1748 // lines with an asterisk in column 1
	Statements = 4832 // everything else
	Labels     = 1624 // distinct labels, and there are no duplicates
	Opcodes    = 131  // distinct mnemonics: 119 operations and 12 directives
)

// ErrAbsent reports that the source is not in this checkout.
var ErrAbsent = errors.New("engine source not present in this checkout")

// SkipMessage tells a reader how to turn a skipped test back on.
const SkipMessage = "not present in this checkout; see references/MANIFEST.md for where to obtain it"

// Load returns the path and contents of the SIL source of SNOBOL4,
// wrapping ErrAbsent when the file is missing.
func Load() (name string, src []byte, err error) {
	root, err := moduleRoot()
	if err != nil {
		return "", nil, err
	}
	name = filepath.Join(root, Engine)
	src, err = os.ReadFile(name)
	if os.IsNotExist(err) {
		return name, nil, fmt.Errorf("%s: %w", Engine, ErrAbsent)
	}
	if err != nil {
		return name, nil, err
	}
	return name, src, nil
}

// moduleRoot walks up from the working directory to the directory
// holding go.mod, because tests run in their own package directory.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above %s", dir)
		}
		dir = parent
	}
}
