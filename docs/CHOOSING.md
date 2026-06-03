# Choosing the right `Options` (and build tags)

qdf has one encode entry point: `Marshal(v, opts)`. The `opts` bit-mask
picks which codecs run. This page is a cheatsheet for the choice.
For the full API reference and data-shape-to-codec mapping see [`docs/USAGE.md`](USAGE.md).

If you don't read anything else, copy one of these:

```go
// hot path / small messages — minimum CPU, no codec machinery
qdf.Marshal(v, qdf.OptSpeed)

// telemetry / logs / event batches — repetitive strings, mixed types
qdf.Marshal(v, qdf.OptBalanced)

// numeric / float vectors — no string interning needed
qdf.Marshal(v, qdf.OptQPack)

// backup / archive — squeeze every byte, CPU is fine
qdf.Marshal(v, qdf.OptCompression)
```

`OptCompression` diverges from `OptBalanced` by one bit:
`OptGorillaFloat`. That bit opts in to the Gorilla XOR codec for
`[]float64` / `[]float32` slices — wire collapses ~70 % on smooth
time-series, encode/decode pay ~10× more CPU on those slices because
the body is bit-level. Anything dominated by float slices that has to
live small (archives, cold storage, paginated history) wants
`OptCompression`; anything hot path stays on `OptBalanced` /
`OptQPack`. Both bundles share every other codec; only the Gorilla
bit differs.

Arrays of flat homogeneous structs are compressed columnar automatically
under `OptBalanced` and above — no flag needed. Within that path, enum-like
string columns (log level, service, region, status) are dictionary-coded
(distinct table + a bit-packed index per row) when that beats per-value
interning, which is a large win on wide low-cardinality dimension columns.

---

## Decision tree

Three questions, in this order:

1. **Does the payload have repeating strings?** (service names, status
   codes, region tags, field names that appear thousands of times)
   - Yes → `OptDense` baseline.
   - No → skip Dense; pick between `OptSpeed` and `OptQPack`.

2. **Does the payload have numeric / boolean slices longer than a few
   dozen elements?** (metric series, embeddings, sensor arrays,
   bitmaps)
   - Yes → add `OptQPack`.
   - No → skip QPack.

3. **Is it an array / stream of the *same struct type*?** (every event
   has the same fields)
   - Yes → add `OptShapeIntern` (only useful with `OptDense`).
   - No → skip.

If you said *yes* to all three you've assembled `OptBalanced`.

---

## What each bit does, concretely

| Bit              | Wire tag | Wins on                                                  | Doesn't help on              | Depends on    |
| ---------------- | -------- | -------------------------------------------------------- | ---------------------------- | ------------- |
| `OptDense`       | `0xE0..` | Repeated strings / `[]byte`. Intern table back-refs.    | Strings unique per call.     | —             |
| `OptQPack`       | `0xE3..` | Numeric / bool slices ≥ ~16 elements.                   | Per-scalar fields.           | —             |
| `OptShapeIntern` | `0xEC`   | Arrays / streams of identical struct types.             | One-off struct values.       | `OptDense`    |
| `OptPairPred`    | `0xEA`   | Conditional pairs (`country` → `city`, `svc` → `host`). | Random / uncorrelated state. | `OptDense`    |
| `OptMTF`         | `0xE9`   | Hot subset reuse on a >128-entry intern table.          | Small intern tables.         | `OptDense`    |
| `OptGorillaFloat`| `0xE7`   | Smooth float time-series. ~70 % wire reduction.         | Random / unrelated floats; latency-sensitive paths (10× CPU/slice). | `OptQPack`    |
| `OptColumnIndex` | `0xEF`   | Wide columnar `[]struct` batches read column-subset.    | Consumers that read all columns; non-columnar payloads. | `OptBalanced` |
| `OptFSST`        | `0xF6`   | High-cardinality columnar string columns (URLs, log lines, paths) where the whole-string dictionary can't help. ~76–79 % wire reduction on URL/log-line corpora. | Low-cardinality or short string columns (dict wins there); single messages; streaming. | `OptQPack` + columnar (`OptBalanced`) |

Dependent bits set without their parent are no-ops — the gating code
ignores them. Reserved bits (6..31) are reserved; never use them.

---

## Scenario recipes

Numbers below: bench/profiles_test.go on Intel i7-9750H, Go 1.26.0,
median of two `-benchtime=300ms` runs. The full table is in
`docs/BENCH.md`.

### Hot path — small single-message encode

You have a 5–10-field event, you encode one at a time, latency budget
under 1 µs, no codec setup amortises. Strings don't repeat across
calls because each call is independent (the encoder pool is shared
but the intern table is per-call when using `Marshal`).

**Recipe:** `qdf.OptSpeed`

```go
b, err := qdf.Marshal(event, qdf.OptSpeed)
```

Result: encode 370 ns vs json 813 ns vs msgpack 520 ns. Decode 368 ns
vs json 1954 ns vs msgpack 766 ns. Wire 70 B vs json 97 B (msgpack
wins on size here at 63 B — its fixmap header is one byte cheaper).

### Telemetry / log batch

You collect 1 000 events at a time. `service` is one of five values,
`region` one of three, `level` one of three. Same struct type for
every row.

**Recipe:** `qdf.OptBalanced`

```go
b, err := qdf.Marshal(rows, qdf.OptBalanced)
```

Wire 40 KB vs json 143 KB vs msgpack 112 KB (3.5× smaller than json,
2.8× vs msgpack). Decode 458 µs vs json 2894 µs (6.3× faster). Encode
353 µs vs json 440 µs (1.25× faster) — the lazy encoder-state alloc and
reset-skip in `OptSpeed` eliminated the old CPU tax; you now get the
size win for free on both encode and decode.

If you control the receiver and care about decode latency more than
encode, this is a clear win.

### Metric series / time-series data

Two parallel slices (`[]int64` timestamps, `[]float64` values), small
header. No repeated strings. Numeric slices are large enough that QPack
codecs (Delta+FOR for monotonic ints, Gorilla XOR for floats) pay off.

**Recipe:** `qdf.OptQPack`

```go
b, err := qdf.Marshal(series, qdf.OptQPack)
```

Result: encode **6.7 µs** vs json 189 µs (28× faster) vs msgpack 129 µs.
Decode 5.3 µs vs json 664 µs (125× faster), msgpack 181 µs (34×). Wire
8.4 KB vs json 37 KB (4.4× smaller). Adding `OptDense` here costs
intern-table CPU for zero wire benefit — skip it.

### Status-code / enum-like int column

A long `[]int` (or `[]int32`/`[]int64`) dominated by a handful of distinct
values — HTTP status codes, log severities encoded as ints, sparse
counter snapshots, finite-state-machine traces. Long stretches of
the same value with short bursts of others is the canonical shape.

**Recipe:** `qdf.OptQPack`

```go
b, err := qdf.Marshal(batch, qdf.OptQPack)
```

A 1024-element status column built from realistic HTTP traffic
(mostly 200, occasional 4xx/5xx incident bursts) lands at **99
bytes** of wire — 42× smaller than json (4.1 KB), 93× smaller than
msgpack (9.2 KB). Encode 7.1 µs vs json 29 µs (4.0× faster);
decode 2.7 µs vs json 155 µs (58× faster). The picker runs a cheap
run-fraction probe over the first 32 elements and only commits to
RLE if the win is real, so unrelated random `[]int` columns stay
on raw / FOR without paying the estimator cost.

### Latency / outlier-heavy int column

A long `[]int`/`[]uint32`/`[]uint64` that is mostly small but carries a rare
tail of large values — request latencies in microseconds with the
occasional slow request, byte counters with the odd jumbo payload,
counters that reset. The spikes are what kill plain Frame-of-Reference:
a handful of large values force every slot to a wide bit count.

**Recipe:** `qdf.OptQPack`

```go
b, err := qdf.Marshal(batch, qdf.OptQPack)
```

The picker evaluates Patched FOR (`tagPackPFor`): it packs the common
case at a reduced bit width and stores the few outliers in an exception
list. On a 1024-element latency column with ~1 % spikes this lands at
**1 215 bytes** versus 2 566 for plain FOR (~53 % smaller). The PFOR
cost estimate uses a conservative upper bound, so it is chosen only
when strictly smaller than every other codec — a clean, tightly-ranged
column keeps FOR and the wire is byte-identical.

### Embedding vector / dense float array

Single `[]float32` or `[]float64` of 100s–1000s of elements. The
identifying string is small.

**Recipe:** `qdf.OptQPack`

```go
b, err := qdf.Marshal(vec, qdf.OptQPack)
```

Result: encode 903 ns vs json 80 µs (**89× faster**) vs msgpack 36 µs
(40×). Decode 904 ns vs json 213 µs (**236× faster**). Wire 3 KB vs
json 8.4 KB. For embeddings this is the difference between a viable
storage format and a bottleneck.

### Map-heavy config

Stable shape, small map fields, one-off encode (not array of). Strings
inside the maps may or may not repeat.

**Recipe:** `qdf.OptBalanced` — usually a wash vs `OptSpeed`. If the
program restarts on every encode, `OptSpeed`. If the encoder is
long-lived and the same payload encodes 100× with the same keys,
`OptBalanced` (the intern table amortises across calls when you use a
`*qdf.Encoder` directly — Marshal allocates a fresh state per call).

For map-heavy payloads, qdf decode is consistently faster than
msgpack thanks to the per-decoder key intern cache (1.7× faster on
the config fixture).

### Backup / archive

Largest dataset you care about. Decode time is "whenever you restore".
You want bytes-on-disk minimised.

**Recipe:** `qdf.OptCompression`

```go
b, err := qdf.Marshal(snapshot, qdf.OptCompression)
```

Result on a 5 000-row archive: wire 125 KB vs json 715 KB (5.6×) vs
msgpack 558 KB (4.4×). Decode 4.1 ms vs json 15.8 ms (3.9×) vs
msgpack 6.1 ms (1.5×). Encode 4.4 ms vs json 2.4 ms — fair for
backup workloads. (Wire shrunk ~34% vs prior measurements: the
dict-cap expansion and time-codec alignment eliminated padding waste
in the columnar container.)

On a 1024-sample smooth metric series `OptCompression` shrinks the
float body from 8398 B (raw-LE under `OptQPack`) to 1671 B (Gorilla
XOR + rANS pass), an 80 % drop. Encode/decode go from ~6.7 µs to
~41 µs — fine when the series is being archived once and queried over
its lifetime, not for the hot ingest path.

### Wide columnar batch, consumers read a column subset

A `[]SomeStruct` with many fields (16-field event rows, wide metric
records) where the readers usually project a handful of columns — a
dashboard reads `ts` + `level`, a billing job reads `amount` only.

**Recipe:** `qdf.OptBalanced | qdf.OptColumnIndex` on the producer, then
read a subset with a typed subset struct (`Unmarshal`) or named columns
(`UnmarshalColumns`).

```go
b, _ := qdf.Marshal(events, qdf.OptBalanced|qdf.OptColumnIndex)

var rows []map[string]any
_ = qdf.UnmarshalColumns(b, &rows, "ts", "level") // skips the other 14
```

Use it when consumers read a subset of columns from wide columnar
batches: the index makes a skip a direct offset add, so decoding a
3-field subset of a 16-field batch is ≈5.7× faster and moves ≈5.3×
fewer bytes than decoding the whole struct. The cost is ~4 B per column on
the wire. **No benefit** if every consumer reads all columns or the
payload is not columnar — leave the bit off and the wire is byte-
identical to plain `OptBalanced`. (Without the index, subset decode
still works correctly; it just decodes-and-discards the rest.) Not
emitted in streaming mode.

### Filter rows from a wide columnar batch

Same wide `[]SomeStruct`, but the reader wants only the rows matching a
condition — `level == "ERROR"`, `code >= 500` — not every row. Use
**predicate pushdown**: pass `qdf.Where(field, pred)` (typed, AND-ed) and
optionally `qdf.Select(fields...)` to `Unmarshal`.

```go
b, _ := qdf.Marshal(events, qdf.OptBalanced|qdf.OptColumnIndex)

var hot []map[string]any
_ = qdf.Unmarshal(b, &hot,
    qdf.Where("level", func(s string) bool { return s == "ERROR" }),
    qdf.Select("ts", "code")) // keep only matching rows, ts+code columns
```

On a wide batch at 1% selectivity it moves ≈4× fewer bytes and runs ≈2.1×
faster than a full decode + manual filter — and no other Go serializer can
do it at all. Full guide: [PREDICATE-PUSHDOWN.md](PREDICATE-PUSHDOWN.md).

### High-cardinality string column (URLs / log lines / paths)

You have a `[]struct` batch where one or more string columns carry many
distinct values that share substrings — request URLs, log messages, file
paths, stack traces. The whole-string dictionary codec (`tagColStrDict`)
can't help here (too many distinct values to win), but the values are not
random either — they share common prefixes, hostnames, method names, etc.

**Reach for `OptCompression`** (bundles `OptFSST` together with
Gorilla/ALP/rANS):

```go
b, err := qdf.Marshal(rows, qdf.OptCompression)
```

Or use `OptFSST` alone without the float codecs or rANS:

```go
b, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptFSST)
```

FSST is **never-larger** — it is emitted only when strictly smaller than the
dictionary and per-value paths. Default tiers (`OptSpeed`, `OptBalanced`
without `OptFSST`) are byte-identical to before.

**If you re-encode the same column shape repeatedly** (same URL space, same
log format), the dominant FSST cost is per-batch symbol-table training. Train
once and reuse with `FSSTDict`:

```go
// Once, at startup or schema change:
d := qdf.TrainFSSTDictStrings(sampleURLs)

// Per batch — skips training; enables FSST + columnar prerequisites:
b, err := d.Marshal(rows, qdf.OptCompression)
```

The reusable dictionary is ~5× faster to encode than per-batch training and
uses far fewer allocations. It is immutable, bounded (≤255 symbols, a few KB),
and safe for concurrent use. The wire is self-describing — each column still
carries its symbol table — so output decodes with a plain `Unmarshal` and
needs no out-of-band dictionary.

**Trade-off:** FSST encode is a storage-tier CPU cost (same family as Gorilla
~10× relative to `OptBalanced`). Decode is cheap and transparent —
`Unmarshal` handles `tagColStrFSST` automatically, works with
`Where`/`Select`, and honours `WithNoCopy()`.

### What about streaming?

`StreamEncoder` carries the intern + shape table across `Encode` calls
in the same stream. That's where Dense really shines:

```go
enc := qdf.NewStreamEncoder(w, qdf.Dense)
for _, ev := range batch {
    if err := enc.Encode(ev); err != nil { return err }
}
enc.Close()
```

For finer control use `NewStreamEncoderWith(w, opts)` and pass the
exact opts you want.

---

## When to use `WithNoCopy()` (decode)

There are no encode-style "decode opts", but `Unmarshal` takes one decode
modifier: `WithNoCopy()`. It makes decoded `string`/`[]byte` values **alias the
input buffer** instead of copying — near-zero allocations, ~1.7× faster on
string-heavy payloads (measured 7002 → 3 allocs/op on a 1000-row batch).

Use it when **the input buffer outlives the decoded values and is never
mutated or reused**:

- a file / blob read fully into memory, decoded once, used read-only;
- an `mmap`-ed region;
- batch analytics over a buffer you own for the whole computation.

Do **not** use it when the buffer is recycled or mutated — most importantly a
pooled server request body. The aliased values become silent garbage once the
buffer is reused (a use-after-free the race detector won't catch). That is why
it is opt-in and the default copies. For a `Decoder`/`StreamDecoder` you drive
directly, the equivalent is `SetNoCopy(true)`.

Works on both the reflect path and codegen types (the generated
`UnmarshalQDFOpts` threads the flag through nested struct decodes). It composes
with `Select`/`Where`.

---

## Anti-patterns

- **`OptDense` on unique strings.** Wire identical to `OptSpeed`, just
  pays CPU. Profile before assuming Dense helps.
- **`OptShapeIntern` on `map[string]any`.** Shape detection is
  per-type, not per-map-instance. `map[string]any` has no fixed
  schema — the bit is a no-op for that path. Use struct values.
- **`OptCompression` on small messages.** Same wire as `OptBalanced`,
  CPU overhead dominates.
- **Hashing or signing the Dense wire.** Dense embeds intern IDs and
  shape IDs that depend on emission order. Two semantically-equal
  payloads can hash differently. Use `OptSpeed` if you sign.
- **`OptQPack` on small numeric slices** (under ~8 elements).
  Per-codec headers (kind byte, varuint count) make tiny slices
  larger than the per-element tag stream. The auto-codec selector
  falls back when nothing wins, so the cost is just one comparison —
  but you can save it by passing `OptSpeed` when you know the slices
  are tiny.
- **Different `opts` across encode/decode.** There are no encode-style
  "decode opts" — `Unmarshal` reads the header flags and the tag stream
  and handles every variant. (The one decode modifier is `WithNoCopy()`;
  see above.) The encoder opts pick what you emit; the decoder reads
  whatever it gets.
- **`WithNoCopy()` on a recycled/mutated buffer.** Decoded values alias
  the input; if it is a pooled server request body or any buffer you
  reuse, the values silently corrupt. Use it only for owned, long-lived,
  immutable input.

---

## Build tags

Compile-time switches, orthogonal to runtime `Options`. Build with
`-tags <name>`.

| Tag           | What it does                                                          | When to use                                                   |
| ------------- | --------------------------------------------------------------------- | ------------------------------------------------------------- |
| `qdf_simd`    | Compiles in AVX2/NEON bit-unpack shared by every bit-packed integer codec (FOR, Delta+FOR, dict, Patched FOR) on the decode side. | amd64 (AVX2, CPUID-gated) or arm64 (NEON, baseline), large numeric payloads under `OptQPack` or `OptBalanced`. Safe to ship even without AVX2 (falls back to scalar). |
| `qdf_reflect2`| Swaps the reflect-based allocator for `github.com/modern-go/reflect2`. | Profile shows `reflect.MakeSlice` / `reflect.MakeMap` on the decode hot path. Otherwise leave off. |

These tags affect linked binary contents (different SIMD code paths,
different allocator dependency). They cannot be toggled at runtime.

Runtime feature toggles (which codec emits) are `Options` bits.
Compile-time switches (which code is linked) are tags. The two
compose: `-tags qdf_simd` plus `qdf.Marshal(v, qdf.OptCompression)`
gives you AVX2-accelerated dense encoding.

---

## Migrating from `encoding/json`

```go
// before
b, _ := json.Marshal(v)
_ = json.Unmarshal(b, &out)

// after
b, _ := qdf.Marshal(v, qdf.OptSpeed)   // drop-in replacement
_ = qdf.Unmarshal(b, &out)
```

Two things to know:

1. qdf reads `qdf:"name"` struct tags first, falls back to `json:"name"`.
   Your existing JSON tags work without change.
2. qdf encodes positive ints as `uint64` on the wire (Go-stdlib-like
   when decoded into `any`). If your code does `v.(int)` on a decoded
   `any`, change to `v.(uint64)` or decode into a concrete type.

For repetitive payloads, switch to `qdf.OptBalanced` once the basic
swap works. The wire is self-describing — receivers using
`qdf.Unmarshal` need no change when the sender flips bits.

---

## Migrating from `vmihailenco/msgpack/v5`

Same drop-in pattern:

```go
b, _ := msgpack.Marshal(v)
// becomes
b, _ := qdf.Marshal(v, qdf.OptSpeed)
```

qdf's Fast wire is roughly the same density as msgpack on tagged
scalars but with a 5-byte header (3-byte magic + version + flags). On
arrays of structs Fast loses a couple of bytes per element vs
msgpack's tightest paths; switch to `OptBalanced` to win them back
and then some.

---

## Putting it together — one program, many call sites

```go
type Service struct {
    Log     io.Writer
    Backup  *backup.Store
}

// Hot ingest path: 100k req/s, single event per call.
func (s *Service) Log(ev Event) {
    b, _ := qdf.Marshal(ev, qdf.OptSpeed)
    s.Log.Write(b)
}

// Periodic batch flush: 1000 events at a time.
func (s *Service) FlushBatch(batch []Event) {
    b, _ := qdf.Marshal(batch, qdf.OptBalanced)
    s.send(b)
}

// Daily snapshot: full state to durable storage.
func (s *Service) Snapshot(state State) error {
    b, _ := qdf.Marshal(state, qdf.OptCompression)
    return s.Backup.Put(b)
}
```

All three call sites share the same encoder pool. The `Options` value
picks the codec on each call; the pool returns the same encoder
configured for that call.

---

## Quick reference card

```
opts                   bytes vs json   encode vs json   decode vs json
─────────────────────  ──────────────  ───────────────  ──────────────
OptSpeed (hot path)        0.72×           2.2×              5.3×
OptQPack (metrics)         0.22×          28×              125×
OptQPack (embedding)       0.37×          89×              236×
OptBalanced (telemetry)    0.28×           1.2×              6.3×
OptCompression (archive)   0.18×           0.5×              3.9×
```

(All "vs json" ratios on the fixtures in `bench/profiles_test.go`.)

Compression rules of thumb:
- **5× smaller than json** on numeric or repetitive payloads.
- **2–3× smaller than msgpack** on the same.
- Encode pays a small CPU tax for Dense codecs; decode is faster
  across every scenario tested.
