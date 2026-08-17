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

|     | Milestone                                  | Status                             | Where                               |
|-----|--------------------------------------------|------------------------------------|-------------------------------------|
| M0  | Scanner                                    | **done** — `86a6d81`               | `pkg/sil/scanner`                   |
| M1  | Operand parser                             | **done** — `adbbae0`               | `pkg/sil/parser`                    |
| M2  | The symbol gate                            | **done** — `c1a4ccc`               | `pkg/sil/symtab`                    |
| M3  | Externals chosen; layout closes            | **done**                           | `pkg/sil/copyseg`, `pkg/sil/layout` |
| M4  | Instruction table and shape validation     | **done**                           | `pkg/sil/op`                        |
| M5  | First vertical slice runs                  | **done**                           | `pkg/sil`, `pkg/sil/asm`            |
| M6  | Instruction batches by §7.5 classification | **done** — 116 of 119; the other three are M7 | `pkg/sil` |
| M7  | Syntax tables and `STREAM`                 | **done** | `pkg/sil`, `pkg/sil/syntab` |
| M8  | The historical source assembles clean      | **done** | `pkg/sil/asm` |
| M9  | Execution to first trap, then to `ENDEX`   | **done** | `pkg/sil/asm` |
| M10 | First SNOBOL4 program                      | **done** | `pkg/sil/asm` |

The front end (M0–M3) is complete: the whole 6,580-line source scans, parses and resolves with no diagnostics, the 37 undefined names it derives are exactly the machine-dependent contract, and with the three COPY segments in it lays out into 17,216 address units with every symbol valued and every expression well formed.

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

The real-number batch follows in `pkg/sil/reals.go`: `ADREAL` `SBREAL` `MPREAL` `DVREAL` `EXREAL` `MNREAL` `INTRL` `RLINT` `REALST` `SPREAL`, and `RCOMP`, the last of the comparisons. `pkg/sil/asm/testdata/reals.sil` checks all eleven with 19 checks. §7.1 marks the whole group optional, at the cost of real arithmetic; it is implemented rather than trapped.

**A real number lives in the address field.** Every figure in §6 draws it there — §6.7 prints `DESCR2` as `R2 F2 V2` — and §3.1.1 says it in words: "the address field must also be large enough to contain any integer or real number (including sign)". §5.3 keeps the two `R`s apart: the `R` in an address field is a real number, the `R` in a value field is the data type code the source defines at line 298. The address field is a Go `int`, so a real is held there as its IEEE 754 bit pattern, and that is what makes it travel: `MOVD`, `GETD`, `PUSH` and the garbage collector move descriptors without knowing what is in them, exactly as on a machine where the address field is a word that sometimes holds an address, sometimes an integer and sometimes a float. The cost is that this machine needs a 64-bit `int`, which a constant in the file checks at compile time.

Two smaller things:

- **§6.88's second sentence sends `R1 = R2` to `GTLOC`.** `EQLOC` is in its own operand list and no other three-way comparison in §6 sends two arms to one place, so it is a slip, and `RCOMP` is implemented as §6.1 words the same case.
- **`SPREAL` does not require the decimal point.** §6.112 note 2 says what the caller supplies rather than what the operation must reject, and note 3 — an empty string is 0.0 — shows it is not a strict literal parser. The source agrees from the other side: `CNVVI` (line 4716) and `CONVR` (line 4703) both try `SPCINT` first and only reach `SPREAL` when the string is not an integer, so a string of digits alone never arrives. What is rejected is everything outside a sign, digits and one point — the exponent, infinity and not-a-number forms Go's own parser would take.

`REALST` gets its own scratch buffer rather than sharing `INTSPC`'s, because §6.49 note 2 and §6.89 note 3 each promise only that a buffer survives until the next use of *that* operation. Both come from past the end of the assembled image; `VM.buffer` is now the one place that does it.

The rest of the stack group and the two remaining branch operations follow in `pkg/sil/control.go`: `PSTACK` `SPUSH` `SPOP` `BRANIC` `SELBRA`, checked from SIL by `pkg/sil/asm/testdata/control.sil`. A specifier is stacked exactly as a pair of descriptors would be — §3.2's "specifiers and descriptors may be stored in the same area indiscriminately" — so `SPUSH` and `SPOP` are `PUSH` and `POP` with `S` in place of `D`, with the same two destinations for overflow and underflow.

`SELBRA` is the point of the batch, and it retires the second half of risk 4. `PC = addr(SELBRA)+I` is the whole of notes 1 and 2: the assembler emits the locations as `BRANCH` instructions and resolves an omitted one to the operation after the vector, so `I <= N` lands on a branch and `I = N+1` lands past them with no case written for either. `control.sil` takes all four arms.

Two adjustments the batch forced:

- **The assembler now writes `N` into the `SELBRA` cell.** §6.98 note 3 asks for a check that `I` is in `1..N+1`, and the machine cannot recover `N` from core — a `BRANCH` belonging to a `SELBRA` is indistinguishable from any other. This is the one place PLAN's "the VM never needs to know M" is not quite free; it costs one operand and buys note 3.
- **`BRANIC` adds `N` rather than asserting it is zero**, which is a deviation from this plan. §6.16 note 1 is a statement about the SNOBOL4 source, not a restriction on the operation — `N` is in §6.16's box like any other operand — and the arithmetic is the same either way, so faulting on a nonzero one would be a restriction this machine invented. Nothing is lost: `N` is resolved by the assembler and cannot change at run time, so the assertion could only ever fire on a source the document permits.

The miscellaneous batch follows in `pkg/sil/misc.go`: `LINKOR` `LOCAPT` `LOCAPV` `LVALUE` `ORDVST` `RPLACE` `SPCINT` `TOP` `VARID`, checked from SIL by `pkg/sil/asm/testdata/misc.sil`. `LOCAPV` is 5.2% of execution time, the fifth entry in §7.4's table.

Three things this batch had to settle:

- **The chain of alternatives is not defined in any one section.** `LINKOR` and `LVALUE` both walk it, and §6.61's figure gives the arithmetic: from a pattern at `A`, the first alternative field is at `A+2D` and holds `N1`, the next is at `A+N1+2D`. So a field holds the offset of the next node *from the base of the pattern*, not from itself. `CPYPAT` confirms it from the other side — copying `X | Y` (the source's `ORPP`, line 2171) copies `X` with `A4 = 0` and `Y` with `A4 = XSIZ`, and §6.21 relocates each alternative field by `F1(X) = X+A4`. A self-relative link would need no relocation when `Y` moves as a unit; one measured from the base needs exactly `XSIZ`. The `LINKOR` on line 2180 then writes `XSIZ` into the end of `X`'s chain, which is `Y`'s first node measured from that same base. Both walks carry a loop guard, since a corrupt chain is otherwise unbounded.
- **The attribute list includes the descriptor at `A+2K*D`.** §6.58's figure draws that row empty, which reads as a terminator, but a block in this system is a title and then the storage its value field measures — the source writes `BLOCK DESCR BLOCK,TTL+MARK,LEN*DESCR` and then `ARRAY LEN` — so the elements run from `A+D` through `A+2K*D` inclusive and the last value descriptor is the last element. `misc.sil` check 14 is exactly that boundary; with an exclusive bound it fails.
- **`ORDVST` is the documented alternative rather than the operation.** §6.74 note 1 offers "perform no operation", §7.1 lists it with that alternative and names what it disables as alphabetization of the post-run dump, and that is what is implemented. The reason is note 3: sorting has to leave out variables whose value is the null string, and §6.74 draws only the parts of a variable it calls relevant — the length at `A`, the link at `A+3D`, the characters at `A+4D`. Where a variable's *value* lives is not in that section, so implementing note 3 would mean deciding it from the source. The single call site, line 5118, runs once at the end of a run and only under `&DUMP`. This is the one operation in M6 not implemented as written.

`VARID`'s hash is this machine's choice: §6.127 note 4 offers an algorithm rather than requiring one, and notes 1 to 3 are the specification — two functions, uncorrelated with the characters and with each other, `K` a descriptor-aligned offset no larger than `(OBSIZ-1)*D` and `M` no larger than `SIZLIM`. FNV-1a forwards and backwards gives both, with no seed, so a run is reproducible.

The input/output and system-dependent batches finish M6, in `pkg/sil/io.go` and `pkg/sil/system.go`: `OUTPUT` `STREAD` `BKSPCE` `ENFILE` `REWIND`, and `MSTIME` `DATE` `LOAD` `LINK` `UNLOAD`. `pkg/sil/asm/testdata/io.sil` runs all ten, and is the first program whose checks are split between the two sides — what the host was asked for is only visible from Go, so the SIL program checks what comes back and the test checks what went out.

This is where the `Host` interface grew from one operation to eight, along the boundary §2.1 draws: SNOBOL4 does its input and output through FORTRAN IV routines and names files by unit reference number, so the machine resolves the number and hands over bytes, and nothing on this side interprets a format. `WriterHost` answers everything it cannot do the way S4D58 licenses — no clock (§6.71 note 4), no calendar (§6.22 note 4), and positioning a writer does nothing rather than failing — which is also what keeps a test run reproducible.

Three decisions:

- **The assembler passes a `FORMAT`'s length with its address.** §6.75's figure gives `OUTPUT`'s format as characters at a location and never says how many, because a FORTRAN IV routine reads to the closing parenthesis. This machine does not read formats, so the count travels alongside; it is the one thing about a `FORMAT` operand that only the assembler knows. Same shape as `SELBRA`'s `N`.
- **`STREAD` pads a short record with blanks.** §6.115 note 1 defers to "FORTRAN IV conventions regarding truncation or reading of additional records", and a FORTRAN A-format read of L characters pads. `L` goes across to the host, so a host that would rather read additional records to fill it may.
- **The external-function group takes §7.1's alternatives**, which footnote 4 says to apply together: `LOAD` branches to `UNDF` (§6.57 note 2 — a program error, so not `INTR10`), `LINK` to `INTR10` (§6.55 note 2 — nothing reaches a `LINK` without a `LOAD` having succeeded, so getting there is an implementation error), and `UNLOAD` performs no operation, which §6.126 note 2 requires rather than permits: the source-language `UNLOAD` undefines ordinary functions too and calls the macro on the way through. This retires risk 10.

**M6 is done.** All 116 executable operations outside M7 are implemented, each with unit tests and at least one assembled SIL program: `slice.sil` `descriptors.sil` `compare.sil` `integers.sil` `specifiers.sil` `nodes.sil` `reals.sil` `control.sil` `misc.sil` `io.sil`. The three left are `STREAM`, `CLERTB` and `PLUGTB`, which need the syntax tables and are M7's own scope. One operation, `ORDVST`, is implemented by the documented alternative rather than as written; see the miscellaneous batch above. §7.1 marks about thirty of these optional, each with the language feature it disables.

**M7 — Syntax tables and `STREAM`.** `STREAM` (35 sites), `PLUGTB`/`CLERTB` (4+4), MDATA generated from Appendix A. Lower risk than it looks: §4.2 states only `SNABTB` is ever mutated, so tables are immutable data with one exception.
*Exit:* a SIL program that plugs `SNABTB`, runs `STREAM`, and reproduces `ANY`/`BREAK`/`NOTANY`/`SPAN`.

The three operations are done, in `pkg/sil/syntax.go`, and the exit criterion is met by `pkg/sil/asm/testdata/stream.sil` — the four combinations of `CLERTB` and `PLUGTB` the source uses at lines 2521 to 2581, with 23 checks. Nothing in it needs a table but `SNABTB`, which §4.2 says is the only one modified during execution and which `CLERTB` builds from nothing every time. That is 119 of 119 operations dispatched; the twelve directives assemble rather than execute.

- **§4.2 lists six actions and only three need code.** `CONTIN`, `GOTO(TABLE)` and `PUT(ADDRESS)` are not indicators: `CONTIN` and `GOTO` both say which table reads the next character, which is the entry's address field either way — §6.19's figure for `CLERTB CONTIN` sets that field to the table itself, so "the current table" and "some other table" are one mechanism — and `PUT` is the value field, which `STREAM` carries along and drops into `STYPE`. Only `STOP`, `STOPSH` and `ERROR` stop the streaming.
- **§6.116's transfer sentence drops a clause.** It reads "if TJ is ERROR, transfer is to ERRROR, while if if TJ is STOPSH, transfer is to SLOC. Otherwise transfer is to RUNOUT", which leaves `STOP` nowhere to go and makes `RUNOUT` mean two things. The sentence before it defines J over "ERROR, STOP, or STOPSH" and the figures give `STOP` its own arm, so it reads "STOP or STOPSH", and `RUNOUT` is the case where no such J exists. Note 2 confirms it from the edge: the null string is `RUNOUT`.

The twenty-five tables of Appendix A follow in `pkg/sil/syntab`, and the assembler loads them.

**The appendix is held verbatim and expanded at load time.** §4.2 asks for exactly that: the tables "are generated from such descriptions using a (SNOBOL4) program in which the character classes and the order of the internal character codes are parameters. The use of some kind of automatic technique to generate the syntax tables is advisable, both to ensure accuracy and because of the large amount of data involved." So `syntab.Appendix` is the 163 lines of the appendix as written — a corpus test compares them against the manual line by line and they match exactly — with a parser for the `BEGIN/FOR/ELSE/END` language and §4.1's character classes in ASCII beside it. The two parameters §4.2 names are the two things supplied over the appendix: the classes, and `ALPHSZ`.

The expansion happens in two steps, which is how §6.20 note 1's "COPY may simply expand into the data required" is split here: `MDATA` declares each table as `ARRAY ALPHSZ`, the right storage with nothing in it, and `pkg/sil/asm` fills the three fields of every entry after emission. The alternative — generating 6,400 lines of `DESCR` into `MDATA` — was rejected because a table's `PUT` codes are symbols of the SNOBOL4 source rather than of the segments, and `copyseg`'s contract test exists to keep the segments from reaching into the program.

Two things the descriptions turn out to guarantee, and are checked rather than assumed:

- **No table names two classes that share a character**, though the classes themselves overlap freely — `DOT`, `BREAK` and `CNT` all contain the full stop, and `TERMINATOR` contains `BLANK`. The appendix never says which line would win, so `Build` refuses an overlap instead of guessing a precedence rule.
- **Every `GOTO` names a table the appendix describes**, and every class a description names is one §4.1 defines — in both directions, which is what catches a class transcribed under the wrong name. All 35 of §4.1's classes are used and all 54 `PUT` symbols are defined by the SNOBOL4 source.

**M8 — The historical source assembles clean.** *Exit:* zero diagnostics over 6,580 lines; entry point is the address of `BEGIN` (line 303, `BEGIN INIT ,` — §6.46 makes `INIT` the first instruction executed; `MLINK` supplies nothing the machine needs); a core listing is byte-stable across runs.

Done, in `pkg/sil/asm`'s corpus test. All three criteria hold: the 6,580 lines go to 17,216 units of core with no diagnostics, `BEGIN` is at address 0 — everything above line 303 is `TITLE`, `COPY` and `EQU`, which assemble nothing — and two assemblies of the same source give byte-identical listings. Every cell in core cites the line that assembled it, and all twenty-five syntax tables are filled with what Appendix A says they hold.

It fell out of M6 and M7 rather than needing work of its own, which is what the front-end-first order was for: M2's symbol gate, M3's layout and M4's shape check had each already been run over the whole source, so the only thing M8 added was emission, and the only thing emission needed was the operations.

**M9 — Execution to first trap, then to `ENDEX`.** *Exit:* every halt is a documented `ENDEX` or a named unimplemented opcode. Track instructions-executed-before-first-trap as a monotone number in the changelog.

Done. The number was tracked twice and then stopped being interesting:

- **23,220 instructions**, and the trap was `BKSIZE` walking off the end of core in the garbage collector. `INIT` was a documented partial from M5 — no dynamic storage — so `FRSGPT` was zero and `GCLAD`'s block walk had no region to end at. The banner had already printed by then.
- **Everything after that is `ENDEX`.** `INIT` now obtains dynamic storage the way §6.46 allows — "space for dynamic storage may be preallocated or obtained from the operating system by `INIT`" — growing core by `DefaultDynamic` descriptors, which §2.2 puts at "about 10,000", and setting the address fields of `HDSGPT`, `FRSGPT` and `TLSGP1`. It is all three or none: every test program here names none of them and does not use dynamic storage, and a program naming some but not all would run against a region half of it cannot see. There is still no timer, which §6.71 note 4 makes optional and `MSTIME` asks the host for.

**M10 — First SNOBOL4 program.** `X = 'HELLO'; OUTPUT = X; END`, compared against a reference SNOBOL4.

Done, in `pkg/sil/asm`'s corpus test. The historical Macro SNOBOL4 implementation compiles the program, executes it, prints `HELLO`, prints its statistics — two statements executed, none failed, one write — and terminates through `ENDEX` at status zero.

The program's own output and the system's are kept apart because they take different paths: the source-language `OUTPUT` variable reaches the host through `STPRNT`, and everything the system says about itself goes through `OUTPUT` (§6.75). Both go under a FORTRAN IV format, which `pkg/fortran` reads, so the test asserts the lines the system was documented to print — "NO ERRORS DETECTED IN SOURCE PROGRAM", `              1 WRITES PERFORMED` — rather than the format strings it printed them with. The `-UNLIST` control card is in the test program so that the compilation listing, which shares the `STPRNT` path, does not mix into the program's output — and getting that far exercises `CARDTB`, the syntax table that tells a control card from a statement.

Eight more programs run alongside it, each reaching a subsystem the first does not: integer arithmetic with precedence, concatenation, `SIZE`, real arithmetic through `SPREAL`/`ADREAL`/`REALST`, pattern matching, `SPAN` through `CLERTB`/`PLUGTB`/`STREAM` over `SNABTB`, `REPLACE` through `RPLACE`, a loop through the interpreter's goto field, and a defined function, which is the call model of M5 running under the SNOBOL4 compiler rather than under a test program. All nine give the right answer.

## What is left

The ten milestones are done. What the machine does not have:

- ~~A FORTRAN IV format interpreter~~ (risk 9). Done; see below.
- ~~A runner.~~ Done. `engines/engines.go` embeds the SIL source and `cmd/sil` runs SNOBOL4 programs on it: `sil hello.sno` prints `HELLO`. Standard output is what the program printed and standard error is what the system printed about itself, because the two leave the machine by different operations — `STPRNT` hands over the characters of a string, `OUTPUT` hands over a FORTRAN format — and only one of them is finished. `-merge` gives the single stream the original printed. Reaching `ENDEX` is exit 0 whatever the program did; `ENDEX`'s operand is the keyword `&ABEND` (§6.29 note 2), not a status, and a nonzero one is reported rather than silently dropped, since this machine has no core dump to give (§6.29 note 1 allows that).
- **`ORDVST`**, which is §6.74 note 1's documented alternative rather than the operation; the post-run dump is unalphabetized. See the miscellaneous batch.
- **`LOAD`, `LINK` and `UNLOAD`**, which are §7.1's alternatives for a machine with no external functions.

## The FORTRAN IV format interpreter

`pkg/fortran`, and risk 9 with it. The boundary did not move: `Host.Print` and `Host.Output` still take the format undigested, which is where §2.1 puts it — "formats used by `STPRNT` are strings that may be formed during program execution and hence must be accepted in their undigested form" — and `cmd/sil` is a host that reads them.

This is the one part of the system S4D58 does not specify. §6.114 note 1 and §6.34 both say "FORTRAN IV format" and stop; the specification is ANSI X3.9-1966 §7. What settles the reading is that the source's own formats produce what the system was documented to print, so the tests are those formats with the values the system passes them.

Six things are needed and all six are in the source's twenty-six `FORMAT` statements and two format strings: Hollerith literals, `Iw`, `Fw.d`, `Aw`, `nX` and `/`, with repeat counts and groups. `Ew.d` and quoted literals are there too, because §2.1 lets a SNOBOL4 program build a format at run time; anything else is an error that names itself rather than a field quietly printed wrong.

Four things were not obvious:

- **Numbers arrive as bits and the format decides what they are.** §6.75 gives `OUTPUT` "the conversion of integers and real numbers given in the address fields", and §3.1.1 puts both in the same field, so which one a value is cannot be read off the value — `I` or `F` is what says. `(1H0,F15.2,...)` is the only real field the system has, and it is fed by `DVREAL`; the corpus test drives it with a clock that ticks a second per `MSTIME` and asserts `         500.00 MS. AVERAGE PER STATEMENT EXECUTED`, which is the whole of that path in one line.
- **A Hollerith count is where a comma stops being a separator.** `(1H0,I15,21H STATEMENTS EXECUTED,,I8,7H FAILED)` looks like it has an empty field. It does not: the twenty-one characters end with the comma, and the `,,` is that comma followed by the separator. `(42H0BELL TELEPHONE LABORATORIES, INCORPORATED,/1H1)` is the same trick the other way — the count stops one short of the trailing comma.
- **Carriage control belongs to the printer, not to the format.** The first character of a record says how far to advance and is not printed. So it is applied by `Lines` and only for the print unit: the SNOBOL4 `PUNCH` variable goes out on unit 7 under `(80A1)`, where column one is data, and eating it would lose the first character of every punched record.
- **Reversion is what wraps a long line.** Reaching the end of a format with the list unfinished starts a new record and goes back to the last group at the outer level, or to the beginning if there is none. `(1X,132A1)` therefore wraps at 132 columns rather than truncating. The same rule read the other way is what makes it print a five-character string as six characters rather than padding to a hundred and thirty-three: format control stops at the first field the list cannot fill.

Overprinting is the one thing a stream of text cannot do. `+` means "do not advance", and two records on one line is something a printer does with ink; the record gets a line of its own, which is what the one place the system uses it wants anyway — the banner underlines `SNOBOL4` by overprinting seven underscores.

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
9. **FORTRAN formats** for the 26 `FORMAT` literals — M6 I/O batch, entirely behind `Host.Print`. Retired, in `pkg/fortran`; the boundary never moved. See below.
10. **`LOAD`/`UNLOAD`/`LINK`** — one site each, 0.000% execution time. Documented trap, or never. Retired in M6, by §7.1's own alternatives: `UNDF`, `INTR10` and no operation, applied together as footnote 4 requires.

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
