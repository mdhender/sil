# testdata/rosetta

SNOBOL4 programs taken from [RosettaCode](https://rosettacode.org),
run on the historical Macro SNOBOL4 implementation and held to what
each task's own description says the answer is.

## The programs are not in this repository

RosettaCode's contributions are licensed CC BY-SA, which is not the
licence this repository carries, so this directory is gitignored the
way `engines/` and `references/` are. A working checkout fetches them:

```sh
go run ./scripts/fetch-rosetta          # everything not already here
go run ./scripts/fetch-rosetta -list    # what the manifest names
go run ./scripts/fetch-rosetta -task fizz
```

`internal/rosetta` is the manifest, and it is where everything this
repository has to say about these programs lives.

## What is committed is the expectation, not the output

The usual way to test against somebody else's program is to run it
once and record what it printed. That is not what happens here. Each
entry in the manifest says what the *task description* fixes about the
answer — FizzBuzz says which of 1 to 100 get which word, Hailstone
says the longest sequence under 100000 is 77031's with 351 elements —
written from the description before the program was fetched.

The difference matters twice:

- A RosettaCode solution that is subtly wrong fails here. A golden
  file recorded from its own output would have blessed it.
- An expectation can be too tight for a solution that satisfies the
  task while printing it differently. Where a description does not fix
  the formatting, the entry asserts only the part that is fixed, and
  says so in its `Note`.

So a failure here is a question — is the machine wrong, or is the
expectation? — and the `Note` is what answers it.

## Pinning

A wiki page changes. Each entry carries the revision (`Oldid`) and the
hash (`SHA256`) of the program its expectation was written against, so
a red test means this repository changed rather than that somebody
edited a wiki page.

Entries start unpinned. The fetcher takes whatever revision is current
and prints the revision and hash as lines to paste back into
`internal/rosetta/tasks.go`. **Read the program before pinning it** —
that reading is the review, and it is where a `Note` written from the
task description stops being a guess about what the program does. A
pinned task is then fetched by revision number and refuses to be
overwritten by anything that does not hash the same.

## Version 3.11 is not the SNOBOL4 most of these were written for

Of the 139 RosettaCode tasks with a SNOBOL4 section, 38 have a program
with an `END` card in column 1 and **10 are written in upper case
throughout**. The rest are CSNOBOL4 — case-insensitive, with `&LCASE`,
`CHAR`, `REVERSE`, `TABLE`. On 3.11 a lower-case program does not fail
anywhere interesting: the compiler does not recognise `end` as the END
card, reads on past the last one, and the card-reading loop in the SIL
source reads again on end-of-file, so it never stops. That is faithful
— a deck without an END card hangs the original too — and it is why
the corpus keeps exactly one such program rather than a hundred.

A program 3.11 cannot run belongs at `Unsupported`, with the reason
and with `Ends` saying how the system gets rid of it (`Diagnosed` or
`Runaway`). Those entries are the useful half of the exercise. The
four here are all different, and two of them are about this machine
rather than the dialect:

| Task | Why 3.11 will not run it |
|---|---|
| Sieve of Eratosthenes | lower-case CSNOBOL4, so there is no END card — runs away |
| Subleq | calls `CHAR` and `SUBSTR`, which came after 1975 |
| Zero to the zero power | S4D58 6.32: `EXPINT` fails when I1 is 0 and I2 is not positive, so `0**0` is an arithmetic error |
| N-queens problem | opens with `&STLIMIT = 10 ** 9`, and `SIZLIM` here is 16777215 — the 24-bit value field chosen in `pkg/sil/copyseg/parms.sil` |

The last two are held rather than dropped, with the task's real answer
still in the entry, so that a change to `EXPINT` or to `SIZLIM` is
announced by a test that starts passing.

## Running them

```sh
go test ./cmd/sil -run TestRosettaCode -v
```

The whole corpus takes about half a second, nearly all of it the one
runaway proving it is one.

Each task skips on its own when its program was not fetched, and
**a skip is not a pass**. With this directory empty, nothing here
checks anything.

That is the limit of what this corpus can do, and why there is a
second one. `internal/programs` holds SNOBOL4 written for this
repository — committed, embedded, in every checkout — and carries the
golden output and the benchmarks. Use these to find out whether the
machine runs SNOBOL4 that nobody here wrote; use those to find out
whether a change altered anything.

## Why this directory keeps a README

The same reason `engines/` does: something has to be committed for the
directory to exist in a fresh checkout, and this is the file that says
why it is otherwise empty.
