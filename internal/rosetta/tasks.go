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

package rosetta

// Tasks is the manifest: which RosettaCode tasks are fetched, and what
// each one's description requires of the output.
//
// # How these ten were chosen
//
// Two filters, and the second one is not the obvious one.
//
// The first is that the task's description has to fix enough of the
// answer to assert without having read the program. Tasks that say
// what to compute and leave every solution free to print it its own
// way are worth very little here, however famous they are: Fibonacci,
// factorial and Towers of Hanoi are all like that, and an expectation
// for one of them would be a transcription of one contributor's
// formatting wearing a task's name. Anagrams and the word-list tasks
// want a dictionary file this runner has no way to give them.
//
// The second is dialect, and it decides most of it. Of the 139
// RosettaCode tasks with a SNOBOL4 section, 38 have a program with an
// END card in column 1 and 10 are written in upper case throughout.
// The rest are CSNOBOL4: case-insensitive, with &LCASE, CHAR, REVERSE,
// TABLE and the rest of what came after 1975. On version 3.11 a
// lower-case program does not fail in any interesting place -- the
// compiler does not recognise `end` as the END card, so it reads on
// past the last one and never stops. Nine of those in a row would say
// the same thing nine times, so one is kept, at Unsupported, to say it
// once. See Ending's Runaway.
//
// # What the four Unsupported entries turned out to be
//
// Not what was expected, and that is the useful part. One is the
// dialect (Sieve). One is a later SNOBOL4's library (Subleq, wanting
// CHAR and SUBSTR). The other two are version 3.11 itself, and neither
// would have been found by reading the instruction set:
//
//   - Zero to the zero power is unsatisfiable by specification. S4D58
//     6.32 says EXPINT transfers to FLOC when I1 is 0 and I2 is not
//     positive, so 0**0 is an arithmetic error and no correct
//     implementation prints 1.
//   - N-queens wants an integer past this machine's ceiling. SIZLIM is
//     16777215 -- a 24-bit value field, chosen in
//     pkg/sil/copyseg/parms.sil -- and the program opens with
//     &STLIMIT = 10 ** 9.
//
// Both are held rather than dropped, so that a change to EXPINT or to
// SIZLIM is announced by a test that starts passing.
//
// # Pinning
//
// Every entry is pinned to the revision and the program its
// expectation was written against. Fetch, read the program that
// arrives, then paste the revision and hash the fetcher prints -- that
// reading is the review, and it is where a Note stops being a guess.
var Tasks = []Task{
	{
		Name:   "Hello world/Text",
		File:   "hello-world-text.sno",
		Oldid:  408091,
		SHA256: "e2b951d9f91109976540594d9c8fa615bab6f8cf7b4917d25718ba25432c5d75",
		Note: "The only task in the corpus whose description fixes the output " +
			"whole, punctuation included, which is why this one asserts the " +
			"whole of it and the others do not.",
		Want: "Hello world!\n",
	},
	{
		Name:   "FizzBuzz",
		File:   "fizzbuzz.sno",
		Oldid:  407793,
		SHA256: "ed89b38369e15fa42159090ffe55d670b9626689e10e973c7c004cbb7a11ebae",
		Note: "The description fixes the whole answer and says nothing about " +
			"layout, so this counts substrings instead of matching text. Of 1 to " +
			"100: 33 are multiples of 3, 20 of 5, and 6 of 15. FizzBuzz contains " +
			"the other two, so the 6 are counted three times over, which is what " +
			"makes 33 and 20 the numbers here rather than 27 and 14. Folded, " +
			"because this program spells them FIZZ and BUZZ.",
		Counts: map[string]int{"Fizz": 33, "Buzz": 20, "FizzBuzz": 6},
		Fold:   true,
	},
	{
		Name:   "Loops/Downward for",
		File:   "loops-downward-for.sno",
		Oldid:  408103,
		SHA256: "c46694dc31cf954af72b5d4243f94215e2d0b0ebab60ba4b0fc24b0f5e558765",
		Note: "The description is the whole of the answer -- the numbers from 10 " +
			"down to 0 -- and there is not much else a solution could print. One " +
			"to a line is the program's choice rather than the task's, so this " +
			"is the entry to loosen to Counts if another solution ever changes " +
			"the layout.",
		Want: "10\n9\n8\n7\n6\n5\n4\n3\n2\n1\n0\n",
	},
	{
		Name:   "Empty string",
		File:   "empty-string.sno",
		Oldid:  406328,
		SHA256: "874806634788f4ad271ab7a1352d4fec0d618af337847880e9b99e378daf72b6",
		Note: "The task fixes the fact rather than the words: the string is " +
			"empty, so a solution that tests it has to take the branch that says " +
			"so. Absent is the real assertion here -- NOT NULL has NULL inside " +
			"it, so Contains alone would pass on the wrong answer.",
		Contains: []string{"NULL"},
		Absent:   []string{"NOT NULL"},
	},
	{
		Name:   "100 doors",
		File:   "100-doors.sno",
		Oldid:  408301,
		SHA256: "a02bbc464e954304d46cf8b4d717eb82047e0a6792b2fcf53d7b98d2a8224f20",
		Note: "The task's answer is that the open doors are the perfect squares. " +
			"The high squares are asserted because a one-digit door number is a " +
			"substring of half the output; 50 and 99 are asserted absent, which " +
			"is what catches a solution that prints all hundred doors with their " +
			"state instead of only the open ones -- and if a solution does that, " +
			"this entry needs rewriting rather than loosening. Reaches ARRAY and " +
			"the node group.",
		Contains: []string{"49", "64", "81", "100"},
		Absent:   []string{"50", "99"},
	},
	{
		Name:   "Roman numerals/Encode",
		File:   "roman-numerals-encode.sno",
		Oldid:  407928,
		SHA256: "f8cb2f01875e2dc7383ef1272400524e25c14d53a9f6d553cb7303fe78779449",
		Note: "The task fixes the encoding, not a particular number, so these " +
			"three are worked from its rules to match the numbers the program " +
			"happens to demonstrate: 1999 is MCMXCIX, 24 is XXIV, 944 is CMXLIV. " +
			"Nothing but a working encoder prints those. If the program's " +
			"demonstration numbers change, encode the new ones by hand rather " +
			"than copying what it printed. The widest reach in the corpus: a " +
			"recursive defined function, RPOS, BREAK, REPLACE and the pattern " +
			"nodes, in nine statements.",
		Contains: []string{"MCMXCIX", "XXIV", "CMXLIV"},
	},
	{
		Name:   "Zero to the zero power",
		File:   "zero-to-the-zero-power.sno",
		Oldid:  405528,
		SHA256: "4cdaf099d8569355c58427543ed21f38747d4e22dbb0b78891e1226be50828de",
		Note: "0**0 is 1, and version 3.11 will not say so. This entry is kept " +
			"unsatisfiable on purpose: the expectation is the task's answer, and " +
			"the day EXPINT stops failing on it the test starts passing and says " +
			"that a documented behaviour changed.",
		Status: Unsupported,
		Reason: "S4D58 6.32: EXPINT transfers to FLOC when I1 is 0 and I2 is not " +
			"positive, so 0**0 is an arithmetic error rather than 1",
		Ends: Diagnosed,
		Want: "1\n",
	},
	{
		Name:   "N-queens problem",
		File:   "n-queens-problem.sno",
		Oldid:  403261,
		SHA256: "f788be7f3bbea72dbbabacf9f352f7f3adc2f53e76f46c6d67b68a9fc7662494",
		Note: "The program sets N to 5, not to the task's 8, so the answer it is " +
			"held to is the number of solutions on a five-by-five board, which " +
			"is 10. Absent catches an eleventh. It never gets that far: the " +
			"second statement is &STLIMIT = 10 ** 9, and 10**9 is past this " +
			"machine's largest integer.",
		Status: Unsupported,
		Reason: "wants an integer past SIZLIM, which is 16777215 here -- a 24-bit " +
			"value field, chosen in pkg/sil/copyseg/parms.sil -- and the program " +
			"opens with &STLIMIT = 10 ** 9",
		Ends:     Diagnosed,
		Contains: []string{"Solution number 10 is:"},
		Absent:   []string{"Solution number 11"},
	},
	{
		Name:   "Subleq",
		File:   "subleq.sno",
		Oldid:  403345,
		SHA256: "e57534332478370688d6780303405ee2f9c6d5738cc0df48110b522be85231c8",
		Note: "A one-instruction machine, given a program that prints Hello, " +
			"world!, which the task fixes exactly. The most demanding task here " +
			"and the one that was expected to be unsupported: a machine " +
			"interpreting a machine interpreting a machine.",
		Status:   Unsupported,
		Reason:   "calls CHAR and SUBSTR, which are a later SNOBOL4's and not in 3.11",
		Ends:     Diagnosed,
		Contains: []string{"Hello, world!"},
	},
	{
		Name:   "Sieve of Eratosthenes",
		File:   "sieve-of-eratosthenes.sno",
		Oldid:  408229,
		SHA256: "fcdbc24ece96898fb6a79f517b99e012560b3d9ae315b793a99486b324acb33f",
		Note: "The one lower-case program kept, and it is kept to record what " +
			"happens to the other hundred: written for a case-insensitive " +
			"SNOBOL4, so 3.11 never sees an END card, so the compiler reads past " +
			"the last one and the card-reading loop reads for ever. Absent is " +
			"what it would have been held to -- 91 is 7 times 13, and a sieve " +
			"that is not sieving prints it -- and it is left in place so that " +
			"the day this program does run, the entry says whether it ran right.",
		Status: Unsupported,
		Reason: "written in lower case for CSNOBOL4, and 3.11's compiler is " +
			"case-sensitive: `end` is a statement, not the END card",
		Ends:     Runaway,
		Contains: []string{"89", "97"},
		Absent:   []string{"91"},
		// A runaway only has to be shown not to stop, and the sooner
		// the suite gets on with it the better.
		Max: 20_000_000,
	},
}
