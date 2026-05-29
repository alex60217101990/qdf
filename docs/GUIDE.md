# qdf — developer guide

This is a deep-dive companion to the README. README answers "what is
this and how do I use it"; this file answers "why is it shaped this
way and how does each feature pay for itself". If you are deciding
whether to depend on qdf, reading code in `state.go` /
`encoder.go` / `decoder.go`, or planning a contribution, start here.
For practical usage and preset selection see [`docs/USAGE.md`](USAGE.md).

The shape of this guide:

1. [TL;DR](#tldr) — when to reach for qdf, when not to.
2. [The mental model](#the-mental-model) — three layers (Fast / QPack
   / Dense) and the bit-mask that picks them.
3. [Wire format primer](#wire-format-primer) — tags you will see in
   hex dumps.
4. [Encoder & decoder lifecycle](#encoder--decoder-lifecycle) — pool,
   state, Reset, watermarks.
5. [Feature deep dive](#feature-deep-dive) — every codec, what it
   does, when it fires, what it costs.
6. [Memory model](#memory-model) — pool eviction, arena, shrink-on-
   Reset.
7. [Performance characteristics](#performance-characteristics) —
   numbers from `bench/profiles_test.go` grouped by scenario.
8. [Build tags](#build-tags) — `qdf_simd`, `qdf_reflect2`.
9. [Streaming, custom marshalers, codegen](#streaming-custom-marshalers-codegen).
10. [Common pitfalls](#common-pitfalls).
11. [Debugging](#debugging).
12. [Internal architecture map](#internal-architecture-map) — where to
    look for what.

Throughout this guide, code snippets compile against the public API
on `main`. Anything prefixed `internal/` is package-private; the
guide names those packages only as references, not as part of the
contract.

---

## TL;DR

qdf is a Go binary serialisation library aimed at the workloads
where `encoding/json` is too slow and `msgpack` leaves wire-size
gains on the table. It ships three layered codecs on a shared base
format:

- **Fast** — msgpack-shaped tag stream with type-specialised slice /
  map fast paths. Comparable to `vmihailenco/msgpack` on CPU, smaller
  by ~30 % on typical payloads.
- **QPack** — adds numeric / bool slice codecs (bit-pack, frame-of-
  reference, delta-FOR, Gorilla XOR) selected per-slice by size
  estimate. Beats msgpack by 2–5× on numeric arrays.
- **Dense** — adds an inline intern table for repeated strings,
  Markov-0 and Markov-1 predictors over intern IDs, Move-to-Front
  rank coding, and struct-shape interning. Beats msgpack by 3–8× on
  telemetry / log payloads.

Pick the layer with `Options`. They are independent bits; a single
`Marshal(v, opts)` entry point reads the mask. `OptSpeed` is the
zero-bit mask (Fast). `OptBalanced` is everything except future
heavy-CPU codecs.

You should NOT use qdf if:

- The wire is consumed by a non-Go reader. No bindings exist for
  Python / Rust / JS today.
- You need RPC / schema evolution semantics (use protobuf / Cap'n
  Proto).
- The payload is < 64 bytes and you marshal less than once per
  second — even `encoding/json` is fine there.

You SHOULD use qdf if:

- Long-lived encoder / decoder pools (services, batch jobs).
- Telemetry / logs / event streams with repeated string keys.
- Numeric vectors (metrics, embeddings, time series).
- Need to keep on-disk archive small without paying msgpack's CPU.

---

## The mental model

```
┌──────────────────────────────────────────────────────────────┐
│  Marshal(v, opts) / Unmarshal(b, &v)                         │
│  AppendMarshal(dst, v, opts)                                 │
│  NewStreamEncoder(w, opts)                                   │
└──────────────────────────────────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────────────────────────────┐
│  Encoder / Decoder                                           │
│   ├── opts Options    (bit-mask: OptDense, OptQPack, …)      │
│   ├── buf []byte      (output)                               │
│   ├── state *encState (lazy — only when OptDense is set)     │
│   └── pool members are reset and returned by sync.Pool       │
└──────────────────────────────────────────────────────────────┘
        │
        ▼
┌─── Fast path ──────────┐ ┌─── QPack path ─────────────┐ ┌─── Dense path ───────────────┐
│  msgpack-shaped tags   │ │  bitpack / FOR / Gorilla   │ │  intern table + LRU + Markov │
│  type-specialised      │ │  on numeric / bool slices  │ │  + shape interning           │
│  slice / map encoders  │ │  picks the smallest form   │ │  + arena-backed key storage  │
└────────────────────────┘ └────────────────────────────┘ └──────────────────────────────┘
```

`Options` is `uint32`. The six bits in use today, in declaration
order:

| Bit | Const             | What it gates                          | Depends on |
|----:|-------------------|----------------------------------------|------------|
|  0  | `OptDense`        | intern table, state-ref tags, arena    | —          |
|  1  | `OptQPack`        | numeric / bool slice codecs (raw-LE / FOR / Delta+FOR / bitpack) | — |
|  2  | `OptShapeIntern`  | struct shape table, `tagMapShape`      | `OptDense` |
|  3  | `OptPairPred`    | Markov-1 predictor over state-refs     | `OptDense` |
|  4  | `OptMTF`          | Move-to-Front rank coding              | `OptDense` |
|  5  | `OptGorillaFloat` | Gorilla XOR codec for `[]float64` / `[]float32` (~70 % wire reduction on smooth time-series, ~10× CPU/slice) | `OptQPack` |

`OptSpeed = 0`. `OptBalanced = OptDense | OptQPack | OptShapeIntern
| OptPairPred | OptMTF`. `OptCompression = OptBalanced | OptGorillaFloat`
— it diverges from balanced by exactly that one bit so workloads that
care about wire size more than encode latency opt in without reaching
for individual flags.

Dependent bits (`OptShapeIntern`, `OptPairPred`, `OptMTF`,
`OptGorillaFloat`) are no-ops without their parent and the encoder
records that fact silently — no error, no warning.
`TestValidity_DependentBitsAreNoOpsWithoutDense` pins the contract.

Reserved bits (6..31) are silent no-ops too. Setting them today does
nothing; setting them tomorrow may opt you into a new codec. This is
deliberate — the constant name is the API, the bit position is an
internal allocation.

### Why a bit-mask and not "modes"?

An earlier prototype had `Marshal`, `MarshalDense`, `MarshalQPack`,
`MarshalDenseQPack`, … one function per combination. With five bits
that explodes to 32 entry points. The bit-mask collapses it to one,
keeps the wire stable across opt-in/out, and lets a downstream
caller toggle codecs by `opts ^ OptMTF` without re-importing
anything.

---

## Wire format primer

Every QDF buffer starts with a 5-byte little-endian header (see
`writeHeader` / `readHeader`). After the header the stream is a
sequence of tag-prefixed values.

Tag ranges:

```
0x00..0x7F  fixint        (positive, value is the tag byte)
0x80..0x9F  fixstr        (length 0..31 packed into the tag)
0xA0..0xBF  fixarr        (length 0..31 packed into the tag)
0xC0        nil
0xC1, 0xC2  false, true
0xC3..0xC6  uint8 .. uint64
0xC7..0xCA  int8 .. int64
0xCB, 0xCC  float32 / float64
0xCD..0xCF  str8, str16, str32
0xD0..0xD2  bin8, bin16, bin32
0xD3, 0xD4  arr16, arr32
0xD5..0xD7  map8, map16, map32
0xD8..0xDF  negfixint     (-1..-8 packed into the tag)
```

So far so msgpack-shaped. The qdf-specific tags start at 0xE0:

```
0xE0  tagInternStr      (Dense)   first occurrence of an interned string
0xE1  tagStateRef       (Dense)   reference: varuint(id)
0xE2  tagInternBin      (Dense)   first occurrence of an interned bytes payload
0xE3  tagPackBool       (QPack)   bitpacked []bool
0xE4  tagPackRaw        (QPack)   raw little-endian numeric slice
0xE5  tagPackFor        (QPack)   Frame-of-Reference bitpacked integer slice
0xE6  tagPackDeltaFor   (QPack)   Delta + zigzag + FOR integer slice
0xE7  tagPackGorilla    (QPack)   Gorilla XOR-coded float slice
0xE8  tagStateRepeat    (Dense)   Markov-0 hit: id == lastID
0xE9  tagStateMTF       (Dense)   MTF rank reference: varuint(rank)
0xEA  tagStatePair      (Dense)   Markov-1 hit: varuint(rank=0 in top-1)
0xEB  tagPackRLE        (QPack)   run-length encoded integer slice
0xEC  tagMapShape       (Dense)   struct shape table reference
0xED  tagPackDict       (QPack)   dictionary-coded integer slice
0xEF  tagColStruct      (QPack)   columnar container for []struct; see below
0xF0..0xF2  tagExt8/16/32         user-extension envelope
0xF3        tagTimestamp          int64 ns since unix epoch
0xF4  tagPackALP        (QPack)   ALP decimal-coded []float64 slice
```

Tags 0xEE, 0xF5..0xFF are reserved.

A `tagStateRef` payload is `varuint(id)`. A `tagStateMTF` payload is
`varuint(rank)` where rank 0 means "most recently emitted". A
`tagStatePair` payload is also `varuint(rank)` but the predictor is
top-1, so the rank byte is always 0 (kept on the wire for parser
compatibility; K=4 was benchmarked and showed negligible hit-rate
gain over K=1 at significant memory cost, so the predictor was kept
at top-1 for simplicity).

`tagMapShape` is a small protocol of its own:

```
0xEC varuint(shapeID)
    if shapeID == 0:  declaration
        varuint(nKeys)
        nKeys × tagStateRef / tagInternStr      key intern IDs
    else:  reference
        nKeys × <value emissions>
```

A struct emission with `OptShapeIntern` enabled either declares the
shape (on first sight) or references it. The decoder maps shape IDs
to ordered key lists and dispatches values into the receiving
struct's fields.

---

## Encoder & decoder lifecycle

```go
// Pool-backed entry points (Marshal, AppendMarshal, Unmarshal) take
// an encoder / decoder out of a sync.Pool, use it for one call, and
// put it back. Each pool entry carries:
//
//   * opts Options              — captured for the call
//   * buf  []byte               — output / cursor
//   * state *encState | *decState — lazy; only allocated when OptDense
//                                   is in the mask
```

The encoder constructor does NOT allocate `state`. It is allocated
on the first `WriteString` / `WriteBytes` that enters the Dense
branch:

```go
// encoder.go simplified
func (e *Encoder) WriteString(s string) {
    ...
    if e.opts.Has(OptDense) {
        if e.state == nil { e.state = newEncState() }
        ...
    }
}
```

This matters: an `OptSpeed` / `OptQPack` workload pays nothing for
the Dense scaffolding, even from the same pool. Stay in Fast / QPack
mode and your encoder is a `*[]byte` with a tag dispatch.

### Reset and the pool

When the entry point returns, `pool.Put(enc)` runs `enc.reset()`:

```go
func (e *Encoder) reset() {
    e.buf = e.buf[:0]    // keep capacity
    if e.state != nil {
        e.state.reset()  // see state.go
    }
}
```

`state.reset()` is where the watermark-based shrink lives. Defaults:

| Structure          | Soft cap | What hits it                                          |
|--------------------|---------:|-------------------------------------------------------|
| `internTable`      | 4 096 ids| flat hash table — drop at `2 * 4096` slots            |
| `lruLink`          | 4 096    | packed `(prev<<16\|next)` chain capacity              |
| `pairPred`         | 4 096    | Markov-1 predictor slice capacity                     |
| `shapes`           | 1 024    | declared struct shapes                                |
| `arena`            | 256 KiB  | total chunk capacity (see internarena)                |
| `stringValues`     | 4 096    | decoder string-cache; truncated with `values` (decode-side only) |

The 128-entry MRU ring side-cache and the four hot scalars
(`lastID`, `lruHead`, `mruHead`, `internLoad`) live inline in
`encState` — fixed-size, no shrink path.

Over the cap → rebuild / drop. Under → reuse in place. The pool
encoder buffer follows the same rule with `maxPooledBuf = 16 MiB`
(see [Memory model](#memory-model)). Long-running services with
bursty traffic stay at the steady-state working set rather than
pinning the historical peak.

### Decoder symmetry

The decoder mirrors every encoder structure: `values [][]byte` for
the resolved intern bytes (aliasing the wire buffer), a parallel
`stringValues []string` that caches the materialised Go-string of
each intern record (populated lazily on the first `ReadString`),
`lruLink` for the packed MTF chain, `pairPred` for Markov-1,
`shapes` for shape-table lookups, and its own `mruRing` so
`tagStateMTF`-encoded ranks resolve in O(1) without walking the
chain. `pairPred`, the LRU chain, and both sides of the MRU ring
are kept in sync per emission so a `tagStateMTF 0` always
resolves to the most-recently-emitted ID on both sides.

`stringValues` is a pure decoder-side optimisation: on dense
payloads (telemetry, archive) the same intern record is read
N times via state-ref / MTF / pair / repeat tags; without the
cache each read paid for a fresh `string(b)` heap copy. With the
cache, the first read materialises and stores the string; the
remaining N-1 reads return the cached value with zero alloc.
Empty interned bytes resolve directly to `""` without touching
the cache.

If the encoder and decoder diverge by a single bookkeeping step, the
stream becomes ambiguous from that point on. Every code path that
touches `state.lastID`, `state.pairRecord`, `state.lruMoveFront`,
or `state.mruPush` runs the same call sequence on both sides. Tests
in `cycle_test.go`, `pair_shape_test.go`, and `mtf_test.go` pin
those sequences.

---

## Feature deep dive

### Intern table (Dense)

```
First sight:  0xE0 varuint(len) <bytes>          tagInternStr
Re-emission:  0xE1 varuint(id)                   tagStateRef
              0xE8                               tagStateRepeat  (id == lastID)
              0xE9 varuint(rank)                 tagStateMTF
              0xEA varuint(rank)                 tagStatePair    (rank 0 today)
```

When `OptDense` is set, `WriteString(s)` and `WriteBytes(b)` look up
the value in `encState.internTable` — a flat open-addressed hash
table (`[]internSlot`, hash precomputed via `hash/maphash.String`
and stored alongside the key + id). On a miss the key is copied
(more on that below) and assigned the next sequential ID; the slot
is installed via linear probing and the table doubles when load
crosses 0.5. On a hit the encoder picks the smallest re-emission
tag:

```
if id == lastID                          → tagStateRepeat   (1 byte)
elif id < 0x80                           → tagStateRef (2)
elif (prev, id) in predictor             → tagStatePair (2)
elif OptMTF and rank-varuint < id-varuint → tagStateMTF (1 + rank-varuint)
else                                     → tagStateRef (1 + id-varuint)
```

The intent: never grow the wire vs the raw `tagStateRef` encoding.
Decode reverses every branch using `lastID`, the LRU, the predictor,
and the intern table — all three structures are kept in sync per
emit. The arithmetic that picks the form is in [`encoder.go`
`emitStateRef`](../encoder.go).

Tuning knobs on the encoder:

- `minIntern` — minimum string length for intern eligibility
  (default 4). Shorter strings go straight to `writeStringInline`
  because the inline encoding would be at most 1 byte longer than
  the state-ref version.
- `maxStateEntries` — hard cap on `state.internLoad` (default
  16 384). After the cap the encoder falls back to inline emission
  for new values. Existing entries continue to hit; the cap only
  blocks unbounded growth on adversarial input. The 16-bit MRU
  ring slot stores IDs up to `0xFFFF - 1`, so the cap also keeps
  the ring sentinel space safe.

Both knobs live on the encoder (set via `NewEncoderOpts`); the
top-level `Marshal` uses defaults.

### Arena allocator (`internal/internarena`)

Replacement for `strings.Clone(key)` on intern misses. The arena is a
chain of `[]byte` slabs; each `Put(s)` copies `s` into the active
slab and returns a packed `uint32` id. `Get(id)` decodes that id back
to a slice that aliases the slab.

Why not `strings.Clone`: one heap allocation per first-occurrence
intern. On a 10 000-string telemetry batch that is 10 000 small heap
blocks, all GC-tracked, all eligible for the next sweep. The arena
collapses them into one growing `[]byte` and keeps the GC scanner out
of the per-key path entirely.

Layout (see [`internal/internarena/arena.go`](../internal/internarena/arena.go)):

- `off uintptr` — byte offset inside `chunks[cur]`. Stored as an
  integer (not a pointer) so the per-Put store does not trip the GC
  write barrier. Same effect as the `next uintptr` trick from
  mcyoung's "Cheating the Reaper in Go", without the `go vet`
  warning for `unsafe.Pointer(uintptr)` round-trips.
- `chunks [][]byte` — every slab the arena ever owned. Doubling
  growth from a 4 KiB seed; the chain keeps prior slabs GC-rooted.
- `locs []uint64` — `(chunk_idx<<48) | (offset<<16) | length`.
  A `Get(id)` is one indexed load + one slice header.

Behaviour on `Reset()`:

```
total chunk cap ≤ DefaultRetainBytes (256 KiB) → keep chunks
total chunk cap >  DefaultRetainBytes           → drop chunks[1..],
                                                  keep chunks[0]
```

`ResetWithLimit(retainBytes)` overrides the cap; `0` disables the
shrink entirely. The contract is documented at the top of
`arena.go`; the key invariant for callers is that `Get` slices alias
the chunk they live in and become invalid after Reset.

`encState.lookupOrAssign` is the only caller. It wraps `arena.Put`
and `arena.Get` and rebinds the intern slot key to an
`unsafestr.String` header that aliases the arena bytes — zero copy,
zero alloc on the hot path.

### Move-to-Front (MTF) — `tagStateMTF`

When the encoder emits a state-ref, the touched intern ID moves to
the head of an LRU chain. The chain is stored in `encState.lruLink`
— a single `[]uint32` slice indexed by ID, with the previous and
next neighbour packed as `(prev << 0) | (next << 16)`. Halving the
array footprint vs separate `prev`/`next` slices roughly halves the
cache lines a `lruMoveFront` has to touch.

With `OptMTF` set, the rank in the chain is encoded as `varuint`.
On streams with locality the rank's varuint is shorter than the raw
id's varuint, so the MTF emission wins.

Rank discovery is the historical bottleneck: walking the linked
list head-first is one pointer chase per step into cache-cold
memory, and a typical telemetry batch needed > 10 % of CPU just for
that walk. `encState.mruRing` (128 uint16 slots, two cache lines)
caches the IDs of the last 128 emissions in order; `mruRank`
scans backward from the head and returns the ring offset on hit.
That offset is the LRU chain rank by construction (every emit
pushes onto the ring), so the encoder skips the chain walk
entirely on hit and falls back to a raw `tagStateRef` on miss —
the chain rank is then ≥ 128 and would not have beaten the raw
varuint anyway. The decoder carries the same ring (`decState.mruRing`)
so `tagStateMTF + rank` decodes via direct indexing in O(1).

Code: `encState.mruRank`, `encState.lruMoveFront`,
`decState.mruIDAtRank` (with the linked-list `lruIDAtRank` as a
forward-compat fallback for rank ≥ ring size).

### Markov-0 — `tagStateRepeat`

When the next intern ID equals `lastID`, emit a single tag byte. No
payload. This catches columnar repetition (every row's `service`
field is the same string) without any predictor state at all — just
the `lastID` register.

Inlined out of `emitStateRef` into `WriteString` / `WriteBytes`
because it is the most common Dense hit and the call site benefits
from avoiding a non-inlinable function call.

### Markov-1 predictor — `tagStatePair`

For each `prev` intern ID we remember its most-recent successor
(top-1; K=4 was evaluated and found to offer negligible hit-rate
gain at significant memory cost, so the predictor remains at K=1). When the next emission matches, write `0xEA 0x00` —
two bytes regardless of how many digits the raw ID has.

Top-1 storage:

```go
pairPred []uint32   // [prev] = succ+1, 0 = empty
```

The `+1` packing lets `clear()` (memclr) reset the slice without an
explicit fill loop, while still letting `succ == 0` be distinguishable
from "no successor recorded".

Hit rate impact vs the prior K=4 ring: the K=4 ring caught cyclic
A→{B,C,D,E,B,C,…} workloads; top-1 misses every step on those. On
stable workloads (A→B repeated, with intermittent noise) the two
agree. Real telemetry sits in the stable regime; the 256 KiB memory
saving is worth the cyclic-workload regression.

Code: `encState.pairLookup`, `encState.pairRecord`, both single
instructions; `decState.pairAtRank` mirrors them.

### Shape interning — `tagMapShape`

A struct emission with `OptShapeIntern` enabled:

1. Look up the struct's `*typeDesc` in the encoder's
   `shapeBindings` slice. Found → emit `0xEC varuint(shapeID)` +
   values. Done.
2. Not found → register a new shape:
   - Reserve the next sequential shape ID.
   - Emit a declaration: `0xEC 0x00 varuint(nKeys) (keys × state-refs)`.
   - Record the binding so subsequent emissions of the same type
     hit the cache.

`encState.shapeAssign` keys on the ordered list of intern IDs of the
struct's fields. Two distinct Go types with the same ordered field
names share a shape — that is desired, because the wire only cares
about names.

Decoder symmetry: `decState.shapes[id-1]` carries the resolved field
names; `tagMapShape varuint(id)` dispatches values into the
receiving struct's fields in declaration order.

Win condition: arrays of identical struct types. The first emission
declares the shape (a few bytes) and every subsequent struct trades
its key emissions for a single shape-ID varuint. On a 100-row
struct array, a per-row saving of ~12 bytes is realistic.

### Map fast paths (`maps_fast_generated.go`)

Generic reflect-driven map encode/decode pays a per-element
`reflect.Value` materialisation. For 27 (K, V) pairs we generate
typed encode / decode functions that bypass reflect entirely:

```go
// key types covered: string, int, int64, uint64
// value types covered: string, bool, int8/16/32/int/int64,
//                      uint8/16/32/uint/uint64, float32/float64,
//                      []byte, []string, any
```

The generator lives at `internal/mapsgen/main.go` and the dispatch
table (`installMapFastPath`) is part of the generated file. To add a
pair: append to the `pairs` slice in the generator, run
`go generate ./...`, commit the regenerated file alongside the
generator change.

Each generated path does roughly:

```go
func encodeMapStringInt64(e *Encoder, p unsafe.Pointer) error {
    m := *(*map[string]int64)(p)
    if m == nil { e.WriteNil(); return nil }
    e.WriteMapHeader(len(m))
    for k, v := range m { e.WriteString(k); e.WriteInt(v) }
    return nil
}
```

For decode the key path interns through `d.keyCache.Make(kb)` — a
small dedupe so high-cardinality repeats stay zero-alloc.

### QPack — numeric / bool slice codecs

`OptQPack` activates a family of codecs that swap msgpack-style
per-element tags for compact bit-packed representations on numeric
and bool slices. The encoder picks the smallest predicted form per
slice; the decoder reads the picked tag.

```
0xE3  tagPackBool      []bool       1 bit per element + tag + varuint(n)
0xE4  tagPackRaw       []intN/uintN raw little-endian, no padding
0xE5  tagPackFor       []uintN      Frame-of-Reference bitpacked
0xE6  tagPackDeltaFor  []intN       Delta + zigzag + FOR
0xE7  tagPackGorilla   []float64    Gorilla XOR coding
0xEB  tagPackRLE       []intN       Run-length encoded (value, runLen) pairs
0xED  tagPackDict      []intN       Dictionary-coded; ≤16 distinct values
```

Selection logic:

- **Bool slice**: always `tagPackBool` (8× smaller than per-element
  tags).
- **Integer slice**: compute `m = min(s)`, find the smallest power-
  of-two `bitsPer` that fits `max(s) - m`. If `qpackForSizeUnsigned`
  beats raw, candidate is FOR. For monotonic / clustered series the
  picker also evaluates Delta+FOR. For run-heavy columns (status
  codes, enum-like values, sparse counters) a cheap run-fraction
  probe over the first 32 elements decides whether to compute the
  full RLE size. For small distinct cardinality (≤ 16) with a wide
  spread where FOR can't pack tightly, the picker evaluates dict
  (`tagPackDict`). The winning estimator wins.
- **Float slice**: by default, raw. Gorilla is opt-in via
  `OptGorillaFloat` (bundled into `OptCompression`). When the bit is
  set, `pickF64Codec` probes the first 32 consecutive XOR pairs; if
  the projected per-sample bit cost stays comfortably below raw 64
  it emits `tagPackGorilla`, otherwise it falls back to raw. The
  probe runs in ~30 ns. Gorilla wins on real-world time-series
  telemetry but loses on white noise — the threshold pick keeps it
  from firing where it would.

The size estimators are pure functions on slice contents; the
encoder calls them once per slice and picks the smallest.

#### SIMD AVX2 (`qdf_simd` build tag)

`qpack_simd_amd64.s` is hand-rolled AVX2 for the QPack integer/bool
inner loops, both directions:

- **Decode** (`VPMOVZX*`) at byte-aligned `bitsPer ∈ {8, 16, 32}`,
  and (`VPBROADCASTQ` + `VPSRLVQ`) at every width `1..28`. Two general
  kernels load one 8-byte window per group and shift each lane by a
  per-group offset selected from a small table: 4 values/iter for
  `b ≤ 14` (`7 + 4·14 < 64`), 2 values/iter for `15 ≤ b ≤ 28`
  (`7 + 2·28 < 64`). Fixed-shift kernels handle the hot `{10,12,14,20}`.
  Widths 29–31 and 33–56 stay on the scalar window.
- **Encode** at `{8, 16, 32}` (`VPSHUFB` byte-gather) and `{10,12,14,20}`
  (`VPSLLVQ` shifts each value to its slot, then a lane-OR reduction folds
  the group into one byte-aligned chunk). Odd / wider encode widths stay
  scalar (they would need cross-chunk byte merging).
- `[]bool` pack (`VPSLLW` + `VPMOVMSKB`).

Widths not listed (odd widths, ≥ ~24 non-aligned) and the float
codecs stay on the scalar path. CPUID-gated at runtime; non-amd64
builds compile a scalar stub. Output is byte-identical to scalar —
the tag is a pure speed switch. See `docs/USAGE.md` for a
plain-language "when to use it" guide.

### reflect2 swap (`qdf_reflect2` build tag)

Optional opt-in. Replaces `reflect.MakeSlice` / `MakeMapWithSize` /
`reflect.New` calls in the decoder with `modern-go/reflect2`'s
unsafe equivalents. Saves a handful of allocations per decode on
slice / map heavy payloads. No change in correctness; the build tag
exists so consumers who do not want a non-stdlib dep can opt out.

### qdfgen (`cmd/qdfgen`)

Code generator that emits `MarshalQDF` / `UnmarshalQDF` methods for
user structs. Generated methods bypass the reflect-based encoder
entirely, calling the typed `Encoder.WriteX` API directly.

```bash
go install github.com/alex60217101990/qdf/cmd/qdfgen@latest
```

```go
//go:generate qdfgen -type Event,User .
```

For workloads with the same handful of struct types serialised
millions of times (RPC payloads, event streams), this is the
biggest single CPU win: -30 % to -60 % vs the reflect path,
depending on struct shape.

The generator is independent of `internal/mapsgen` — that one
generates the map type-switch dispatch inside `package qdf`; this
one generates per-user-type marshalers in the user's package.

### Columnar struct-array codec (`tagColStruct`, 0xEF)

When the encoder sees a `[]SomeStruct` under `OptBalanced`
(Dense + ShapeIntern), it checks whether the element type is a flat,
homogeneous struct — fields of kind `int*`, `uint*`, `float*`, `bool`,
`string`, or `[]byte`, no custom marshalers, no nested structs. If it
is, a per-array probe samples up to 16 elements and estimates the
columnar wire size vs row-major. On a commit the encoder transposes
the slice:

- Numeric (`int*`/`uint*`) and bool columns are gathered into a
  scratch slice and passed through the existing QPack codec selector
  (FOR, Delta+FOR, RLE, dict, bitpack). Each column's full-length
  slice is encoded as a single QPack payload.
- Float columns are emitted as raw-LE slices (Gorilla is opt-in via
  `OptGorillaFloat`).
- String and `[]byte` columns are emitted as M consecutive inline
  string/bytes values through the normal intern path; repeated values
  collapse to state-refs.

The wire layout is `0xEF, varuint(M), varuint(shapeID)` followed by K
column payloads in declaration order. `shapeID == 0` declares a new
columnar shape inline (field names + kinds); subsequent arrays of the
same struct type reuse the ID. The decoder scatters columns back into
the output `[]SomeStruct`.

This path is **automatic under `OptBalanced`** — there is no flag to
set. The probe is conservative: when the estimated columnar size does
not clearly beat row-major, the encoder falls back to the existing
row-major path. Non-struct-array payloads and incompressible inputs
are byte-and-speed identical to the pre-columnar behaviour.

Measured ~11× smaller than `OptSpeed` on a numeric-heavy event-batch
fixture. The columnar path subsumes the older column-conditional
repeat codec, which has been removed.

---

## Memory model

### Pool lifecycle

`Marshal` / `Unmarshal` are pool-backed. The pool grows under load,
contracts at GC time (the standard `sync.Pool` policy). Each pool
entry carries the encoder / decoder plus its associated state. State
allocation is lazy: an encoder that never enters Dense mode never
allocates `encState`.

### Watermark shrink-on-Reset

The naive pool design has a problem: a single outlier payload (a
1 MiB event with thousands of unique strings) grows the encoder's
intern table, LRU chain, pair predictor, and arena to fit. The
pool holds onto the encoder. Every subsequent call inherits the
peak footprint, forever.

Fix: each `reset()` checks the post-call capacity against a soft
cap. Over the cap → drop the backing array; under the cap → reuse
in place. Defaults in `state.go`:

```go
const (
    maxRetainedIDs       = 4096   // flat intern table threshold
    maxRetainedLRUCap    = 4096   // packed lruLink slice cap
    maxRetainedPairCap   = 4096   // pair predictor slice cap
    maxRetainedShapeCap  = 1024   // shape table cap
    internTableInitSize  = 64     // table size after over-cap rebuild
)
```

`internarena.DefaultRetainBytes = 256 KiB` is the arena's
counterpart cap. The pool encoder buffer follows the same rule
with `maxPooledBuf = 16 MiB` in `qdf.go`; outputs bigger than that
are dropped from the pool instead of pinning multi-megabyte
buffers across goroutines forever.

Tests in `shrink_test.go` pin the contract: after a burst that
exceeds every cap, `reset()` shrinks at least one structure;
post-reset capacity is bounded; new payloads still encode
correctly.

### Reset shape — flat table

The flat intern table is contiguous, so the shrink path is a
straight rebuild when the capacity is over the soft cap, and a
memclear of the in-place slots otherwise:

```go
if cap(e.internTable) > maxRetainedIDs*2 {
    e.internTable = make([]internSlot, internTableInitSize)
} else {
    for i := range e.internTable {
        e.internTable[i] = internSlot{}
    }
}
e.internLoad = 0
```

That `*2` accounts for the load-factor headroom: at load 0.5 the
backing array is twice the active entry count, so an active set of
4 096 ids sits on an 8 192-slot table — both are "steady state"
and neither triggers the rebuild. The rest of Reset is in-place
slice truncation or `clear()`.

### Arena and GC pressure

The arena's whole point is to keep GC-tracked allocations off the
intern path. On a 1 000-string telemetry batch, the per-key
`strings.Clone` would create 1 000 separately-tracked heap blocks;
the arena creates one slab that hosts all 1 000 payloads. The next
GC cycle scans one root instead of 1 000.

If you are GC-pressure-sensitive (long-running services, low-latency
SLOs) and you marshal many Dense payloads per second, the arena's
contribution is measurable: every percentage point of
`runtime.MemStats.PauseTotalNs` you spend in scanning intern objects
moves into "scan one arena slab" instead.

---

## Performance characteristics

Numbers from `bench/profiles_test.go`, Intel i7-9750H, Go 1.26.0,
median of 3 × 2 s runs. See `docs/BENCH.md` for the full matrix
and how to reproduce.

| Scenario        | json encode | msgpack encode | qdf encode | qdf vs msgpack |
|-----------------|-------------|----------------|------------|----------------|
| HotPath         | 650 ns      | 423 ns         | 400 ns     | **-5 %**       |
| TelemetryBatch  | 387 µs      | 477 µs         | 288 µs     | **-40 %**      |
| MetricSeries    | 161 µs      | 112 µs         | 4.09 µs    | **-96 %**      |
| EmbeddingVec    | 63 µs       | 25.6 µs        | 837 ns     | **-97 %**      |
| Config          | 2.23 µs     | 1.83 µs        | 1.39 µs    | **-24 %**      |
| Archive         | 1.81 ms     | 2.57 ms        | 1.93 ms    | **-25 %**      |

| Scenario        | json decode | msgpack decode | qdf decode | qdf vs msgpack |
|-----------------|-------------|----------------|------------|----------------|
| HotPath         | 1.55 µs     | 631 ns         | 287 ns     | **-55 %**      |
| TelemetryBatch  | 2.46 ms     | 994 µs         | 381 µs     | **-62 %**      |
| MetricSeries    | 565 µs      | 152 µs         | 4.16 µs    | **-97 %**      |
| EmbeddingVec    | 162 µs      | 40.5 µs        | 715 ns     | **-98 %**      |
| Config          | 6.03 µs     | 2.87 µs        | 1.68 µs    | **-42 %**      |
| Archive         | 11.8 ms     | 4.70 ms        | 2.20 ms    | **-53 %**      |

qdf wins across every workload in the matrix after the May 2026
series (MRU-ring + flat-intern-table + packed-lruLink + large-
payload buffer work; commits `ada9fd7`, `2ea3b48`, `02d6aac`) and
the follow-up branch `perf/decode-intern-cache` (cached decode
interning + mruRank unroll + PreIntern + codegen bench;
commits `7090e25`, `c0517e8`, `95d3c21`, `001864b`). Largest
absolute wins remain the QPack-friendly numeric workloads
(MetricSeries / EmbeddingVec) where Delta+FOR / Gorilla collapse
`float64`/`int64` columns to near-zero bytes per element, and
dense-friendly columnar workloads (TelemetryBatch / Archive)
where intern + Markov + MTF + shape interning bury repeated
keys and values.

Wire size, telemetry_1k payload:

```
json     112 KiB     1.00× baseline
msgpack   90 KiB     0.80×
qdf       18 KiB     0.16×
```

Roughly 5× smaller than msgpack on telemetry, 3–4× on archive
payloads.

---

## Build tags

| Tag             | Effect                                                                                                                                                                                                                                                                                | Platform                              |
|-----------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| `qdf_simd`      | AVX2 for the QPack integer/bool codecs: decode at every `bits ∈ 1..28` plus `32`, encode at `{8,10,12,14,16,20,32}`, `[]bool` pack. Output byte-identical to scalar. Runtime CPUID gate; non-AVX2 amd64 falls back transparently; other arches compile a stub. See docs/USAGE.md for when to use it. | amd64; AVX2 detected at run time      |
| `qdf_reflect2`  | Swap `reflect.MakeSlice` / `MakeMapWithSize` / `reflect.New` for `modern-go/reflect2` unsafe equivalents.                                                                                                                                                                             | none — pure Go                        |

Combine freely:

```bash
go build -tags qdf_simd ./...
go build -tags qdf_reflect2 ./...
go build -tags "qdf_simd qdf_reflect2" ./...
```

The default build (no tags) is the baseline behaviour. Tests run
under every combination in CI.

---

## Streaming, custom marshalers, codegen

### Streaming API

```go
enc := qdf.NewStreamEncoder(w, qdf.OptBalanced)
for _, ev := range events {
    enc.Encode(&ev)
}
enc.Close()
```

State (intern table, shape table, predictors) persists across
`Encode` calls — that is the whole point of streaming. A 10 000-row
event log shares one intern table; the second event onwards trades
its keys for state-refs.

`StreamDecoder` is symmetric. Reset semantics are unchanged: when
the stream encoder is returned to its pool (`Close`), the state
shrinks via the same watermark logic.

### Custom marshalers (`Marshaler` / `Unmarshaler`)

```go
type Marshaler interface {
    MarshalQDF(*Encoder) error
}

type Unmarshaler interface {
    UnmarshalQDF(*Decoder) error
}
```

Types that implement either interface bypass the descriptor cache
and the per-field dispatch entirely. The codegen tool emits methods
that satisfy these interfaces; you can hand-write them when the
generator's output is not what you want.

### Codegen (`cmd/qdfgen`)

```bash
go install github.com/alex60217101990/qdf/cmd/qdfgen@latest
```

```go
//go:generate qdfgen -type Event,User .
```

Emits `<package>_qdf.go` with concrete `MarshalQDF` /
`UnmarshalQDF` for each named type. The generated code calls the
qdf encoder / decoder API directly — no reflect, no runtime
descriptor lookup.

See `cmd/qdfgen/README.md` for the flag set and supported tags.

---

## Common pitfalls

**1. Forgetting that Dense options need `OptDense`.**

```go
// This silently encodes with Fast mode — no intern table.
b, _ := qdf.Marshal(v, qdf.OptShapeIntern|qdf.OptMTF)

// Want Dense? Always include OptDense.
b, _ := qdf.Marshal(v, qdf.OptDense|qdf.OptShapeIntern|qdf.OptMTF)
```

Dependent bits are silent no-ops without their parent. The Options
docstring spells it out; the test suite pins it; if a payload looks
suspiciously big, your first move is to print the mask and confirm
`OptDense` is set.

**2. Comparing wire across opt sets.**

```go
b1, _ := qdf.Marshal(v, qdf.OptSpeed)
b2, _ := qdf.Marshal(v, qdf.OptBalanced)
bytes.Equal(b1, b2)  // false — different encoding shape entirely
```

The opts mask is captured into the buffer header. The decoder reads
it back and dispatches accordingly. Two Marshal calls with different
opts produce different bytes by design.

**3. Holding `Decoder.ReadStringBytes` slices across calls.**

```go
b, _ := d.ReadStringBytes()
// b aliases the input buffer (or the intern table). After the next
// Read call, the buffer cursor advances and the slice might still
// look valid but its contents are not guaranteed stable.
```

`ReadString()` is the safe variant — it allocates a Go string copy.
Use `ReadStringBytes()` only when you immediately copy / hash / parse
the bytes.

**4. Building two structs with the same field names in different
orders.**

```go
type A struct { X int; Y int }
type B struct { Y int; X int }
```

Each gets a distinct shape ID. The wire grows by one declaration
per type. The intern table sees each field name once across both
types. Round-trip is fine.

But: don't try to decode an A-wire into a B. The shape table maps
shapeID → ordered field-name list. The decoder looks up the
receiving struct's field by name, so field order in the receiving
struct does not matter — but if you Unmarshal an A-wire into a B
the decoder will set `B.X` and `B.Y` to A's `X` and `Y` values
respectively. Different physical layout, same logical mapping. This
is a feature (rename-tolerant), but if you depend on order, you
will be surprised.

**5. Float slice compression — `OptQPack` vs `OptCompression`.**

Float slices are the only codec that trades latency for size in the
preset bundles. `OptQPack` (and therefore `OptBalanced`) keeps floats
on raw-LE bulk — predictable ~4 µs per 1024-sample slice, 8 B per
sample. `OptCompression` adds `OptGorillaFloat`; the encoder probes
the first 32 XOR pairs (~30 ns) and emits Gorilla XOR when the
projected per-sample cost stays comfortably below 64 bits.

Empirically, on `bench/profiles_test.go` Intel i7-9750H, Go 1.26.0:

| series                | opts          | wire (B) | encode  | decode  |
|-----------------------|---------------|---------:|--------:|--------:|
| random walk (1024×f64)| `OptQPack`    | 8391     | ~4.0 µs | ~4.2 µs |
| random walk (1024×f64)| `OptCompression` | 8391  | ~4.1 µs | ~4.3 µs |
| smooth (1024×f64)     | `OptQPack`    | 8398     | ~4.2 µs | ~4.0 µs |
| smooth (1024×f64)     | `OptCompression` | 2307  | ~41 µs  | ~40 µs  |

Reading the table: under `OptCompression`, smooth time-series
collapses ~72 % on the wire but encode/decode pay ~10× more CPU per
slice because Gorilla works at the bit level. On random-walk floats
the probe rejects Gorilla so the path stays raw and OptCompression
matches OptQPack exactly — wire and latency. The wrong choice is
`OptCompression` on hot-path metric ingest with smooth values; the
right choice is `OptCompression` on archival snapshots and offline
batches where the wire matters more than encode latency.

**6. `map[string]any` with positive integer values.**

```go
in := map[string]any{"n": int64(42)}
b, _ := qdf.Marshal(in, qdf.OptBalanced)
var out map[string]any
qdf.Unmarshal(b, &out)
// out["n"] is uint64(42), not int64(42)
```

`decodeAny` returns positive integers as `uint64`. This is shared
with msgpack and is documented behaviour; the test suite uses
`reflect.DeepEqual` so it shows up clearly. If you need a stable Go
type, decode into a concrete struct field instead of `any`.

---

## Debugging

### Hex dump

For tiny test payloads, hex-dump the buffer and read the tags.

```go
b, _ := qdf.Marshal(v, qdf.OptBalanced)
fmt.Printf("% x\n", b)
```

Use [wire format primer](#wire-format-primer) to identify tags.
`0x80..0x9F` is a fixstr (length in the low 5 bits); `0xE0` opens
an intern declaration; `0xE1` opens a state-ref; etc.

### Bench introspection

```bash
go test -bench=Profile -benchmem -count=3 -benchtime=2s ./bench
```

The "B/op" column is allocations per op; "allocs/op" is the count.
A regression on either is the canonical signal that a code change
upset the inliner or introduced a `reflect.Value` allocation.

For deeper inspection:

```bash
go test -bench=Profile_TelemetryBatch -cpuprofile=cpu.prof ./bench
go tool pprof -text cpu.prof | head -20
```

The hot functions you should see on Dense encode:

```
emitStateRef
appendUvarint
lookupOrAssign
runtime.mapaccess (intern lookup)
runtime.mapassign (intern install)
```

If you see `runtime.heapBitsSetType` high up, the arena path is not
being taken — usually because `OptDense` is off.

### Tracing single emissions

The tests in `golden_test.go` build small payloads byte-by-byte and
assert the exact wire. Copy one of those patterns when adding a new
codec or modifying an existing tag's emit logic. The golden
fixtures double as worked examples.

---

## Internal architecture map

Top-level files in the repo:

```
qdf.go                  — package API: Marshal, Unmarshal, AppendMarshal
encoder.go              — Encoder type, WriteX methods, emitStateRef
decoder.go              — Decoder type, ReadX methods, tag dispatch
state.go                — encState / decState: intern table, LRU, pair predictor, shapes
wire.go                 — tag constants, varuint helpers

reflect_encode.go       — typeDesc cache, reflect-based encode / decode
                          (reflect-alloc helpers moved to
                          internal/reflectutil/)

maps_fast.go            — //go:generate directive, doc stub
maps_fast_generated.go  — codegen output for 27 (K, V) map pairs
slices_fast.go          — typed slice fast paths
qpack*.go               — QPack codecs (raw, bool, FOR, delta, gorilla, simd)

stream.go               — StreamEncoder / StreamDecoder
marshaler.go            — Marshaler / Unmarshaler interfaces

internal/internarena/   — bump-pointer byte arena
internal/intern/        — short-string interner used by decoder
internal/bufpool/       — reusable []byte pool
internal/unsafestr/     — string ↔ []byte unsafe alias helpers
internal/mapsgen/       — generator for maps_fast_generated.go
internal/endian/        — NativeIsLittle build-tag const (replaces old root endian_*.go)
internal/reflectutil/   — MakeSlice / SliceData / MakeMap helpers
                          with reflect / reflect2 backends
                          (replaces old root reflect_alloc*.go)
internal/codegen_test/  — fixtures and tests for cmd/qdfgen output

cmd/qdfgen/             — code generator emitting MarshalQDF / UnmarshalQDF

bench/                  — separate go module with benchmarks vs json / msgpack
docs/BENCH.md           — bench numbers and reproduction instructions
docs/CHOOSING.md        — opts cheatsheet for end-users
docs/GUIDE.md           — this file
```

Test files (`*_test.go`) sit next to the code they exercise. The
ones worth knowing if you are debugging a wire-format question:

- `golden_test.go` — exact hex fixtures for every tag.
- `cycle_test.go` — round-trip across Marshal → Unmarshal cycles.
- `pair_shape_test.go` — Markov-1 + shape interning behaviour.
- `mtf_test.go` — Move-to-Front rank coding.
- `tag_matrix_test.go` — every tag emitted at least once across opts.
- `interface_matrix_test.go` — typed and reflect paths both round-
  trip the same payloads byte-for-byte.

When you change a wire-format-relevant function, the first thing to
re-run is `go test -run 'TestGolden|TestCycle|TestTagMatrix' ./...`.
If those pass, the wire is unchanged and you are unlikely to have
broken downstream consumers.

---

## Further reading

- README — high-level overview and quick start.
- `docs/BENCH.md` — full benchmark matrix and methodology.
- `docs/CHOOSING.md` — opts cheatsheet for end-users.
- `example_*_test.go` files at the repo root — pkg.go.dev surfaces
  these per-symbol. Cover Marshal / AppendMarshal / MarshalT
  basics, OptSpeed-vs-OptBalanced wire size, low-level Encoder
  with PreIntern, StreamEncoder + StreamDecoder, custom
  Marshaler / Unmarshaler, and Decoder fast-paths (SetNoCopy,
  PeekTag, IsNil).
- mcyoung, "Cheating the Reaper in Go" — background reading on the
  arena allocator tricks (uintptr cursor, chunk-keep slice,
  Reset-keeps-chunks).
- protobuf-go, `encoding/protowire` — reference for the varint
  encoding qdf inherits.
