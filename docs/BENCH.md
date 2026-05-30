# QDF Benchmark Results

> **The numbers below are a point-in-time snapshot.** For the
> continuously-updated results (re-run on every push to `main`) see the
> **live dashboard: <https://alex60217101990.github.io/qdf/dev/bench/>** —
> read [`BENCHMARKS.md`](BENCHMARKS.md) for how to read the trend graphs
> and tell a real change from shared-runner noise.

Measured on Darwin amd64 / Intel i7-9750H @ 2.6 GHz. Go 1.26.0.
`go test -bench=. -benchmem -benchtime=2s` in `bench/`.

Operating modes compared:

- **qdf_fast** — default build. Reflect-based with specialized fast
  paths for common slice (`[]string`, `[]int*`, `[]uint*`,
  `[]float32/64`, `[]bool`) and map types.
- **qdf_qpack** — Fast mode + QPack codecs (bit-packed bool, raw-LE,
  Frame-of-Reference, Delta+FOR, RLE, dictionary, Patched FOR for
  numeric/bool slices; Gorilla XOR and ALP decimal for float64 under
  `OptCompression`). Auto-selects the smallest predicted form per
  slice.
- **qdf_dense** — qdf_qpack + inline state-table interning for
  repeating strings (logs, columnar telemetry). Enum-like string columns
  in a struct array are dictionary-coded (distinct table + bit-packed
  index per row) when that beats per-value interning.
- **qdf_codegen** — code-generated `MarshalQDF`/`UnmarshalQDF` from
  `cmd/qdfgen` (no runtime reflection).
- **qdf_fast + qdf_reflect2** (opt-in build tag) — swap
  `reflect.MakeSlice` / `MakeMapWithSize` for `modern-go/reflect2`
  unsafe equivalents.
- **qdf_fast + qdf_simd** (opt-in build tag) — AVX2 asm
  bit-unpack at the byte-aligned widths (8 / 16 / 32 bits per slot);
  ~50 GB/s, runtime CPUID gate, scalar fallback otherwise.

vs. `encoding/json` (stdlib) and `github.com/vmihailenco/msgpack/v5`.

Round-trip verified by `TestSizes` and the `TestFastPath_*` suite. Race
coverage by `TestPool_ConcurrentEncoders` + the full `-race` test sweep.

## TL;DR

- **Encode**: qdf beats msgpack and json across the board. Wins are
  proportional to payload structure: 6× on map-heavy, 12-16× on
  numeric vectors, 4-6× on log batches.
- **Decode**: qdf beats both on every payload. 2-7× over msgpack,
  4-9× over json.
- **QPack (numeric/bool slices)**: 5× smaller wire than json, 21×
  faster encode, 80× faster decode on a mixed numeric payload. Delta
  +FOR reaches 512× compression on monotonic timestamp vectors.
- **Large realistic payload (~150 MiB)**: qdf_dense encodes 7.5×
  faster than json, decodes 8.1× faster, with a working-set delta
  of just 9.7 MiB for a 43.7 MiB output (json's encode allocator
  delta is 199 MiB for the same payload).
- **Generic `MarshalT[T]`**: -1 alloc and 25-40 % faster than the
  `any`-boxing entry points on small/medium payloads. Same wire.
- **`MarshalDirect[T Marshaler]`**: 1.55× faster than `Marshal` on
  generated types, 1 alloc per call, 3× less peak memory. Fast-mode
  only (Dense falls back to the reflect path so the intern table is
  resolved correctly).
- **`tagStateRepeat` (Dense, Markov-0 predictor)**: ~50 % size cut on
  payloads with repeating service / region / level fields, on top of
  the existing Dense intern table.
- **`tagStatePair` (Dense, Markov-1 predictor, `0xEA`)** + **`tagMapShape`
  (struct shape interning, `0xEC`)**: another -31 % on the
  TelemetryBatch fixture (73 104 → 50 129 bytes, **5.0× vs JSON**) by
  eliding per-record struct headers and exploiting conditional
  transitions between intern IDs.
- **`qdf_simd` (AVX2)**: 22-53× faster bit-unpack at byte-aligned
  widths (~50 GB/s, memory-bound). CPUID-gated; runtime falls back
  cleanly on older amd64.
- **Realistic / unique-data**: pool wins survive. UniqueLog is
  1.4-1.6× faster than json/msgpack encode and 3-4× faster on decode.
- **Concurrent**: parallel decode is 1.4-1.7× faster than json/msgpack.
- **Size**: Dense mode = **34% of json** on log batches (was 43%
  before shape interning), **20% of json** on the TelemetryBatch
  realistic-corpus fixture. No external compression layer.
- **Codegen** halves the per-call overhead vs reflect path on small
  fixed-schema structs (`Sample` decode: 695 ns → vs json 5899 ns =
  **8.5× faster**).
- All build-tag combinations (default, `qdf_reflect2`, `qdf_simd`,
  both) build, test under -race, and fuzz-pass cleanly.

## Scenario profiles (per-call `Options` recipes)

Six representative workloads, each encoded with the `Options`
combination the [`docs/CHOOSING.md`](CHOOSING.md) recipe recommends,
vs `encoding/json` and `vmihailenco/msgpack/v5` on the same fixture.
Numbers from `bench/profiles_test.go`, median of two
`-benchtime=300ms` runs on Intel i7-9750H, Go 1.26.0.

### Wire size (bytes)

| Scenario        | Recipe            | json    | msgpack | **qdf**     | vs json  | vs msgpack |
| --------------- | ----------------- | ------: | ------: | ----------: | -------: | ---------: |
| hot_path        | `OptSpeed`        |      97 |  **63** |          72 |   0.74×  |    1.14×   |
| telemetry_1k    | `OptBalanced`     | 142 881 | 111 637 |  **40 563** | **0.28×**|  **0.36×** |
| metric_1024     | `OptQPack`        |  37 258 |  19 512 |   **8 391** | **0.22×**|  **0.43×** |
| embed_768       | `OptQPack`        |   8 385 |   3 864 |   **3 103** |   0.37×  |    0.80×   |
| config          | `OptBalanced`     |     250 | **197** |         225 |   0.90×  |    1.14×   |
| archive_5k      | `OptCompression`  | 714 795 | 558 510 | **192 238** | **0.27×**|  **0.34×** |

### Encode latency (ns/op, median of 2 runs)

| Scenario      | json      | msgpack   | qdf        | qdf vs json |
| ------------- | --------: | --------: | ---------: | ----------: |
| hot_path      |     678   |     444   |    **349** |   **1.94×** |
| telemetry_1k  |  368 422  |  485 416  |    777 273 |       0.47× |
| metric_1024   |  149 639  |  121 974  |  **4 013** |  **37.3×**  |
| embed_768     |   58 545  |   28 729  |    **716** |  **81.7×**  |
| config        |    2 042  |    1 657  |      2 103 |       0.97× |
| archive_5k    | 1 767 838 | 2 507 389 |  4 257 244 |       0.42× |

### Decode latency (ns/op, median of 2 runs)

| Scenario      | json       | msgpack   | qdf        | qdf vs json  |
| ------------- | ---------: | --------: | ---------: | -----------: |
| hot_path      |     1 557  |     618   |    **282** |   **5.5×**   |
| telemetry_1k  | 2 280 074  |  897 409  |    485 601 |   **4.7×**   |
| metric_1024   |   524 417  |  140 243  |  **3 845** | **136×**     |
| embed_768     |   160 376  |   38 833  |    **618** | **260×**     |
| config        |     5 739  |   2 678   |    1 551   |     3.7×     |
| archive_5k    | 13 414 681 | 5 105 266 |  2 394 553 |   **5.6×**   |

**Reading the numbers:**

- **Decode is faster than json + msgpack across every scenario.** The
  per-decoder key-intern cache, pooled `*Decoder`, and self-describing
  wire avoid the schemaless-tag walk JSON pays per call.
- **Encode is faster on small and numeric scenarios, slower on big
  Dense ones.** `OptBalanced` / `OptCompression` pay a CPU tax to
  reduce wire size by 3–5×. That's the trade.
- **`OptQPack` on numeric payloads is the dramatic case** — 37–80×
  faster encode, 130–260× faster decode, 4× smaller wire than json.
  If your hot path moves floats around, the bit is essentially free
  and the wins are large.
- **hot_path msgpack edges qdf on size** (63 B vs 72 B) because the
  5-byte qdf header (`'QDF' + version + flags`) plus per-field tags
  are slightly looser than msgpack's fixmap on a 5-field struct.
  Speed and decode latency still favour qdf.

Reproduce: `go test -C bench -bench=BenchmarkProfile_ -benchmem -benchtime=300ms`

## Encode — synthetic (single payload reused)

| Payload          | json    | msgpack | qdf_fast    | qdf_dense | vs msgpack | vs json |
| ---------------- | ------- | ------- | ----------- | --------- | ---------- | ------- |
| Tiny             | 192     | 286     | 227         | 398       | 1.26×      | 0.85× ⚠ |
| Flat (20 fields) | 1182    | 1262    | **480**     | 948       | **2.63×**  | **2.46×** |
| Nested (4 deep)  | 446     | 793     | **331**     | 786       | **2.40×**  | **1.35×** |
| Deep16           | 1157    | 3060    | **477**     | 814       | **6.42×**  | **2.43×** |
| Wide ×1000       | 991k    | 991k    | **213k**    | 289k      | **4.66×**  | **4.66×** |
| LogBatch ×1000   | 998k    | 624k    | **171k**    | 562k      | **3.65×**  | **5.84×** |

Throughput (encode, MB/s):

| Payload     | json | msgpack | qdf_fast |
| ----------- | ---- | ------- | -------- |
| Wide1k      | 211  | 144     | **604**  |
| LogBatch1k  | 252  | 306     | **1 086** |
| Float64Vec512 | 100 | 170    | **1 700**+ |

## Encode — realistic (fresh unique payload per iteration)

These benchmarks construct a different `LogEntry` per loop iteration so
the pool's buffer-reuse heuristic and the type-descriptor cache have to
handle real variability rather than re-encoding the same byte sequence.

| Bench                        | json | msgpack | qdf_fast | vs msgpack | vs json |
| ---------------------------- | ---- | ------- | -------- | ---------- | ------- |
| UniqueLog (serial)           | 2419 | 2080    | **1510** | 1.38×      | 1.60×   |
| MixedTypes (rotating shape)  | 1223 | 1121    | **692**  | 1.62×      | 1.77×   |
| RandomSize (Wide len ∈{1,10,100,1000}) | 313k | 283k | **76k** | 3.72× | 4.12× |
| UniqueLog (RunParallel)      | 667  | 682     | **487**  | 1.40×      | 1.37×   |

The pool wins are **not** an artifact of synthetic loops — qdf still
beats both encoders on speed AND on allocations under unique-data
conditions. Alloc count differential is dominated by payload-generation
helpers (randomHex allocates 2 per call); the encoder itself is steady
at ~3 allocs/op for fast path.

## Decode — synthetic

| Payload          | json    | msgpack | qdf_fast    | vs msgpack | vs json |
| ---------------- | ------- | ------- | ----------- | ---------- | ------- |
| Tiny             | 729     | 383     | **170**     | 2.25×      | 4.29×   |
| Flat             | 4411    | 1862    | **1122**    | 1.66×      | 3.93×   |
| Nested           | 2483    | 1206    | **425**     | 2.84×      | 5.84×   |
| Deep16           | 7363    | 4170    | **1972**    | 2.11×      | 3.73×   |
| Wide ×1000       | 4.14M   | 1.95M   | **1.06M**   | 1.84×      | 3.89×   |
| LogBatch ×1000   | 3.42M   | 1.32M   | **469k**    | 2.81×      | 7.30×   |

## Decode — realistic (different bytes per iteration)

| Bench                              | json | msgpack | qdf_fast | vs msgpack | vs json |
| ---------------------------------- | ---- | ------- | -------- | ---------- | ------- |
| UniqueLog (serial)                 | 3569 | 1296    | **509**  | 2.55×      | 7.01×   |
| UniqueLog (RunParallel, contention)| 750  | 330     | **199**  | 1.66×      | 3.77×   |

## Map-heavy payload (40-entry `map[string]string` + `map[string]int`)

The type-specific fastpath (no reflect.MapRange / SetMapIndex on the hot
path) is the difference between losing and winning here.

| Op     | json   | msgpack | qdf_fast | qdf_dense | vs msgpack |
| ------ | ------ | ------- | -------- | --------- | ---------- |
| Encode | 8446   | 5282    | **1391** | 2019      | **3.80×**  |
| Decode | 18508  | 7972    | **3108** | -         | **2.57×**  |

Allocs: encode 84 / 46 / **3** / 9. Decode 124 / 112 / **71**. qdf wins
both ways without an opt-in build tag.

## Float-slice payload (`[]float32`/`[]float64` × 512 elements)

| Bench                | json   | msgpack | qdf_fast | qdf_simd | vs msgpack |
| -------------------- | ------ | ------- | -------- | -------- | ---------- |
| Encode []float32×512 | 30 054 | 18 270  | **1821** | **1508** | **12.1×**  |
| Encode []float64×512 | 39 819 | 23 856  | **2450** | **1735** | **13.8×**  |
| Decode []float32×512 | 72 928 | 28 521  | **3960** | 3928     | **7.20×**  |

The wire format matches the native little-endian layout on amd64 and
arm64, so the bulk float path collapses to a tight LE-store loop.

## QPack codecs — head-to-head

Payload: 256 booleans, 512 monotonic `uint64` (timestamps), 512 `int64`,
256 `float64` — the shape a metrics or columnar batch produces.

| Format        | Bytes  | Encode ns/op | Decode ns/op |
| ------------- | -----: | -----------: | -----------: |
| json          | 10 739 |       48 000 |      200 000 |
| msgpack       | 11 808 |       64 000 |       80 000 |
| qdf_fast      |  6 694 |        6 500 |       14 000 |
| **qdf_qpack** |  **2 132** | **2 300** |  **2 600** |
| **qdf_dense** |  **2 134** | **2 500** |  **2 500** |

QPack auto-selects, per slice, by predicted wire size: bit-packed
bool, raw-LE bulk, Frame-of-Reference + bit-pack, Delta + zigzag +
FOR, run-length, a low-cardinality dictionary codec, and Patched FOR
for integer slices. Float slices add two codecs under `OptCompression`
— Gorilla XOR for smooth series and ALP decimal for quantized/decimal
grids — picked against raw-LE only when strictly smaller, since their
bit-level work costs CPU that pays off only when size matters more.
On a 2-decimal `metric_quant_1024` fixture ALP brings `OptCompression`
to 2 592 B versus 8 238 B at `OptBalanced` (3.2× smaller).

On a 1024-element monotonic Unix-second timestamp vector the Delta+FOR
codec collapses the wire from 8 201 bytes (raw) to **16 bytes** —
a 512× reduction.

On a `latency_spikes_1024` fixture (sub-millisecond latencies with a
~1 % tail of large spikes) Patched FOR packs the common case at 9 bits
and patches the outliers from an exception list, landing at 1 215 B
versus 2 566 B for plain FOR — **~53 % smaller**. Plain FOR loses here
because the rare spikes force every slot to ~20 bits. The picker uses a
conservative cost upper bound, so PFOR is chosen only when strictly
smaller — clean columns keep FOR with a byte-identical wire.

## QPack micro-benchmarks (in-package, `qdf_simd` tag where noted)

Bit-unpack throughput at 1024 elements per call, median of 5:

| Bits | Scalar (orig)   | Pure-Go 128-bit window | AVX2 (qdf_simd)   | vs scalar |
| ---: | --------------: | ---------------------: | ----------------: | --------: |
|    8 |      3 450 ns/op |               3 450 ns/op |    **152 ns/op** | **22×**   |
|   16 |      4 790 ns/op |               3 530 ns/op |    **145 ns/op** | **33×**   |
|   32 |      8 070 ns/op |               4 330 ns/op |    **152 ns/op** | **53×**   |
|   56 |     13 340 ns/op |          **5 412 ns/op** |   (no asm path)  | **2.47×** |

The AVX2 path (`VPMOVZX{B,W,D}Q` + `VMOVDQU`) hits ~50 GB/s on the
byte-aligned widths — effectively memory-bandwidth bound. Non-byte-
aligned widths (1..7, 9..15, 17..31, 33..56) stay on the pure-Go
sliding-window decoder, which still beats the original byte-at-a-time
loop by 1.85× to 2.5×.

## Generic API (no `any` boxing)

`MarshalT[T]` / `UnmarshalT[T]` skip the `interface{}` conversion and
the reflect copy that `Marshal(v any)` needs for value-typed inputs.
Same wire output, fewer allocations per call.

| Op (mixed struct, 5 u64 + 3 bool) | `Marshal(any)` | `MarshalT[T]` | Speedup |
| --------------------------------- | -------------: | ------------: | ------: |
| encode (n=0 empty slice)          |    285 ns/op   |    170 ns/op  |  1.67×  |
| encode (n=4 small slice)          |    287 ns/op   |    175 ns/op  |  1.65×  |
| encode (n=64)                     |    460 ns/op   |    350 ns/op  |  1.30×  |
| allocs                            |    3           |    2          |  -1     |
| heap bytes                        |    192 B       |    112 B      |  -80 B  |

## Direct entry points (no `descOf`)

`MarshalDirect[T Marshaler]` / `UnmarshalDirect[T Unmarshaler]` go one
step further: with the receiver method known at compile time (`qdfgen`
output or hand-written), they skip the descriptor cache lookup and the
runtime interface assertion that `encodeMarshaler` does inside the
reflect path. Same wire bytes; Fast-mode only.

| Encode (Sample fixture, 11 fields) | ns/op | B/op | allocs |
| ---------------------------------- | ----: | ---: | -----: |
| json                               | 1800  |  576 |      8 |
| qdf_reflect (`Marshal`)            |  580  |  480 |      3 |
| qdf_codegen (`(*T).MarshalQDF`)    |  530  |  504 |      6 |
| **qdf_direct (`MarshalDirect`)**   | **364** | **160** | **1** |

Encode is 1.55× faster than the reflect path, with one third of the
allocations and 3× less peak memory. Decode runs at parity with the
reflect path when the `UnmarshalQDF` method is well-written
(`qdfgen` uses `Decoder.InternKey` for keys and matches the pooled
key-intern cache the reflect path relies on). Ad-hoc receivers that
build a fresh Decoder per call regress decode noticeably — the
docstring spells this out.

## Markov-0 state-ref predictor (`tagStateRepeat`)

Dense mode now collapses a state-ref whose ID equals the immediately
preceding emission to a single byte (the `0xE8` tag). The encoder side
also invalidates the chain on any inline-string emission so a later
`tagStateRepeat` cannot resurrect a stale ID across an uninterned
value. Wire savings on synthetic single-token vs alternating-token
batches:

| n elements | alternating (no predictor hit) | all-same (predictor every time) | delta   |
| ---------: | -----------------------------: | ------------------------------: | ------: |
|         16 |                          81 B  |                          49 B   |  -40 %  |
|        256 |                         563 B  |                         291 B   |  -48 %  |
|       1024 |                       2 099 B  |                       1 059 B   |  -50 %  |

Real workloads where the predictor pays off:

- Log batches: same `service`, `region`, `level`, `env` across most
  events.
- Columnar rows where a few "tag" columns rarely change.
- Repeated nested keys produced by the reflect / codegen field-name
  emit path.

Forward-compat note: a reader that does not implement `0xE8` fails
with `ErrBadTag` on first contact rather than silently mis-decode. The encoder only emits the tag in Dense mode, so Fast
buffers stay byte-identical to previous versions.

## Move-To-Front state-ref coding (`tagStateMTF`)

Dense additionally encodes a state-ref's LRU rank instead of its raw
intern ID when the rank's varuint is strictly shorter (`0xE9` tag).
Catches "hot subset defined late in intern order" patterns that the
Markov-0 predictor on its own misses.

Synthetic stress: 256 unique strings, every intern ID > 200 so the
raw varuint is 2 bytes, followed by 4 000 references rotating
through a hot subset of 8 items.

    OptBalanced (Markov-0 + MTF on)        10 824 bytes
    OptDense + OptPairPred (Markov-0 only) ~15 000 bytes (estimated, raw refs)
    OptDense alone                         ~18 000 bytes

The encoder picks `tagStateMTF` only when its rank varuint is strictly
shorter than the raw id varuint, so the wire never grows over plain
`tagStateRef`. The decoder mirrors the LRU chain so all three forms
(repeat / ref / mtf) co-exist on the same wire.

## Markov-1 pair predictor (`tagStatePair`, `0xEA`)

Dense keeps a per-prev ring of the last four successor IDs and emits
`0xEA + 1-byte rank` when the next ID is in that ring AND the raw
state-ref would need a multi-byte varuint. Catches conditional
patterns Markov-0 misses — `country` → `city`, `service` → `region`,
`level` → `host` — where the transition is predictable but the values
themselves do not repeat back-to-back.

Selection rule (encoder, `emitStateRef`):

    bestTag = tagStateRef ; bestLen = uvarintLen(id)
    if pair-hit && 1 < bestLen     → tagStatePair, payload = rank
    if mtf-hit && rankLen < bestLen → tagStateMTF,  payload = rank

The "strictly shorter" rule means the predictor never wins on small
state tables (ids ≤ 127) — the raw state-ref already uses a single
byte. It engages on streams with intern tables ≥ ~130 entries, which
is exactly where the byte cost of raw IDs would otherwise inflate.

## Shape interning (`tagMapShape`, `0xEC`)

Dense routes every struct emission through `tagMapShape` instead of
the generic `tagMap8/16/32` path:

    first emit:   0xEC, 0, varuint(N), [N x key],     [N x value]
    later emits:  0xEC, varuint(shapeID),             [N x value]

`shapeID` is assigned per encoder lifetime and addressed by `*typeDesc`
on the encoder side, so different struct types never collide on the
same id and types are looked up in O(N) over a tiny binding slice
(typical: 1 – 4 entries per stream).

Per-record saving on an array of identical-shape structs: roughly
`N × 2` bytes for the elided state-refs covering the key names, plus
the `tagMap8` header. For the 1 000-event TelemetryBatch fixture the
TelemetryBatch wire dropped **73 104 → 50 129 bytes** (-31 %) purely
from the shape codec layered on top of Markov-0 / MTF / Markov-1.

Forward-compat: a reader that does not implement `0xEC` fails with
`ErrBadTag` on first contact. Fast mode is unaffected — it
never emits the tag.

## Realistic corpus

Built-in `realistic_corpus_test.go` builders produce three shapes
that mirror real telemetry workloads. Numbers below are encoded
sizes (`TestSizes_RealisticCorpus`) plus encode latency
(`BenchmarkCorpus_TelemetryBatch1000`, Intel i7-9750H, 3 runs).

### TelemetryBatch (1 000 events, repeating service / region / level)

|              | bytes   | vs json   | encode ns/op |
| ------------ | ------: | --------: | -----------: |
| json         | 252 497 |     1.00× |            — |
| qdf_fast     | 186 674 |     0.74× |       272 k  |
| qdf_qpack    | 186 674 |     0.74× |       261 k  |
| **qdf_dense**| **50 129** | **0.20×** |    1.0 M    |

Dense pays ~4× on CPU for a **5.0× size reduction** vs JSON and
**3.7× vs qdf_fast** — string-intern + Markov-0 + MTF + Markov-1 pair
+ shape interning collapse the repeating service / region / level /
host fields and the per-record struct headers. QPack does not help
much here because the per-event numeric fields are scalar (TS, Span,
Trace, Duration) rather than slice-shaped.

### MetricSeries (1 024 numeric timestamps + values)

|              | bytes   | vs json   |
| ------------ | ------: | --------: |
| json         |  30 043 |     1.00× |
| qdf_fast     |  14 442 |     0.48× |
| **qdf_qpack**| **8 307** | **0.28×** |
| qdf_dense    |   8 315 |     0.28× |

Here QPack pulls its weight: the `[]int64` timestamp column is
monotonic and Delta+FOR compresses it to near-zero bytes per
element; the `[]float64` value column uses raw-LE bulk. Dense and
QPack converge because the string overhead is tiny.

## Large payload (~150 MiB JSON-equivalent)

Driven by `bench/largepayload_test.go`. The builder emits a struct
of N records, every record carrying every qdf-supported field type:
scalar ints/floats/bools, low-cardinality hot strings (service /
region / level / host), unique-per-record UUIDs, nested map +
string slice, `[]int32` path, `[]byte` blob, `[]float64` vector.

Numbers below come from `TestSizes_LargePayload` (sizes, N = 200 000)
and `TestMem_LargePayload` (encode/decode latency + working-set
delta, N = 100 000) on Intel i7-9750H, Go 1.26.0. Both helpers skip
under `-short`. Reproduce:

```bash
go test -C bench -run TestSizes_LargePayload -count=1 -timeout=10m
go test -C bench -run TestMem_LargePayload   -count=1 -timeout=10m
```

### Encoded size (200 000 records)

| Format        |    bytes  |  MiB   | vs json |
| ------------- | --------: | -----: | ------: |
| json          | 149 006 973 | 142.10 |   1.00× |
| msgpack       |  97 508 774 |  92.99 |   0.65× |
| qdf_fast      |  96 462 436 |  91.99 |   0.65× |
| qdf_qpack     |  94 008 854 |  89.65 |   0.63× |
| **qdf_dense** | **92 820 231** | **88.52** | **0.62×** |

Dense's compression ceiling here is set by the 200 000 unique UUIDs
which cannot dedupe (each ~36 bytes literal). The win shows in
encode/decode latency and memory below.

### Encode / decode latency + working-set delta (100 000 records)

| Format        | bytes (MiB) | encode (ms) | decode (ms) | encode heap delta (MiB) |
| ------------- | ----------: | ----------: | ----------: | ----------------------: |
| json          |       71.08 |       1 070 |       1 744 |                  199.10 |
| msgpack       |       46.51 |         296 |         597 |                   64.01 |
| qdf_fast      |       46.01 |     **142** |     **300** |                   94.14 |
| qdf_qpack     |       44.84 |         147 |         231 |                   92.95 |
| **qdf_dense** |   **43.70** |         169 |     **216** |                **9.73** |

Speedups vs json: qdf_fast encode **7.5×**, qdf_dense decode **8.1×**.
Speedups vs msgpack: qdf_fast encode **2.1×**, qdf_dense decode
**2.8×**.

The most surprising line is qdf_dense's encode heap-delta of
**9.7 MiB** for a 43.7 MiB output. `Marshal(v, OptBalanced)` reuses a pooled
encoder buffer plus the intern table; the produced wire is `slices.
Clone`-d for the caller, but the pool buffer survives and shrinks
per-call working-set proportionally. json builds a fresh buffer per
call and the allocator delta is 4.5× the output size. msgpack falls
between the two.

Decode heap deltas are omitted from the table: forced `runtime.GC()`
inside the timing window reclaims the encoded buffer in the same
sample so the delta reads negative for several formats and the
number stops being useful. The latency column captures the real
decode work.

## Codegen (no reflection) — `Sample` fixture

11-field struct with nested struct, slice, map, pointer, fixed array,
`[]byte`, `time.Time`.

| Op     | json   | qdf_fast (reflect) | qdf_codegen | gen vs reflect | gen vs json |
| ------ | ------ | ------------------ | ----------- | -------------- | ----------- |
| Encode | 1714   | 702                | **646**     | 1.09×          | **2.65×**   |
| Decode | 5899   | 792                | **695**     | 1.14×          | **8.49×**   |

The codegen gap over the tuned reflect path is modest because the reflect
path already uses pool, cached descriptors, and `unsafe.Pointer + offset`
field access. Codegen wins more on decode where reflect must alloc
map/slice values through `reflect.MakeMap`.

## Encoded size

| Payload          | json    | msgpack | qdf_fast    | qdf_dense   | dense vs json |
| ---------------- | ------- | ------- | ----------- | ----------- | ------------- |
| Tiny             | 24      | **16**  | 22          | 25          | 1.04×         |
| Flat             | 210     | 134     | **132**     | 138         | 0.66×         |
| Nested           | 103     | **76**  | 86          | 96          | 0.93×         |
| Deep16           | 239     | 139     | 166         | **63**      | **0.26×**     |
| Wide ×1000       | 212 901 | 135 626 | 128 632     | **66 702**  | **0.31×**     |
| LogBatch ×1000   | 251 902 | 185 639 | 185 649     | **85 440**  | **0.34×**     |

A few rows are slightly **larger** under Dense than under Fast on
tiny / non-repeating payloads (`Tiny`, `Flat`, `Nested`). That is the
expected 1-3 byte cost of the shape declaration prelude on a one-shot
struct emit — Markov / shape predictors need at least one repeat to
amortise. Wins start at `Deep16` and grow with payload size.

## Memory (the second axis — every bit as important as speed)

### Bytes allocated per decode (lower is better)

| Payload          | json     | msgpack  | qdf_fast | dense | qdf vs json | qdf vs msgpack |
| ---------------- | -------- | -------- | -------- | ----- | ----------- | -------------- |
| Tiny             | 248      | 77       | **29**   | 29    | 0.12×       | 0.38×          |
| Flat             | 448      | 272      | **224**  | 224   | 0.50×       | 0.82×          |
| Nested           | 664      | 160      | **112**  | 112   | **0.17×**   | 0.70×          |
| Deep16           | 1200     | 312      | **264**  | 264   | 0.22×       | 0.85×          |
| Wide ×1000       | 638 353  | 409 221  | **220 591** | 220 587 | 0.35× | 0.54×        |
| LogBatch ×1000   | 442 536  | 407 698  | **251 838** | 251 860 | **0.57×** | **0.62×**  |
| MapHeavy unique  | 4912     | 3089     | **2359** | -     | 0.48×       | 0.76×          |
| Float32Vec512    | 4384     | 4282     | **2113** | -     | 0.48×       | 0.49×          |
| MapStringAny (repeated keys) | 790 | -    | **345** | -     | **0.44×**   | -              |

### Allocations per decode (lower is better)

| Payload                | json | msgpack | qdf_fast (no intern) | qdf_fast (with intern wins) |
| ---------------------- | ---- | ------- | -------------------- | --------------------------- |
| Nested                 | 15   | 6       | **5**                | -                           |
| Wide ×1000             | 5020 | 5007    | **5003**             | -                           |
| LogBatch ×1000         | 7019 | 7007    | **7003**             | -                           |
| MapHeavy 40-entry      | 124  | 112     | 71 → **32**          | **−39 allocs from intern**  |
| MapHeavy repeated keys | 71   | 46      | -                    | **26**                      |
| MapStringAny repeated  | 37   | -       | -                    | **3**  (**12× fewer**)      |
| Float32Vec512          | 16   | 8       | **3**                | -                           |

The intern win is biggest when:
1. The same key set appears across many maps (typical for analytics /
   tracing telemetry).
2. The Decoder is recycled via `sync.Pool` (Unmarshal already does this).

### Key intern cache (`internal/intern`)

- 256-slot direct-mapped hash table, fixed size, lives in the Decoder.
- On cache hit: zero-allocation string return.
- On miss: one `string(b)` copy, stored. Collisions overwrite (no chain).
- Survives across `qdf.Unmarshal` calls because the decoder is pooled.
- Same shape as `go-json-experiment/json`'s intern.go.

### Streaming memory footprint

`BenchmarkStream_LogBatch1k_Dense` encode through `bytes.Buffer` sink:

| Sender             | total bytes written | over 1000 entries |
| ------------------ | ------------------- | ----------------- |
| json (one Marshal) | 251 902             | 251 B/entry       |
| msgpack            | 185 639             | 186 B/entry       |
| qdf_fast (Marshal) | 185 649             | 186 B/entry       |
| qdf_dense (Stream) | **107 416**         | **107 B/entry**   |

Stream Dense halves the bytes on the wire vs single-shot Fast because
the intern table is shared across the entire stream — every repeated
`level`, `service`, `host`, `region`, `msg` value collapses to a
1-2-byte state reference after first sight.

### Safety: hostile-input memory bounds

The decoder validates length-prefixed payloads against the remaining
buffer (`Decoder.CheckLength`) before any `make`. A malicious wire
encoding claiming a 2-billion-element map can NOT cause an OOM — it
returns `ErrShortBuffer` instead. Verified by the fuzz suite (3 M+
iterations across `FuzzDecoder_NeverPanics` and `FuzzRoundTrip_StringSlice`,
plus reproducers stored in `testdata/fuzz/`).

## Correctness coverage

What the test suite verifies (all under `-race`):

- **Primitives round-trip** at every wire-format boundary (int8/16/32/64,
  uint8/16/32/64, float32/64, fixstr boundary, str8/16/32 thresholds).
- **Boundary integers**: every power-of-two boundary from 0 to MaxInt64
  and the corresponding negatives down to MinInt64.
- **Strings**: 0, 1, 31 (fixstr edge), 32, 255 (str8 edge), 256,
  65535 (str16 edge), 65536, 1 MiB. Unicode + invalid-UTF-8 round-trip.
- **Fast-paths**: every specialized slice/map type vs the generic path.
- **Truncated input**: every prefix from 0 to N-1 decodes without panic.
- **Bad magic / bad version**: errors, not panics.
- **Cross-mode interop**: Fast-encoded bytes decode through the
  auto-detecting Decoder; same for Dense.
- **Streaming**: single-type and mixed-type (`Sale` interleaved with
  `Sig`) Dense stream preserves intern table across messages.
- **Concurrency**: 32 goroutines × 500 iterations × marshal+unmarshal
  with `-race`, plus dedicated `TestRace_AppendMarshal`. No races, no
  cross-call state bleed.
- **Fuzz**: 400 k+ iterations across `FuzzDecoder_NeverPanics` and
  `FuzzRoundTrip_StringSlice`. Never panics, round-trips agree.
- **Codegen**: round-trip through generated `MarshalQDF` / `UnmarshalQDF`
  plus interop with `qdf.Unmarshal` on the same bytes.

## Reproducing

```bash
# Default build
go test -race -count=1 ./...

# Cross-format bench (Marshal at OptSpeed / OptQPack / OptBalanced vs json/msgpack)
cd bench && go test -bench='BenchmarkQPack_' -benchmem -benchtime=2s

# Whole-suite bench
cd bench && go test -bench=. -benchmem -benchtime=2s -timeout=10m

# QPack codec micro-benchmarks (root module)
go test -bench='BenchmarkQPack' -benchmem -benchtime=2s

# AVX2 bit-unpack (asm under qdf_simd; CPUID-gated at run time)
go test -tags qdf_simd -bench='BenchmarkBitUnpackFast' -benchmem -benchtime=2s

# Build-tag race sweeps
go test -tags qdf_reflect2          -race ./...
go test -tags qdf_simd              -race ./...
go test -tags "qdf_simd qdf_reflect2" -race ./...

# Codegen
cd internal/codegen_test
go test -run TestGenerate .
go test -bench=. -benchmem -benchtime=2s

# Fuzz
go test -run=^$ -fuzz=FuzzDecoder_NeverPanics      -fuzztime=30s
go test -run=^$ -fuzz=FuzzRoundTrip_StringSlice    -fuzztime=30s
go test -run=^$ -fuzz=FuzzQPackBool                -fuzztime=30s
go test -run=^$ -fuzz=FuzzQPackRawUint64           -fuzztime=30s
```
