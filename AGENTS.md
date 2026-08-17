# AGENTS.md

## Mission

This repository implements the SNOBOL Implementation Language (SIL) as a virtual machine in Go.

It also provides an assembler for documented SIL assembly source and, ultimately, a command-line environment capable of running the historical SNOBOL4 implementation on that VM.

The architecture is:

```text
SNOBOL4 SIL source
        |
        v
   SIL assembler
        |
        v
  assembled image
        |
        v
      SIL VM
        |
        v
SNOBOL4 program execution
```

The central rule is:

> **Implement SIL, not SNOBOL4.**

Do not bypass the SIL machine by implementing SNOBOL4 operations directly in Go.

## Primary Reference

The primary instruction-set reference is:

**Ralph E. Griswold, _Implementing SNOBOL4 in SIL: Version 3.11_, S4D58, February 1981.**

Other historical documents and implementations may clarify behavior, but they do not justify silently changing documented SIL semantics.

When documentation is ambiguous:

1. identify the ambiguity;
2. locate additional historical evidence;
3. write a test demonstrating the chosen interpretation;
4. document the decision.

Do not guess based on what a modern VM, compiler, or regular-expression engine would do.

## Agent Role

Act primarily as a **guide, reviewer, debugger, and research assistant**.

The developer should normally write the production implementation.

For a new feature:

1. explain the relevant SIL concept;
2. identify the documentation that defines it;
3. propose the smallest test case;
4. help write or review the failing test;
5. let the developer implement the production code;
6. review the implementation;
7. help diagnose test failures;
8. suggest refactoring only after correctness is established.

Do not autonomously implement large portions of the VM unless explicitly requested.

Do not jump several milestones ahead.

## Working Method

Develop the system as small executable slices.

Prefer:

```text
parse one instruction
-> assemble it
-> execute it
-> verify machine state
```

over:

```text
build complete assembler
-> build complete VM
-> attempt to run SNOBOL4
```

Every new instruction should ideally become executable soon after it can be parsed.

## Architecture Boundaries

Maintain clear boundaries between:

```text
SIL source syntax
assembler
assembled image
SIL machine
host environment
SNOBOL4 workload
```

Avoid leaking assumptions between these layers.

### Assembler

The assembler understands SIL source representation.

It should:

- tokenize and parse source;
- maintain symbols;
- resolve labels;
- validate operands;
- emit an executable representation;
- produce useful source diagnostics.

It should not implement instruction behavior.

### Assembled Image

The image is the contract between assembler and VM.

Keep it simple and inspectable initially.

Do not design an elaborate object-file format or linker unless the historical source or development workflow requires one.

### Virtual Machine

The VM owns SIL execution semantics.

It should:

- maintain machine state;
- fetch instructions;
- execute instructions;
- update the program counter;
- manipulate SIL-defined runtime structures;
- expose host services only through explicit boundaries.

It should not parse SIL source.

### SNOBOL4

The historical SNOBOL4 implementation is a program running on the machine.

Do not add VM instructions or Go shortcuts merely because they make a particular SNOBOL4 routine easier to execute.

If the source uses a documented SIL operation, implement that operation.

## Instruction Implementation

Keep SIL instruction names recognizable in the source tree.

For an instruction such as:

```text
INCRA WSTAT,1
```

there should be a straightforward path from:

```text
assembler opcode
```

to:

```text
VM implementation
```

to:

```text
instruction tests
```

Avoid translating SIL into an unrelated internal instruction language unless there is a demonstrated need.

### Instruction Tests

Every implemented instruction needs tests.

Tests should establish:

- valid operands;
- state changes;
- program-counter behavior;
- branch behavior if applicable;
- stack effects if applicable;
- memory effects if applicable;
- error behavior.

Where useful, test boundary values.

Treat these tests as an executable SIL specification.

## Machine Model

Do not model SIL as though it were a conventional modern CPU unless the documentation says so.

SIL concepts such as:

```text
descriptors
specifiers
addresses
data blocks
dynamic storage
```

should be represented according to their documented semantics.

A Go representation is an implementation detail.

For example, if a SIL descriptor has multiple logical fields, prefer a type that makes those fields visible rather than encoding the entire concept as an opaque Go value.

Do not use unsafe Go pointers to mimic historical machine addresses unless a compelling requirement emerges.

Prefer deterministic virtual addresses or indexes controlled by the VM.

## Assembler Development

The assembler should preserve SIL terminology and syntax.

Do not "improve" the assembly language into a new syntax.

Support the historical source as written wherever practical.

### Diagnostics

Assembler errors should include source location.

Prefer:

```text
snobol.sil:143: undefined symbol FOO
```

over:

```text
undefined symbol
```

Keep source locations in parsed structures long enough to provide useful diagnostics.

### Forward References

Expect assembly-language behavior such as labels referenced before their definition.

Use explicit symbol resolution rather than requiring source reordering.

### Parser Complexity

Start with the simplest parser that correctly accepts the required SIL source.

Do not introduce a parser generator without a concrete reason.

A handwritten lexer/parser is preferred initially.

## VM Execution

Keep the fetch/execute loop explicit and boring.

Conceptually:

```go
for !m.Halted {
    inst := m.Fetch()
    if err := m.Execute(inst); err != nil {
        return err
    }
}
```

The real API may differ.

Avoid clever dispatch mechanisms until profiling demonstrates a need.

Correctness and inspectability are more important than dispatch performance.

## Tracing

Design debugging support early.

At minimum, make it possible to trace:

```text
program counter
instruction
operands
```

without changing normal program output.

Later tracing may include:

```text
register changes
descriptor changes
specifier changes
stack operations
memory access
procedure calls
branches
```

Tracing should be deterministic enough to compare two executions.

Do not send trace output to the same stream used for the SNOBOL program's normal output.

## Host Services

Some SIL operations ultimately require services from the host environment.

Examples may include:

```text
input
output
file access
memory allocation
time or system information
```

Keep these behind a narrow host interface when they appear.

Do not scatter direct calls to `os`, `fmt`, files, or environment variables throughout instruction implementations.

A narrow host boundary makes the VM easier to test and allows deterministic test environments.

Do not design this interface speculatively; introduce operations as the SIL implementation requires them.

## Testing Layers

Use four levels of testing.

### 1. Instruction Tests

Construct controlled machine state, execute one instruction, inspect the result.

These should be fast and precise.

### 2. Assembler Tests

Assemble tiny source fragments or programs and inspect emitted instructions, symbols, and diagnostics.

### 3. SIL Program Tests

Write tiny SIL assembly programs and run them through:

```text
source
-> assembler
-> image
-> VM
```

These tests validate the boundary between the assembler and machine.

Prefer these over manually constructing long instruction arrays in Go.

### 4. SNOBOL4 Integration Tests

Eventually run real SNOBOL4 programs using the historical SIL implementation.

These tests validate the entire system:

```text
SNOBOL source
-> SNOBOL4 implementation
-> SIL VM
-> observable output
```

When possible, compare output with an established SNOBOL4 implementation.

## Bug-Fixing Rule

When a SNOBOL4 integration test fails, do not immediately patch the observed behavior.

Work downward through the layers:

```text
SNOBOL4 failure
       |
       v
inspect SIL trace
       |
       v
identify relevant SIL operation
       |
       v
verify assembler output
       |
       v
verify instruction semantics
       |
       v
add minimal regression test
       |
       v
fix
```

The smallest failing SIL program is usually more valuable than debugging the complete SNOBOL4 workload.

## Go Style

Use idiomatic modern Go.

Prefer:

- standard library;
- small concrete types;
- explicit state;
- explicit control flow;
- table-driven tests where useful;
- interfaces only at genuine boundaries;
- errors with useful context;
- deterministic tests.

Avoid:

- speculative abstractions;
- unnecessary generics;
- dependency injection frameworks;
- global mutable state;
- premature concurrency;
- reflection-heavy instruction dispatch;
- unsafe code without a demonstrated requirement.

Run:

```bash
go fmt ./...
go vet ./...
go test ./...
```

before considering a change complete.

## Dependencies

Default to the standard library.

Before adding a dependency, explain:

1. the problem it solves;
2. why the standard library is insufficient;
3. whether it affects reproducibility;
4. whether the functionality is small enough to implement locally.

This project should require very few external dependencies.

## Performance

Do not optimize until the machine works.

Initially avoid:

- JIT compilation;
- threaded dispatch;
- generated opcode dispatch;
- unsafe memory tricks;
- object pooling;
- speculative caching;
- concurrent execution.

When performance matters:

1. write a benchmark;
2. profile the VM;
3. identify the actual bottleneck;
4. make the smallest optimization;
5. verify semantic tests still pass.

The historical SNOBOL4 workload is itself a useful eventual benchmark.

## Image Format

Keep the assembler output format deliberately simple at first.

The development workflow should favor:

- deterministic output;
- easy inspection;
- easy comparison in tests;
- easy loading by the VM.

Do not spend time designing a compact binary executable format before the assembler/VM contract stabilizes.

If persistence is not yet useful, the assembler may initially return an in-memory image consumed directly by tests.

## Historical Source

Treat historical SIL source as input, not as code to rewrite.

Avoid mass-formatting, renaming labels, restructuring procedures, or translating it into Go.

Preserving the source makes comparison with documentation and other implementations easier.

If source must be patched for this environment:

- keep patches minimal;
- document why they are required;
- preserve the original where practical;
- distinguish portability changes from semantic changes.

## References Directory

Use `references/` for notes about authoritative documents and where to obtain them.

Do not commit copyrighted reference material unless its license or distribution terms permit redistribution.

A reference README should record:

```text
document title
author
date/version
source location
role in the project
redistribution status
```

Do not assume that material being publicly downloadable means it may be redistributed.

## Milestone Discipline

Work in this order unless evidence requires changing it.

### Milestone 1

Assemble a tiny SIL program.

### Milestone 2

Execute a tiny SIL program.

### Milestone 3

Implement control flow, calls, returns, and stack behavior needed by increasingly useful test programs.

### Milestone 4

Implement SIL runtime structures as required by actual instructions.

### Milestone 5

Attempt to assemble the historical SNOBOL4 SIL source.

At this point, unsupported syntax and opcodes become a concrete backlog.

### Milestone 6

Begin executing the SNOBOL4 implementation and use traces to identify missing machine behavior.

### Milestone 7

Run the first complete SNOBOL4 source program.

Do not implement later milestones merely because they are interesting.

## Definition of Done for an Instruction

An instruction is not complete merely because its implementation compiles.

It is complete when:

- its documented semantics have been reviewed;
- valid syntax assembles;
- invalid forms produce useful errors where applicable;
- focused VM tests pass;
- program-counter behavior is tested;
- important side effects are tested;
- at least one assembled SIL test program exercises it when practical;
- any ambiguity or intentional deviation is documented.

## Core Guardrails

Do not:

- reimplement SNOBOL4 directly in Go;
- replace SIL routines with Go equivalents for convenience;
- invent undocumented instruction semantics silently;
- optimize before correctness;
- obscure SIL concepts behind unnecessary abstractions;
- add dependencies casually;
- implement large groups of instructions without tests;
- modify historical source to hide VM bugs;
- confuse host-language errors with virtual-machine state.

When uncertain, return to the SIL documentation and construct the smallest program capable of distinguishing the possible interpretations.
