# qdf-bench

Representative qdf serialize/deserialize benchmark over **real** data: the
[adalanche-sampledata](https://github.com/lkarlslund/adalanche-sampledata)
local-machine dumps (rich nested Active Directory `localmachine.Info` — hundreds
of services/tasks/groups per host). It serializes then deserializes each dump
across qdf's valid option and build-tag matrix and prints ser/deser ns/op, B/op,
allocs/op, and wire size — for both a **typed struct** and a dynamic
**`map[string]any`** representation of the same data. Every measurement is gated
on a lossless round-trip check, so the numbers can't lie about correctness.

## Get the data

```sh
git clone https://github.com/lkarlslund/adalanche-sampledata
```

Only the `goad/localmachine/*.json` files are used (the `*.objects.msgp.lz4` AD
dumps are skipped — they need adalanche's own codec).

## Run

One build (default tags):

```sh
go run ./cmd/qdf-bench -datapath ./adalanche-sampledata
```

Full build-tag matrix (`none`, `qdf_reflect2`, `qdf_simd`, both) — build tags are
compile-time, so this builds one binary per combo:

```sh
GO=~/.gvm/gos/go1.26.0/bin/go ./cmd/qdf-bench/run.sh ./adalanche-sampledata
```

Flags: `-datapath` (required), `-iters N` (iterations per measured op, default
200). Extra flags after the datapath are forwarded by `run.sh`.

## Reading the table

```
repr   bundle    ser_ns  ser_B  ser_alloc ser_rssKiB deser_ns deser_B deser_alloc deser_rssKiB wire_B
typed  Balanced  1339939 317180 738       12         1232810  567352  6483        60           208995
map    Balanced  2035585 518943 1572      12         2109494  816897  12932       4            222659
```

- `repr` — `typed` (Go structs) vs `map` (`map[string]any`). Typed is qdf's strong
  path: roughly half the allocs and smaller wire.
- `bundle` — option preset: `Speed`, `Balanced`, `Compression`, plus
  `+ColIndex` / `+MapShape` / `+Canonical` variants.
- `ser_*` / `deser_*` — per-op **encode / decode** cost, averaged over the sample
  files. **Each is isolated to the qdf call only**: the value is dereferenced once
  before timing, the timed loop runs nothing but `qdf.Marshal` / `qdf.Unmarshal`
  (the round-trip check happens outside the loop), and a sink prevents the
  compiler from eliding the call.
  - `*_ns` — nanoseconds per op.
  - `*_B` / `*_alloc` — bytes and allocations per op, from `runtime.MemStats`
    deltas (the same quantities as `go test -bench`'s `B/op` and `allocs/op`).
    **This is the precise per-op memory.**
  - `*_rssKiB` — process resident-set growth measured **around only that op's
    loop** (`getrusage` Maxrss delta). Coarse and high-water by nature: the first
    large op warms the heap and shows growth; later ops show ~0 because the heap
    is already sized. Use `*_B` for exact per-op memory; this column is the
    process-footprint view, scoped to encode vs decode.
- `wire_B` — encoded size (average per file).
- A `whole-process peak RSS` line follows — the entire run (loading + both reprs
  + all bundles), context only.

## Notes for hacking on it

- `types.go` mirrors adalanche's `localmachine.Info`; the deeply variable
  `Task.Definition` is kept as `map[string]any` (it genuinely is). If the sample
  schema changes, the round-trip gate will fail loudly — update the structs.
- `tags_*.go` stamp the active build tags via `init()`; `tags.go` reports them.
- The bench loads everything once, then times marshal/unmarshal with a fixed
  `-iters` loop (fast and deterministic vs `testing.Benchmark`'s auto-tuning).
