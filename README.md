# SNOBOL4 SIL Virtual Machine

A Go implementation of the SNOBOL Implementation Language (SIL) abstract machine, together with an assembler and command-line runner capable of executing the documented Macro SNOBOL4 implementation.

The project follows the same general approach as a traditional machine-language system:

```text
SNOBOL4 implementation written in SIL
                |
                v
          SIL assembler
                |
                v
        assembled VM image
                |
                v
             SIL VM
                |
                v
           SNOBOL4 runtime
```

The immediate goal is **not to reimplement SNOBOL4 directly in Go**.

Instead, the goal is to implement the machine on which the documented SNOBOL4 implementation expects to run.

This keeps a clean boundary between:

- the historical SNOBOL4 implementation;
- the documented SIL instruction set;
- our assembler;
- our Go virtual machine; and
- the command-line environment hosting it.

## Why This Approach?

SNOBOL4 was designed for portability. Its implementation was expressed using the SNOBOL Implementation Language, an abstract machine whose operations could be implemented on different host computers.

Rather than translating SNOBOL4 semantics into a new Go interpreter, this project treats SIL as a real instruction-set architecture.

That gives us a much more concrete target:

```text
SIL documentation
       |
       v
instruction semantics
       |
       +------> assembler
       |
       +------> virtual machine
                       |
                       v
                historical SIL code
```

If the VM correctly implements SIL and the assembler correctly translates documented SIL source, the existing SNOBOL4 implementation becomes our primary integration test.

This is deliberately similar to implementing a CPU emulator and assembler rather than rewriting software originally written for that CPU.

## Reference Target

The primary SIL reference is:

**Ralph E. Griswold, _Implementing SNOBOL4 in SIL: Version 3.11_, S4D58, February 1981.**

Supporting references include:

- _The SNOBOL4 Programming Language_, Griswold, Poage, and Polonsky
- _The Macro Implementation of SNOBOL4_
- S4D54, _Transporting the SIL Version of SNOBOL4: An Overview_
- surviving Macro SNOBOL4 source material
- CSNOBOL4, where useful as an executable behavioral reference

The project should distinguish carefully between:

1. documented SIL behavior;
2. original SNOBOL4 behavior;
3. later CSNOBOL4 extensions.

The first compatibility target is the documented SIL machine needed by the historical SNOBOL4 implementation.

## Project Components

The project has three principal executable pieces.

### SIL Assembler

The assembler reads documented SIL assembly source and converts it into an executable representation for the VM.

Conceptually:

```text
source.sil
    |
    v
 lexer/parser
    |
    v
 symbols + instructions + data
    |
    v
 assembled image
```

The assembler should preserve the structure and terminology of SIL rather than inventing a substantially different assembly language.

### SIL Virtual Machine

The VM implements the SIL abstract machine.

Conceptually:

```go
type Machine struct {
    PC     Address
    // registers
    // stack
    // memory
    // descriptors
    // specifiers
    // runtime state
}
```

Instructions are fetched and executed according to their documented SIL semantics:

```text
fetch
  |
decode
  |
execute
  |
update machine state
  |
next instruction
```

SIL instructions such as:

```text
POP
VEQLC
RCALL
GETSPC
BRANCH
LOCSP
STPRNT
AEQLC
INCRA
INTSPC
```

should exist as recognizable operations in the implementation.

Do not prematurely translate groups of SIL instructions into higher-level Go implementations of SNOBOL behavior. The VM is intended to execute SIL.

### Command-Line Runner

The runner loads an assembled image, initializes the SIL machine, connects its I/O environment, and executes it.

Eventually the user-facing path should be approximately:

```bash
snobol program.sno
```

Internally:

```text
program.sno
     |
     v
SNOBOL4 running on SIL VM
     |
     v
 stdin / stdout / stderr
```

Development tools may also expose the SIL layer directly:

```bash
silasm source.sil -o image
silrun image
```

The exact command names and image format should emerge from implementation.

## Proposed Repository Layout

A likely starting structure is:

```text
.
├── cmd/
│   ├── silasm/
│   ├── silrun/
│   └── snobol/
├── asm/
│   ├── lexer.go
│   ├── parser.go
│   ├── assembler.go
│   └── symbols.go
├── sil/
│   ├── machine.go
│   ├── instruction.go
│   ├── memory.go
│   ├── descriptor.go
│   └── specifier.go
├── image/
│   └── image.go
├── runtime/
│   └── ...
├── testdata/
│   └── ...
└── references/
    └── README.md
```

This is a working hypothesis, not an architecture mandate. Packages should be introduced when implementation pressure justifies them.

## The Assembled Image

The assembler and VM need a stable contract.

An assembled image will likely contain some combination of:

```text
instruction stream
static data
entry point
symbol information
initial memory state
metadata useful for diagnostics
```

During early development, prefer a simple representation that is easy to inspect and test.

A Go structure serialized with a straightforward format is preferable to designing a sophisticated binary executable format prematurely.

A binary image format can be introduced later if it provides a concrete benefit.

## Instruction Representation

Keep SIL instructions recognizable.

A possible representation is:

```go
type Opcode uint16

type Instruction struct {
    Op   Opcode
    Args []Operand
}
```

The actual operand model should be derived from the SIL documentation.

Avoid forcing every SIL operation into an artificial uniform encoding if the documented machine does not naturally work that way.

The first objective is semantic fidelity and debuggability.

## Machine State

Implement machine state from the documentation rather than from assumptions about modern CPUs.

SIL concepts such as descriptors and specifiers are part of the machine model and should be represented explicitly when required.

Do not replace them with Go strings, slices, pointers, or interfaces merely because those host-language types appear convenient.

Host types are implementation tools; SIL types define the virtual machine.

## Assembly

The assembler should be developed independently enough that its output can be inspected without executing it.

Important early capabilities include:

- source locations
- labels
- symbols
- constants
- instruction parsing
- operand parsing
- forward references
- diagnostics
- generation of an executable image

Diagnostics should report source filename and line number whenever possible.

For example:

```text
macro.sil:143: undefined symbol FOO
```

## Execution and Tracing

A SIL VM will be dramatically easier to debug if execution can be traced.

Plan for a development mode capable of showing information such as:

```text
PC
instruction
operands
selected machine state
branch destination
procedure call/return
```

For example:

```text
004231  INCRA WSTAT,1
004232  BRANCH RTN1
004817  ...
```

Tracing should be optional and should not contaminate normal SNOBOL program output.

More sophisticated tracing can be added only when needed.

## Testing Strategy

The project should be built from the bottom up while testing vertically whenever possible.

There are several useful levels of tests.

### Instruction Tests

Each SIL instruction should have focused tests establishing its documented behavior.

Given:

```text
initial machine state
instruction
```

assert:

```text
resulting machine state
next PC
side effects
```

These tests form an executable specification of SIL.

### Assembler Tests

Assembler tests should cover:

- lexical forms
- labels
- operands
- constants
- forward references
- invalid instructions
- invalid operands
- duplicate symbols
- undefined symbols

Prefer tiny complete SIL programs over testing parser internals unnecessarily.

### VM Program Tests

Write small SIL programs specifically to test the machine.

For example:

```text
initialize value
increment it
compare it
branch
terminate
```

Assemble and execute the source rather than constructing instructions directly.

This tests the assembler/VM boundary.

### SNOBOL4 Integration Tests

The strongest tests eventually run actual SNOBOL4 programs through the historical implementation executing on our SIL VM.

Start with extremely small programs and grow the corpus as the VM becomes capable.

Where practical, compare observable behavior with a known SNOBOL4 implementation.

## Development Milestones

### Milestone 1: Assembly Skeleton

Implement enough syntax to assemble a tiny SIL program.

The result should include:

- parsed instructions
- labels
- resolved addresses
- an entry point

No attempt should yet be made to support the entire instruction set.

### Milestone 2: Minimal VM

Implement:

- machine creation
- instruction fetch
- dispatch
- program counter
- a handful of simple instructions
- termination

A hand-written SIL program should assemble and run.

### Milestone 3: Calls and Control Flow

Implement enough of SIL to support:

- branches
- comparisons
- calls
- returns
- stack behavior

Write SIL test programs that exercise nested control flow.

### Milestone 4: SIL Data Model

Implement the documented runtime structures required by the SNOBOL4 implementation, including descriptors and specifiers as they become necessary.

Do this from the SIL documentation rather than trying to predict the entire model in advance.

### Milestone 5: SNOBOL4 Bootstrap

Begin assembling the actual SNOBOL4 SIL source.

Treat every assembler error or unsupported opcode as information about the next required piece.

The source itself now becomes a requirements list.

### Milestone 6: First SNOBOL4 Program

Reach the point where the VM can execute enough of the implementation to run a minimal SNOBOL4 program.

This is the project's first major compatibility milestone.

### Milestone 7: Compatibility Expansion

Use progressively more demanding SNOBOL4 programs to expose missing or incorrect SIL behavior.

Fix the VM rather than adding SNOBOL-specific shortcuts.

## Guiding Principle

The central rule of the project is:

> **Implement SIL, not SNOBOL4.**

SNOBOL4 is the workload that proves the SIL implementation.

If a SNOBOL4 program exposes a problem, first ask:

```text
Is the assembler translating SIL correctly?
```

then:

```text
Is the VM implementing the SIL instruction correctly?
```

and only then investigate whether the historical source or host environment requires special handling.

Avoid solving failures by recognizing what the SNOBOL4 code is "trying to do" and implementing that behavior directly in Go.

That would turn the project into a SNOBOL4 reimplementation and undermine the purpose of using SIL.

## Status

Experimental.

The first objective is to obtain and catalog the authoritative SIL documentation and source, then implement the smallest assembler/VM slice capable of executing a hand-written SIL program.
