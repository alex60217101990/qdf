# Decode performance: the four levers

Decode is allocation-bound, not CPU-bound: on a string-heavy payload ~96 % of
the allocations are string/`[]byte` bodies copied out of the wire buffer. The
default decode copies (safe, owned results) and is already at its floor for that
contract. Four opt-in levers cut the copies further. **None changes the default
behavior** — you reach for them per call / per type / per field.

Pick by how your data and your buffers are shaped:

| Lever | Cuts | Cost / when |
| --- | --- | --- |
| Decode fewer fields | the copy of every field you don't ask for | free — just declare a smaller struct |
| `[N]byte` for IDs | wire size + the string alloc for fixed-width ids | free — change the field type |
| `WithNoCopy` | every copy (aliases the input) | input must outlive the result and not be reused |
| `WithArena` | per-string allocs across an epoch | caller scopes an arena (see [ARENA.md](ARENA.md)) |

---

## 1. Decode fewer fields (subset-struct projection)

The decoder skips any wire field that the target struct does not declare — it
advances past the value without copying it. So to read a subset of a record,
decode into a struct that contains **only the fields you want**:

```go
// Wire was produced from a 7-field LogEntry. Read just two of them:
type LogView struct {
    Level string `qdf:"level"`
    Trace string `qdf:"trace"`
}

var v LogView
_ = qdf.Unmarshal(buf, &v)   // svc/host/span/msg/... are Skip()'d, never copied
```

Measured (7-field record): full decode **7 allocs**, the 2-field view **3
allocs** — the four unread string fields cost nothing. Works on the reflect path
and on codegen types (the generated decoder `Skip()`s unknown keys). The wire is
unchanged; this is purely a decode-side choice of target type.

Use it whenever a consumer only needs some columns (routing on one field,
filtering, projecting for a downstream system). It is the row-major analog of
[`Select` / predicate pushdown](PREDICATE-PUSHDOWN.md) for columnar payloads.

---

## 2. Fixed-width ids as `[N]byte`

A trace id / span id / UUID / hash is fixed-width binary. Storing it as a hex
**string** doubles the data (32 hex chars for 16 bytes) and costs a string alloc
on decode. Declaring the field as a fixed byte array stores the raw bytes and
decodes in place:

```go
type Span struct {
    TraceID [16]byte `qdf:"trace_id"`   // 16 raw bytes, not 32 hex chars
    SpanID  [8]byte  `qdf:"span_id"`
}
```

A `[N]byte` field encodes as one flat binary blob (identical wire to a `[]byte`
of length N) and decodes with a single memcpy straight into the struct's inline
array — **zero allocation**. Versus a 32-char hex string for the same id: wire
**43 → 27 bytes** on the record, and no decoded-string alloc. Type-driven, works
in every mode, no flag.

(If you must keep the id as a Go `string`, it is already wire-optimal — a short
string is a 1-byte-tag blob. The win here is choosing raw bytes over hex text,
not a different encoding.)

---

## 3. Zero-copy decode (`WithNoCopy`)

`WithNoCopy()` returns string/`[]byte` values that **alias the input buffer**
instead of copying — near-zero allocations, roughly 2× faster on string-heavy
batches.

```go
// data must outlive `out` and must NOT be reused/mutated while `out` is in use.
var out LogBatch
_ = qdf.Unmarshal(data, &out, qdf.WithNoCopy())
```

Measured on a 1000-row string-heavy batch: **7002 → 3 allocs/op, ~−38 % B/op,
~1.8× faster** (411 µs → 226 µs).

> ⚠️ The decoded values point INTO `data`. Safe when the input is **owned and
> not recycled** — an mmap, a file read fully into memory, or a buffer you
> allocate fresh per message and let live as long as the decoded value (the GC
> keeps it alive through the aliasing headers). UNSAFE on a pooled / reused
> buffer (e.g. a `sync.Pool` request body): the aliases dangle the moment the
> buffer is reused — a use-after-free the race detector cannot catch. This is
> why it is opt-in.

For a `Decoder`/`StreamDecoder` you drive yourself, use `SetNoCopy(true)`.

---

## 4. Decode arena (`WithArena`)

When the wire buffer is **recycled** (so `WithNoCopy` is unsafe) but you still
want to cut per-string allocations, pass a [`qdf.Arena`](ARENA.md): it copies
the strings out (so the buffer can return to its pool immediately) but into one
dense bump block per epoch instead of one allocation per string.

```go
a := qdf.NewArena()
for i, msg := range batch {
    _ = qdf.Unmarshal(msg, &out[i], qdf.WithArena(a))
}
// out's strings live in `a`; the GC frees it when out is dropped.
```

Measured (telemetry log batch ×1000, epoch loop): **−35 % time, allocs
3502 → 3**. Safe by default (owned arena, GC-managed); see
[`docs/ARENA.md`](ARENA.md) for the full lifetime contract and the off→on tables
across realistic corpora.

---

## Combining

These compose: a subset struct with `[N]byte` id fields, decoded with
`WithNoCopy` (owned input) or `WithArena` (pooled input), reads the minimum data
with the minimum copies. The default — plain `Unmarshal` into the full struct —
stays safe and unchanged when you reach for none of them.
