# Structural delta — `Diff` / `Apply`

`Diff` computes the structural difference between two values of the same type
and `Apply` merges that difference back onto a base. The patch carries the new
value of every location that changed and nothing else, so when one field of a
large record set moves you ship a few dozen bytes instead of re-encoding the
whole thing.

```go
patch, _ := qdf.Diff(old, updated, qdf.OptBalanced) // only the changes
_ = qdf.Apply(&base, patch)                          // base == updated, in place
```

This is the primitive behind any *state synchronisation* workload: Kubernetes-style
resource reconciliation, config hot-reload (ship the change, not the whole
config), database change-data-capture and event sourcing, realtime / game
netcode (send only the entities that moved), and live dashboards. It works
straight from the two values, type-generically — no IDL, no `FieldMask` to
maintain by hand, no diff code to write per type.

---

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

type Config struct {
	Replicas int               `qdf:"replicas"`
	Image    string            `qdf:"image"`
	Env      map[string]string `qdf:"env"`
	Args     []string          `qdf:"args"`
}

func main() {
	old := Config{
		Replicas: 3,
		Image:    "app:v1",
		Env:      map[string]string{"LOG": "info", "REGION": "eu"},
		Args:     []string{"--serve"},
	}
	// Only Replicas and one Env key change.
	updated := old
	updated.Replicas = 5
	updated.Env = map[string]string{"LOG": "debug", "REGION": "eu"}

	patch, err := qdf.Diff(old, updated, qdf.OptBalanced)
	if err != nil {
		panic(err)
	}
	fmt.Printf("patch: %d bytes\n", len(patch)) // tiny — only the changes

	// Apply merges the patch onto a base IN PLACE, reconstructing `updated`.
	base := old
	if err := qdf.Apply(&base, patch); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", base) // == updated
}
```

The patch is self-describing. The receiver hands the bytes straight to `Apply`
and never has to be told which `Options` the producer used.

---

## API

```go
func Diff[T any](old, new T, opts Options) ([]byte, error)
func AppendDiff[T any](dst []byte, old, new T, opts Options) ([]byte, error)
func Apply[T any](base *T, patch []byte) error
func ApplyArena[T any](base *T, patch []byte, arena *Arena) error
```

`Diff` returns a patch carrying only the difference `new − old`. `AppendDiff`
does the same but appends to a caller-owned `dst`, so a hot loop can reuse one
buffer with no per-call allocation for the output. `Apply` merges a patch onto
`*base`, leaving every location the patch does not mention exactly as it was.
`ApplyArena` is `Apply` with a decode arena for patches that replace many
strings — see [Batching replaced strings](#batching-replaced-strings-with-an-arena).

`opts` is the same `Options` bit-mask `Marshal` takes — `OptSpeed`,
`OptBalanced`, `OptCompression`, and the modifiers. The tier only affects how
individual replaced values are codec-encoded and whether the optional rANS pass
runs over the patch body; the patch's op-level wire is identical across tiers.

> **`Apply` mutates `*base` in place.** It overwrites the changed locations and
> leaves the rest untouched — it does not allocate a fresh value. If you need to
> keep the original around, clone it first (a struct copy plus a copy of any
> slice/map fields) and apply onto the clone.

### Errors

| Error                    | When |
| ------------------------ | ---- |
| `ErrInvalidPatch`        | The patch is truncated, has a bad magic/version, or is otherwise malformed. |
| `ErrPatchSchemaMismatch` | The patch was built for a different type than the supplied base. |
| `ErrPatchBaseMismatch`   | The supplied base does not match the `old` the patch was computed against. |
| `ErrCycleDetected`       | `Diff` was handed a cyclic or pathologically deep value. It is rejected, not crashed. |

---

## What gets diffed

`Diff` covers the same type surface as the rest of qdf:

- **Structs**, including nested and embedded structs. A struct is diffed
  field-by-field; only changed fields land in the patch.
- **Scalars** — every integer / unsigned / float / bool width, plus `string`,
  `[]byte`, and `[N]byte` fixed-byte arrays.
- **Slices and arrays** — diffed positionally. Element `i` of `new` is compared
  against element `i` of `old`.
- **Maps** — diffed per key. Added or changed keys carry their new value;
  removed keys carry a tombstone.
- **Pointers** — a presence change (nil ↔ non-nil) replaces; two non-nil
  pointers recurse into what they point at.

The `nil`-vs-empty distinction is preserved: a transition between `nil` and an
empty non-nil slice/map/`[]byte` (think `null` vs `[]` in JSON) is a real change
and ships as a whole-value replace, never silently collapsed.

A changed location is always expressed at the finest granularity that still
shrinks the patch. An unchanged field, element, or key is omitted entirely — it
costs no bytes. A scalar change, a presence change, or a `nil`-vs-empty
transition ships the whole new value with the normal value codec. A nested
struct, or two non-nil pointer/map/slice values, recurses and carries a
sub-patch describing only *its* changed locations.

The core rule on the apply side is the mirror image: **absent means unchanged.**
If a location is not listed, `Apply` leaves the corresponding location in `base`
exactly as it found it.

---

## Deletion

| Container | How a removed element is expressed |
| --------- | ---------------------------------- |
| **map**   | A tombstone in the patch's delete set; `Apply` deletes the key from `base`'s map. |
| **slice** | A positional shrink: the patch carries the new (shorter) length and `Apply` truncates. |
| **array** | Arrays are fixed-length, so there is no deletion; a removed value is a zero/nil replace at its index. |
| **struct field** | Struct shape is fixed; a "removed" value is just a zero/nil replace. |

Slices are matched positionally, so deleting an element from the **middle** is
expressed as positional shifts of the tail plus a truncate. The result is
correct, but it reships the shifted tail, because every index past the deletion
now holds a different value. Deleting or appending at the **end** is cheap;
inserting or deleting in the middle is not. See
[Limitations](#limitations) for the keyed-diff alternative that would avoid this.

---

## Wire format

A patch is not a normal qdf blob — it carries its own magic so a patch can never
be mistaken for a full value or vice versa.

```
+-----+-----+-----+-----+-----+--------- 8 ---------+----- 8 (optional) -----+
| 'Q' | 'D' | 'P' | ver |flags|       schemaFP      |   baseFP (if flag set) |
+-----+-----+-----+-----+-----+---------------------+------------------------+
| body — the root patch (rANS-framed iff the rANS flag is set)              |
+--------------------------------------------------------------------------+
```

- 3-byte magic `'Q' 'D' 'P'` plus a 1-byte version.
- 1 flag byte: *dense* (field names interned in the body), *rANS* (body is
  rANS-compressed), *baseFP present*.
- An 8-byte little-endian **schemaFP** (always present).
- An optional 8-byte little-endian **baseFP** (present unless you opt out).

Every changed location is one op byte plus a payload. The op is either a
*replace* (the whole new value, encoded with the normal qdf value codec) or a
*merge* (a recursive sub-patch). The three container sub-patches are:

| Sub-patch  | Layout |
| ---------- | ------ |
| struct     | `varuint(nChanged)`, then `nChanged ×` `{varuint(fieldIdx), op}` — sparse, only changed fields. |
| slice      | `varuint(newLen)`, `varuint(nEntries)`, then `nEntries ×` `{varuint(idx), op}` — positional; `newLen` grows or shrinks the slice. |
| map        | `varuint(nUpdate)`, `nUpdate ×` `{key, op}` updates, then `varuint(nDelete)`, `nDelete ×` `{key}` tombstones. |

---

## Safety

`Apply` will not silently corrupt a value applied against the wrong type or the
wrong base. Two fingerprints guard it.

The **schema fingerprint** (always present) is a hash of the type's shape —
kind, field names, and recursive field/element kinds. `Apply` recomputes it from
the target type and rejects a patch built for a different type with
`ErrPatchSchemaMismatch`. A patch can never be mis-parsed as if it were for
another struct.

The **base fingerprint** (present by default) is an order-independent hash of
`old`; map-bearing values fingerprint deterministically regardless of map
iteration order. `Apply` recomputes it over the supplied base and rejects a base
that has diverged from the original `old` with `ErrPatchBaseMismatch`. This is
what turns "applied a patch onto the wrong base" from a corrupt merge into a
loud error.

The patch body is self-contained and bounded. Every length-prefixed read is
bounded by the input, so a hostile or truncated patch returns an error — it
never panics and never OOMs. Under `OptCompression` an order-0 rANS entropy pass
runs *after* the structural diff, over the already-minimal patch body, and only
when it actually shrinks (never larger). The diff does the structural work; rANS
squeezes whatever byte entropy is left.

The path is covered by a round-trip oracle fuzzer
(`Apply(base, Diff(old, new)) == new`) and a hostile-patch fuzzer that feeds
malformed bytes to `Apply`, both run into the millions of executions.

---

## Performance

`Diff` costs CPU to compute, but you ship a fraction of the bytes — exactly the
trade a state-sync or CDC workload wants. Measured on `BenchmarkDiffApply`
(Intel i7-9750H, Go 1.26): a 1000-record slice in which a **single field of a
single element** changes.

| Operation         | time      | output size |
| ----------------- | --------: | ----------: |
| `Diff`            | ~0.51 ms  | **42 B**    |
| `Apply`           | ~0.25 ms  | —           |
| full `Marshal`    | ~0.20 ms  | **26 677 B**|

The patch is 42 bytes against the 26 677-byte full marshal — about **635×
smaller** for one changed field. `Apply` only touches the locations the patch
mentions; the rest of the base is left in place, untouched.

### Skipping the base fingerprint

By default `Diff` embeds a fingerprint of `old` in the patch and `Apply` walks
the supplied base to recompute and verify it. That walk is what catches a
wrong base — and on a large base with a tiny patch it dominates `Apply`'s cost,
because you pay a full reflect walk of the whole value to validate a few changed
bytes.

`OptDeltaNoBaseFingerprint`, set in the `Diff` / `AppendDiff` opts, omits the
fingerprint and skips the check on both sides:

```go
patch, _ := qdf.Diff(old, updated, qdf.OptBalanced|qdf.OptDeltaNoBaseFingerprint)
// ... ship patch ...
_ = qdf.Apply(&base, patch) // no base walk, no ErrPatchBaseMismatch possible
```

In our benchmark this takes `Apply` from ~0.25 ms to ~0.02 ms — roughly **10×**
faster — because `Apply` no longer reflect-walks the large base. The cost is the
safety net: with no fingerprint, applying onto a divergent base silently merges
onto the wrong value instead of returning `ErrPatchBaseMismatch`. Use it only in
trusted pipelines where the caller *guarantees* `Apply`'s base is exactly the
value `Diff` was computed against (a single-writer cache, a CDC stream you
control end-to-end). When in doubt, leave it off — the schema fingerprint is
always present regardless, so a wrong *type* is still rejected.

### Batching replaced strings with an arena

`Apply` decodes each string the patch replaces into its own heap allocation, so a
patch that replaces many strings — a large changed `[]string` or
`[]struct{…string…}` field, or a map with many changed string values — costs one
allocation per string. `ApplyArena` copies those replaced strings into the
contiguous bump blocks of a caller-provided `Arena` instead:

```go
arena := qdf.NewArena()
err := qdf.ApplyArena(&base, patch, arena)
// base's replaced string fields now alias arena's memory; keep arena
// reachable for as long as you use base. Call arena.Reset() to reuse it
// for the next epoch (which invalidates strings from the previous one).
```

In our benchmark, applying a patch that rewrites every string in a 1000-row
slice drops from ~3000 allocations to a handful (≈ −99%) and ~25% wall-clock.
The result, errors, and wire are identical to `Apply` — only where the replaced
string bytes live differs.

This helps **only** when a patch replaces many strings; a typical small patch
changes few, and `Apply` is the simpler choice. The lifetime rule is the same as
the decode arena: strings written into `base` alias the arena and stay valid only
while it is reachable, so use one arena per epoch and drop it when the values
built from it are done. Unchanged string fields already in `base` are not touched
and never alias the arena.

### The `qdf_reflect2` build tag

`Apply` allocates new slices and maps through the same helper the rest of qdf
uses on decode. Building with `-tags qdf_reflect2` swaps that helper's backend
to [modern-go/reflect2](https://github.com/modern-go/reflect2), which skips
reflect's per-call type checks — so a patch that **creates** a map or grows a
slice allocates faster. Diff is unaffected (it encodes, it does not allocate
destination containers), and the default build (without the tag) is unchanged.
This only matters for `Apply`/`ApplyArena` on patches that add map keys to a
previously-nil map or grow slices; for small in-place patches the difference is
negligible.

---

## Keyed slices

By default slices are matched **positionally**: reordering a slice, or inserting
or deleting in the middle, reships the shifted tail. For slices whose elements
have a stable identity, tag the identity field with `,key` on its `qdf` tag and
the slice is matched by that key instead:

	type Entity struct {
	    ID   string `qdf:"id,key"`
	    X, Y float64
	}

Now a `[]Entity` patch matches elements by `ID`. Reordering the slice ships only
the new key order (no element values); inserting, deleting, or moving an element
touches just that element — where the positional diff would reship everything
after the change. Value-only edits that keep the same order ship no order list at
all. In a benchmark, a full reverse of a 200-element slice produced a keyed patch
~30% smaller than a full re-encode (and far smaller than a positional diff).

The key field must be comparable — a scalar, string, or `[N]byte`. If keys are
not unique, the diff transparently falls back to the positional path (still
correct). One `,key` field per element type.

## Columnar column-level diff

The positional diff walks a `[]struct` element by element: a changed row ships
the whole element's per-field patch. For a large batch where only a few
*columns* move — telemetry, metrics, event rows — that scatters the changes
across every touched element. When the diff is **equal-length** (same number of
elements old and new) and the element is **pure-columnar** (every field is a
column the columnar encoder understands, with no map/slice/nested residual), at
least `16` elements, the diff instead groups the changes **by column** and emits
a compact column patch.

Each changed column is encoded independently, in whichever of three modes is
smallest for that column:

- **sparse** — the changed row indices (gap-encoded, ascending) plus the new
  values of just those cells, for a column touched in few rows;
- **arithmetic-delta** — a full-length per-row delta added back on apply, for a
  numeric (or bool) column that moved in most rows but by small amounts;
- **dense-whole** — the entire new column re-shipped, when neither of the above
  is cheaper.

The per-column bodies reuse the existing column codecs — FOR / delta / RLE /
dictionary for numerics, the dictionary-or-raw bulk path for strings, the
sec/nsec sub-columns for `time.Time` — so each column is packed as tightly as a
fresh columnar encode would pack it.

This needs **no flag**: it is chosen automatically, and only when the resulting
column patch is **smaller** than the positional patch for the same change
(never-larger — if the changes are scattered across many columns, the positional
patch wins and is used). Apply mirrors it, scattering each decoded column back
into its rows.

v1 covers pure-columnar elements only — a **hybrid** element (one carrying a
`map` or non-columnar field) always uses the positional path — and every column
kind except **nullable string** (`*string`), which also falls back to
positional. Float columns never use arithmetic delta (floating-point subtraction
does not round-trip exactly), so they pick sparse or dense-whole.

## Baseline registry

`Apply[T](base *T, patch)` requires the caller to hold the exact `old` value the
patch was diffed against. In a state-sync stream — a server pushing successive
deltas of an evolving value to clients — the consumer would have to remember the
previous state to apply the next patch. `BaselineRegistry[T]` does that
bookkeeping: it keeps a small content-addressed cache of recent baselines and
resolves a patch's base from it, so the consumer applies a chain of patches
without threading `old` by hand.

	// NewBaselineRegistry returns an empty registry for baselines of type T.
	func NewBaselineRegistry[T any]() *BaselineRegistry[T]

	// Register stores *v as a resolvable baseline and returns its content id.
	func (r *BaselineRegistry[T]) Register(v *T) uint64

	// Apply resolves the patch's baseline by its embedded fingerprint, applies
	// the patch onto a fresh deep copy, registers the result, and returns it.
	func (r *BaselineRegistry[T]) Apply(patch []byte) (*T, error)

	// Len reports the number of live baselines (prunes dead entries).
	func (r *BaselineRegistry[T]) Len() int

**No wire change.** A patch already carries `baseFP`, an 8-byte content hash of
`old` (on by default; see *Skipping the base fingerprint*). Phase 1 uses it only
to *reject* a mismatched base; the registry reuses the very same `baseFP` as a
*lookup key*. The producer ships an ordinary `Diff` — there is no new tag, flag,
or field. The registry is a consumer-side construct layered on the existing
format.

	// producer holds prev; ships an ordinary diff (baseFP embedded by default)
	patch, _ := qdf.Diff(prev, cur, qdf.OptBalanced)

	// consumer
	reg := qdf.NewBaselineRegistry[State]()
	s0 := &decodedFullState        // a *State the consumer decoded and keeps alive
	reg.Register(s0)               // bootstrap from a full value
	s1, err := reg.Apply(patch)    // resolves s0 by baseFP, applies, returns *s1
	// keep s1 to apply the next patch in the chain; drop it to let the GC reclaim it.

**Non-pinning lifetime.** The registry holds each baseline through a
`weak.Pointer`, never a strong reference, so the GC reclaims any baseline the
**caller** has dropped — zero leak, automatic reclamation. The flip side is the
caller's contract: a baseline stays resolvable only while the caller keeps a
strong reference to it. In a chain, keep the previous `*T` reachable across the
`Apply` that consumes it (the returned pointer is the natural anchor for the next
step). Resolving a baseline the caller has dropped is a clean miss
(`ErrBaselineEvicted`); recover by re-syncing a full value.

`Apply` **deep-clones** the resolved baseline before applying — the registry
keeps the original intact, since another patch may branch off it — and
auto-registers the result so it can base the next patch. The clone preserves
nil-vs-empty exactly, so its fingerprint matches the original's; the Phase 1
`baseFP` check is the backstop that turns any clone discrepancy into an error,
never silent corruption.

`OptDeltaNoBaseFingerprint` is incompatible with the registry: such a patch
carries no id, so `Apply` returns `ErrBaselineRequired`. Baselines are addressed
by a 64-bit fingerprint; two distinct values sharing one (probability ~N²/2⁶⁴,
negligible for thousands of live baselines) collide on the same key, the later
registration wins, and the earlier baseline then reads as `ErrBaselineEvicted`.

The registry is safe for concurrent `Register` / `Apply` / `Len`.

## Limitations

A few things are not currently supported and may come later:

- **Column-level diff for `[]struct`** — when a slice of flat structs is stored
  columnar, shipping only the columns that changed instead of per-element ops.

A few things are not currently supported and may come later:

- **Column-level diff for `[]struct`** — when a slice of flat structs is stored
  columnar, shipping only the columns that changed instead of per-element ops.
- **Content-addressed baselines** — addressing the base by a content hash so a
  patch can be applied against a baseline fetched from a store, rather than the
  caller holding `old`.
