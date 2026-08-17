# SIL: build order and first milestones

## Context

`mdhender/sil` aims to run the historical Macro SNOBOL4 implementation by implementing the machine it was written for — SIL — rather than reimplementing SNOBOL4 in Go. The repo is currently a skeleton: descriptor/specifier types transcribed from S4D58 §3, two implemented instructions (`ACOMP`, `ACOMPC`), and an empty `main`.

The sibling project `../maclo` just did the same thing successfully for ML/I on LOWL, so the question is which half to build first and what to carry over. This plan answers that and lays out milestones through "the historical source assembles clean."

Three decisions were taken up front:

- **Mode:** AI-heavy, as maclo was. `AGENTS.md`'s "Agent Role" section says the opposite and must be updated (M0 task), or it will contradict every session.
- **Externals:** hybrid — `PARMS`/`MLINK` as hand-written SIL source that `COPY` includes; the 25 syntax tables generated in Go from S4D58 Appendix A.
- **Dispatch:** method per instruction with an S4D58 citation in the doc comment, as `pkg/sil/macros.go` already does, driven from a table.

## Recommendation: the assembler front end first

Build the parser and symbol resolver before any instruction semantics, then the VM behind an already-validated assembler.

"VM first" looks attractive because maclo unit-tested opcodes from hand-built core and needed no assembler to do it. But that would mean 119 instructions written against a machine model (`DESCR`, `CPA`, specifier layout) that nothing validates until something real assembles, and `AGENTS.md` explicitly prefers assembled SIL programs over hand-built instruction arrays. "Assembler first" looks blocked on the 37 external symbols, but it is not: **the entire reference graph of the historical source closes without choosing a single external value and without knowing what any instruction does**, because SIL's columnar format means every identifier outside a quoted literal is a reference and every label field is a definition. That yields a full-corpus exact-match gate in the first few days (M2 below) and retires the biggest unknown — does the real source actually fit the grammar — before a line of the machine exists.

## What the research established

Verified directly against the source and the manual during planning:

| Claim                                                                                             | Status                                                                                                                                                                       |
|---------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 131 mnemonics = 119 instructions + 12 directives; 4,832 statements; 1,624 labels, zero duplicates | Confirmed; S4D58 §7.4's own occurrence table matches the census exactly                                                                                                      |
| Columns: label 1–6, blank 7, opcode 8–13, blank 14–15, operand 16+                                | §7.6. Operand field ends at first blank/tab **not inside quotes**                                                                                                            |
| `BRANCH LOC,PROC` needs no linkage restore                                                        | §6.15 + all 21 sites target `SCNR`. S/360 base-register artifact; **hard part removed**                                                                                      |
| `PROC ,` vs `PROC name` needs no frame state                                                      | §6.78 note 2: *"has no functional use and may be implemented as LHERE"*; **hard part removed**                                                                               |
| `RCALL`/`RRTURN` layout                                                                           | §6.87/§6.95 print it verbatim (see below); maclo's `GOTBL` is **not** the right model                                                                                        |
| `*` must bind tighter than `+`/`-`                                                                | Exactly one site: `sil-v3.11.sil:5475` `OBEND DESCR OBLIST+DESCR*OBOFF,0,0`, where `OBOFF`=254 and `OBLIST` is relocatable. Left-to-right silently corrupts a bin-list bound |
| Unary minus exists                                                                                | `sil-v3.11.sil:2694` and `:2706` — `GETAC TVAL,PDLPTR,-2*DESCR`                                                                                                              |
| ~30 instructions are optional                                                                     | §7.1 lists them with the feature each disables, so **first-cut surface is ~89, not 119**                                                                                     |
| Syntax tables are specified                                                                       | Appendix A defines all 25 in a `BEGIN/FOR/ELSE/END` language; §4.2 recommends generating them mechanically                                                                   |
| Faults branch into the program                                                                    | §6.77 note 1 and §7.3: transfer to `INTR10`, never a Go error                                                                                                                |

§7.4 also gives execution-time percentages, which set the implementation priority: `RCALL` 8.9%, `GETD` 7.4%, `RRTURN` 6.2%, `INCRA` 5.6%, `LOCAPV` 5.2%, `GETDC` 5.0%, `POP` 4.3%, `AEQLC` 3.6%, `PUSH` 3.1%, `PUTDC` 3.1%.

Note the file we have is 6,580 lines with the columns 73–80 sequence field stripped; §7.6 describes 6,611 card images with it intact. Do not write a parser that expects it.

## Milestones

Each exit criterion is mechanically checkable. M0–M2 involve zero instruction semantics.

A milestone is **done** when its exit criterion is checked by a test that *ran* — the whole-source tests skip when `engines/sil-v3.11.sil` is absent, and a skip is not a pass.

|     | Milestone                                  | Status               | Where             |
|-----|--------------------------------------------|----------------------|-------------------|
| M0  | Scanner                                    | **done** — `86a6d81` | `pkg/sil/scanner` |
| M1  | Operand parser                             | **done** — `adbbae0` | `pkg/sil/parser`  |
| M2  | The symbol gate                            | **done** — `c1a4ccc` | `pkg/sil/symtab`  |
| M3  | Externals chosen; layout closes            | **done**             | `pkg/sil/copyseg`, `pkg/sil/layout` |
| M4  | Instruction table and shape validation     | **done**             | `pkg/sil/op`      |
| M5  | First vertical slice runs                  | next                 |                   |
| M6  | Instruction batches by §7.5 classification | TODO                 |                   |
| M7  | Syntax tables and `STREAM`                 | TODO                 |                   |
| M8  | The historical source assembles clean      | TODO                 |                   |
| M9  | Execution to first trap, then to `ENDEX`   | TODO                 |                   |
| M10 | First SNOBOL4 program                      | TODO                 |                   |

The front end (M0–M3) is complete: the whole 6,580-line source scans, parses and resolves with no diagnostics, the 37 undefined names it derives are exactly the machine-dependent contract, and with the three COPY segments in it lays out into 16,506 address units with every symbol valued and every expression well formed.

Two measurements in this plan were taken from an early census and turned out to be wrong once a real parse existed. The source contains **unary minus** (`GETAC TVAL,PDLPTR,-2*DESCR`, lines 2694 and 2706), and **536** statements carry null operands rather than 668 — the difference is 123 lone-comma statements, which S4D58 7.6 defines as *no operands* rather than empty ones, plus 9 whose only nulls sit inside parenthesised lists.

**M0 — Scanner.** Split every line into `{Line, Label, Op, Operand, Comment}` per §7.6, operand terminated at the first unquoted blank/tab. Update `AGENTS.md`'s Agent Role section to match the chosen mode.
*Exit:* 1,748 comment lines; 4,832 statements; 1,624 distinct labels, zero duplicates; every label matches `^[A-Z][A-Z0-9]{0,5}$`; re-joining the fields reproduces each line byte-for-byte.

**M1 — Operand parser.** `operandlist := item (',' item)*`; `item := ε | expr | '(' [item] (',' [item])* ')' | quoted`; `expr` with unary minus and `*` binding tighter than `+`/`-`. Nulls first-class (592 of them, across 536 statements); parens never nest.
*Exit:* all 4,832 parse; max top-level arity 6; max paren depth 1; a unit test pins `OBLIST+DESCR*OBOFF` as `OBLIST+(DESCR*OBOFF)`.

**M2 — The symbol gate.** Accumulate every identifier outside a quoted literal as a reference, every label field as a definition, resolve, report undefined *accumulated* (not first-error).
*Exit:* the undefined set is exactly these 37 names —
`ALPHA ALPHSZ AMPST BIOPTB CARDTB COLSTR CONTIN CPA DESCR ELEMTB EOSTB ERROR FNC FRWDTB GOTOTB IBLKTB LBLTB LBLXTB MARK MDATA MLINK NUMBTB PARMS PTR QTSTR SIZLIM SNABTB SPEC STOP STOPSH STTL TTL UNITI UNITO UNITP UNOPTB VARATB`
asserted as a literal sorted slice, in a test that skips when `engines/sil-v3.11.sil` is absent. If the test teaches the parser that `COPY`'s operand is a segment name rather than a symbol, the assertion becomes 34 + 3 segments — either is fine, but **state which the test encodes**, since it is the only per-opcode knowledge in the milestone.

This gate proves every name is *defined*. It says nothing about symbol *values*, which need a location counter, which needs `DESCR` and `CPA`. Hence M3.

**M3 — Externals chosen; layout closes.** Supply `PARMS`/`MDATA` as SIL text (§6.20 note 1 permits `COPY` to expand into text), one address unit per instruction, run the location counter.
*Exit:* zero undefined symbols, and these four identities **computed rather than asserted**: `PRMSIZ == PRMTRM-PRMTBL-DESCR`; `OBLIST == OBSTRT-LNKFLD` with `LNKFLD == 3*DESCR`; `BUFLEN == BUFEXT*CPA`; `OBEND == OBLIST+DESCR*OBOFF`. Plus a relocatable/absolute discipline check (`reloc-reloc`=abs, `reloc±abs`=reloc, `reloc*anything`=error).

Done, in `pkg/sil/copyseg` (the three segments, and `COPY` expansion into the line stream between the scanner and the parser) and `pkg/sil/layout` (the location counter and the value discipline). Four things the milestone did not anticipate:

- **`MLINK` is empty.** §6.20 wants entry points for machine-language subroutines and I/O packages; this machine has neither, so the segment is comment-only. It stays a segment because `COPY MLINK` is in the read-only source.
- **Nothing else needed per-opcode knowledge.** The location counter knows the twelve directives of §7.5 and treats everything else as one address unit. `COPY` had to move ahead of the parser, since it is what supplies the symbols; it is the assembler's only per-operation knowledge until M4.
- **`E`, the syntax table entry width (§5.3), is `DESCR`.** A table entry has three fields — next table address, indicator, put field (§5.1) — which is the shape of a descriptor, so a table is `ARRAY ALPHSZ`. All twenty-five of Appendix A's tables are declared, twelve of them reachable only through `GOTO(TABLE)`; contents are M7.
- **The four identities do not constrain `SPEC`.** `SPEC` reaches the location counter only through `STRING`, and the only relation the source states across a run of strings is `BUFEXT`, which measures that run with the sizes being checked. Assembling `STRING` as a descriptor plus characters passes all four. M5 is the first thing that can catch it. Recorded in the corpus test.

Risk 3 is retired further than planned: the whole assembly is rerun with `DESCR=2, SPEC=4, CPA=4` and every identity rechecked, which is where `ARRAY N` emitting `N` units instead of `N*DESCR` shows up — a fault `DESCR=1` cannot see.

**M4 — Instruction table and shape validation.** Still no semantics.
*Exit:* all 4,832 statements type-check against the table. Free oracle: for each of the 516 `*_` markers, the preceding statement's entry has `Terminates: true` (353 are `BRANCH`; the rest must be `BRANIC`, `RRTURN`, `SELBRA` with all slots filled, or `ENDEX`).

Done, in `pkg/sil/op`. All 4,832 statements fit, and the source uses all 131 entries. Four corrections the transcription of §6's boxes did not predict, each confirmed against the section before being made: `RRTURN`'s `DESCR` (§6.95 note 2), `MAKNOD`'s `DESCR6` (§6.62), `OUTPUT`'s argument list (undocumented, but `N=0` is what an omitted list means and 15 sites rely on it), and `BRANCH`'s `LOC`, which §5.2's blanket rule would have made optional and §6.15 does not.

The `*_` oracle turned out weaker than the plan assumed and a second, much stronger one turned up:

- `Terminates` is true for four operations, not five. §6.87 note 6 and §6.98 note 2 both let `RCALL` and `SELBRA` pass control to the operation following, even with every location supplied, so the 55 markers after them are claims about what the called procedure returns. The check that does hold over all 516 is "the preceding statement leaves no branch point unfilled". It is necessary, not sufficient, and — tested — it does not catch a *missed* branch slot.
- **No symbol in the source is used in two roles.** Over all 4,832 statements, 1,131 names each appear in exactly one of descriptor / specifier / branch point / constant / flag / syntax table / format. Misclassifying any slot in either direction puts a name in two roles: calling `AEQLC`'s `EQLOC` a constant produces 43 collisions, calling `GETDC`'s `DESCR2` a specifier produces 23. Four slots are excluded because their overlap is the documented meaning — `DESCR`'s `A` field, `EQU`'s operand, `SlotProc` (a procedure entry is also a branch target, by §6.15), and `SlotList`, which counts as its element kind.

`MinArgs` is a method rather than a field: it is entirely implied by which operands are `Optional`, and a stored copy could only drift. `Cells` became `Size`, an eight-value enum, because four of the six data directives size in `DESCR`, `SPEC` or `CPA` and those are not known until `PARMS` is read; `layout` reads the enum and supplies the numbers.

**M5 — First vertical slice runs.** See below.
*Exit:* `go test ./pkg/sil` drives a hand-written SIL program from source through assembler and VM to byte-exact output on an in-process buffer, exercising an alternate return and a fall-through return.

**M6 — Instruction batches by §7.5 classification**, cheapest-first, prioritised within batch by §7.4 frequency: descriptor move/set → address-field arithmetic → comparisons → flags → value/size → specifiers → tree/pattern nodes → real numbers → I/O → OS-dependent.
*Exit per batch:* every instruction has a test covering operand validity, state change, PC behaviour, branch behaviour, and error behaviour (`AGENTS.md`'s Definition of Done).

**M7 — Syntax tables and `STREAM`.** `STREAM` (35 sites), `PLUGTB`/`CLERTB` (4+4), MDATA generated from Appendix A. Lower risk than it looks: §4.2 states only `SNABTB` is ever mutated, so tables are immutable data with one exception.
*Exit:* a SIL program that plugs `SNABTB`, runs `STREAM`, and reproduces `ANY`/`BREAK`/`NOTANY`/`SPAN`.

**M8 — The historical source assembles clean.** *Exit:* zero diagnostics over 6,580 lines; entry point is the address of `BEGIN` (line 303, `BEGIN INIT ,` — §6.46 makes `INIT` the first instruction executed; `MLINK` supplies nothing the machine needs); a core listing is byte-stable across runs.

**M9 — Execution to first trap, then to `ENDEX`.** *Exit:* every halt is a documented `ENDEX` or a named unimplemented opcode. Track instructions-executed-before-first-trap as a monotone number in the changelog.

**M10 — First SNOBOL4 program.** `X = 'HELLO'; OUTPUT = X; END`, compared against a reference SNOBOL4.

## The call model (verified against §6.87 / §6.95)

Two descriptor-valued VM registers, `CSTACK` and `OSTACK` — not program symbols, so they are machine state. `STACK` *is* a program symbol (`sil-v3.11.sil:6352`) and reaches the VM through maclo's `Symbols map[string]int` channel. **There is no Go-side return stack**: frames live in `Core` inside the program's own `STACK` block as descriptors, because the garbage collector walks them. `ISTACK` (§6.50) sets `OSTACK.A = 0`, `CSTACK.A = Symbols["STACK"]`.

Frame layout on `RCALL`, with `A = CSTACK.A`:

```
A + 1*D          <- old OSTACK.A          (flags must be 0, §6.87 note 8)
A + 2*D          <- LOC (the return point)
A + 3*D          <- DESCRN                 arguments in REVERSE of PUSH order
   ...
A + (2+N)*D      <- DESCR1
CSTACK.A = A + (2+N)*D ;  OSTACK.A = A
```

`RCALL DESCR,PROC,(args),(exits)` assembles as **M+2 consecutive cells**: the `RCALL` word; a synthetic `RETVAL` cell holding the result descriptor's address (0 if the slot was null — §6.87 note 3); then `BRANCH LOC1..LOCM`, with **null exit slots resolved at emit time to the fall-through address** (note 5). `LOC` = `addr(RCALL)+1`.

`RRTURN DESCR,N` (§6.95) restores `CSTACK.A = A`, `OSTACK.A = A0`, optionally stores the returned descriptor, and sets **`PC = LOC + N`**. That single expression is the whole dispatch: `N=1` lands on the first `BRANCH`; `N=M+1` lands on `addr(RCALL)+M+2`, the fall-through, satisfying note 6 with no special case. **The VM never needs to know `M`.**

This is strictly better than maclo's `GOTBL` (`pkg/lowl/vm/step.go:281`), which stores an index in `Registers.JumpValue` and linear-scans contiguous words comparing `ValueTwo`. **Do not port that mechanism.** The VM should assert `Core[LOC].Op == RETVAL` so a corrupted stack traps at the return rather than executing data.

`SELBRA DESCR,(LOC1..LOCN)` (§6.98) is the same machinery — N+1 cells, `PC = addr(SELBRA)+I`, `I=N+1` falls through. **Write the branch-vector emitter once and call it from both.** 18 sites, one with 12 exits (`:1861`).

`BRANCH LOC[,PROC]` emits one cell; if `PROC` is present, validate it names a `PROC` label and discard it. `BRANIC DESCR,N` sets `PC = Core[Core[DESCR].A + N].A`; §6.16 note 1 says `N` is always zero — assert it. `PROC` emits nothing; it defines the label and records primary-vs-secondary for tracing and for validating `BRANCH LOC,PROC`.

Stack overflow and underflow **branch to `INTR10`** (§6.77 note 1, §7.3), never a Go error — same discipline as maclo's `ERLSO` handling.

## The Core model

**`Core []Cell`, one address unit per cell, `DESCR = 1`, `SPEC = 2`, `CPA = 1`.**

Every appearance of `DESCR` in the source is linear (`k*DESCR`, `label-label-DESCR`, `DESCR+SPEC`, `-2*DESCR`, `DESCR*OBOFF`) — no masking, no division, no alignment tricks — so `DESCR` may be any positive integer, and 1 makes `Cell` and "descriptor" the same thing. §3.1.1 licenses it: *"Descriptors do not have to address individual characters of strings"* — character indexing lives in the specifier's offset field.

Rejected: `[]byte` with `DESCR = 8`. Faithful to the S/360, but it forces marshalling on every access, destroys the "core doubles as a listing" property, and violates `AGENTS.md`'s rule to keep descriptor fields visible rather than opaque.

```go
type Cell struct {
    Op  OpCode   // Nop for pure data
    A   int      // descriptor address field                  S4D58 3.1.1
    F   Flags    // TTL MARK PTR FNC STTL                      S4D58 3.1.2
    V   int      // value field                                S4D58 3.1.3
    Ch  int      // one character, when this cell holds string data  S4D58 3.3
    Ops [6]int   // operand slots for instruction cells
    Src SourceRef // Line, Label, Op text, Operand text, Comment, Continuation
}
```

`Src` is maclo's highest-leverage idea (`pkg/lowl/vm/vm.go:68`): core becomes self-describing, the listing is free, and every trace line cites a source line. Add `Label` and `Comment` — SIL's 3,437 trailing comments make traces far more readable than maclo's.

A **specifier is two adjacent cells** (§3.2): `cells[a]` gives address/flag/value, `cells[a+1].A` the offset, `cells[a+1].V` the length. This is required, not cosmetic — the source moves specifiers half at a time.

Keep `CPA` a named constant rather than a literal `1`, and plan a later run with `CPA = 4` as the test that nothing silently assumed 1. Size `Core` at load time (`image + dynamic`); do **not** copy maclo's fixed `Core [65536]Word`.

External values to adopt: `DESCR`=1, `SPEC`=2, `CPA`=1, `ALPHSZ`=256, `SIZLIM`=2²⁴−1, flags `TTL MARK PTR FNC STTL`=1/2/4/8/16, `UNITI UNITO UNITP`=5/6/7.

## The instruction table

One table indexed by the enum, read by `String`, `Lookup`, the validator, the emitter, and the disassembler. Model on `maclo/pkg/l/stmt/table.go` — **do not** reintroduce the `codes.go`/`stringer.go`/`lookup.go` trio maclo still maintains by hand under `pkg/lowl/op/`, which its own CLAUDE.md names as a mistake.

```go
type Slot uint8
const (
    SlotNone Slot = iota
    SlotDescr; SlotSpec; SlotBranch; SlotProc; SlotConst
    SlotFlag; SlotUnit; SlotTable; SlotList; SlotLiteral; SlotSegment
)

type Operand struct {
    Slot     Slot
    ElemKind Slot   // element kind when Slot == SlotList
    Optional bool   // a null here is legal
    Name     string // "GTLOC", "DESCR1" -- used verbatim in diagnostics
}

type Entry struct {
    Kind       Kind
    Mnemonic   string
    Cat        Category  // S4D58 7.5 grouping, verbatim
    Operands   []Operand
    MinArgs    int
    Cells      int   // address units emitted; -1 = computed (RCALL, SELBRA, data)
    Directive  bool
    Terminates bool  // cross-checked against the 516 "*_" markers
    Doc        string // "S4D58 6.87"
}
```

Three tests hold it together: index equals `Kind`; the 131 mnemonics equal §6's alphabetical list (whose ordinal *is* the section number, so `Doc` must read `6.<ordinal>`); and `Terminates` matches the `*_` markers.

Two payoffs. `SlotBranch` + `Optional` is where §5.2 lives — **the assembler resolves every null branch slot to the fall-through address at emit time**, so the VM never sees a "0 means fall through" case and the existing `ACOMP` in `pkg/sil/macros.go` needs no change. And `Cat` copied from §7.5 gives the M6 batching for free.

## First vertical slice (M5)

Twelve executable macros — `INIT` `ISTACK` `RCALL` `RRTURN` `POP` `MOVD` `SETAC` `SUM` `ACOMP` `BRANCH` `STPRNT` `ENDEX` — plus directives `TITLE` `COPY` `EQU` `PROC` `LHERE` `DESCR` `STRING` `END`. `ACOMP`/`ACOMPC` already exist, so the slice starts one instruction ahead.

The program adds two numbers in a procedure, returns by exit 1 if the sum is within a limit and exit 2 if not, and the caller prints one of two strings:

```
       TITLE   'Vertical slice'
       COPY    PARMS
LIMIT  EQU     100
BEGIN  INIT    ,
       ISTACK  ,
       RCALL   SUMCL,PLUS,(ACL,BCL),(,BIG)
       STPRNT  RETCL,PRBLK,OKSP
       BRANCH  DONE
*_
BIG    STPRNT  RETCL,PRBLK,BIGSP
DONE   ENDEX   ZEROCL
*_
PLUS   PROC    ,
       POP     XCL
       POP     YCL
       MOVD    ZCL,XCL
       SUM     ZCL,ZCL,YCL
       ACOMP   ZCL,LIMCL,PLUSBG
       RRTURN  ZCL,1
*_
PLUSBG RRTURN  ZCL,2
```

It proves the columnar parse, null operand slots (`(,BIG)`), an expression, forward references, `RCALL` branch-vector emission, reversed argument order with `POP` retrieval, `RRTURN`'s `PC = LOC+N` on both arms, `ACOMP`'s omitted `EQLOC`/`LTLOC` resolving to fall-through, `STRING` laying a specifier plus characters, and `ENDEX`. Flipping `ACL` from 40 to 400 fires the other branch — one source edit, two goldens.

`STPRNT`'s format operand is an undigested FORTRAN IV format (§6.114), and a FORTRAN format interpreter is a subproject. Keep it behind a `Host` interface owned by the VM package, per `maclo/pkg/lowl/vm/host.go`, with a writer-backed fallback:

```go
type Host interface {
    Print(unit int, format []byte, s []byte) error // STPRNT 6.114, OUTPUT 6.75
    Read(unit int, buf []byte) (n int, err error)  // STREAD 6.115
    Time() int                                     // MSTIME 6.71
}
```

The slice's host ignores `format`. Keep it receiving raw format bytes so the interpreter lands later without moving the boundary.

Also carry over from maclo: `Assemble(nodes, Options) (*VM, error)` returning a populated VM rather than a file — **no image format gets designed**; `Options{Trace io.Writer; Listings bool}` zero-value-silent; `Step` public with `Run` a thin wrapper and the cycle cap a *field*, not a constant.

## Risks, in retirement order

1. **Expression precedence** (`:5475`) — M1. One line; no test program will ever find it.
2. **Parser doesn't fit the source** — M0–M2. The 37-name gate is exact; 38 means a real discovery.
3. **`DESCR`/`CPA` chosen wrong** — M3, by computing the four identities rather than asserting them, and by rerunning the assembly on a second set of parameters. Retired for `DESCR` and `CPA`; `SPEC` waits for M5.
4. **Call model wrong** — M5. Hardest joint decision; a 30-line program falsifies it immediately. **Do not defer behind sixty easy instructions.**
5. **Table drift** — M4, via the three cross-checks. Retired, but by the one-role oracle rather than by the `*_` markers; §6 being alphabetical makes `Doc` self-checking, since an operation's rank in the alphabet is its section number.
6. **First-error bailout unusable at 6,580 lines** — M2. maclo's retrospective names this as its own mistake; accumulate per stage.
7. **Syntax tables** — M7. Appendix A is the spec; budget for 25 tables where the source names only 13.
8. **`PLUGTB`/`CLERTB`** — M7, and optional per §7.1 (cost: `ANY`/`BREAK`/`SPAN`/`NOTANY`).
9. **FORTRAN formats** for the 26 `FORMAT` literals — M6 I/O batch, entirely behind `Host.Print`.
10. **`LOAD`/`UNLOAD`/`LINK`** — one site each, 0.000% execution time. Documented trap, or never.

Struck from the risk register by the manual: `BRANCH LOC,PROC` (§6.15) and `PROC ,` vs `PROC name` (§6.78 note 2).

## Verification

- `go test ./pkg/sil` at every milestone; `go fmt ./...` and `go vet ./...` before any milestone is called done.
- Corpus tests **skip when `engines/sil-v3.11.sil` is absent** (it is gitignored and not redistributable), keyed on the file itself so the skip expires by itself — maclo's pattern. A skip and a pass are different things; check which you are looking at.
- M2/M3/M4 each assert against the whole 6,580-line file, so they are regression tests for every later change.
- M5 onward, each instruction needs the five-part test from `AGENTS.md`'s Definition of Done, plus at least one assembled SIL program exercising it.
- Trace output goes to a stream separate from program output (`AGENTS.md`), so a run can be diffed against a previous run.

## Files

- `pkg/sil/types.go` — `am`, `descriptor`, `specifier` become `Cell`, `Core`, `CSTACK`/`OSTACK`, `Symbols`
- `pkg/sil/macros.go` — the doc-comment-plus-method convention every instruction follows
- new: scanner, operand parser, symbol table, instruction table, assembler, `Step`, `Host`
- `pkg/sil/copyseg/{parms,mlink,mdata}.sil` — the machine-dependent segments; Go generator for the Appendix A syntax table contents still to come
- `AGENTS.md` — update "Agent Role" to match the chosen mode (M0)
- `engines/sil-v3.11.sil` — read-only input. Lines 303, 2694, 5475, 6336, 6343, 6352, 6580 are the ones this plan leans on
- `references/s4d58-sil-v3.11.pdf` — §5.2 branch points, §6.15/6.78/6.87/6.95/6.98 call model, §7.1 optional macros, §7.4 frequencies, §7.5 classification, §7.6 source format, Appendix A syntax tables
