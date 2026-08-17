# engines

The SIL source of the historical Macro SNOBOL4 implementation, and the
`//go:embed` that carries it into the binary.

## The source is not in this repository

`sil-v3.11.sil` is assumed not to be ours to redistribute, so
`.gitignore` keeps it out and a working checkout fetches it:

```sh
curl -o engines/sil-v3.11.sil \
  https://raw.githubusercontent.com/atdt/snoflake/master/external/v311.sil
```

`references/MANIFEST.md` records the origin of that file and of the
Arizona documents beside it.

## What builds without it

Everything. `engines.go` embeds this directory as it finds it, and
`Source` reports `ErrAbsent` when the file was not here at build time.
The corpus tests that read the source skip the same way, keyed on the
file itself, so the skip expires by itself when the file arrives.

**A skip is not a pass.** With `sil-v3.11.sil` absent, nothing checks
that SNOBOL4 assembles or runs, and `cmd/sil` has no engine to run.

## Why this directory keeps a README

`go:embed` refuses a pattern that matches no files, and it has no way
to say that a file may be absent. So the pattern is a bare `*`, and
this file is committed to guarantee it matches something.
