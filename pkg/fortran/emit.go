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
	"math"
	"strconv"
	"strings"
)

// Running a format, and the three numeric fields.

// emit runs a format over an output list and returns the records it
// produced.
//
// Three rules of format control decide where a record ends, and all
// three are in here:
//
//   - A / ends the record and starts the next one.
//   - Reaching a data descriptor with nothing left in the list ends
//     format control, and the record so far is written. That is what
//     makes (1X,132A1) print a five-character string as six characters
//     rather than padding it to a hundred and thirty-three.
//   - Reaching the end of the format with the list not yet empty
//     starts a new record and goes back to the last group at the outer
//     level, or to the beginning if there is no group. That is what
//     wraps a long string across records rather than truncating it.
func emit(items []item, l list) ([]Record, error) {
	p := &printer{list: l}
	from := items

	for {
		if err := p.items(from); err != nil {
			return nil, err
		}
		if p.stopped || l.left() == 0 {
			break
		}
		// The list still has something and the format is out of
		// fields to put it in. If a whole pass took nothing, going
		// round again would take nothing again.
		if p.took == 0 {
			return nil, fmt.Errorf("the format has no field for the %d values left", l.left())
		}
		p.took = 0
		p.record()
		from = reversion(items)
	}
	p.record()
	return p.out, nil
}

// reversion is where format control goes back to when the format runs
// out before the list does: the last group at the outer level, or the
// whole format when there is none.
func reversion(items []item) []item {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].kind == kGroup {
			return items[i:]
		}
	}
	return items
}

type printer struct {
	list    list
	rec     []byte
	out     []Record
	stopped bool // a data descriptor found the list empty
	took    int  // items taken since the last reversion
}

// record ends the record being built and starts an empty one.
func (p *printer) record() {
	p.out = append(p.out, Record(p.rec))
	p.rec = p.rec[:0]
}

func (p *printer) items(items []item) error {
	for _, it := range items {
		if p.stopped {
			return nil
		}
		if err := p.item(it); err != nil {
			return err
		}
	}
	return nil
}

func (p *printer) item(it item) error {
	for n := 0; n < it.repeat; n++ {
		if p.stopped {
			return nil
		}
		switch it.kind {
		case kText:
			p.rec = append(p.rec, it.text...)

		case kBlank:
			p.rec = append(p.rec, ' ')

		case kSlash:
			p.record()

		case kGroup:
			if err := p.items(it.group); err != nil {
				return err
			}

		case kChars:
			// Aw is w characters of the string, so a field that runs
			// off the end of it ends format control like any other.
			for i := 0; i < it.width; i++ {
				if p.list.left() == 0 {
					p.stopped = true
					return nil
				}
				c, err := p.list.char()
				if err != nil {
					return err
				}
				p.took++
				p.rec = append(p.rec, c)
			}

		default: // kInt, kReal, kExp
			if p.list.left() == 0 {
				p.stopped = true
				return nil
			}
			v, err := p.list.number()
			if err != nil {
				return err
			}
			p.took++
			p.rec = append(p.rec, field(it, v)...)
		}
	}
	return nil
}

// field converts one value into the w columns a descriptor asks for.
// A value that does not fit fills them with asterisks, which is what
// FORTRAN does rather than widen a field and put every column after it
// in the wrong place.
func field(it item, v Value) string {
	var s string
	switch it.kind {
	case kInt:
		s = strconv.Itoa(v.Int())
	case kReal:
		s = strconv.FormatFloat(v.Real(), 'f', it.decimals, 64)
	default:
		s = exponent(v.Real(), it.decimals)
	}
	if len(s) > it.width {
		return strings.Repeat("*", it.width)
	}
	return strings.Repeat(" ", it.width-len(s)) + s
}

// exponent writes a real number the way Ew.d does: a sign if it is
// negative, then a zero, a point, d digits, and E with a signed
// two-digit exponent. The digits are scaled so that the first of them
// is significant, which is what puts the value in 0.1 to 1 times a
// power of ten.
func exponent(f float64, d int) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	exp := 0
	mantissa := f
	if f != 0 {
		exp = int(math.Floor(math.Log10(math.Abs(f)))) + 1
		mantissa = f / math.Pow(10, float64(exp))
		// Rounding the mantissa to d places can carry it up to one,
		// which belongs in the exponent instead.
		if rounded, _ := strconv.ParseFloat(strconv.FormatFloat(mantissa, 'f', d, 64), 64); math.Abs(rounded) >= 1 {
			mantissa /= 10
			exp++
		}
	}
	sign := "+"
	if exp < 0 {
		sign, exp = "-", -exp
	}
	return fmt.Sprintf("%sE%s%02d", strconv.FormatFloat(mantissa, 'f', d, 64), sign, exp)
}
