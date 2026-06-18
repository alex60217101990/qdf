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

Nothing to do — the bench **downloads the sample data itself**. On start it
fetches the [adalanche-sampledata](https://github.com/lkarlslund/adalanche-sampledata)
tarball into a temp dir, extracts only the `goad/localmachine/*.json` dumps
(the `*.objects.msgp.lz4` AD dumps are skipped — they need adalanche's own
codec), and removes the whole temp tree when the run ends. No archive is left on
disk.

Pass `-datapath` to point at a **local clone** instead (offline runs, or to
avoid re-downloading across repeated runs):

```sh
git clone https://github.com/lkarlslund/adalanche-sampledata
go run ./cmd/qdf-bench -datapath ./adalanche-sampledata
```

## Run

One build (default tags), auto-downloading the data:

```sh
go run ./cmd/qdf-bench
```

Full build-tag matrix (`none`, `qdf_reflect2`, `qdf_simd`, both) — build tags are
compile-time, so this builds one binary per combo:

```sh
./cmd/qdf-bench/run.sh
```

Flags: `-datapath` (optional — local clone; default downloads to a temp dir),
`-iters N` (iterations per measured op, default 200), `-cpuprofile` / `-memprofile`
(write a `go tool pprof` profile). Extra flags are forwarded by `run.sh`.

## Reading the table

```
repr   bundle    dec     ser_ns  ser_B  ser_alloc ser_liveKiB deser_ns deser_B deser_alloc deser_liveKiB wire_B
typed  Balanced  copy    1443609 368788 741       264         1321929  575798  6427        590           208992
map    Balanced  copy    2098801 656730 1576      298         2176466  828749  12858       822           222694
map    Balanced  nocopy  2098801 656730 1576      298         1985046  625924  10794       598           222694
map    Balanced  arena   2098801 656730 1576      298         2173360  848768  10968       877           222694
```

- `repr` — `typed` (Go structs) vs `map` (`map[string]any`). Typed is qdf's strong
  path: roughly half the allocs and smaller wire.
- `bundle` — encode option set. Covers every `Opt*` flag at least once: the three
  presets (`Speed` / `Balanced` / `Compression`), `Dense`, `Dense+QPack`, and the
  `Bal+*` increments (`Gorilla`, `RANS`, `FSST`, `ColIndex`, `MapShape`,
  `Canonical`) plus `Comp+ColIndex`. (`OptDeltaNoBaseFingerprint` is omitted — it
  only affects `Diff`/`Apply`, not this Marshal/Unmarshal path.)
- `dec` — decode mode. `copy` is the default (every string copied out). `nocopy`
  and `arena` are the decode-side `QueryOption`s — they only ride the dynamic
  `Unmarshal` path, so they appear on `map` rows only (the typed `UnmarshalT` API
  takes no options and always runs `copy`). They change **decode** cost only;
  `ser_*` and `wire_B` are identical across a bundle's decode modes.
  - `nocopy` (`WithNoCopy`) — decoded strings alias the input buffer: near-zero
    string allocs, faster decode, **lowest `deser_liveKiB`** (borrows, copies
    nothing), lifetime-bound to the buffer.
  - `arena` (`WithArena`) — copied string bodies bump-pack into a reused arena
    (reset per op to model per-message reuse). It copies the same bytes as `copy`
    (so `deser_liveKiB` ≈ `copy`) but in far fewer allocations (lower
    `deser_alloc` → less GC pressure) — that, not resident size, is its win.
- `ser_*` / `deser_*` — per-op **encode / decode** cost, averaged over the sample
  files. **Each is isolated to the qdf call only**: the value (and decode
  `QueryOption`) is prepared once before timing, the timed loop runs nothing but
  `qdf.Marshal` / `qdf.Unmarshal`, a sink prevents the compiler from eliding the
  call, and a warmup pass primes the pools/heap so first-call lazy costs never
  land in the measured window.
  - `*_ns` — nanoseconds per op, the **minimum over 3 re-timed rounds**: GC
    pauses and scheduler preemption only ADD wall-clock, so the fastest round is
    the cleanest estimate of the op's own CPU (without disabling GC, which would
    understate the GC-bound decode path).
  - `*_B` / `*_alloc` — bytes and allocations **allocated** per op, from
    `runtime.MemStats` deltas across one GC-settled round (the same quantities as
    `go test -bench`'s `B/op` and `allocs/op`). This is the total churned per op.
  - `*_liveKiB` — **resident** memory one result occupies: the live-heap delta
    measured while `liveSamples` (24) results are held alive simultaneously, then
    divided by the sample count. Holding many at once amortizes the span-level GC
    granularity that makes a single-result reading noisy. For the `arena` decode
    mode the samples decode into one un-reset arena, so this figure includes the
    arena buffer the decoded strings alias. (This replaced an earlier `*_rssKiB`
    column built from `getrusage` Maxrss — a monotonic process high-water mark
    that showed a value only for the first op to grow the heap and 0 for every op
    after, i.e. not attributable per op. That was a measurement bug, now fixed.)
- `wire_B` — encoded size (average per file).
- **Reference rows** follow the qdf matrix: `bundle` = `json` (`encoding/json`)
  and `msgpack` (`github.com/vmihailenco/msgpack/v5`), `dec` = `-`. They run the
  exact same typed and map values through the same isolated harness, so qdf's
  wire size, allocs, and CPU can be read directly against a familiar baseline. A
  reference round-trip that doesn't DeepEqual the source is reported as a warning
  (a reference codec needn't preserve values the way qdf does) and still timed.
- A `whole-process peak RSS` line follows. This is the **whole process**, not a
  per-op cost: it holds all sample files in both representations at once, plus the
  Go runtime baseline, and is a high-water mark inflated by retained (not-yet-
  returned-to-OS) freed heap spans after gigabytes of churn over the run. A heap
  profile (`-memprofile`) confirms the *live* heap at the end is only a couple of
  MiB — the figure is GC/runtime span retention, not a qdf leak. Use `*_B` and
  `*_liveKiB` for actual per-op memory.

## Codegen vs reflect

After the matrix the bench prints a **codegen** section: the `[]Service` and
`[]Task` slices gathered from every host, encoded both through the reflection
path (the real `Service` / `Task` types) and through qdfgen-generated
`MarshalQDF` / `UnmarshalQDF` (the `GenService` / `GenTask` defined types —
regenerate with `go generate ./cmd/qdf-bench`). `GenTask` keeps the dynamic
`map[string]any` Definition field, so its codegen row also proves qdfgen now
handles interface (`any`) fields via a reflect fallback — code generation is no
longer limited to fully static schemas.

Reading it: codegen emits a fixed Fast-framed body and **ignores Options** (a
`Marshaler` is opts-invariant by contract), so it does not get the cross-record
shape-intern / columnar transposition that `Balanced` reflection applies to a
homogeneous `[]struct`. On these repetitive slices that makes reflect+`Balanced`
the smaller, lower-alloc choice; codegen's win is on single / heterogeneous
records and nested encoder threading, where there is nothing to intern.

## Streaming

A **streaming** section encodes the whole batch through each codec's streaming
encoder and decodes it back, per value: qdf `StreamEncoder` / `StreamDecoder`
vs `encoding/json` and msgpack `NewEncoder` / `NewDecoder`, for both the typed
and map representations. qdf streams the `summaryBundle` options.

## Profiling

```sh
go run ./cmd/qdf-bench -memprofile mem.prof          # heap profile, written after the run
go tool pprof -alloc_space -top mem.prof             # where bytes are allocated
go tool pprof -inuse_space -top mem.prof             # what is live at the end

go run ./cmd/qdf-bench -cpuprofile cpu.prof
go tool pprof -top cpu.prof
```

On this data the cumulative allocation is dominated by the dynamic
`map[string]any` decode (`decodeAny` + `popOrMakeMap` ≈ 60 % of all bytes) —
every value is boxed in an `interface{}` and every map is allocated. The typed
struct path allocates ~half as much. String-body copies are the next chunk, and
are what `nocopy` / `arena` cut.

Profiling already paid off once: it found `encodeReflect` taking a `reflect.New`
per by-value `any`, so every nested `map[string]any` / `[]any` cost an
allocation — fixed with concrete fast paths (typed encode 739 → 4 allocs/op,
map encode 1571 → 5). The remaining encode-CPU gap vs msgpack is the
intern / shape bookkeeping that buys the 2.4–4× smaller wire (an opt-out via
`OptSpeed`), plus the opt-in rANS / FSST codecs in the Compression bundles;
decode is at the structural floor of dynamic Go (interface boxing + map
allocation + string copies). No further free lever was found.

## Tests

```sh
go test ./cmd/qdf-bench
```

Fully self-contained — `TestMain` downloads the sample data to a temp dir, runs
a lossless round-trip across the **full** encode-option matrix (and every decode
mode for the map path), then cleans up. Nothing to clone by hand. Set
`QDF_BENCH_DATAPATH=<clone>` (or `QDF_BENCH_SAMPLE=<file>`) to run offline against
a local copy; with no data reachable the tests skip rather than fail.

## Notes for hacking on it

- `types.go` mirrors adalanche's `localmachine.Info`; the deeply variable
  `Task.Definition` is kept as `map[string]any` (it genuinely is). If the sample
  schema changes, the round-trip gate will fail loudly — update the structs.
- `tags_*.go` stamp the active build tags via `init()`; `tags.go` reports them.
- The bench loads everything once, then times marshal/unmarshal with a fixed
  `-iters` loop (fast and deterministic vs `testing.Benchmark`'s auto-tuning).
