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
| M5  | First vertical slice runs                  | **done**             | `pkg/sil`, `pkg/sil/asm` |
| M6  | Instruction batches by §7.5 classification | in progress — 80 of 119 operations | `pkg/sil`        |
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

Done, in `pkg/sil` (the machine) and `pkg/sil/asm` (the front end plus emission). `pkg/sil/asm/testdata/slice.sil` is the program; one operand differs between the two runs and the two goldens follow. Risk 4 is retired: the call model is right as written.

Four corrections to the model this plan sketched:

- **The branch vector is BRANCH instructions, not inert cells.** §6.87 prints the return code at `LOC` as `OP DESCR1 / BRANCH LOC1 / ... / BRANCH LOCM`, so `PC = LOC+N` lands on a real branch and executes it. Emitting them as data traps at the first return.
- **`STPRNT`'s format is a string structure, not a specifier.** §6.114's figure puts the length in the title's value field and the characters at `A2+4D`; the source's `BCDFLD EQU 4*DESCR` and §6.13's "4 descriptors (including the title) in a string structure in addition to the string itself" agree. The statically assembled `STRING` formats reach that shape through initialization — line 322 runs `GENVAR` over each one and stores the structure — which is also how a format built at run time arrives (§2.1).
- **The return slot is assembly-time data.** The descriptor the value comes back in is known when the `RCALL` is assembled, so the assembler writes it and `RCALL` only checks it. `RRTURN` asserts the cell is a return point, which is where a corrupted stack traps.
- **`Cells` on the table entry became `Size`**, and `MinArgs` a method — see M4.

The `Host` interface has one operation, `Print`, because `STPRNT` is the only thing that needs one yet. `INIT` is a documented partial: no dynamic storage and no timer, so `FRSGPT`, `HDSGPT` and `TLSGP1` are not set.

**M6 — Instruction batches by §7.5 classification**, cheapest-first, prioritised within batch by §7.4 frequency: descriptor move/set → address-field arithmetic → comparisons → flags → value/size → specifiers → tree/pattern nodes → real numbers → I/O → OS-dependent.
*Exit per batch:* every instruction has a test covering operand validity, state change, PC behaviour, branch behaviour, and error behaviour (`AGENTS.md`'s Definition of Done).

Batches 1 and 2 are done, in `pkg/sil/descriptors.go`: `GETD` `GETDC` `PUTD` `PUTDC` `MOVDIC` `MOVBLK` `ZERBLK` `PUSH`, and `ADJUST` `BKSIZE` `DECRA` `INCRA` `GETAC` `PUTAC` `GETSIZ` `GETLG` `GETLTH` `MOVA` `SETAV`. With `MOVD`, `POP` and `SETAC` from M5 that is both §7.5 groups complete, and the top six of §7.4's frequency table.

Alongside the unit tests, `pkg/sil/asm/testdata/descriptors.sil` is a SIL program that checks itself: fifteen round trips and identities the document states, compared with `ACOMP`, ending with the number of the check that failed or zero. Two things it caught that a per-operation test would not have: a `MOVBLK` that copies from the title rather than past it moves the same data to the same place, so only §6.66 note 1 — "the descriptor at A1 is not altered" — distinguishes it.

Two operations transfer to a label rather than faulting, and they are different labels: `POP` underflow goes to `INTR10` (§6.77 note 1, §7.3) and `PUSH` overflow to `OVER` (§6.80 note 1), which the source defines at line 5233 as `SETAC ERRTYP,21`.

Batches 3, 4 and 5 follow in `pkg/sil/compare.go` and `pkg/sil/fields.go`: the comparisons `AEQL` `AEQLC` `AEQLIC` `CHKVAL` `DEQL` `LCOMP` `LEQLC` `LEXCMP` `TESTF` `TESTFI` `VCMPIC` `VEQL` `VEQLC`, the flag operations `SETF` `SETFI` `RESETF` `RSETFI`, and the value-field ones `INCRV` `MOVV` `PUTVC` `SETSIZ` `SETVA` `SETVC`. `RCOMP` is the only comparison left and waits for real numbers. `pkg/sil/asm/testdata/compare.sil` checks all of them the same way, with 37 checks that branch to `FAIL` on the arm the document says they must not take.

**§6.53 is wrong about `LEXCMP` and the source proves it.** The prose reads "If C11...C1N1 < C21...C2M, transfer is to GTLOC ... if > ... LTLOC", which inverts the operand names. Eleven of the twelve `LEXCMP` sites give `GTLOC` and `LTLOC` the same target — §6.53 note 5 says that is the usual case, which is why the error survived — but `LGT` at line 4485 does not:

```
LGT    PROC    ,
       ...
       AEQLC   YPTR,0,,RETNUL      Null is less than anything
       LEXCMP  XSP,YSP,RETNUL,FAIL,FAIL
```

`LGT(X,Y)` succeeds when X is lexically greater, and `RETNUL` is how a primitive succeeds, so `GTLOC` means `SPEC1` greater. Implemented that way, with the deviation recorded in the method's doc comment and checked by `compare.sil`.

Flags are set and reset bitwise rather than by the addition §6.101 and §6.91 write, because their own note 1 -- "the other flags are left unchanged" -- and §3.1.2's "a set of bits that are individually tested, turned on, and turned off" require it: adding a flag already present would carry into the next one.

The integer arithmetic group follows in `pkg/sil/integers.go`: `SUBTRT` `MULT` `MULTC` `DIVIDE` `EXPINT` `MNSINT`, joining `SUM`, `INCRA` and `DECRA`. They share one helper, which is what keeps a failed operation from leaving a half-computed descriptor behind: out of range takes `FLOC` and writes nothing. `pkg/sil/asm/testdata/integers.sil` checks each result and each failure arm from SIL.

§6.32 does not say what a nonzero base raised to a negative power gives. Its own words settle it rather than a guess: `FLOC` is taken "if the result is out of the range available for integers", and a proper fraction is not an integer at all — except for the two bases whose negative powers still are, 1 and −1.

The specifier batch follows in `pkg/sil/specifiers.go`: the whole-specifier moves `SETSP` `GETSPC` `PUTSPC`, and the ones that alter parts of a specifier — `ADDLG` `APDSP` `FSHRTN` `GETBAL` `INTSPC` `LOCSP` `PUTLG` `REMSP` `SETLC` `SHORTN` `SUBSP` `TRIMSP`. That is §7.5's first two groups complete except `STREAM`, which is M7. `pkg/sil/asm/testdata/specifiers.sil` checks all fifteen with 28 checks, mostly by comparing what an operation produced against a `STRING` of the answer with `LEXCMP`.

Three things the sections did not say outright:

- **§6.90's prose and its figure disagree, and the figure is right.** `REMSP` is described as "the deletion of a specified length at the end", which would give `O2,L2-L3`; the figure gives `O2+L3,L2-L3`, and so does note 3's "see also `FSHRTN`". The source settles it — all six sites read "Get specifier to unscanned portion" (line 2532), "Remove part matched" (line 2800), "Get tail of subject" (line 2272), and in each `SPEC3` is what was just matched at the front.
- **`INTSPC`'s buffer is the machine's, not the program's.** §6.49 note 2 calls it "local to `INTSPC`", nothing in the source names it, and its contents are promised to nobody past the next `INTSPC`. It is taken from past the end of the assembled image the first time `INTSPC` runs, so the image is exactly what the assembler laid out and a core listing is unaffected. Putting it in `MDATA` was the alternative and was rejected: `copyseg`'s corpus test asserts that the segments define exactly the contract the source needs and nothing more.
- **`LOCSP`'s offset `4*CPD` is where a string structure's characters begin.** §6.13 gives a structure four descriptors including the title, the source says the same thing as `BCDFLD EQU 4*DESCR`, and CPD is `DESCR*CPA`. §6.37's `GETBAL` needed one thing the figure does not draw: `N = 0` leaves no `CL+1` to examine, so no balanced string exists and transfer is to `FLOC`.

The tree and pattern-node batch follows in `pkg/sil/nodes.go`: `ADDSIB` `ADDSON` `INSERT` `MAKNOD` `CPYPAT`, checked from SIL by `pkg/sil/asm/testdata/nodes.sil`. One reading decision governs the whole file and turned out to be checkable:

- **A blank field in a figure is not a zero.** These entries name some fields of an altered descriptor and leave others blank — `ADDSIB` alters only the value field at `A3+CODE`, `MAKNOD` only the address fields of two of the four descriptors it writes. §6.100 settles it: `SETAV`'s figure reads `V 0 0`, with the zeros written, so a field the document does not name is a field the operation does not touch. Both the unit tests and `nodes.sil` give those fields a value beforehand, so leaving them alone is checked rather than assumed.
- **§6.21's two figure headings contradict each other and the source says which is right.** The input of `R2+4D` is guarded by "if V7 = 3" and the output of `R1+4D` by "if V3 = 7". `V7` is the value field of what `MAKNOD` wrote from its `DESCR5`, and that value is the node's size: the three-descriptor nodes are built from `NMECL`, value 2 (line 5521), the four-descriptor ones from `ATOPCL` and `CHRCL`, value 3 (lines 5501, 5504) — and those are exactly the `MAKNOD` calls that pass a `DESCR6`. The section stride `(1+V7)*D` says the same thing again. `V3` is never read at all.
- **`FATHER`, `LSON`, `RSIB` and `CODE` are the program's**, by §6.4 note 2, so the machine reads them out of the assembly. An assembly that leaves one undefined faults rather than guessing.

**Remaining, by §7.5 group:** real numbers (11), miscellaneous (9), I/O (5), OS-dependent (5), the rest of the stack group (`PSTACK`, `SELBRA`, `BRANIC`, `SPOP`, `SPUSH`), and `STREAM`/`CLERTB`/`PLUGTB`, which belong with M7. §7.1 marks about thirty of these optional, each with the language feature it disables.

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
4. **Call model wrong** — M5. Retired. A 44-line program takes both returns, and `PC = LOC+N` is the whole dispatch, as long as the vector is assembled as BRANCH instructions.
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
