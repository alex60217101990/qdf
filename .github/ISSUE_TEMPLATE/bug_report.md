---
name: Bug report
about: Decoder panic, encoder produces wrong wire, round-trip mismatch
title: '[bug] '
labels: bug
---

<!-- Security-critical issues (panic on hostile input, OOM, memory
     corruption) should be reported through a private Security
     Advisory instead: see SECURITY.md. -->

## What happened

<!-- Encode/decode operation, observed vs expected. -->

## Reproducer

```go
// Minimal Go program that demonstrates the bug.
```

Or a fuzz-corpus seed under `testdata/fuzz/...`.

## Environment

- qdf commit: `...`
- Go version: `go version`
- GOOS / GOARCH:
- Build tags (`qdf_simd`, `qdf_reflect2`, both, none):

## Additional context

<!-- Wire bytes (`hex.Dump`), allocator output, race detector report,
     etc. -->
