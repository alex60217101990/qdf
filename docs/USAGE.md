# qdf Usage Guide

Practical how-to for the common cases. For the "why is it shaped this
way" deep-dive see [`docs/GUIDE.md`](GUIDE.md). For a quick opts
cheatsheet see [`docs/CHOOSING.md`](CHOOSING.md).

---

## Quick start

### Marshal / Unmarshal

```go
import "github.com/alex60217101990/qdf"

type Event struct {
    Service string    `qdf:"service"`
    Level   string    `qdf:"level"`
    TS      int64     `qdf:"ts"`
    Msg     string    `qdf:"msg"`
}

b, err := qdf.Marshal(event, qdf.OptBalanced)
if err != nil { ... }

var out Event
err = qdf.Unmarshal(b, &out)
```

`Marshal` returns a freshly-allocated `[]byte` owned by the caller.
`Unmarshal` reads any buffer produced by `Marshal` — the wire
self-describes its codec dialect, so the decoder does not need to know
which `Options` the encoder used.

### Generic variant

`MarshalT[T]` skips the `any`-boxing that `Marshal` does on value
types. Same wire, one fewer allocation on small/medium structs.

```go
b, err := qdf.MarshalT(event, qdf.OptBalanced)
out, err := qdf.UnmarshalT[Event](b)
```

### Buffer reuse with AppendMarshal

On hot paths that produce many messages, reuse the output slice to
avoid per-call allocations:

```go
var buf []byte
for _, ev := range events {
    var err error
    buf, err = qdf.AppendMarshal(buf[:0], ev, qdf.OptSpeed)
    if err != nil { ... }
    sink.Write(buf)
}
```

Pass `buf[:0]` to reset length while keeping capacity. The returned
slice is valid until the next call.

---

## Choosing an Options preset

| Preset | What it is | When to use |
|---|---|---|
| `OptSpeed` | No codecs. Raw tag stream. | Hot request path, single events, latency under 1 µs. |
| `OptBalanced` | Dense interning + QPack + shape interning + Markov predictors + MTF. | Telemetry, logs, event batches with repeating string fields or numeric slices. Good default. |
| `OptCompression` | `OptBalanced` + Gorilla XOR and ALP decimal float coding + a final order-0 rANS entropy pass over the whole body. | Cold storage, backup, archive. Wire size matters more than encode CPU. |

Decision list:

- Hot request path, one event per call → `OptSpeed`
- Telemetry / logs / event batches, general use → `OptBalanced`
- Float time-series being archived, wire size dominates → `OptCompression`
- Numeric vectors only (metrics, embeddings), no repeated strings → `qdf.OptQPack`

### The rANS entropy pass

`OptCompression` adds `OptRANS`: after the body is encoded, a static order-0
rANS pass entropy-codes the whole body. It is **never larger** — the encoder
keeps the plain body unless the rANS form is strictly smaller — and it kicks in
only for bodies above ~512 B, so small messages are untouched. It is a
whole-buffer pass, so `StreamEncoder` ignores it (streaming stays per-message).

It pays off where the encoded body still has byte-level redundancy that the
structural codecs do not remove — most visibly **string/hex-heavy** data:

| workload | `OptCompression` w/o rANS | with rANS |
| -------- | ------------------------: | --------: |
| trace batch (unique hex IDs) | 33 607 | **21 003** (−37%) |
| log batch (repeated fields) | ~8 800 | **~5 200** (−40%) |
| smooth metric series (Gorilla) | 2 307 | **1 671** (−27%) |
| already-dense numeric (FOR/dict) | — | unchanged (rANS declines) |

The cost is encode/decode CPU: roughly **4–6× slower** on the bodies where it
fires (it does extra entropy-coding work). That trade is why it lives only in
`OptCompression` — use it for archives and cold storage, not hot paths. You can
opt out while keeping the float codecs with `OptCompression &^ OptRANS`.

---

## Composing your own bitmask

Presets are just `|`-combined constants. You can build your own:

```go
b, err := qdf.Marshal(v, qdf.OptDense|qdf.OptQPack|qdf.OptShapeIntern)
```

Bit reference:

| Bit | Constant | Effect | Requires |
|-----|----------|--------|----------|
| 0 | `OptDense` | Inline intern table; repeated strings/bytes → back-references by ID. | — |
| 1 | `OptQPack` | Numeric/bool slice codecs: bit-pack bools, FOR, Delta+FOR, RLE, dict. Auto-picks per slice. | — |
| 2 | `OptShapeIntern` | Struct shape table: first emit declares field order, subsequent emits write only shape ID + values. | `OptDense` |
| 3 | `OptPairPred` | Markov-1 successor predictor over intern IDs (two-byte hit when transition is predictable). | `OptDense` |
| 4 | `OptMTF` | Move-to-Front rank coding over intern IDs (shorter varuint when LRU rank < raw ID). | `OptDense` |
| 5 | `OptGorillaFloat` | Gorilla XOR codec for `[]float64`/`[]float32`. ~70% wire reduction on smooth time-series; ~10× CPU/slice. | `OptQPack` |
| 6 | `OptRANS` | Order-0 rANS entropy pass over the whole body. Never larger (applied only when it shrinks); ~4–6× CPU where it fires; whole-buffer (not for `StreamEncoder`). | — |

Dependent bits without their parent are silent no-ops — the encoder
does not error, it just ignores them. `OptSpeed = 0` is the zero value.

```go
// This silently encodes in Fast mode — no intern table.
// OptShapeIntern and OptMTF need OptDense.
qdf.Marshal(v, qdf.OptShapeIntern|qdf.OptMTF)

// Correct: include OptDense explicitly.
qdf.Marshal(v, qdf.OptDense|qdf.OptShapeIntern|qdf.OptMTF)
```

---

## How your data shape maps to codecs

QPack (`OptQPack`) auto-selects the best codec per slice at encode
time. You do not need to hint it — just set the bit.

| Data shape | Codec fired | Preset that enables it |
|---|---|---|
| `[]bool` | bit-pack (1 bit/elem) | `OptQPack` |
| `[]intN`/`[]uintN`, clustered range | Frame-of-Reference (FOR) | `OptQPack` |
| `[]int64`/`[]uint64`, monotonic or time-series | Delta+FOR | `OptQPack` |
| `[]intN`, run-heavy (status codes, enum-like, sparse counters) | RLE (value, runLen pairs) | `OptQPack` |
| `[]intN`, small distinct cardinality (≤16), wide value range | Dictionary codec | `OptQPack` |
| `[]float64`/`[]float32`, smooth time-series | Gorilla XOR | `OptCompression` (or `OptGorillaFloat`) |
| `[]float64`, quantized/decimal grid (prices, percentages, latencies) | ALP decimal (integer-mantissa FOR + exception list) | `OptCompression` |
| Repeated strings / `[]byte` across messages | Intern table + state-ref | `OptDense` |
| Arrays of identical struct type | Shape interning | `OptBalanced` |
| Predictable field transitions (e.g. service→region) | Markov-1 pair predictor | `OptBalanced` |
| `[]SomeStruct` where fields are int/uint/float/bool/string/[]byte | Columnar transpose: numeric fields get FOR/delta/RLE/dict, repeated string fields collapse via intern. Automatic — no flag. The encoder probes each array and falls back to row-major when columnar would not win. | `OptBalanced` (Dense + ShapeIntern) |

The encoder probes a slice's structure before committing. For
integer slices it evaluates FOR, Delta+FOR, RLE, and dict by
predicted wire size and picks the smallest. For `[]float64` under
`OptCompression` it picks the smallest of raw-LE, a Gorilla XOR
projection (32-sample probe), and an ALP decimal estimate; the ALP
estimate is a conservative upper bound, so ALP is chosen only when it
is strictly smaller than both — no smooth-float workload regresses.

FOR/Delta+FOR win on tight or monotonic integer ranges. RLE wins when
a handful of distinct values repeat in long runs. Dict wins when the
cardinality is small but values are spread wide (e.g. HTTP status
codes 200/301/404/500 scattered randomly — no long runs, not a tight
range). Gorilla wins on smooth sensor data; it loses on white noise.
ALP wins on quantized/decimal streams (2-decimal metrics, prices,
latencies) that sit on a fixed grid Gorilla cannot exploit.

---

## The `qdf:"name"` struct tag

qdf reads `qdf:"name"` struct tags to determine the wire field name.
If `qdf:"name"` is absent it falls back to `json:"name"`, then the
field name as-is.

```go
type Record struct {
    UserID   int64  `qdf:"user_id"`
    Email    string `qdf:"email"`
    Internal string `qdf:"-"`   // skip this field
}
```

The wire field name matters for back-compatibility and for the Dense
intern table: two struct types with the same `qdf:"name"` field set
share the same shape ID on the wire. If you rename a field tag, old
decoders that `Unmarshal` into a struct that still has the old tag
will silently not populate that field (the wire key no longer matches).

---

## Concurrency contract

`Marshal`, `AppendMarshal`, and `Unmarshal` are safe for concurrent
use from many goroutines. Each call leases an encoder or decoder from
an internal `sync.Pool`, does its work, and returns it. No shared
mutable state between calls.

A single `*Encoder`, `*Decoder`, or `*StreamEncoder` instance is
**not** safe to share across goroutines. Use one per goroutine, or
protect with a mutex.

A single Dense document is a sequential stream: values back-reference
each other via intern IDs that only make sense in emission order. One
document cannot be encoded or decoded in parallel. To parallelize a
large dataset:

```go
// Split into independent shards. Each shard encodes separately;
// each resulting []byte is self-contained and decodes independently.
shards := splitIntoChunks(bigSlice, chunkSize)
results := make([][]byte, len(shards))
var wg sync.WaitGroup
for i, shard := range shards {
    wg.Add(1)
    go func(i int, s []MyStruct) {
        defer wg.Done()
        b, err := qdf.Marshal(s, qdf.OptBalanced)
        if err != nil { ... }
        results[i] = b
    }(i, shard)
}
wg.Wait()
// Decode results[i] independently in any goroutine.
```

---

## Tuning checklist

- Pick the preset that matches your workload (see table above). Do not
  reach for `OptCompression` on hot paths.
- If your payload is purely numeric (metrics, embeddings) with no
  repeated strings, use `OptQPack` alone — `OptDense` adds CPU for no
  wire benefit.
- Reuse the output buffer with `AppendMarshal(buf[:0], ...)` on hot
  paths to eliminate per-call allocations.
- If you have smooth float time-series going to cold storage, use
  `OptCompression` — the Gorilla codec cuts float wire size ~70% at
  the cost of ~10× more CPU per slice, which is fine for archival.
- For large datasets that need to be processed in parallel, split them
  into independent shards and `Marshal` each one separately. Each
  `[]byte` decodes independently with no coordination.
- The `qdfgen` code generator (`cmd/qdfgen`) emits typed
  `MarshalQDF`/`UnmarshalQDF` methods that bypass reflect entirely.
  Worth it for hot paths encoding the same struct type millions of
  times — typically 30–60% faster than the reflect path on encode.
- Build with `-tags qdf_simd` on amd64 to accelerate the numeric-slice
  codecs with AVX2. It is opt-in and changes nothing else — see the
  section below for when it actually helps.

---

## SIMD acceleration (`qdf_simd` build tag)

`qdf_simd` is an **opt-in build tag** that swaps hand-written AVX2 assembly
into the inner loops of the numeric-slice (QPack) codecs:

```bash
go build -tags qdf_simd ./...
go test  -tags qdf_simd ./...
```

It is **off by default**. Turning it on does **not** change the wire format,
the API, or the output — encoded bytes are byte-for-byte identical with and
without the tag. It is purely a speed switch, safe to flip.

### What it speeds up

Only the QPack codecs that pack fixed-width integers and booleans:

| Operation | Accelerated widths | Typical speedup vs scalar |
| --------- | ------------------ | ------------------------- |
| Integer **decode** (FOR) | 8, 16, 32 (byte-aligned, `VPMOVZX`) | ~3–8× |
| Integer **decode** (FOR) | 1–14 (4 values/iter, `VPSRLVQ`) | ~7–11× |
| Integer **decode** (FOR) | 15–28 (2 values/iter, `VPSRLVQ`) | ~5–7× |
| Integer **encode** (FOR) | 8, 16, 32 (`VPSHUFB`) | ~2–5× |
| Integer **encode** (FOR) | 10, 12, 14, 20 (`VPSLLVQ` + lane-OR) | ~4–5.5× |
| `[]bool` pack | — | large |

Decode is the most broadly accelerated: every bit width from 1 to 28 (plus
32) has a SIMD path. Only widths 29–31 and 33–56 still run the scalar
window — uncommon for real FOR deltas.

Everything else runs the same scalar code whether or not the tag is set:
strings, maps, struct shapes, varints, the Gorilla and ALP float codecs, and
integer widths not listed above. The tag simply doesn't touch them.

### When it helps

- Payloads dominated by **large numeric or boolean slices** that go through
  QPack/FOR: metrics, counters, sensor batches, columnar numeric rows,
  integer-quantized embeddings. The bigger the slices, the bigger the win —
  it is a per-element inner-loop speedup, so fixed overhead dominates on small
  slices.

### When it does *not* help

- Small payloads, single objects, config-shaped data (few or short slices) —
  the overhead swamps any gain.
- String-heavy, map-heavy, or struct-shape-heavy payloads — those paths are
  scalar regardless of the tag.
- Float time-series (Gorilla / ALP) — not SIMD-accelerated yet.

### Architectures

- **amd64 with AVX2** is the only accelerated target. A runtime CPUID check
  gates the asm: on a CPU without AVX2 it transparently falls back to scalar,
  so the binary still runs correctly everywhere.
- **Other architectures** (arm64, etc.) compile a scalar stub — building with
  the tag is safe but gives no speedup. (arm64 NEON is a possible future
  addition, not implemented today.)
- No special environment is required — plain `-tags qdf_simd` is enough.

### Recommendation

Use the default build unless profiling shows that encode/decode of numeric
slices is a hot spot **and** your payloads carry big integer/bool slices. In
that case enable `qdf_simd` and measure on your real data
(`go test -bench` with and without the tag) — if your data is string- or
struct-heavy, expect little change. Because the output is identical and the
fallback is automatic, it is safe to enable broadly; worst case it is a no-op.
