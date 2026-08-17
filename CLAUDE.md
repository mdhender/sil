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

`go.mod` declares `go 1.16`; the toolchain in use is newer. Run fmt/vet/test before considering a change complete.

## State of the code

Early skeleton, well before Milestone 1. There is no lexer, parser, assembler, image format, or fetch/execute loop yet — only a handful of instruction methods hung on a stub machine. Do not assume infrastructure described in `README.md` exists.

- `pkg/sil` — the abstract machine. `am` in `types.go` currently holds only `pc`. `descriptor`, `specifier`, `characterString`, and `syntaxTableEntry` are transcribed from S4D58 §3 and carry the documentation in their comments.
- `cmd/sil` — the runner entry point; empty.
- `README.md`'s "Proposed Repository Layout" (`asm/`, `sil/`, `image/`, `runtime/`) is an aspirational sketch and does **not** match the tree. The actual layout after the `refactor project layout` commit is `cmd/` + `pkg/`. Introduce packages when implementation pressure justifies them, per AGENTS.md.

## Instruction conventions

Follow the pattern established by `pkg/sil/macros.go` (`ACOMP`, `ACOMPC`) when adding an instruction:

- Method on `*am` named exactly as the SIL opcode, uppercase: `func (s *am) ACOMP(...)`. Receiver is `s`.
- Preceding doc comment paraphrases the S4D58 entry — description, `Data Input` / `Data Altered`, `Programming Notes`, cross-references — and ends with a section citation: `// S4D58.PDF: 6.1`. This is the primary way documented semantics stay attached to the code; keep it up.
- Branch-style instructions assign the destination to `s.pc` directly on each arm (see `ACOMP`'s GTLOC/EQLOC/LTLOC), rather than returning a target.
- Every `.go` file starts with the BSD 2-clause header block found in the existing files.
- `SETAC`, `Add`, and `Branch` are empty stubs; `add.go` keeps the historical SIL source for `ADD` in a comment as the specification to implement against.

## Uncommitted source material

`engines/` and `references/` are gitignored except for their `.gitignore` and `MANIFEST.md` — the SIL source and the Arizona PDFs are assumed not redistributable. A working checkout needs these fetched locally:

- `engines/sil-v3.11.sil` — the historical Macro SNOBOL4 SIL source (~6600 lines), the eventual assembler input and requirements list. Origin recorded in `references/MANIFEST.md`: `https://raw.githubusercontent.com/atdt/snoflake/master/external/v311.sil`.
- `references/s4d58-sil-v3.11.pdf` — Griswold, *Implementing SNOBOL4 in SIL: Version 3.11*, S4D58 (Feb 1981). The primary instruction-set reference; every instruction comment cites it. Also `s4d54` (transporting), `s4d57` (implementation), `s4d59` (terminology), `s4n24` (errata).

`engines/README.md` says the source is embedded in the application, so the directory must not be empty — there is no `//go:embed` directive in the tree yet.

Treat the historical `.sil` source as read-only input. Do not reformat, rename labels, or restructure it; patch only for portability, minimally, and document why.
