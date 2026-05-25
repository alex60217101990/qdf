# qdf

[![ci](https://github.com/alex60217101990/qdf/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/alex60217101990/qdf/actions/workflows/ci.yml)
[![codeql](https://github.com/alex60217101990/qdf/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/alex60217101990/qdf/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/alex60217101990/qdf.svg)](https://pkg.go.dev/github.com/alex60217101990/qdf)
[![Go Report Card](https://goreportcard.com/badge/github.com/alex60217101990/qdf)](https://goreportcard.com/report/github.com/alex60217101990/qdf)
![Go](https://img.shields.io/badge/go-1.26-blue?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)
![Status](https://img.shields.io/badge/status-alpha-orange)

<p align="center">
  <img src="images/qdf-logo-new.png" alt="qdf — Quantum Density Format" width="640">
</p>

**Quantum-inspired data serialization format designed for ultra-fast
and lightweight data exchange.**

qdf gives you `encoding/json`-style ergonomics (`Marshal` / `Unmarshal` /
`NewEncoder` / `NewDecoder`) and a compact tagged binary wire format
with an optional inline string-interning table. The reflect path ships
with specialised encoders for the common slice / map shapes. A
`go:generate` tool emits reflection-free `MarshalQDF` / `UnmarshalQDF`
methods when you need the last 10–15 %.

---

## Where the name comes from

**Quantum Density Format** is a borrowed metaphor, not physics. A
classical CPU cannot store qubits; what qdf borrows from quantum
information theory is a way of thinking about repetitive structured
data.

The starting observation: in a payload like

```json
{ "users": [
    {"country": "LT"},
    {"country": "LT"},
    {"country": "LT"} ] }
```

the byte `"LT"` is written three times even though, from an
information-theoretic point of view, it is a single state with high
probability. Classical formats (JSON, MessagePack, CBOR, Protobuf)
treat data as `tree → bytes`. qdf treats it as

> `data = (state set, references, entropy weights)`

— a finite set of distinct values plus a stream that "collapses" each
position onto one of them. The wire dialect that ships today realises
the practical subset of that idea:

| Quantum-inspired concept | What it maps to in qdf today |
| ------------------------ | ---------------------------- |
| **State set** — the discrete values a position can take. | The Dense-mode intern table. The first occurrence of a string or `[]byte` value writes it in full and assigns it a stable ID. |
| **Wavefunction collapse** — a single observation picks one state. | Every subsequent occurrence emits a `state_ref` tag plus a varint ID. Decoding "collapses" the reference back to its stored value. |
| **Density** — keep only what carries information; minimise entropy `H(D \| C)` against the existing context. | Repeated keys / values across a message (and across a Dense stream) cost 1–3 bytes rather than their full length. Numeric and bool slices use QPack codecs (FOR, Delta+FOR, Gorilla XOR, raw-LE bulk) that auto-select the smallest predicted form per slice. Field-name headers in generated code are pre-encoded once per type and concatenated without further work. |
| **Probabilistic / residual coding** — predict, then store only the deviation. | QPack's Delta+FOR codec is exactly this for monotonic integer sequences: encode the first value plus the bit-packed residual against a `aᵢ = aᵢ₋₁ + minΔ` predictor. Gorilla does the same for floats by XOR-ing each sample against the previous one and storing the differing bits only. |
| **Entanglement** — correlated values that constrain each other (e.g. `city = "Vilnius"` ⇒ `country = "Lithuania"`). | Partial. The Dense encoder runs a Markov-0 predictor on intern IDs (`tagStateRepeat`, `0xE8`) and a Move-To-Front rank coder (`tagStateMTF`, `0xE9`). The first collapses immediate repeats to one byte; the second catches "hot subset interned late" patterns by encoding LRU rank instead of raw id whenever that is shorter. On telemetry payloads with repeating service / region / level fields the pair delivers a ~3.5× wire reduction vs JSON on its own. A full conditional-probability table across non-adjacent fields is still future work and would occupy more of the reserved `0xEA..0xEF` block. |
| **Arithmetic / range coding (rANS)** — push the encoded stream to its Shannon limit. | Not yet. The state-table + back-reference pair plus QPack already captures most of the practical win on telemetry workloads; rANS would buy further compression at a CPU cost. |

The conceptual roadmap (state table → entanglement graph → predictive
encoder) is preserved in the codebase: tag space and the encoder
interface are designed to let those layers slot in above the current
one without a format break. The MVP that ships now is the part of the
idea that has a positive CPU / size tradeoff on day-one workloads —
logs, traces, columnar rows, embedding vectors.

The realistic conclusion: the real shift is not the "quantum" framing
but the move from "data as bytes" to "data as a model of the world
that produced it".

---

## Why another format

`encoding/json` is verbose and slow. `vmihailenco/msgpack/v5` is fast
but allocates a lot on decode and has no built-in deduplication for
repeated keys / values — which is the dominant cost on real telemetry,
log, and trace payloads where the same handful of strings (service
names, log levels, region codes, span attributes) appears thousands of
times per batch.

qdf attacks two costs at once:

1. **Wire size on repetitive data.** A Dense-mode encoder maintains an
   inline interning table: the first time `"region":"eu-west-1"`
   appears it is written in full; subsequent occurrences emit a 2-byte
   back-reference. Streams share that table across messages, so a
   1000-event log batch with eight distinct region codes carries each
   code once. The decoder follows the same protocol — no out-of-band
   dictionary, no two-pass encode.

2. **CPU and allocations on decode.** The reflect path uses a cached
   per-type descriptor with `unsafe.Pointer + field offset` access
   instead of `reflect.Value.Field(i)`, plus specialised type-specific
   decoders for `[]string`, `[]int*`, `[]float*`, `map[string]string`,
   `map[string]int`, `map[string]any`, etc. A direct-mapped key
   interner deduplicates map keys across decode calls without a pool
   or chained hash table. On a `map[string]any` decode with repeated
   keys the allocator runs **3 times per call** versus json's 37.

The format is single-pass and streaming-safe in both directions. There
is no schema language, no IDL, no compile-time registration — types
are picked up via Go struct tags (`qdf:"name"`, falling back to
`json:"name"`).

### Wire layout

```
+-----+-----+-----+-----+-----+
| 'Q' | 'D' | 'F' | ver |flags|
+-----+-----+-----+-----+-----+
| body (tagged values)        |
+-----------------------------+
```

A 5-byte header (3-byte magic + 1-byte version + 1-byte flags) and a
tagged body. The flag bit distinguishes Fast from Dense; the decoder
auto-detects.

Tag space is msgpack-inspired with a few additions:

| Range                | Purpose                                       |
| -------------------- | --------------------------------------------- |
| `0x00..0x7F`         | positive fixint (0..127)                      |
| `0x80..0x9F`         | fixstr (len 0..31)                            |
| `0xA0..0xBF`         | fixarr (len 0..31)                            |
| `0xC0..0xCF`         | nil, bool, int / uint / float of width 8–64   |
| `0xCD..0xCF`         | str8 / str16 / str32                          |
| `0xD0..0xD2`         | bin8 / bin16 / bin32                          |
| `0xD3..0xD4`         | arr16 / arr32                                 |
| `0xD5..0xD7`         | map8 / map16 / map32                          |
| `0xD8..0xDF`         | negfixint (-1..-8)                            |
| `0xE0..0xE2`         | intern\_str / state\_ref / intern\_bin (Dense)|
| `0xE3`               | QPack: bit-packed `[]bool`                    |
| `0xE4`               | QPack: raw-LE numeric slice                   |
| `0xE5`               | QPack: Frame-of-Reference bit-packed ints     |
| `0xE6`               | QPack: Delta + zigzag + FOR (monotonic ints)  |
| `0xE7`               | QPack: Gorilla XOR-coded float slice          |
| `0xE8`               | Dense: state-ref repeat (Markov-0 predictor)  |
| `0xE9`               | Dense: state-ref MTF rank (Move-To-Front)     |
| `0xEA..0xEF`         | reserved (full entanglement / rANS / future)  |
| `0xF0..0xF3`         | ext / timestamp                               |

The 5th header byte holds two flag bits: `FlagDense` (`0x01`) for the
intern dialect, and `FlagQPack` (`0x02`) as an early hint that the body
may carry codec tags from the `0xE3..0xE7` block. A legacy decoder that
does not know the QPack tags fails with `ErrBadTag` on first contact;
it never decodes a packed payload as scalar by accident.

All multi-byte integers and floats are little-endian. amd64 and arm64
are the supported targets.

---

## Quick start

```bash
go get github.com/alex60217101990/qdf
```

Requires Go 1.26.

```go
package main

import (
    "fmt"

    "github.com/alex60217101990/qdf"
)

type Event struct {
    ID      int               `qdf:"id"`
    Source  string            `qdf:"source"`
    Payload []byte            `qdf:"payload"`
    Attrs   map[string]string `qdf:"attrs"`
}

func main() {
    in := Event{
        ID:     42,
        Source: "ingest",
        Attrs:  map[string]string{"region": "eu-west-1", "version": "v3"},
    }

    b, err := qdf.Marshal(in)
    if err != nil {
        panic(err)
    }
    fmt.Printf("wire: %d bytes\n", len(b))

    var out Event
    if err := qdf.Unmarshal(b, &out); err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", out)
}
```

### Three wire dialects

| Mode           | API                | When to use                                                  |
| -------------- | ------------------ | ------------------------------------------------------------ |
| `qdf.Fast`     | `Marshal`          | Default. Tightest CPU cost; size comparable to msgpack.      |
| `qdf.Dense`    | `MarshalDense`     | Repetitive payloads — logs, telemetry, columnar rows. Strings intern once; numeric and bool slices also use QPack codecs. |
| QPack (Fast+codecs) | `MarshalQPack` | Fast mode + QPack codecs for numeric/bool slices. Best when the payload has few unique strings but lots of numeric data (vectors, embeddings, timestamps). |

```go
b, _ := qdf.MarshalDense(rows)  // string interning + QPack
b, _ := qdf.MarshalQPack(rows)  // QPack only, no string interning
```

A single decoder handles all three. `qdf.Unmarshal` reads the header
flag and picks the right path automatically.

### Generic API (Go 1.18+ generics)

`Marshal(v any)` boxes the argument through `interface{}` and then
makes one reflect copy for value-typed inputs. The generic helpers
skip both: `T` is fixed at the call site, `reflect.TypeFor[T]()`
resolves at compile time, and `unsafe.Pointer(&v)` points directly at
the caller's stack.

| Generic                    | Equivalent to       |
| -------------------------- | ------------------- |
| `MarshalT[T any]`          | `Marshal`           |
| `MarshalQPackT[T any]`     | `MarshalQPack`      |
| `MarshalDenseT[T any]`     | `MarshalDense`      |
| `AppendMarshalT[T any]`    | `AppendMarshal`     |
| `UnmarshalT[T any]`        | `Unmarshal`         |

Wire output is byte-identical. The win is on the encode side: one
fewer allocation per call (-80 B) and 25–40 % faster on small/medium
payloads.

```go
buf, _ := qdf.MarshalT(event)
var out Event
_ = qdf.UnmarshalT(buf, &out)
```

### Direct entry points (skip reflection entirely)

`MarshalT` still goes through `descOf` to walk the type. Types that
already implement `MarshalQDF` / `UnmarshalQDF` (either hand-written or
generated by `cmd/qdfgen`) can bypass the descriptor cache and the
runtime type assertion via the *Direct* generics. The type parameter
is constrained to the `Marshaler` / `Unmarshaler` interface, so the
method call is resolved at compile time and the compiler inlines it.

| Direct                                                  | When to use                                                   |
| ------------------------------------------------------- | ------------------------------------------------------------- |
| `MarshalDirect[T Marshaler](v T) ([]byte, error)`       | Hot paths where the type is known to have generated codecs.   |
| `AppendMarshalDirect[T Marshaler](dst []byte, v T)`     | Same, when the caller owns the destination buffer.            |
| `UnmarshalDirect[T Unmarshaler](data []byte, out T)`    | Inverse of `MarshalDirect`. Falls back to `Unmarshal` on `FlagDense` because generated code does not maintain the intern table. |

Wire is identical to `Marshal` / `Unmarshal`; the path is Fast-mode
only (Dense or QPack stay on the reflect path so the intern table is
correctly resolved). On the `cmd/qdfgen` `Sample` fixture
(11-field struct):

| Encode      | ns/op   | B/op | allocs |
| ----------- | ------: | ---: | -----: |
| json        |    1800 |  576 |      8 |
| qdf_reflect |     580 |  480 |      3 |
| qdf_codegen |     530 |  504 |      6 |
| **qdf_direct** | **364** | **160** | **1** |

Decode-side performance is bounded by the receiver's `UnmarshalQDF`.
The reflect path is heavily pooled (Decoder pool + per-decoder key
intern cache); generated code from `qdfgen` uses `Decoder.InternKey`
and matches it. Hand-rolled receivers that allocate a new Decoder per
call will not.

```go
buf, _ := qdf.MarshalDirect(&event)
var out Event
_ = qdf.UnmarshalDirect(buf, &out)
```

### QPack: math-driven slice codecs

QPack auto-selects, per slice, the codec with the smallest predicted
wire form. Round-trip is lossless for every type (NaN/±Inf survive on
floats).

| Codec      | Trigger                                            | Math                                                                  |
| ---------- | -------------------------------------------------- | --------------------------------------------------------------------- |
| Bitpack    | `[]bool`                                           | 1 bit per element, LSB-first per byte. 8× smaller for free.           |
| Raw-LE     | numeric slice, large delta range                   | Unsafe-slice cast → single `memmove` of LE bytes. 10-50× faster.      |
| FOR        | numeric slice, clustered values                    | Frame-of-Reference: store `min` and `ceil(log₂(max-min+1))`-bit deltas.|
| Delta+FOR  | monotonic / near-monotonic integers                | Δᵢ = aᵢ - aᵢ₋₁, zigzag bias, FOR over the deltas.                     |
| Gorilla    | float slices (explicit opt-in via low-level API)   | XOR with previous, run-length leading/meaningful-bit window. (Facebook VLDB 2015.) |

Head-to-head on a mixed 256-bool / 512-monotonic-u64 / 512-i64 / 256-f64
payload (Intel i7-9750H):

| Format      | Bytes  | Encode (ns/op) | Decode (ns/op) |
| ----------- | -----: | -------------: | -------------: |
| json        | 10,739 |         48,000 |        200,000 |
| msgpack     | 11,808 |         64,000 |         80,000 |
| qdf_fast    |  6,694 |          6,500 |         14,000 |
| qdf_qpack   |  2,132 |          2,300 |          2,600 |
| qdf_dense   |  2,134 |          2,500 |          2,500 |

For sequences where the deltas alone collapse, the gain is much
larger — e.g. 1024 consecutive Unix-second timestamps shrink from
8201 bytes (raw) to 16 bytes (Delta+FOR), a 512× reduction.

### Streaming

```go
var w bytes.Buffer
enc := qdf.NewStreamEncoder(&w, qdf.Dense)
for _, ev := range events {
    if err := enc.Encode(ev); err != nil {
        return err
    }
}
enc.Close()

dec := qdf.NewStreamDecoder(&w)
defer dec.Close()
for {
    var ev Event
    if err := dec.Decode(&ev); err == io.EOF {
        break
    } else if err != nil {
        return err
    }
    // ...
}
```

Dense streams preserve the intern table across messages — the second
occurrence of `"region":"eu-west-1"` in the batch is a 2-byte reference
rather than a 13-byte string.

### Zero-extra-copy encode

`AppendMarshal` lets callers own the destination buffer:

```go
out, err := qdf.AppendMarshal(out[:0], v)
```

Pair with a goroutine-local buffer to drop the per-call allocation
entirely.

---

## Code generation

For fixed-schema types where every nanosecond matters, generate
reflection-free methods:

```bash
go install github.com/alex60217101990/qdf/cmd/qdfgen@latest
```

```go
//go:generate qdfgen -type Event,User .
type Event struct {
    ID     int    `qdf:"id"`
    Source string `qdf:"source"`
    // ...
}
```

`qdfgen` writes `<package>_qdf.go` with concrete `MarshalQDF` /
`UnmarshalQDF` methods using only the public `qdf` API. No `reflect.*`
at runtime. See [`cmd/qdfgen/README.md`](cmd/qdfgen/README.md) for
flags and supported types.

On the test fixture (11-field struct with nested struct, slice, map,
pointer, fixed array, `[]byte`, `time.Time`) the generated code is
**2.65× faster** than `encoding/json` on encode and **8.49× faster**
on decode.

---

## Benchmarks

Darwin amd64 / Intel i7-9750H @ 2.6 GHz, Go 1.26.0, `-benchtime=2s`,
`-race` cross-check. Full numbers, realistic / unique-data scenarios,
memory tables and reproduction commands are in
[`docs/BENCH.md`](docs/BENCH.md).

### Encode — ns/op

| Payload                  | json    | msgpack | qdf\_fast | qdf\_dense | vs json    | vs msgpack |
| ------------------------ | ------: | ------: | --------: | ---------: | ---------: | ---------: |
| Tiny (`{id,name}`)       |     192 |     286 |       227 |        398 |       0.85× | 1.26×      |
| Flat (20 primitive fields) |   1182 |    1262 |   **480** |        948 | **2.46×**  | **2.63×**  |
| Nested (4 deep)          |     446 |     793 |   **331** |        786 | **1.35×**  | **2.40×**  |
| Deep linked-list (16)    |    1157 |    3060 |   **477** |        814 | **2.43×**  | **6.42×**  |
| Wide × 1000              |   991 k |   991 k |  **213 k**|      289 k | **4.66×**  | **4.66×**  |
| Log batch × 1000         |   998 k |   624 k |  **171 k**|      562 k | **5.84×**  | **3.65×**  |
| Map-heavy (40 entries)   |    8446 |    5282 |  **1391** |       2019 | **6.07×**  | **3.80×**  |
| `[]float32` × 512        |  30 054 |  18 270 |  **1821** |          – | **16.5×**  | **10.0×**  |
| `[]float64` × 512        |  39 819 |  23 856 |  **2450** |          – | **16.3×**  | **9.74×**  |
| UniqueLog (fresh per iter) |    2419 |    2080 |  **1510** |          – | **1.60×**  | **1.38×**  |

### Decode — ns/op

| Payload                  | json    | msgpack | qdf\_fast | vs json   | vs msgpack |
| ------------------------ | ------: | ------: | --------: | --------: | ---------: |
| Tiny                     |     729 |     383 |   **170** | **4.29×** | **2.25×**  |
| Flat                     |    4411 |    1862 |  **1122** | **3.93×** | **1.66×**  |
| Nested                   |    2483 |    1206 |   **425** | **5.84×** | **2.84×**  |
| Deep16                   |    7363 |    4170 |  **1972** | **3.73×** | **2.11×**  |
| Wide × 1000              |  4.40 M |  1.83 M | **990 k** | **4.44×** | **1.85×**  |
| Log batch × 1000         |  3.29 M |  1.32 M | **449 k** | **7.31×** | **2.93×**  |
| Map-heavy (40 entries)   |  16 621 |    8096 |  **3108** | **5.35×** | **2.61×**  |
| `[]float32` × 512        |  72 928 |  28 521 |  **3960** | **18.4×** | **7.20×**  |
| UniqueLog (fresh bytes)  |    3569 |    1296 |   **509** | **7.01×** | **2.55×**  |
| UniqueLog (RunParallel)  |     667 |     682 |   **487** | **1.37×** | **1.40×**  |

### Memory — bytes per decode

| Payload                  | json    | msgpack | qdf\_fast | vs json    | vs msgpack |
| ------------------------ | ------: | ------: | --------: | ---------: | ---------: |
| Tiny                     |     248 |      77 |    **29** | **0.12×**  | **0.38×**  |
| Nested                   |     664 |     160 |   **112** | **0.17×**  | 0.70×      |
| Log batch × 1000         | 442 536 | 407 698 | **251 838** | **0.57×** | **0.62×**  |
| Map-heavy                |    4912 |    3089 |  **2359** | **0.48×**  | 0.76×      |
| `[]float32` × 512        |    4384 |    4282 |  **2113** | **0.48×**  | **0.49×**  |
| MapStringAny (rep. keys) |     790 |       – |   **345** | **0.44×**  | –          |

### Allocations per decode

| Payload                  | json | msgpack | qdf\_fast |
| ------------------------ | ---: | ------: | --------: |
| Nested                   |   15 |       6 |     **5** |
| Map-heavy (40 entries)   |  124 |     112 |    **32** |
| MapStringAny (rep. keys) |   37 |       – |     **3** |
| `[]float32` × 512        |   16 |       8 |     **3** |

### Encoded size

| Payload                  | json    | msgpack | qdf\_fast | qdf\_dense | dense vs json |
| ------------------------ | ------: | ------: | --------: | ---------: | ------------: |
| Flat                     |     210 |     134 |   **132** |        137 | 0.65×         |
| Deep16                   |     239 |     139 |       166 |    **122** | **0.51×**     |
| Wide × 1000              | 212 901 | 135 626 |   128 632 |**106 660** | **0.50×**     |
| Log batch × 1000         | 251 902 | 185 639 |   185 649 |**107 416** | **0.43×**     |

On a 1000-entry Dense **stream** the same log batch encodes to
**107 bytes per entry**, against 251 for json and 186 for msgpack —
shared intern table across messages.

### Realistic corpus

Numbers from `realistic_corpus_test.go` builders that mirror real
telemetry workloads. Full breakdown plus encode latency in
[`docs/BENCH.md`](docs/BENCH.md).

#### TelemetryBatch (1 000 log events, repeating service / region / level)

| Format       |   bytes | vs json    | encode ns/op |
| ------------ | ------: | ---------: | -----------: |
| json         | 252 497 |      1.00× |            — |
| qdf_fast     | 186 674 |      0.74× |       272 k  |
| qdf_qpack    | 186 674 |      0.74× |       261 k  |
| **qdf_dense** | **73 104** | **0.29×** |    1.0 M     |

Dense pays ~4× CPU for **3.5× smaller wire** vs JSON — string-intern
+ Markov-0 + MTF crush the repeating service / region / level / host
fields. QPack does not help much here (per-event numeric fields are
scalar, not slice-shaped).

#### MetricSeries (1 024 numeric timestamps + values)

| Format        |   bytes | vs json    |
| ------------- | ------: | ---------: |
| json          |  30 043 |      1.00× |
| qdf_fast      |  14 442 |      0.48× |
| **qdf_qpack** | **8 307** | **0.28×** |
| qdf_dense     |   8 315 |      0.28× |

Here QPack pulls its weight: `[]int64` timestamp column is monotonic
and Delta+FOR compresses it to near-zero bytes per element; the
`[]float64` value column rides raw-LE bulk. Dense and QPack converge
because string overhead is tiny.

### Reproduce

```bash
# Default build
cd bench
go test -bench=. -benchmem -benchtime=2s -timeout=10m

# QPack head-to-head (Marshal / MarshalQPack / MarshalDense
# / json / msgpack on the same numeric+bool payload)
go test -bench='BenchmarkQPack_' -benchmem

# QPack micro-benchmarks (codec internals, in the root module)
cd ..
go test -bench='BenchmarkQPack' -benchmem

# AVX2 bit-unpack (asm, requires CPUID AVX2 at run time)
go test -tags qdf_simd -bench='BenchmarkBitUnpackFast' -benchmem

# Realistic unique-data scenarios
cd bench
go test -bench='BenchmarkEncode_UniqueLog|BenchmarkDecode_UniqueLog|BenchmarkEncode_MixedTypes|BenchmarkEncode_RandomSize' -benchmem

# Encoded sizes
go test -run TestSizes -v

# Codegen vs reflect on the Sample fixture
cd ../internal/codegen_test
go test -run TestGenerate .
go test -bench=. -benchmem -benchtime=2s
```

---

## Build tags

Opt-in fast paths. None are required for the headline numbers; the
defaults already include the specialised slice / map encoders, the
QPack codecs, and a 128-bit sliding-window bit-unpacker for FOR / Delta+FOR.

| Tag             | Effect                                                                                                  | Build prerequisite |
| --------------- | ------------------------------------------------------------------------------------------------------- | ------------------ |
| `qdf_reflect2`  | Swap `reflect.MakeSlice` / `MakeMapWithSize` for `modern-go/reflect2` unsafe equivalents. Smaller decode allocations on map / slice heavy workloads. | none — pure Go     |
| `qdf_simd`      | AVX2 fast path for QPack bit-unpack at bits ∈ {8, 16, 32}. 22-53× over the pure-Go path on those widths (≈ 50 GB/s, memory-bandwidth bound). Runtime CPUID gate via `golang.org/x/sys/cpu`; older amd64 without AVX2 transparently falls back to the scalar zero-extend. Non-amd64 targets compile a stub. | amd64; AVX2 detected at run time |

```bash
go build -tags qdf_reflect2 ./...
go build -tags qdf_simd ./...
go build -tags "qdf_simd qdf_reflect2" ./...   # combined
```

The `qdf_simd` path only changes behaviour for QPack-encoded numeric
slices whose chosen codec is FOR or Delta+FOR at one of the byte-aligned
bit widths; other widths and other codecs run the pure-Go path
unchanged.

---

## Correctness

The test suite runs under `-race` and covers:

- Primitive round-trip across every wire-tag boundary.
- Boundary integers from `math.MinInt64` to `math.MaxUint64`.
- Strings at the fixstr / str8 / str16 / str32 boundaries (0, 1, 31,
  32, 255, 256, 65535, 65536, 1 MiB) plus Unicode and invalid UTF-8.
- Every specialised fast path against the generic reflect path.
- Truncated input — every prefix from 0 to N-1 decodes to an error,
  never a panic.
- Bad magic / bad version — errors, not panics.
- Cross-mode interop (Fast bytes decoded via Dense decoder and vice
  versa).
- Streaming with mixed message types under Dense intern.
- Concurrency: 32 × 500 marshal+unmarshal goroutines under `-race`.
- QPack codecs: each codec (bitpack-bool, raw-LE, FOR, Delta+FOR,
  Gorilla) has its own round-trip suite covering edge cases — empty
  slices, single elements, near-`MaxUint64`, `MinInt64`, NaN / ±Inf /
  ±0 / denormals, monotonic / constant / mixed-direction sequences.
- `TestCompleteness_AllModes` runs one big payload (every QPack-
  eligible type, every edge case, nested structs, maps, string
  interning) through `Marshal`, `MarshalQPack`, and `MarshalDense`,
  then asserts bit-for-bit IEEE-754 equality for floats and
  `reflect.DeepEqual` elsewhere.
- `TestCompleteness_StreamingDense` exercises three messages through
  `NewStreamEncoder` in Dense mode (carries QPack + shared intern
  table); `TestCompleteness_FuzzRandomStructsQPack` generates 50
  random struct shapes and round-trips them through all three
  encoders.
- AVX2 bit-unpack (under `qdf_simd`) has parity tests against scalar
  zero-extend for `n` in `{0,1,2,3,4,5,7,8,9,15,16,17,1000,4096}` at
  bits ∈ {8, 16, 32}, plus a `bitPackU64LE → bitUnpackU64LE` round-
  trip.
- `tagStateRepeat` (Markov-0 state-ref predictor): tests that the
  predictor fires on runs of identical interned values, does not fire
  on alternation, invalidates correctly across an inline-string
  emission, stays in sync with `Decoder.Skip`, and round-trips through
  the public `MarshalDense` / `Unmarshal` boundary.
- **Property-based round-trip fuzzers** (`fuzz_property_test.go`)
  drive a deterministic value generator from the fuzz bytes, encode
  through every Marshal entry point, and assert
  `Unmarshal(Marshal(v)) == v`. Targets cover Int64Slice,
  Uint64Slice, Float64Slice, BoolSlice, MapStringInt, StructTriad,
  and an AllModesAgree fuzzer that asserts the three encoder
  dialects decode to the same Go value. Caught one Markov-0
  predictor bug in `encodeStruct` (commit `dfc30ae`).
- **Golden-file wire pinning** (`testdata/golden/*.bin`,
  `golden_test.go`): every representative payload has its
  Marshal / MarshalQPack / MarshalDense bytes committed to disk. A
  later wire-format change either matches the bytes byte-for-byte
  or fails the test. Map-shaped cases skip the byte-pin half
  (Go map iteration is randomised) but keep the decode-and-compare
  half. Regenerate intentionally via `go test -run TestGolden -update`.
- **Deterministic truncation matrix** (`truncation_test.go`, ~2700
  sub-tests): for every representative payload, every prefix
  `payload[:i]` is fed through Unmarshal into six destination
  types, every 4th byte is mutated to one of nine "bad"
  alternatives (00, FF, 80, 55, AA, and every QPack tag), and
  malformed 5-byte headers are exhaustively rejected. The decoder
  must return a typed error in every case; panic budget is zero.
- **Unknown-field skip** (`unknown_field_test.go`): encoder writes a
  10-field struct, decoder declares only 3; subsequent declared
  fields still decode correctly. Each of the 10 fields is also
  pulled out alone via its own single-field destination type.
- **Pathological-shape stress** (`stress_nesting_test.go`):
  256-deep linked list, 512-key map, balanced binary tree
  (depth 12), 1 MiB string + 1 MiB []byte + 100k-element u64/f64
  vectors, and a 2000-deep chain under `debug.SetMaxStack(64 MiB)`.
- **Realistic payload corpus** (`realistic_corpus_test.go`):
  telemetry batches with repeating service/region/level fields,
  metric time-series, wide column-store rows with recursive
  Children — round-tripped through every Marshal entry point and
  cross-checked for three-way agreement.
- **Allocation-budget tests** (`alloc_budget_test.go`): each hot
  entry point has an upper-bound allocation budget enforced via
  `testing.AllocsPerRun`. A regression in alloc count fails CI
  rather than silently bloating downstream callers.
- **Tag and named-type matrix** (`tag_matrix_test.go`): every
  primitive type is exercised through a named alias (`type MyInt
  int64`), the tag fallback chain `qdf` → `json` → field name is
  verified, the `qdf:"-"` skip directive is checked via a
  `map[string]any` decode, and embedded-struct (non-)flattening is
  pinned.
- **Differential testing** (`bench/diff_test.go`): qdf round-trips
  agree semantically with `encoding/json` and
  `github.com/vmihailenco/msgpack/v5` on a shared payload,
  including NaN / ±Inf / ±0 preservation against msgpack.
- **Pointer-cycle detection** via depth counter: the encoder
  increments `Encoder.depth` on every pointer dereference and
  returns `ErrCycleDetected` once the count exceeds
  `DefaultMaxDepth` (10000). Cheaper than a per-pointer set (no
  allocation per call) and catches both genuine `*T → *T` cycles
  and pathologically deep payloads. Override the cap via
  `Encoder.SetMaxDepth`.
- **OOM protection** on every length-prefixed read path: a
  hostile varuint encoding a value > 2^62 cannot drive the
  decoder cursor negative because the byte-count check happens
  in `uint64` against `(len(d.buf) - d.i) * 8 / bitsPer` *before*
  any signed cast. Applies to tagPackBool/Raw/For/DeltaFor/Gorilla
  in both the slice-read and Skip paths.
- **Stream Close idempotency**: both `StreamEncoder.Close` and
  `StreamDecoder.Close` are safe to call multiple times.
- Fuzz: `FuzzDecoder_NeverPanics`, `FuzzRoundTrip_StringSlice`,
  `FuzzQPackBool`, `FuzzQPackRawUint64`, `FuzzRoundTrip_Int64Slice`,
  `FuzzRoundTrip_Uint64Slice`, `FuzzRoundTrip_Float64Slice`,
  `FuzzRoundTrip_BoolSlice`, `FuzzRoundTrip_MapStringInt`,
  `FuzzRoundTrip_StructTriad`, `FuzzRoundTrip_AllModesAgree` —
  persistent corpus under `testdata/fuzz/`. 10 M+ executions
  clean across the suite after the Skip-overflow and
  encodeStruct-predictor fixes; both repros are saved under
  `testdata/fuzz/`.

Length prefixes are validated against the remaining buffer
(`Decoder.CheckLength`) before any `make`, so a hostile payload
claiming a multi-billion-element map cannot OOM the process. The
QPack-tag `Skip` paths perform the same overflow-safe check on
uint64 element counts before any signed cast, so a 10-byte varuint
encoding a value > 2^62 cannot drive the decoder cursor negative.

```bash
go test -race -count=1 ./...

# Property-based round-trip fuzz (each fuzzer asserts
# Unmarshal(Marshal(v)) == v on randomly generated values)
go test -run=^$ -fuzz=FuzzRoundTrip_StructTriad     -fuzztime=60s
go test -run=^$ -fuzz=FuzzRoundTrip_AllModesAgree   -fuzztime=60s
go test -run=^$ -fuzz=FuzzRoundTrip_Int64Slice      -fuzztime=30s
go test -run=^$ -fuzz=FuzzRoundTrip_Uint64Slice     -fuzztime=30s
go test -run=^$ -fuzz=FuzzRoundTrip_Float64Slice    -fuzztime=30s
go test -run=^$ -fuzz=FuzzRoundTrip_BoolSlice       -fuzztime=30s
go test -run=^$ -fuzz=FuzzRoundTrip_MapStringInt    -fuzztime=30s

# Decoder safety fuzz (asserts never-panic on hostile input)
go test -run=^$ -fuzz=FuzzDecoder_NeverPanics       -fuzztime=30s
go test -run=^$ -fuzz=FuzzRoundTrip_StringSlice     -fuzztime=30s
go test -run=^$ -fuzz=FuzzQPackBool                 -fuzztime=30s
go test -run=^$ -fuzz=FuzzQPackRawUint64            -fuzztime=30s

# Differential vs msgpack and encoding/json
go test -C bench -run=TestDiff -count=1

# Regenerate wire-format golden fixtures (intentional after a wire
# bump only)
go test -run=TestGolden -update

# Build-tag combinations
go test -tags qdf_reflect2          -race ./...
go test -tags qdf_simd              -race ./...
go test -tags "qdf_simd qdf_reflect2" -race ./...
```

---

## Project layout

```
qdf/
├── qdf.go                  public API: Marshal, MarshalDense, MarshalQPack, Unmarshal, AppendMarshal
├── qdf_generic.go          generic helpers: MarshalT, MarshalDenseT, MarshalQPackT, UnmarshalT, AppendMarshalT
├── qdf_direct.go           reflection-free shortcut: MarshalDirect / UnmarshalDirect / AppendMarshalDirect
├── encoder.go              *Encoder + tag emitters
├── decoder.go              *Decoder + length validation + key intern
├── stream.go               streaming encoder/decoder
├── state.go                Dense intern table (with Markov-0 last-state-ref predictor)
├── maps_fast.go            type-specific map encoders/decoders
├── slices_fast.go          type-specific slice encoders/decoders + QPack dispatch
├── qpack.go                shared QPack constants, raw-LE codec, per-slice codec selector
├── qpack_for.go            Frame-of-Reference bit-packed integer codec
├── qpack_delta.go          Delta + zigzag + FOR for monotonic integers
├── qpack_gorilla.go        Gorilla XOR float codec
├── qpack_bitpack_fast.go   128-bit sliding-window bit-unpacker (default fast path)
├── qpack_simd_amd64.s      AVX2 zero-extend for bits ∈ {8,16,32} (under qdf_simd)
├── qpack_simd_amd64.go     CPUID gate + dispatch (under qdf_simd)
├── qpack_simd_stub.go      scalar fallback for everything else
├── endian_le.go/be.go      native-endian guard for unsafe slice aliasing
├── floats_default.go       default float-slice bulk path
├── floats_simd.go          tighter loop for float-slice encode under qdf_simd
├── reflect_alloc*.go       slice/map allocator (default vs reflect2)
├── reflect_encode.go       reflect-based encode/decode with descriptor cache
├── wire.go                 tag constants + varint
├── errors.go
├── fuzz_test.go                  decoder-safety fuzzers
├── fuzz_property_test.go         property-based round-trip fuzzers
├── golden_test.go                wire-format goldens (testdata/golden/*.bin)
├── truncation_test.go            prefix / mutation / header rejection matrix
├── unknown_field_test.go         decoder skips unknown fields cleanly
├── stress_nesting_test.go        deep-chain, wide-map, tree, big-primitive stress
├── realistic_corpus_test.go      telemetry / metric-series / wide-row round-trips
├── alloc_budget_test.go          AllocsPerRun upper-bound checks per hot entry
├── tag_matrix_test.go            named-type, qdf:"-", json fallback, embedded
├── cycle_test.go                 deep finite chain + documented cycle limitation
├── entanglement_test.go          tagStateRepeat Markov-0 predictor coverage
├── qpack_completeness_test.go    end-to-end coverage across all Marshal entries
├── oom_protection_test.go        hostile huge-length-prefix rejection
├── concurrent_stress_test.go     1000-goroutine round-trip, pool churn, per-G streams
├── interface_matrix_test.go      `any` field dynamic-type matrix
├── reset_and_edges_test.go       Encoder.Reset coverage + pointer/misc edges
├── stream_edges_test.go          chunked I/O, EOF mid-message, Close idempotency
├── internal/
│   ├── bufpool/        size-classed sharded byte-slice pool
│   ├── intern/         decoder-side string-key intern cache
│   └── unsafestr/      zero-copy string ↔ []byte
├── cmd/
│   └── qdfgen/         code generator (separate module)
├── bench/              comparison benchmarks (separate module)
└── docs/
    └── BENCH.md        full benchmark report
```

---

## Status

Alpha. The wire format is stable for the `0x01` version byte; future
versions will bump it. The public API may change before the first
tagged release — pin a commit if you depend on it.

## License

MIT. See [LICENSE](LICENSE).
