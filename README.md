# qdf

[![ci](https://github.com/alex60217101990/qdf/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/alex60217101990/qdf/actions/workflows/ci.yml)
[![codeql](https://github.com/alex60217101990/qdf/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/alex60217101990/qdf/actions/workflows/codeql.yml)
[![codecov](https://codecov.io/gh/alex60217101990/qdf/branch/main/graph/badge.svg)](https://codecov.io/gh/alex60217101990/qdf)
[![benchmarks](https://img.shields.io/badge/benchmarks-dashboard-blue)](https://alex60217101990.github.io/qdf/)
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
| **Density** — keep only what carries information; minimise entropy `H(D \| C)` against the existing context. | Repeated keys / values across a message (and across a Dense stream) cost 1–3 bytes rather than their full length. Numeric and bool slices use QPack codecs (FOR, Delta+FOR, RLE, dictionary, Gorilla XOR, ALP decimal, raw-LE bulk) that auto-select the smallest predicted form per slice; slices of homogeneous structs transpose to per-column codecs, and an enum-like string column is dictionary-coded (distinct table + bit-packed index per row) when that beats per-value interning. Field-name headers in generated code are pre-encoded once per type and concatenated without further work. |
| **Probabilistic / residual coding** — predict, then store only the deviation. | QPack's Delta+FOR codec is exactly this for monotonic integer sequences: encode the first value plus the bit-packed residual against a `aᵢ = aᵢ₋₁ + minΔ` predictor. Gorilla does the same for floats by XOR-ing each sample against the previous one and storing the differing bits only. |
| **Entanglement** — correlated values that constrain each other (e.g. `city = "Vilnius"` ⇒ `country = "Lithuania"`). | Three stacked predictors on the Dense state stream. **Markov-0** (`tagStateRepeat`, `0xE8`) collapses an immediate repeat of the previously emitted state-ref to a single byte. **MTF rank** (`tagStateMTF`, `0xE9`) encodes the LRU rank of the touched ID when that varuint is shorter than the raw id. **Markov-1 pair** (`tagStatePair`, `0xEA`) keeps the last 4 successors observed after each prev ID and encodes a hit as `0xEA + 1-byte rank` — wins when the prev → curr transition is predictable and the raw id needs a multi-byte varuint. The encoder picks the shortest of the four variants per emission, so the wire is never larger than the plain `tagStateRef` encoding. A full conditional-probability table beyond order-1 stays in the reserved `0xEB / 0xED..0xEF` block. |
| **Shape interning** — repeated structure means the *layout* itself is information. | Dense mode emits structs (and `map[string]any` of stable shape via the reflect-struct path) through `tagMapShape` (`0xEC`). First occurrence declares the shape inline: `0xEC, 0, varuint(N), N × key`. Subsequent occurrences of the same shape emit `0xEC + varuint(shapeID) + N × value` — keys are *not* re-emitted. Per-record saving on an array of identical-shape structs is `N × 2` bytes for the elided state-refs plus the map header. The shape table is per-stream, addressed by `*typeDesc` on the encoder and by sequential ID on the wire. Shapes never collide across types because the encoder keys the binding on the descriptor pointer. |
| **Arithmetic / range coding (rANS)** — push the encoded stream to its Shannon limit. | Shipped behind `OptRANS` (in `OptCompression`): a static order-0 rANS pass over the whole body, applied only when it shrinks (never larger). Squeezes the residual byte-entropy the structural codecs leave — e.g. −37 % on trace batches — at ~4–6× CPU, so it stays in the opt-in compression tier. |

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
| `0xEA`               | Dense: state-ref pair rank (Markov-1)         |
| `0xEB`               | QPack: run-length encoded integer slice       |
| `0xEC`               | Dense: struct/map shape declare / reuse       |
| `0xED`               | QPack: dictionary-coded integer slice         |
| `0xEE`               | QPack: Patched FOR integer slice (outliers)   |
| `0xEF`               | QPack: columnar `[]struct` container          |
| `0xF0..0xF3`         | ext / timestamp                               |
| `0xF4`               | QPack: ALP decimal-coded `[]float64` slice    |
| `0xF5`               | QPack: dictionary-coded string column          |
| `0xF6..0xFF`         | reserved (rANS, n-gram graph, future)         |

The 5th header byte holds two flag bits: `FlagDense` (`0x01`) for the
intern dialect, and `FlagQPack` (`0x02`) as an early hint that the body
may carry codec tags from the QPack codec range (`0xE3..0xEF`, `0xF4..0xF5`). A reader that does
not implement the QPack tags fails with `ErrBadTag` on first contact;
it never decodes a packed payload as scalar by accident.

> **Alpha note:** qdf is pre-1.0. The wire format is still being
> shaped — tags may shift, version may bump, and "older" decoders
> here mean "earlier in alpha", not stable releases. No
> backwards-compat promise yet.

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

    b, err := qdf.Marshal(in, qdf.OptSpeed)
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

### Encode profile is per-call

There is one encode entry point — `Marshal(v, opts)` — and the
`opts` bit-mask picks which codecs run for that specific call. The
convenience bundles cover the common tradeoffs:

| Bundle             | When to use                                                  |
| ------------------ | ------------------------------------------------------------ |
| `qdf.OptSpeed`     | Fast path. Tightest CPU cost; size comparable to msgpack. Drop-in for `encoding/json` behaviour. |
| `qdf.OptBalanced`  | Repetitive payloads — logs, telemetry, columnar rows. Strings intern once; numeric and bool slices use QPack codecs; struct shapes intern; Markov-1 + MTF run on state-refs. |
| `qdf.OptCompression` | `OptBalanced` plus the heavier wire-size codecs: Gorilla XOR for smooth float series, ALP for quantized/decimal `[]float64`, and a final order-0 rANS entropy pass over the whole body (never larger). Trades encode CPU for smaller wire — pick it for backup / cold storage. |
| custom mix         | Or-combine individual bits (`OptDense \| OptQPack \| OptShapeIntern …`) when one of the bundles is one click off the desired tradeoff. |

```go
b, _ := qdf.Marshal(event,    qdf.OptSpeed)        // hot path
b, _ := qdf.Marshal(batch,    qdf.OptBalanced)     // telemetry / logs
b, _ := qdf.Marshal(snapshot, qdf.OptCompression)  // backup / archive
b, _ := qdf.Marshal(payload,                       // tuned mix
    qdf.OptDense|qdf.OptQPack|qdf.OptShapeIntern)
```

A single `qdf.Unmarshal` reads everything. The wire header is self-
describing; the receiver never has to know which option mix the
sender picked.

Picking the right combo for a concrete workload — hot path, telemetry,
metric series, embeddings, backup — is covered in
[`docs/CHOOSING.md`](docs/CHOOSING.md), with head-to-head numbers vs
json and msgpack per scenario.

### Dense mode internals — what each bit buys you

`OptDense` activates the inline intern table. The four codec bits
that compose `OptBalanced` layer on top of it. They share one rule:
**the encoder picks the shortest variant per emission, so Dense wire
is never larger than the same payload at `OptSpeed`.** If the
predictors do not pay, the byte cost is identical to a plain
state-ref.

| Tag | Predictor | Wins when | Doesn't help when |
| --- | --------- | --------- | ----------------- |
| `0xE8` | Markov-0: same ID as the previous emission | Runs of the same value (`region`, `region`, `region`). | Distinct values in a row. |
| `0xE9` | MTF: encode LRU rank instead of raw id | Heavy reuse of a small hot subset of strings that were interned early. | Random access pattern with no recency. |
| `0xEA` | Markov-1: top-1 predictor of the previous emission's successor | Correlated pairs — `country` → `city`, `service` → `region`. Only emits when the raw id needs ≥ 2 varuint bytes, so the wire never grows on small state tables. | Unique transitions, low-cardinality state. |
| `0xEC` | Shape interning: declare struct shape once, reuse by id | Arrays / streams of the same struct type. Per-record saving is ≈ `N × 2` bytes (elided key state-refs + map header). | One-shot encodes of a unique struct type. |

The state table is shared **per stream** in `StreamEncoder` /
`StreamDecoder`, so a batch of 1 000 log events with eight distinct
region codes ships each region once. `Marshal*` calls do not share —
each call starts with an empty table.

**Where Dense is worth it:** logs, traces, telemetry, columnar rows,
event batches, audit streams, snapshot dumps — anything where a small
vocabulary of strings and a stable struct shape repeat thousands of
times. Expected envelope: 25 – 55 % smaller than JSON at 2–6× the
encode speed.

**Where Dense is *not* worth it:** single-shot encodes of small
unique payloads (Markov / shape predictors never reach their
amortisation point, you pay ≤ 2 bytes of prelude for nothing), or
strict-throughput paths where decode CPU is the bottleneck — Dense
decode is ~15 % slower than Fast because of the state-table lookups.
Reach for `OptSpeed` or `OptQPack` there.

**Where it is wrong:** *cryptographic / forensic* contexts that need
byte-stable wire across implementations and versions. Dense embeds
predictor state, intern-ID ordering, and shape IDs that depend on
emission order. Use `OptSpeed` if you hash or sign the wire.

### Bit reference

| Bit | What it gates |
| --- | -------------- |
| `OptDense`        | Inline intern table; required by everything below. |
| `OptQPack`        | Numeric / bool slice codecs (FOR, Delta+FOR, Gorilla, bit-pack). |
| `OptShapeIntern`  | `tagMapShape` for struct emissions (declare once, reuse by id). |
| `OptPairPred`     | Markov-1 successor predictor (`tagStatePair`, `0xEA`). |
| `OptMTF`          | Move-to-Front rank coding on state-refs (`tagStateMTF`, `0xE9`). |

| Bundle | Composition |
| ------ | ----------- |
| `OptSpeed`       | `0` — Fast mode, no codecs. |
| `OptBalanced`    | `OptDense \| OptQPack \| OptShapeIntern \| OptPairPred \| OptMTF`. |
| `OptCompression` | `OptBalanced` + Gorilla XOR + ALP decimal floats + an order-0 rANS entropy pass (never larger). Trades CPU for wire size; for backup / cold storage. |

`Options` is a `uint32` carried by value, so `Marshal` and
`AppendMarshal` add **zero per-call allocations** over the pool /
output-clone overhead. Encoders are pooled in a single shared pool
(`encPool`) keyed only by buffer; the configuration is applied on
each acquire via `applyOpts`. Markov-0 (`tagStateRepeat`, `0xE8`)
is always on inside `OptDense` because it costs nothing on the wire
when the predictor misses.

A note on tuning knobs that do *not* live on the bit-mask: the
intern threshold (`SetIntern`) and the cycle-depth ceiling
(`SetMaxDepth`) stay on the `*Encoder` itself. Use the low-level
encoder API for those.

### Generic API (Go 1.18+ generics)

`Marshal(v any, opts)` boxes the argument through `interface{}` and
then makes one reflect copy for value-typed inputs. The generic
helpers skip both: `T` is fixed at the call site,
`reflect.TypeFor[T]()` resolves at compile time, and
`unsafe.Pointer(&v)` points directly at the caller's stack.

| Generic                    | Equivalent to                 |
| -------------------------- | ----------------------------- |
| `MarshalT[T any]`          | `Marshal` (typed)             |
| `AppendMarshalT[T any]`    | `AppendMarshal` (typed)       |
| `UnmarshalT[T any]`        | `Unmarshal` (typed)           |

Wire output is byte-identical. The win is on the encode side: one
fewer allocation per call (-80 B) and 25–40 % faster on small/medium
payloads.

```go
buf, _ := qdf.MarshalT(event, qdf.OptSpeed)
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
| Patched FOR| integer slice with rare outliers (latency spikes)  | FOR body at a reduced width `b` + an exception list for the few values that don't fit. ~50% smaller than FOR on spiky columns. |
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

> **May 2026 perf series** (commits `ada9fd7`, `2ea3b48`, `02d6aac`,
> `7090e25`, `c0517e8`, `95d3c21`, `001864b`) rebuilt the encode +
> decode hot paths: MRU ring side-cache for MTF rank, flat
> open-addressed intern table replacing `map[string]uint32`,
> packed `lruLink`, large-payload buffer probe-and-grow, cached
> decode interning (`decState.stringValues`), 4-way `mruRank`
> unroll, opt-in `Encoder.PreIntern` API, and the `qdfgen`
> codegen path wired into the bench matrix. qdf is now strictly
> faster than `msgpack` on encode AND decode for every workload
> in the matrix (-5 % HotPath, -40 % TelemetryBatch, -96 %
> MetricSeries on encode; -55…-98 % on decode). See
> [`docs/GUIDE.md`](docs/GUIDE.md#performance-characteristics)
> for the up-to-date Profile_* matrix.

### Encode — ns/op

| Payload                  | json    | msgpack | qdf\_fast | qdf\_dense | vs json    | vs msgpack |
| ------------------------ | ------: | ------: | --------: | ---------: | ---------: | ---------: |
| Tiny (`{id,name}`)       |     192 |     286 |       227 |        398 |       0.85× | 1.26×      |
| Flat (20 primitive fields) |   1182 |    1262 |   **480** |        948 | **2.46×**  | **2.63×**  |
| Nested (4 deep)          |     446 |     793 |   **331** |        786 | **1.35×**  | **2.40×**  |
| Deep linked-list (16)    |    1157 |    3060 |   **477** |        814 | **2.43×**  | **6.42×**  |
| Wide × 1000              |   991 k |   991 k |  **213 k**|      289 k | **4.66×**  | **4.66×**  |
| Log batch × 1000         |   998 k |   624 k |  **186 k**|  **365 k** | **5.36×**  | **3.35×**  |
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
| Flat                     |     210 |     134 |   **132** |        138 | 0.66×         |
| Deep16                   |     239 |     139 |       166 |     **63** | **0.26×**     |
| Wide × 1000              | 212 901 | 135 626 |   128 632 | **66 702** | **0.31×**     |
| Log batch × 1000         | 251 902 | 185 639 |   185 649 | **85 440** | **0.34×**     |

On a 1000-entry Dense **stream** the same log batch encodes to
**85 bytes per entry**, against 251 for json and 186 for msgpack —
shared intern table across messages plus per-record struct headers
collapsed via shape interning (`tagMapShape`).

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
| **qdf_dense** | **50 129** | **0.20×** |    1.0 M     |

Dense pays ~4× CPU for **5.0× smaller wire** vs JSON — string-intern
+ Markov-0 + MTF + Markov-1 pair + shape interning crush the
repeating service / region / level / host fields AND elide per-record
struct headers. QPack does not help much here (per-event numeric
fields are scalar, not slice-shaped).

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

#### Large payload (~150 MiB, 200 000 records, every supported type)

`bench/largepayload_test.go` builds a 200 000-record corpus that
exercises every qdf-supported field type (scalars, hot/cold strings,
nested map, `[]int32` / `[]float64`, `[]byte`, UUIDs) and measures
size + encode/decode latency + working-set memory delta across json,
msgpack, qdf_fast, qdf_qpack, qdf_dense.

Sizes (200 000 records):

| Format       |   bytes (MiB) |  vs json |
| ------------ | ------------: | -------: |
| json         |       142.10  |    1.00× |
| msgpack      |        92.99  |    0.65× |
| qdf_fast     |        91.99  |    0.65× |
| qdf_qpack    |        89.65  |    0.63× |
| **qdf_dense** |     **71.19** | **0.50×** |

Latency + memory (100 000 records):

| Format        | encode (ms) | decode (ms) | encode heap delta |
| ------------- | ----------: | ----------: | ----------------: |
| json          |       1 070 |       1 744 |          199 MiB  |
| msgpack       |         296 |         597 |           64 MiB  |
| qdf_fast      |     **142** |         300 |           94 MiB  |
| qdf_qpack     |         147 |         231 |           93 MiB  |
| **qdf_dense** |         169 |     **216** |       **9.7 MiB** |

Reproduce:

```bash
go test -C bench -run TestSizes_LargePayload -count=1 -timeout=10m
go test -C bench -run TestMem_LargePayload   -count=1 -timeout=10m
```

Both helpers skip under `-short`; the size test allocates ~600 MiB
during the run, the mem test ~400 MiB.

### Reproduce

```bash
# Default build
cd bench
go test -bench=. -benchmem -benchtime=2s -timeout=10m

# QPack head-to-head: Marshal(_, OptSpeed) vs Marshal(_, OptQPack)
# vs Marshal(_, OptBalanced) vs json / msgpack on the same payload
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

### Profile-guided optimisation (downstream)

Go's PGO applies to the main package being built, so a library like
qdf cannot ship its own profile. If you build a service that imports
qdf, collect a representative profile of *your* workload and drop it
next to your `main`:

```bash
# Run your service / load test under cpuprofile.
go test -bench=. -cpuprofile=cpu.pprof -benchtime=10s ./...
mv cpu.pprof default.pgo            # same dir as your main package
go build .                          # auto-picks up default.pgo
```

The Go toolchain will then recompile the qdf functions on the hot
path with PGO-driven inlining and devirtualisation. Typical gain is
5–15 % across the encode/decode pipeline on top of the numbers above.

---

## Build tags

Opt-in fast paths. None are required for the headline numbers; the
defaults already include the specialised slice / map encoders, the
QPack codecs, and a 128-bit sliding-window bit-unpacker for FOR / Delta+FOR.

| Tag             | Effect                                                                                                  | Build prerequisite |
| --------------- | ------------------------------------------------------------------------------------------------------- | ------------------ |
| `qdf_reflect2`  | Swap `reflect.MakeSlice` / `MakeMapWithSize` for `modern-go/reflect2` unsafe equivalents. Smaller decode allocations on map / slice heavy workloads. | none — pure Go     |
| `qdf_simd`      | SIMD fast paths for the QPack integer/bool codecs. **amd64 (AVX2):** decode bits ∈ {8,16,32} (`VPMOVZX`) and every width 1–28 (`VPBROADCASTQ`+`VPSRLVQ`), encode {8,16,32} (`VPSHUFB`) and {10,12,14,20} (`VPSLLVQ`+lane-OR), `[]bool` pack; runtime CPUID gate, non-AVX2 falls back. **arm64 (NEON):** decode 1–28 plus 32 (encode/bool scalar for now), baseline. ~3-11× over pure-Go on accelerated widths. Output byte-identical to scalar; other arches compile a stub. See `docs/USAGE.md`. | amd64 (AVX2) / arm64 (NEON) |

```bash
go build -tags qdf_reflect2 ./...
go build -tags qdf_simd ./...
go build -tags "qdf_simd qdf_reflect2" ./...   # combined
```

The `qdf_simd` path only changes the speed of QPack-encoded numeric and
bool slices at the accelerated widths; string / map / struct paths, the
float codecs, and other bit widths run the pure-Go path unchanged. It is
a pure speed switch — the wire format and output are identical.

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
  interning) through `OptSpeed`, `OptQPack`, and `OptBalanced`,
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
  the public `Marshal(v, OptBalanced)` / `Unmarshal` boundary.
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
  OptSpeed / OptQPack / OptBalanced bytes committed to disk. A
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

## Status

Alpha. The wire format is stable for the `0x01` version byte; future
versions will bump it. The public API may change before the first
tagged release — pin a commit if you depend on it.

## License

MIT. See [LICENSE](LICENSE).
