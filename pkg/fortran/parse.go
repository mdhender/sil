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

package fortran

import (
	"fmt"
	"strconv"
)

// The format itself: parsing one, and running one.
//
// A format is a parenthesised list of items separated by commas, with
// the record separator / acting as its own separator. An item is an
// edit descriptor, optionally repeated, or a parenthesised group,
// optionally repeated. Blanks between items are ignored, which is what
// makes the count of a Hollerith field the only place in a format
// where a blank matters.

type kind uint8

const (
	kText  kind = iota // nH, or a quoted literal
	kBlank             // nX
	kSlash             // /
	kGroup             // ( ... )
	kInt               // Iw
	kReal              // Fw.d
	kExp               // Ew.d
	kChars             // Aw
)

// data reports whether a descriptor takes something from the output
// list. Format control stops at the first one of these that finds the
// list empty.
func (k kind) data() bool { return k >= kInt }

type item struct {
	kind     kind
	repeat   int // how many times, and for X how many blanks
	width    int // w
	decimals int // d
	text     string
	group    []item
}

// parse reads a format. The whole of it must be one parenthesised
// list, which is what FORMAT assembles and what a SNOBOL4 program
// builds.
func parse(format []byte) ([]item, error) {
	p := &parser{src: format}
	p.spaces()
	if !p.take('(') {
		return nil, fmt.Errorf("format %q: it does not start with (", format)
	}
	items, err := p.items()
	if err != nil {
		return nil, fmt.Errorf("format %q: %w", format, err)
	}
	if !p.take(')') {
		return nil, fmt.Errorf("format %q: it does not end with )", format)
	}
	p.spaces()
	if p.at < len(p.src) {
		return nil, fmt.Errorf("format %q: %q follows the closing )", format, p.src[p.at:])
	}
	return items, nil
}

type parser struct {
	src []byte
	at  int
}

func (p *parser) spaces() {
	for p.at < len(p.src) && p.src[p.at] == ' ' {
		p.at++
	}
}

func (p *parser) take(c byte) bool {
	if p.at < len(p.src) && p.src[p.at] == c {
		p.at++
		return true
	}
	return false
}

// count reads a repeat count, and reports whether there was one.
func (p *parser) count() (int, bool) {
	start := p.at
	for p.at < len(p.src) && p.src[p.at] >= '0' && p.src[p.at] <= '9' {
		p.at++
	}
	if p.at == start {
		return 0, false
	}
	n, err := strconv.Atoi(string(p.src[start:p.at]))
	if err != nil {
		return 0, false
	}
	return n, true
}

// items reads a list up to the closing parenthesis, which it leaves
// for the caller.
func (p *parser) items() ([]item, error) {
	var out []item
	for {
		p.spaces()
		if p.at >= len(p.src) {
			return nil, fmt.Errorf("it ends inside a list")
		}
		if p.src[p.at] == ')' {
			return out, nil
		}

		n, counted := p.count()
		if !counted {
			n = 1
		}
		p.spaces()
		if p.at >= len(p.src) {
			return nil, fmt.Errorf("it ends after a repeat count")
		}

		it, err := p.descriptor(n, counted)
		if err != nil {
			return nil, err
		}
		out = append(out, it)

		// A comma separates two items and the record separator
		// separates itself, so a comma after one is optional and a
		// second one is not a field.
		p.spaces()
		p.take(',')
	}
}

// descriptor reads one edit descriptor or group, with the repeat count
// already read.
func (p *parser) descriptor(n int, counted bool) (item, error) {
	c := p.src[p.at]
	p.at++

	switch c {
	case '(':
		group, err := p.items()
		if err != nil {
			return item{}, err
		}
		if !p.take(')') {
			return item{}, fmt.Errorf("a group is not closed")
		}
		return item{kind: kGroup, repeat: n, group: group}, nil

	case '/':
		return item{kind: kSlash, repeat: n}, nil

	case 'H', 'h':
		// The count is the number of characters that follow, and they
		// are taken exactly as they are: a Hollerith field is the one
		// place in a format where a comma or a parenthesis is text.
		if !counted {
			return item{}, fmt.Errorf("H has no count")
		}
		if p.at+n > len(p.src) {
			return item{}, fmt.Errorf("%dH wants %d characters and only %d are left", n, n, len(p.src)-p.at)
		}
		text := string(p.src[p.at : p.at+n])
		p.at += n
		return item{kind: kText, repeat: 1, text: text}, nil

	case '\'':
		text, err := p.quoted()
		if err != nil {
			return item{}, err
		}
		return item{kind: kText, repeat: n, text: text}, nil

	case 'X', 'x':
		return item{kind: kBlank, repeat: n}, nil

	case 'I', 'i':
		w, err := p.width(c)
		return item{kind: kInt, repeat: n, width: w}, err

	case 'F', 'f', 'E', 'e':
		w, err := p.width(c)
		if err != nil {
			return item{}, err
		}
		if !p.take('.') {
			return item{}, fmt.Errorf("%c%d has no decimal places", c, w)
		}
		d, ok := p.count()
		if !ok {
			return item{}, fmt.Errorf("%c%d. has no decimal places", c, w)
		}
		k := kReal
		if c == 'E' || c == 'e' {
			k = kExp
		}
		return item{kind: k, repeat: n, width: w, decimals: d}, nil

	case 'A', 'a':
		// A on its own is one character, which is what makes A1 and A
		// the same field.
		w, ok := p.count()
		if !ok {
			w = 1
		}
		return item{kind: kChars, repeat: n, width: w}, nil
	}
	return item{}, fmt.Errorf("%q is not an edit descriptor this reads", string(c))
}

// width reads the w of a descriptor that must have one.
func (p *parser) width(c byte) (int, error) {
	w, ok := p.count()
	if !ok {
		return 0, fmt.Errorf("%c has no width", c)
	}
	if w < 1 {
		return 0, fmt.Errorf("%c%d has no width", c, w)
	}
	return w, nil
}

// quoted reads a literal, with the opening quote already taken. Two
// quotes together are one quote, which is how a literal holds the
// character that ends it.
func (p *parser) quoted() (string, error) {
	var out []byte
	for p.at < len(p.src) {
		c := p.src[p.at]
		p.at++
		if c != '\'' {
			out = append(out, c)
			continue
		}
		if p.at < len(p.src) && p.src[p.at] == '\'' {
			out = append(out, '\'')
			p.at++
			continue
		}
		return string(out), nil
	}
	return "", fmt.Errorf("a literal is not closed")
}
