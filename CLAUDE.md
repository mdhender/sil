# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Read AGENTS.md first

`AGENTS.md` is the authoritative working agreement for this repo (agent role, milestone order, testing layers, guardrails). It is not summarized here — read it before making changes. The two rules that override everything else:

- **Implement SIL, not SNOBOL4.** Never satisfy a failing SNOBOL4 program by recognizing what its code is "trying to do" and writing the equivalent in Go.
- The developer normally writes the production implementation; act as guide, reviewer, and test-writer unless asked to implement.

## Commands

```bash
go build ./...
go fmt ./...
go vet ./...
go test ./...

go test ./pkg/sil -run TestACOMP -v   # single test
go run ./cmd/sil                      # runner (currently an empty main)
```

Run fmt/vet/test before considering a change complete.

## State of the code

`PLAN.md` holds the milestone table and is the authority on what is done. As of M6 batch 5 a SIL program goes from source through the assembler into the machine and runs. Fifty-four of the 119 operations are implemented; the rest fault with their own name.

- `pkg/sil/diag` — accumulated diagnostics with source locations. Every stage appends and keeps going.
- `pkg/sil/scanner` — columns to fields (S4D58 §7.6). Knows no operations.
- `pkg/sil/parser` — the operand field to items and expressions. Knows no operations.
- `pkg/sil/copyseg` — the `PARMS`, `MLINK` and `MDATA` segments as embedded SIL text, and `COPY` expansion into the line stream. This is where every machine-dependent constant is chosen. The only per-operation knowledge in the front end.
- `pkg/sil/symtab` — definitions and references; closes the reference graph without any operation knowledge.
- `pkg/sil/op` — the instruction table: all 131 operations of §6 with their operand signatures, §7.5 classification, size shape and section citation. One table; `Lookup`, `String`, the shape checker and the location counter all read it. Do not add a parallel enum, stringer, or lookup file.
- `pkg/sil/layout` — the location counter and symbol values, with the relocatable/absolute discipline. Reads sizes from `op`.
- `pkg/sil` — the machine. `Cell` and `Core` (`cell.go`), `VM`/`Step`/`Run` (`vm.go`), the `Host` boundary (`host.go`), and one method per operation (`macros.go` for the vertical slice, `descriptors.go` for §7.5's descriptor and address-field groups, `compare.go` for the comparisons, `fields.go` for the flag and value-field groups).
- `pkg/sil/asm` — `Assemble(file, src, Options) (*sil.VM, diag.List)`: the whole front end in one call, then emission into core. There is no image format and none is wanted yet.
- `cmd/sil` — the runner entry point; empty.
- `internal/corpus` — locates the historical source for tests and holds the counts they assert. Corpus tests skip when it is absent; **a skip is not a pass**.
- `README.md`'s "Proposed Repository Layout" (`asm/`, `sil/`, `image/`, `runtime/`) is an aspirational sketch and does **not** match the tree. Introduce packages when implementation pressure justifies them, per AGENTS.md.

## Instruction conventions

Follow the pattern established by `pkg/sil/macros.go` (`ACOMP`, `ACOMPC`) when adding an instruction:

- Add its entry to the dispatch in `VM.execute`; the operand signature is already in `pkg/sil/op`'s table.
- Method on `*VM` named exactly as the SIL opcode, uppercase: `func (s *VM) ACOMP(...)`. Receiver is `s`.
- Operands are addresses in core, or constants, already resolved. Every branch point is a real address — the assembler turns an omitted one into the fall-through (§5.2) — so no method handles "omitted".
- Return `error` only if the operation can fail. A condition the SNOBOL4 system handles itself goes through `s.interrupt`, which branches to `INTR10` (§7.3); a machine failure goes through `s.fault`.
- Preceding doc comment paraphrases the S4D58 entry — description, `Data Input` / `Data Altered`, `Programming Notes`, cross-references — and ends with a section citation: `// S4D58.PDF: 6.1`. This is the primary way documented semantics stay attached to the code; keep it up.
- Branch-style instructions assign the destination to `s.PC` directly on each arm (see `ACOMP`'s GTLOC/EQLOC/LTLOC), rather than returning a target. `Step` has already advanced `s.PC`, so an operation that does not branch leaves it alone.
- Every `.go` file starts with the BSD 2-clause header block found in the existing files.

## Uncommitted source material

`engines/` and `references/` are gitignored except for their `.gitignore` and `MANIFEST.md` — the SIL source and the Arizona PDFs are assumed not redistributable. A working checkout needs these fetched locally:

- `engines/sil-v3.11.sil` — the historical Macro SNOBOL4 SIL source (~6600 lines), the eventual assembler input and requirements list. Origin recorded in `references/MANIFEST.md`: `https://raw.githubusercontent.com/atdt/snoflake/master/external/v311.sil`.
- `references/s4d58-sil-v3.11.pdf` — Griswold, *Implementing SNOBOL4 in SIL: Version 3.11*, S4D58 (Feb 1981). The primary instruction-set reference; every instruction comment cites it. Also `s4d54` (transporting), `s4d57` (implementation), `s4d59` (terminology), `s4n24` (errata).

`engines/README.md` says the source is embedded in the application, so the directory must not be empty — there is no `//go:embed` directive in the tree yet.

Treat the historical `.sil` source as read-only input. Do not reformat, rename labels, or restructure it; patch only for portability, minimally, and document why.
