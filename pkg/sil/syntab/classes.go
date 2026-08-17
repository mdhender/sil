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

package syntab

// The character classes of S4D58 4.1.
//
// 4.1 gives each class a descriptive name "to avoid dependence on a
// particular machine" and then lists the graphics for each, saying
// "ASCII graphics are used as a point of reference". This machine's
// characters are ASCII bytes -- one to a cell, ALPHSZ of them -- so
// the reference is the definition here, and that is the whole of what
// makes these classes machine dependent.
//
// The classes overlap: DOT and BREAK and CNT all contain the full
// stop, ALPHANUMERIC contains LETTER and NUMBER, and TERMINATOR
// contains BLANK and five graphics besides. That is harmless because
// no table names two classes that overlap; Build checks it rather than
// assuming it, since an overlap would be a transcription error and the
// appendix says nothing about which line would win.

const (
	blank = ' '
	tab   = '\t'
)

// Classes is 4.1's table, by name.
var Classes = map[string][]byte{
	"ALPHANUMERIC": span('A', 'Z', 'a', 'z', '0', '9'),
	"AT":           {'@'},
	"BLANK":        {blank, tab},
	"BREAK":        {'.', '_'},
	"CMT":          {'*'},
	"CNT":          {'+', '.'},
	"COLON":        {':'},
	"COMMA":        {','},
	"CTL":          {'-'},
	"DOLLAR":       {'$'},
	"DOT":          {'.'},
	"DQUOTE":       {'"'},
	"EOS":          {';'},
	"EQUAL":        {'='},
	"FGOSYM":       {'F'},
	"KEYSYM":       {'&'},
	"LEFTBR":       {'<', '['},
	"LEFTPAREN":    {'('},
	"LETTER":       span('A', 'Z', 'a', 'z'),
	"MINUS":        {'-'},
	"NOTSYM":       {'~'},
	"NUMBER":       span('0', '9'),
	"ORSYM":        {'|'},
	"PERCENT":      {'%'},
	"PLUS":         {'+'},
	"POUND":        {'#'},
	"QUESYM":       {'?'},
	"RAISE":        {'^'},
	"RIGHTBR":      {'>', ']'},
	"RIGHTPAREN":   {')'},
	"SGOSYM":       {'S'},
	"SLASH":        {'/'},
	"SQUOTE":       {'\''},
	"STAR":         {'*'},
	"TERMINATOR":   {';', ')', '>', ',', ']', blank, tab},
}

// span expands pairs of endpoints into the characters between them.
func span(ends ...byte) []byte {
	var out []byte
	for i := 0; i+1 < len(ends); i += 2 {
		for c := ends[i]; c <= ends[i+1]; c++ {
			out = append(out, c)
		}
	}
	return out
}
