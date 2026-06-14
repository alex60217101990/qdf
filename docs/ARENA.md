# Decode Arena (`WithArena`)

A caller-owned bump allocator that collapses the per-string allocations of a
decode into a handful of dense blocks. Opt-in, safe by default, and adaptive —
it helps in proportion to how many strings a payload carries and never makes a
string-poor payload slower.

For the general decode story see [`GUIDE.md`](GUIDE.md); for the zero-copy
alternative see [`WithNoCopy`](#arena-vs-withnocopy) below.

---

## Why it exists

Decoding is **allocation-bound, not CPU-bound.** Profile any string-heavy
`Unmarshal` and ~96 % of the allocations come from one place: copying each
string/`[]byte` body out of the wire buffer into an owned Go `string`. The
codec/tag parsing is a rounding error next to it.

The default decode is already at its floor for *one* string — `string(b)`
allocates exactly `len` bytes (un-zeroed) and `memmove`s into them; you cannot
beat that for a single owned string. The waste is in the **count**: a record
with seven string fields pays seven separate heap allocations, and the GC then
scans seven objects.

The arena removes the count. Every copied string is bump-appended into one
contiguous block:

```
default:   [str0] [str1] [str2] ...        7 allocations, 7 GC objects
arena:     [str0|str1|str2|...]            ~0 allocations (amortized), 1 GC object
```

The strings are byte-for-byte the same; only *where they live* changes. Across
an epoch of many decodes the block allocation amortizes to near zero, the
strings sit next to each other (cache-friendly), and the GC walks one block
instead of N strings.

> **Why not automatic?** A per-message arena (allocated and thrown away every
> decode) measurably *regresses* under parallel load — the per-decode block
> allocation costs more than the small string allocations it replaces, because
> Go's tiny allocator already packs short strings efficiently. The win only
> appears when the block is **amortized across an epoch**, and only the caller
> knows where an epoch ends. So the arena is caller-owned and explicit.

---

## When to use it

| Use an arena when… | Skip it when… |
| --- | --- |
| You decode **many** messages in a loop (batch, stream, request handler) | A single one-shot `Unmarshal` with nothing to amortize over |
| Records are **string-heavy** (logs, traces, directory/AD exports, events) | Records are numeric-heavy (metric series) — it won't hurt, but the win is small |
| You want the wire buffer back in a pool **immediately** after decode | You already use `WithNoCopy` on a long-lived/mmap'd buffer (that's zero-copy, even cheaper) |
| You can scope an epoch (a request, a batch) and reset/drop the arena at its end | You can't reason about when the decoded strings die |

The win scales with string density and is never negative:

| Corpus | Shape | time | allocs/op (off → on) |
| --- | --- | ---: | ---: |
| **Telemetry log batch** (1 000 events) | string-heavy | **−35 %** | 3 502 → **3** |
| **AD / directory export** (200 users, 11 string fields + map) | string-heavy | **−26 %** | 4 856 → **605** |
| **Event batch** (500, `[]byte` payload + 1 string) | mixed | **−13 %** | 1 003 → **504** |
| **IoT sensor batch** (numeric series + map tags) | numeric-heavy | **−4 %** | 291 → **164** |

*Darwin amd64 / i7-9750H, Go 1.26, reflect decode path, epoch loop with
`Reset` per message. Codegen path (`UnmarshalerArena`): −11 % seq / −17 %
parallel, allocs 8 → 2.*

The same win applies to the **Dense and Compression** tiers, not just Speed —
the interned first-sight string copies are arena-backed too:

| Corpus (Balanced/Dense wire) | time | allocs/op (off → on) |
| --- | ---: | ---: |
| **Telemetry log batch** (500 events) | **−31 %** | 3 502 → **3** |
| **AD / directory export** (200 users) | **−21 %** | 3 006 → **606** |
| **IoT sensor batch** (numeric series + map tags) | neutral | 213 → **168** |

A Dense decode now reaches the same arena allocation floor as Speed; before, it
stayed near its no-arena allocation count because the intern materialisations
bypassed the arena.

---

## API

```go
type Arena struct{ /* opaque */ }

func NewArena() *Arena                       // construct
func (a *Arena) Reset()                      // rewind for reuse (see safety)

func WithArena(a *Arena) QueryOption         // pass to Unmarshal
func (d *Decoder) SetArena(a *Arena)         // or to a Decoder/StreamDecoder you drive
```

A nil arena (or none) means "decode normally" — the default copying path is
completely unchanged.

---

## Usage

### Pattern 1 — one arena per epoch, then drop it (safe, recommended)

The arena lives exactly as long as the decoded values. The garbage collector
frees it for you once those values die — a returned string keeps its block
alive through an interior pointer, so you never track the buffer by hand.

```go
func handle(messages [][]byte) []Event {
    a := qdf.NewArena()              // one arena for this batch
    out := make([]Event, len(messages))
    for i, msg := range messages {
        if err := qdf.Unmarshal(msg, &out[i], qdf.WithArena(a)); err != nil {
            return nil
        }
    }
    return out                       // out's strings live in `a`; GC frees it
                                     // when `out` is dropped. Nothing to free.
}
```

This is the safe default: **no `Reset`, no lifetime bookkeeping.** Allocate an
`Arena`, use it for one epoch's worth of values, let it go out of scope.

### Pattern 2 — reuse one arena across epochs with `Reset` (max performance)

If you process an unbounded stream and want **zero** steady-state allocation,
reuse a single arena and `Reset` it at each epoch boundary. `Reset` rewinds the
bump cursor and reuses the same block — no allocation at all after warm-up.

```go
a := qdf.NewArena()
for {
    msg, ok := next()
    if !ok { break }

    a.Reset()                        // ⚠ invalidates every string from the
                                     //   PREVIOUS iteration — see safety
    var ev Event
    if err := qdf.Unmarshal(msg, &ev, qdf.WithArena(a)); err != nil {
        return err
    }
    process(ev)                      // ev valid here; done before next Reset
}
```

This is the fastest mode but carries the lifetime contract below.

### Pattern 3 — driving a `Decoder`/`StreamDecoder` directly

```go
d := qdf.NewDecoderOnBuf(buf)
d.SetArena(a)
// ... typed Read* / generated UnmarshalQDFArena ...
```

---

## Safety / lifetime contract

> **Strings returned by an arena decode ALIAS the arena's memory.** They are
> valid only while the arena's blocks are intact.

Two rules, one of which the compiler and race detector **cannot** enforce:

1. **Drop-to-free is always safe.** If you never call `Reset`, the arena and
   its blocks are ordinary GC-managed memory. A decoded string keeps its block
   alive; when the last value dies, the GC reclaims the block. You cannot
   create a dangling pointer this way. This is Pattern 1.

2. **`Reset` is a manual use-after-free contract.** `Reset` rewinds the cursor
   so the next decode overwrites the same memory. Any string returned *before*
   the `Reset` then silently points at reused bytes. Call `Reset` **only once
   every value decoded since the last `Reset` (or since `NewArena`) is dead.**
   The race detector will not catch a violation — it is manual memory, not a
   data race. If you are unsure, drop the arena and `NewArena` again (Pattern 1).

Other guarantees:

- **`[]byte` fields are never arena-backed.** A decoded `[]byte` is
  caller-mutable; sharing a block between a mutable slice and neighbouring
  strings would let a write corrupt them. `[]byte` always gets its own copy,
  arena or not.
- **An `Arena` is not safe for concurrent use.** Give each goroutine its own
  (`Arena` is cheap — a few words). The benchmarks above use one arena per
  `RunParallel` goroutine.
- **Values are byte-identical** to a normal decode — only their backing storage
  differs. `reflect.DeepEqual(arenaDecoded, plainDecoded)` holds for structs,
  maps, nested slices, and `[]byte` alike.

---

## How it composes

- **Codegen types.** `cmd/qdfgen` emits `UnmarshalQDFArena(src, noCopy, a)`
  alongside `UnmarshalQDF`/`UnmarshalQDFOpts` (which now delegate to it with a
  nil arena). `Unmarshal(buf, &v, WithArena(a))` threads the arena into a
  generated type and through its **nested** fields via `UnmarshalNestedArena`.
  External `Unmarshaler` types without the `UnmarshalerArena` method simply fall
  back to a copying decode — fully backward compatible.
- **Interned / repeated strings (Dense / Compression).** Under
  `OptDense`/`OptCompression` the encoder interns recurring fields and the wire
  carries `tagInternStr` (first sight) + `tagStateRef`/`tagStateRepeat`
  (references). The decoder materialises each interned id into an owned string
  **exactly once** and caches it; every later reference reuses that cached string
  with no re-copy. When an arena is attached, that one first-sight
  materialisation is bump-appended into the arena like any other copy — so a
  Dense or Compression decode amortises its string copies the same way a
  Speed-mode decode of inline strings does. Interning is **not** defeated: the
  per-id copy still happens at most once; the arena only changes *where* that
  single copy lives. (Earlier the intern path took its own `string(b)` heap
  copy and bypassed the arena entirely, so Dense/Compression decodes — the tiers
  you run for wire size — saw no arena benefit; that gap is now closed.)
- **`WithNoCopy`.** If both are set, `WithNoCopy` wins: aliasing the input skips
  the copy entirely, so there is nothing for the arena to do.

---

## Arena vs `WithNoCopy`

Both cut the decode copy; they trade off differently.

| | `WithArena` | `WithNoCopy` |
| --- | --- | --- |
| Copies | one amortized block per epoch | **zero** (strings alias the input) |
| Holds in memory | only the string bytes, packed | the **whole** input buffer (tags + numbers + strings) |
| Wire buffer reusable right after decode? | **yes** (strings copied out) | **no** (strings alias it) |
| Safe with a pooled / recycled wire buffer? | **yes** | **no** — classic use-after-free |
| Best for | server handlers that pool the read buffer; batch/stream decode | mmap'd or long-lived caller-owned input |

Rule of thumb: if the wire buffer is **recycled** (a `sync.Pool` request body),
use `WithArena` — it copies the strings out so the buffer can go back to the
pool immediately. If the wire buffer is **long-lived and owned** (an mmap'd
file), `WithNoCopy` is strictly cheaper. See the `WithNoCopy` godoc for its full
contract.

---

## Internals

The bump core is [`internal/bumparena`](../internal/bumparena/bumparena.go):

- **Geometric block growth** (like a `Vec`/`std::vector`): 512 B first block,
  doubling up to a 64 KiB cap — few allocations on a large epoch, little waste
  on a small one. One block (and thus the retention of any single returned
  string) is bounded by the cap.
- **No-zero allocation** (`internal/unsafestr.DirtBytes`, `mallocgc` with
  `needzero=false`): the copy overwrites exactly what is exposed, so zeroing the
  block first is pure waste. (Go's own `string(b)` allocates the same way.)
- **Hot/cold split**: the fast path (value fits the current block) is tiny and
  inlines into the decoder; the rare block allocation is out of line.

It deliberately does **not** carry the encoder's interning id-table (see
`internal/internarena`): a decode returns strings directly and never looks a
value up by id, so the id-table would be pure overhead.
