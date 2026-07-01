---
name: A payload where qdf loses
about: A data shape where qdf is bigger or slower than it should be
title: '[perf] qdf loses on '
labels: perf
---

<!-- The deep-dive posts invite this on purpose: if you found a payload
     where qdf is larger or slower than json / msgpack / protobuf and you
     think it shouldn't be, this is the place. Measured beats anecdotal. -->

## The data shape

<!-- Fields and their types, string cardinality (how repetitive), batch
     size (records per message / messages per stream). A struct definition
     is ideal. -->

## The numbers

<!-- qdf vs the baseline you're comparing against: wire bytes and/or ns/op,
     which qdf option tier (OptSpeed / OptBalanced / OptCompression), and
     whether it was streaming. State how you measured. -->

| codec | wire bytes | encode | decode |
| --- | ---: | ---: | ---: |
| qdf (`Opt...`) |  |  |  |
| baseline (`...`) |  |  |  |

## Reproducer

```go
// A runnable snippet, or a fixture / generator for the data.
```

## Environment

- qdf commit: `...`
- Go version: `go version`
- GOOS / GOARCH:
- Build tags (`qdf_simd`, `qdf_reflect2`, both, none):
