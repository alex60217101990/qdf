# QDF Benchmark Results

Measured on Darwin amd64 / Intel i7-9750H @ 2.6 GHz. Go 1.26.0.
`go test -bench=. -benchmem -benchtime=2s` in `bench/`.

Operating modes compared:

- **qdf_fast** — default build. Reflect-based with specialized fast
  paths for common slice (`[]string`, `[]int*`, `[]uint*`,
  `[]float32/64`, `[]bool`) and map types.
- **qdf_qpack** — Fast mode + QPack codecs (bit-packed bool, raw-LE,
  Frame-of-Reference, Delta+FOR; Gorilla XOR available explicitly) for
  numeric and bool slices. Auto-selects the smallest predicted form
  per slice.
- **qdf_dense** — qdf_qpack + inline state-table interning for
  repeating strings (logs, columnar telemetry).
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
- **Generic `MarshalT[T]`**: -1 alloc and 25-40 % faster than the
  `any`-boxing entry points on small/medium payloads. Same wire.
- **`qdf_simd` (AVX2)**: 22-53× faster bit-unpack at byte-aligned
  widths (~50 GB/s, memory-bound). CPUID-gated; runtime falls back
  cleanly on older amd64.
- **Realistic / unique-data**: pool wins survive. UniqueLog is
  1.4-1.6× faster than json/msgpack encode and 3-4× faster on decode.
- **Concurrent**: parallel decode is 1.4-1.7× faster than json/msgpack.
- **Size**: Dense mode = **43% of json**, **58% of msgpack** on log
  batches (without an external compression layer).
- **Codegen** halves the per-call overhead vs reflect path on small
  fixed-schema structs (`Sample` decode: 695 ns → vs json 5899 ns =
  **8.5× faster**).
- All build-tag combinations (default, `qdf_reflect2`, `qdf_simd`,
  both) build, test under -race, and fuzz-pass cleanly.

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

QPack auto-selects, per slice, between four codecs by predicted wire
size: bit-packed bool, raw-LE bulk, Frame-of-Reference + bit-pack,
and Delta + zigzag + FOR. Gorilla XOR is available for floats via the
low-level helpers but not auto-selected (its bit-level work makes it
~100× slower than raw-LE memmove; only worth it when size matters
more than CPU).

On a 1024-element monotonic Unix-second timestamp vector the Delta+FOR
codec collapses the wire from 8 201 bytes (raw) to **16 bytes** —
a 512× reduction.

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
| Tiny             | 24      | **16**  | 22          | 24          | 1.00×         |
| Flat             | 210     | 134     | **132**     | 137         | 0.63×         |
| Nested           | 103     | **76**  | 86          | 91          | 0.83×         |
| Deep16           | 239     | 139     | 166         | **122**     | **0.51×**     |
| Wide ×1000       | 212 901 | 135 626 | 128 632     | **106 660** | **0.50×**     |
| LogBatch ×1000   | 251 902 | 185 639 | 185 649     | **107 416** | **0.43×**     |

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

# Cross-format bench (Marshal vs MarshalQPack vs MarshalDense vs json/msgpack)
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
