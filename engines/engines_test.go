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

package engines_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mdhender/sil/engines"
	"github.com/mdhender/sil/internal/corpus"
)

// The embed carries the same bytes the corpus tests read from disk.
//
// Two things could go wrong quietly and this is what catches them: an
// embed pattern that matches nothing useful, so that a binary ships
// with no engine and only says so when someone runs it; and an embed
// that goes stale, since a build that did not notice the file changed
// would run a different SNOBOL4 from the one the tests assembled.
func TestTheEmbeddedSourceIsTheSourceOnDisk(t *testing.T) {
	name, embedded, err := engines.Source()
	if errors.Is(err, engines.ErrAbsent) {
		t.Skipf("%s: %s", name, corpus.SkipMessage)
	}
	if err != nil {
		t.Fatal(err)
	}

	path, onDisk, err := corpus.Load()
	if errors.Is(err, corpus.ErrAbsent) {
		// The embed has it and the disk does not, which means the
		// file was removed after this binary was built.
		t.Fatalf("%s is embedded but %s is gone", name, path)
	}
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(embedded, onDisk) {
		t.Errorf("the embedded %s is %d bytes and %s is %d",
			name, len(embedded), path, len(onDisk))
	}
	t.Logf("%s: %d bytes", name, len(embedded))
}

// A build without the source still builds, and says which file is
// missing rather than running an empty machine.
func TestSourceNamesWhatIsMissing(t *testing.T) {
	name, _, err := engines.Source()
	if err != nil && !errors.Is(err, engines.ErrAbsent) {
		t.Errorf("reported %v, want a wrapped ErrAbsent", err)
	}
	if name != engines.Engine {
		t.Errorf("named %q, want %q", name, engines.Engine)
	}
}
