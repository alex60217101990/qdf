# Choosing the right `Options` (and build tags)

qdf has one encode entry point: `Marshal(v, opts)`. The `opts` bit-mask
picks which codecs run. This page is a cheatsheet for the choice.

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

Result: encode 349 ns vs json 678 ns vs msgpack 444 ns. Decode 282 ns
vs json 1557 ns vs msgpack 618 ns. Wire 72 B vs json 97 B (msgpack
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
2.8× vs msgpack). Decode 485 µs vs json 2280 µs (4.7× faster). Encode
pays a CPU tax (777 µs vs json 368 µs) because shape declaration plus
intern table cost up front — that's the trade you make for the size
win.

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

Result: encode **4 µs** vs json 150 µs (37× faster) vs msgpack 122 µs.
Decode 4 µs vs json 524 µs (130× faster), msgpack 140 µs (35×). Wire
8.4 KB vs json 37 KB (4.4× smaller). Adding `OptDense` here costs
intern-table CPU for zero wire benefit — skip it.

### Embedding vector / dense float array

Single `[]float32` or `[]float64` of 100s–1000s of elements. The
identifying string is small.

**Recipe:** `qdf.OptQPack`

```go
b, err := qdf.Marshal(vec, qdf.OptQPack)
```

Result: encode 716 ns vs json 58 µs (**80× faster**) vs msgpack 29 µs
(40×). Decode 618 ns vs json 160 µs (**260× faster**). Wire 3 KB vs
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

Result on a 5 000-row archive: wire 192 KB vs json 715 KB (3.7×) vs
msgpack 558 KB (2.9×). Decode 2.4 ms vs json 13.4 ms (5.6×) vs
msgpack 5.1 ms (2.1×). Encode pays 4.5 ms vs json 1.8 ms — fair for
backup workloads.

On a 1024-sample smooth metric series `OptCompression` shrinks the
float body from 8398 B (raw-LE under `OptQPack`) to 2307 B (Gorilla
XOR), a 72.5 % drop. Encode/decode go from ~4.2 µs to ~41 µs — fine
when the series is being archived once and queried over its lifetime,
not for the hot ingest path.

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
- **Different `opts` across encode/decode.** There are no "decode
  opts" — `Unmarshal` reads the header flags and the tag stream and
  handles every variant. The encoder opts pick what you emit; the
  decoder reads whatever it gets.

---

## Build tags

Compile-time switches, orthogonal to runtime `Options`. Build with
`-tags <name>`.

| Tag           | What it does                                                          | When to use                                                   |
| ------------- | --------------------------------------------------------------------- | ------------------------------------------------------------- |
| `qdf_simd`    | Compiles in AVX2 bit-unpack for FOR / Delta+FOR codecs (decode side). | amd64, large numeric payloads under `OptQPack` or `OptBalanced`. CPUID-gated at runtime — safe to ship even if the target doesn't have AVX2 (falls back to scalar). |
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
OptSpeed (hot path)        0.74×           2.0×              5.5×
OptQPack (metrics)         0.22×          37×              130×
OptQPack (embedding)       0.37×          80×              260×
OptBalanced (telemetry)    0.28×           0.5×              4.7×
OptCompression (archive)   0.27×           0.4×              5.6×
```

(All "vs json" ratios on the fixtures in `bench/profiles_test.go`.)

Compression rules of thumb:
- **5× smaller than json** on numeric or repetitive payloads.
- **2–3× smaller than msgpack** on the same.
- Encode pays a small CPU tax for Dense codecs; decode is faster
  across every scenario tested.
