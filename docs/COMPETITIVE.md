# Competitive benchmarks — qdf vs json / msgpack / protobuf / flatbuffers

> **Every `json` number on this page is `encoding/json` v1, measured on the
> amd64 host named in [docs/BENCH.md](BENCH.md).** Go 1.27's
> `encoding/json/v2` narrows some of these gaps and none of them decisively:
> measured here it gains **0–21% on decode** and **0–9% on encode**, and on a
> payload with no maps to sort and no HTML to escape it gains **nothing at all**,
> because in Go 1.27 `encoding/json` *is* json/v2 under `DefaultOptionsV1` and
> the difference between them is exactly the price of those options. Its real win
> is allocation — up to 60% fewer on a large payload. The full v1 / v2 /
> hand-written-`jsontext` comparison is in
> [BENCH.md](BENCH.md#json-v2-and-what-json-actually-costs).


Realistic, batch-shaped payloads encoded by every codec at the **same record
count**, so wire size and memory are directly comparable.

**Environment:** Intel i7-9750H · Go 1.26 · darwin/amd64 · 2026-06-01.
Speed numbers are machine-specific; wire size and allocation counts are
deterministic. Reproduce with `scripts/compare.sh` (runs the `bench/` module
suite at `-count=6` plus the `EMIT_MEM=1` memory report). Competitor deps live
only in `bench/go.mod`; the root module stays dependency-free.

## Fixtures
| name | shape |
| --- | --- |
| RTB 1024 | OpenRTB-style nested bid requests, enum-heavy, hex IDs |
| IoT 32×256 | 32 devices × 256 samples; float64 sensor series + monotonic ns timestamps |
| OTLP 4×512 | nested trace spans, repeated op-names / attr keys, hex trace/span IDs |
| Logs 1024 | structured logs: enum level, repeated service/host, hex trace IDs, map fields |
| Events 1024 | typed events with raw `[]byte` payloads |

## Wire size (bytes, lower = better)

| fixture | json | msgpack | protobuf | flatbuffers | qdf_speed | qdf_balanced | qdf_compression |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| RTB 1024 | 559 294 | 428 404 | 327 700 | 648 808 | 436 517 | 258 167 | **203 360** |
| IoT 32×256 | 469 058 | 224 534 | 207 562 | 203 224 | 224 604 | 158 474 | **148 177** |
| OTLP 4×512 | 1 027 033 | 793 192 | 561 860 | 960 160 | 807 235 | 240 686 | **179 181** |
| Logs 1024 | 245 037 | 193 476 | 156 479 | 278 104 | 195 530 | 89 631 | **62 149** |
| Events 1024 | 122 857 | 84 712 | 64 978 | 88 944 | 85 742 | 39 650 | **39 639** |

**qdf is smaller than protobuf on every batch fixture** at the compression
tier — RTB −38 %, IoT −29 %, OTLP −68 %, Logs −60 %, Events −39 %. qdf dedups and
columnar-compresses *across* records (intern + dictionary + Delta/FOR + Gorilla/Chimp);
protobuf, msgpack, json and flatbuffers encode each record independently, so
repeated strings, enum values and smooth float series re-pay their cost every
row. flatbuffers is the largest — it trades size for zero-copy random access.

## Memory — bytes allocated per encode+decode cycle (lower = better)

The steady-state allocation rate is what drives GC pressure and resident set
under sustained load in a container.

**Fairness note:** qdf's `Marshal` pools its encoder internally, so a naïve
comparison against the default `proto.Marshal` (no pooling) flatters qdf. The
table below is the **fair** comparison — every codec reuses its output buffer
the way a throughput-conscious caller would: qdf via `AppendMarshal`, protobuf
via `proto.MarshalOptions.MarshalAppend`, msgpack via a pooled encoder,
flatbuffers via `Builder.Reset`, json via a reused `bytes.Buffer`. Bytes
allocated **per encode cycle** (decode allocates a fresh target on all codecs —
symmetric, excluded here):

| fixture | json | msgpack | protobuf | flatbuffers | qdf_speed | qdf_balanced | qdf_compression |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| RTB 1024 | 3.17 M | 2.02 M | 2.51 M | **4 K** | 1.48 M | 4.38 M | 5.28 M |
| IoT 32×256 | 557 K | 420 K | 227 K | **128** | 212 K | 212 K | 791 K |
| OTLP 4×512 | 2.44 M | 2.02 M | 2.05 M | **10 K** | 1.21 M | 2.10 M | 2.93 M |
| Logs 1024 | 1.06 M | 693 K | 1.02 M | **4 K** | 550 K | 561 K | 880 K |
| Events 1024 | 203 K | 170 K | 184 K | **4 K** | 113 K | 163 K | 312 K |

**Honest verdict once both sides reuse buffers:**
- **flatbuffers wins encode allocation outright** (4 K–128 B/cycle) — it builds
  in place with no heap traffic after warm-up. This is its core design win and
  was *masked* in a default-API comparison (where it allocates a fresh builder).
- **`qdf_speed` / `qdf_qpack` beat protobuf on 4 of 5 fixtures** (≈40–50 % less
  on RTB and OTLP), and tie on IoT. They are the lowest-allocating of the
  marshal-into-`[]byte` codecs.
- **`qdf_balanced` / `qdf_compression` do NOT win memory** — their richer codecs
  (intern table, Gorilla/ALP scratch) cost transient buffers. They trade memory
  and CPU for the wire-size win above; pick them when bytes-on-wire dominate.
- Default-API allocation *counts* (no reuse) are even more lopsided — e.g. IoT
  encode `qdf_balanced` 3 allocs/op vs `proto.Marshal` 385 — but that is mostly
  qdf's built-in pooling vs protobuf's pool-less default call, **not** a format
  intrinsic, so the fair reuse table above is the one to trust.

## Active-Directory export — mixed structs (hybrid columnar)

A realistic LDAP/AD user export: 5 000 `ADUser` records, each a *mixed* struct —
~14 scalar/string columns (objectGUID, UPN, mail, names, department, title,
timestamps, enabled flag) plus a `[]string` group-membership field and a
`map[string]string` extra-attributes field. Cardinality mirrors a real org:
GUID/UPN/email unique, names occasionally repeating, groups/departments
moderately repeating. The two multi-valued fields are exactly what used to force
the whole struct down the row-major path; **hybrid columnar** (under
`OptCompression`) now transposes the eligible columns and keeps the rest
row-major. Same data through each library (i7-9750H · Go 1.26, 5 000 rows):

| 5 000 AD users | wire | encode | decode |
| --- | ---: | ---: | ---: |
| `encoding/json` | 3 833 KB | 14.1 ms | 57.8 ms |
| `msgpack` | 3 370 KB | 8.7 ms | 16.7 ms |
| **qdf `OptBalanced`** (default) | **1 830 KB** | **6.6 ms** | **7.9 ms** |
| **qdf `OptCompression`** | **616 KB** | 32.1 ms | 13.2 ms |

- **`OptBalanced` is a clean sweep**: smaller *and* faster than json and msgpack
  on every axis — 2.1× smaller than json / 1.8× than msgpack on wire, 2.1× faster
  encode than json, and **≈7× faster decode than json** (≈2× faster than msgpack),
  at roughly half their allocations.
- **`OptCompression`** is **6.2× smaller than json / 5.5× smaller than msgpack**
  and still decodes faster than both; it pays encode CPU (tANS/FSE entropy pass +
  FSST + hybrid transpose) for the bytes — the backup/cold-storage trade.
- protobuf/flatbuffers are not in this row: they need a generated schema for
  `ADUser`, which an ad-hoc Go struct doesn't carry. On the five schema-fixtures
  above qdf is already smaller than protobuf at the compression tier.

## Honest caveats
- **`qdf_speed` wire ≈ msgpack** — the speed tier skips columnar compression; it
  is the drop-in `encoding/json` replacement, not the size play.
- **`qdf_compression` encode is slower** (Gorilla/Chimp/ALP float cost: ~70 MB/s on the
  IoT float batch vs `qdf_balanced` ~1100 MB/s). Use `OptBalanced` for the
  size win without the CPU hit; reserve `OptCompression` for cold storage.
- **protobuf and flatbuffers win raw decode throughput** — generated code and
  zero-copy access beat qdf's reflection path. `WithNoCopy()` narrows this for
  owned-input callers (aliases the buffer like flatbuffers; ~1.7× faster,
  near-zero allocs — see below); closing the rest (codegen that emits the
  columnar wire) is tracked in the speed backlog.
- Single-tiny-message size favours protobuf (schema field numbers, no columnar
  amortisation < 16 rows). qdf's win is *batches* of structured records.

## Selective decode — a capability the others don't have

The size and memory tables above all decode the **whole** payload. qdf can do
something none of json / msgpack / protobuf can: decode **only the columns you
ask for** and **filter rows with a predicate pushed into the decoder**, over a
self-describing batch with no schema (`Unmarshal(data, &out, Select(...),
Where(...))` on an `OptColumnIndex` payload). Unselected columns are skipped on
the wire — never materialised.

Measured on a wide `[]struct` batch (i7-9750H):

| operation | time | memory | vs full |
| --- | ---: | ---: | --- |
| full decode | ~117 µs | 290 KB | baseline |
| `Select` two columns (projection) | **~25 µs** | **55 KB** | **~5× faster, ~5× less memory** |
| `Where` predicate pushdown | ~50 µs | 68 KB | vs decode-all-then-filter ~133 µs / 301 KB → **~2.5× faster, ~4.4× less** |

protobuf and msgpack must decode the entire message before you can read one
field. flatbuffers offers zero-copy random *field* access — a different model
that needs a compiled schema and does not do predicate-filtered column
projection across a batch in one call. For "store a wide batch, read a few
columns or filter rows later" — the columnar-warehouse access pattern — qdf is
the only one of these formats that reads less than the whole thing.

## Zero-copy decode — closing the decode-alloc gap

The memory table above excluded decode as "symmetric" — every codec allocates a
fresh target. That is only true for qdf's *default* (copying) decode. With
`WithNoCopy()` qdf decodes string/`[]byte` fields as **aliases of the input
buffer** — the same trick that gives flatbuffers its zero-copy reads, but over
qdf's self-describing batch wire and into ordinary Go structs.

Measured on a 1000-row string-heavy batch (i7-9750H): default decode 7002
allocs/op → `WithNoCopy()` **3 allocs/op**, **−38 % B/op, ~1.7× faster**. Works
on the reflect path and codegen types alike.

The catch is the same as flatbuffers' zero-copy: the decoded values are valid
only while the input buffer lives and is unmodified — so it is opt-in, not the
default. Use it for owned, long-lived input (mmap, a file in memory, batch
analytics); never on a recycled server request buffer. A safe arena-backed
variant (owned copies, ~1 alloc, no lifetime caveat) is a tracked follow-up.

## When to pick which
- **Smallest wire on batches, no schema** → qdf (`OptBalanced` for balanced
  CPU, `OptCompression` for minimum bytes).
- **Read a few columns / filter rows out of a stored batch** → qdf, uniquely
  (`Select` / `Where` pushdown — ~5× faster, ~5× less memory than full decode).
- **Lowest encode allocation, schema on hand** → flatbuffers (zero-copy build).
- **Fastest decode / zero-copy random field access** → flatbuffers.
- **Schema-based RPC, smallest single message, max encode throughput** → protobuf.
- **Maximum interop / human-readable** → json.
