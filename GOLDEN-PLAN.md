# Bring CSNOBOL4 in as a differential oracle

## Context

The question was whether to start a native Go SNOBOL4 interpreter now, using SIL as
the oracle, or using CSNOBOL4. Reading the repo's own documents turns the question
around.

`PLAN.md:272` already names this as the next thing, and names it the largest open
assumption in the repository:

> **"Correct" here means "agrees with the documents", not "agrees with SNOBOL4".**
> There is no differential test against CSNOBOL4 or any other implementation. Every
> golden in `internal/programs` is written by hand from what the program should do
> [...] but it means a misreading of the manual shared between the implementation and
> the test would pass both. **This is the largest open assumption in the repository
> and it is not on the risk register. Getting a reference SNOBOL4 in as an oracle is
> what would retire it.**

That loop is the problem: the same person read S4D58, wrote `pkg/sil`, and wrote the
goldens `pkg/sil` is checked against. Four assembly stages agreeing does not break it,
because they all read the same manual the same way. **SIL is therefore not yet entitled
to be an oracle for anything** — diffing a second implementation against it would
propagate the unvalidated reading into a second place and return agreement that looks
like confirmation.

CSNOBOL4 breaks the loop precisely, because it was written from the Green Book and from
SNOBOL4 itself, independently of this repository's reading of S4D58.

**Decision taken:** do the CSNOBOL4 validation and stop there. No Go interpreter now.
If one is built later it is a *modern* SNOBOL4 (deliberate divergence, not a
compatibility clone) and it lives in a **sibling repo**, because `AGENTS.md` and
`README.md` both name a native Go SNOBOL4 as this project's explicit anti-goal —
README: it "would turn the project into a SNOBOL4 reimplementation and undermine the
purpose of using SIL." That decision is recorded at the end and is not part of this work.

**Outcome:** `PLAN.md`'s "What needs qualifying" loses its first and largest item, and
"What can be claimed" gains a bullet backed by a test that ran.

## Availability

`brew install snobol4` — CSNOBOL4 2.3.4, bottled, BSD-2-Clause, one dependency
(`openssl@4`). Not currently installed. No `spitbol` in Homebrew; not needed.

## What the corpus can and cannot validate today

Measured on this checkout, and this is the part that shapes the work.

**Pattern matching is nearly absent.** Only 3 of the 22 committed programs match a
pattern at all, from the machine's own statistics:

| program | pattern matches performed |
|---|---|
| `roman` | 309 |
| `reverse` | 69 |
| `charcode` | 6 |
| the other 19 | 0 |

Pattern matching is SNOBOL4's defining feature and the most intricate part of the
machine — the node group, backtracking, cursor position, `&ANCHOR`, `&FULLSCAN`,
unevaluated expressions, conditional and immediate value assignment. A differential
test over today's corpus would validate arithmetic, control flow and I/O, and would
barely touch the region where a shared misreading is most likely to hide. **Extending
the corpus is the larger half of this work, not a follow-on.**

**Two of the five deviating readings are reached only by synthetic probes.** `PLAN.md`
records five places where this implementation deviates from the printed text on its own
judgment. Counting opcodes in `-trace` output across the pattern-matching programs:

| reading | `roman` | `reverse` | `charcode` | covered by a real program? |
|---|---|---|---|---|
| `LEXCMP` §6.53 | 477 | 133 | 51 | yes |
| `REMSP` §6.90 | 1188 | 128 | 6 | yes |
| `STREAM` §6.116 | 888 | 182 | 303 | yes |
| `RCOMP` §6.88 | 0 | 0 | 0 | **no — probe only** |
| `EXPINT` §6.32 | 0 | 0 | 0 | **no — probe only** |

`RCOMP` (real comparison) and `EXPINT` (integer exponentiation) are judgment calls
about what the document meant, validated by nothing but the same judgment. They are the
sharpest single target for an outside opinion.

**`charcode` is a free result.** It carries the one `PENDING:` directive — `&ALPHABET`
reads as 256 blanks because `ALPHA` is never filled in. CSNOBOL4 says independently what
that program should print, which either confirms the diagnosis or corrects it.

## Design

### The golden is the contract; two implementations meet it

The differential test does **not** re-run SIL and diff against CSNOBOL4. `TestPrograms`
already asserts SIL matches each `.out`. The new test asserts CSNOBOL4 matches the same
hand-written `.out`. Two independent implementations agreeing on a golden written from
the manual is exactly the evidence that retires the assumption, and it keeps the goldens
as the single statement of intent rather than introducing a second recorded baseline.

This also preserves the repo's rule that a golden is written from what the program
*should* do, never recorded from a run — CSNOBOL4's output is evidence about the golden,
never a source for it.

### A divergence is a question, not a verdict

CSNOBOL4 is not v3.11. It is a later reimplementation with extensions and bug fixes, so
each program has three possible outcomes, and the test must classify rather than assert
equality:

- **Agrees** — the evidence being sought.
- **Diverges, adjudicated** — a recorded dialect difference, with a `Reason` and a
  citation. Data, not a failure.
- **Diverges, unexplained** — a real finding. Either `pkg/sil` misread S4D58 or CSNOBOL4
  departed from 3.11; settle it against S4D58 and the Green Book, and record which.

Model this on `internal/rosetta`, which already solved the same problem: `Status`,
`Reason`, `Ends`, and a mandatory `Note` saying where the expectation came from, with
`Validate` run unconditionally to reject an entry that diverges without saying why.
Reuse that discipline verbatim — a divergence recorded without a reason is the failure
mode to design against.

### Valid and invalid programs compare differently

The 15 valid programs compare **byte-for-byte on stdout** after `programs.Normalize`.

The 6 `ERRORS:` programs cannot. Their `.out` goldens contain a v3.11-format compilation
listing (statement numbers in a fixed column) and v3.11 error text — `ERROR 24 IN
STATEMENT    1 AT LEVEL  0` — which CSNOBOL4 will never reproduce. For those, compare
semantically: **did it reject the program, and at which statement**. Say so explicitly
in the test rather than quietly excluding them.

`noend` (`RUNAWAY:`) needs a wall-clock timeout on the CSNOBOL4 side, since there is no
`-max` there.

### Skip discipline

CSNOBOL4 is an external binary, so the oracle is absent in most checkouts. Follow
`internal/corpus` exactly: key the skip on the artifact itself via `exec.LookPath`, not
on an env var or build tag, so the skip expires the moment the tool arrives. **A skip is
not a pass** — log the ran/skipped counts the way `TestRosettaCode` does, because a run
where everything skipped looks identical to a run where everything passed.

## Work, in order

**1. Get CSNOBOL4 in and characterise it.** `brew install snobol4`. Establish by hand,
before writing any test: whether it accepts the corpus's card format and column-72
statement field; what its equivalent of `-UNLIST`/`-LIST` is; whether it prints a banner
and where; whether it separates the program's printing from the system's report the way
`STPRNT` vs `OUTPUT` does here, or merges them. Write down what has to be normalized.
This step decides the shape of everything after it — do not skip it.

**2. `internal/corpus`: add the reference tool.** Alongside `Engine`/`Load`/`ErrAbsent`,
add a `Reference` lookup returning the binary path, its own `ErrAbsent`, and a skip
message pointing at `brew install snobol4`. Same shape as what is there.

**3. New package `internal/reference`.** Two things: run a program through the external
binary (source in, two streams and a timeout out), and hold the adjudication manifest —
one entry per program that does not agree, with `Reason` and `Note`, plus a `Validate`
that runs unconditionally. Reuse `programs.Normalize`; add whatever step 1 showed is
needed and document each one where it is defined.

**4. `TestReference` over `programs.All()`.** Subtest per program. Valid programs
byte-compare against `p.Want`; `ERRORS:` programs compare on rejection and statement
number; `RUNAWAY:` expects the timeout. A program that agrees needs no manifest entry;
one that diverges needs one, or the test fails. Report ran/skipped.

**5. Adjudicate the first round.** Every divergence gets read against S4D58 and the
Green Book and lands in one of two places: a manifest entry citing the dialect
difference, or a fix. **This is the step with the actual value in it** — expect it to
be where the time goes, and expect at least one genuine finding.

**6. Extend the corpus where it is thin.** New programs in `internal/programs/sno`,
each with a hand-written `.out`, written to the manual first and only then run — the
existing discipline. Priority follows the measurements above:

- **Pattern matching**, the bulk of it: `ARB`, `SPAN`, `BREAK`, `ANY`, `NOTANY`, `LEN`,
  `POS`/`RPOS`, `TAB`/`RTAB`, `REM`, alternation, concatenation, replacement,
  conditional (`.`) and immediate (`$`) value assignment, `&ANCHOR` both ways,
  backtracking that actually backtracks, and unevaluated expressions (`*X`) for
  recursive patterns.
- **`RCOMP` and `EXPINT`**: real comparison and `**`, including the edge `EXPINT` was
  read against — `0 ** 0`, which §6.32 makes an arithmetic error here and which
  RosettaCode's "Zero to the zero power" is pinned `Unsupported` on. CSNOBOL4 settles
  whether that reading is right.
- Defined data types, `TABLE`, indirect reference (`$`), `CODE`/`EVAL`.

Each new program goes through both `TestPrograms` and `TestReference`, so it validates
SIL and gets a second opinion in the same commit.

**7. The Rosetta cross-check.** Run the 10 `testdata/rosetta` programs through CSNOBOL4
and check the four `Unsupported` reasons are dialect, not defect. `sieve-of-eratosthenes`
is the pointed one — it is `Unsupported` *because it was written for CSNOBOL4*, so it
should run there and confirms the reason rather than the excuse. Gitignored, so this
skips independently.

**8. Update `PLAN.md`.** Move the first item of "What needs qualifying" into "What can
be claimed", stating what agreement was measured over and what it does not cover.
Add the corpus extension to the milestone record.

## Files

- `internal/corpus/corpus.go` — add the reference-tool lookup beside `Engine`.
- `internal/reference/` (new) — runner, manifest, `Validate`, `TestReference`.
- `internal/programs/sno/*.sno` + `.out` (new) — the pattern-matching and
  arithmetic-edge programs from step 6.
- `PLAN.md` — the claim, and the qualification that goes away.
- `README.md:393` ("Where practical, compare observable behavior with a known SNOBOL4
  implementation") stops being aspirational; also worth fixing the stale `## Status`
  section while there.

Reused as-is, not rebuilt: `programs.All/Get/Normalize/Diagnostics/Card/Columns`,
`corpus.Root/ErrAbsent/SkipMessage`, and `rosetta`'s `Status`/`Reason`/`Note`/`Validate`
manifest pattern.

## Verification

```bash
brew install snobol4
go build ./... && go vet ./... && go fmt ./...
go test ./internal/reference/ -v          # must report programs ran, not skipped
go test ./... -count=1                    # 813 tests today; no failures, no skips
```

The run is only meaningful with `engines/sil-v3.11.sil` present *and* `snobol4` on
`PATH`. Confirm the test log names a nonzero ran-count; a green suite with everything
skipped is the failure this is designed to catch.

Two spot checks that should be done by hand, because they are the point:

- `charcode` — what CSNOBOL4 prints for `&ALPHABET` either confirms the `PENDING:`
  diagnosis or replaces it.
- `0 ** 0` — whether CSNOBOL4 makes it an error settles the `EXPINT` §6.32 reading and
  the Rosetta entry pinned on it.

## What is deferred, and on what terms

A native Go SNOBOL4 interpreter, if built:

- **A modern SNOBOL4** — better diagnostics, Unicode, arbitrary-precision integers,
  modern I/O. Deliberate divergence, so the corpus is a conformance *baseline* it
  departs from feature by feature, with each departure recorded, not a spec it must
  match.
- **A sibling repo** importing `mdhender/sil`, leaving this project's central rule
  intact. Note that `internal/programs` is under `internal/`, so a separate module
  cannot import it — the corpus would need to move to `pkg/` or be vendored, which is
  a decision for that day.
- **Not before this work lands.** The whole argument above is that SIL is not yet
  entitled to be an oracle. After step 8 it is.
