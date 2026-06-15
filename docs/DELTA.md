# Structural delta — `Diff` / `Apply`

qdf can compute a **structural delta** between two values of the same type and
ship only what changed. `Diff(old, new)` walks both values field-by-field,
element-by-element, key-by-key and emits a self-describing patch that carries
the *new* value of every location that differs and nothing else; `Apply(base,
patch)` merges that patch back onto a base value in place to reconstruct `new`.
This is the missing primitive for **state synchronisation** — Kubernetes-style
resource reconciliation, config hot-reload (ship the change, not the whole
config), database change-data-capture and event sourcing, realtime / game
netcode (send only the entities that moved), and live dashboards or
collaborative editing. No mainstream Go wire format — protobuf, msgpack,
flatbuffers, capnproto, gob — computes a structural delta natively;
protobuf's `FieldMask` is caller-specified, not computed. qdf does it from the
two values, type-generically, with no IDL and no hand-rolled diff code.

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

The patch is self-describing: the receiver passes the bytes straight to
`Apply` without being told which `Options` the producer used.

---

## API reference

```go
func Diff[T any](old, new T, opts Options) ([]byte, error)
func AppendDiff[T any](dst []byte, old, new T, opts Options) ([]byte, error)
func Apply[T any](base *T, patch []byte) error
```

| Function     | Purpose |
| ------------ | ------- |
| `Diff`       | Compute a patch carrying only the structural difference `new − old`. |
| `AppendDiff` | Same, appending to a caller-owned `dst` (zero extra allocation for the output buffer). |
| `Apply`      | Merge `patch` onto `*base` **in place**, reconstructing `new`. Locations the patch marks unchanged are left exactly as they were in `base`. |

`opts` is the same `Options` bit-mask as `Marshal`: `OptSpeed` / `OptBalanced`
/ `OptCompression` (the latter enables the optional rANS post-pass over the
patch body, see [Safety](#safety)). The `op`-level wire is the same across
tiers; the tier only affects how individual replaced values are codec-encoded
and whether rANS runs.

Errors:

| Error                    | Meaning |
| ------------------------ | ------- |
| `ErrInvalidPatch`        | The patch blob is truncated, has a bad magic / version, or is otherwise malformed. |
| `ErrPatchSchemaMismatch` | The patch was built for a different type than the supplied base (see [Safety](#safety)). |
| `ErrPatchBaseMismatch`   | The patch carries a base fingerprint that does not match the supplied base value — the base has diverged from the original `old`. |

> ⚠️ **`Apply` mutates `*base` in place.** It overwrites the changed fields of
> `base` and leaves the rest untouched — it does not allocate a fresh value. If
> you need to keep the original `base` around, **clone it first** (e.g. a struct
> copy plus a copy of any slice/map fields) and apply onto the clone.

---

## What gets diffed

`Diff` supports the same type surface as the rest of qdf:

- **Structs**, including arbitrarily **nested** structs. A struct is diffed
  field-by-field; only changed fields appear in the patch (a sparse
  `tagStructPatch`).
- **Scalars** — every integer / unsigned / float / bool width, plus `string`,
  `[]byte`, and `[N]byte` fixed-byte arrays.
- **Slices and arrays** — diffed **positionally** (index-aligned). Element `i`
  of `new` is compared against element `i` of `old`.
- **Maps** — diffed **per key**: added/changed keys carry their new value,
  removed keys carry a tombstone.

### Granularity tiers

A changed location is expressed at the **finest granularity that still
shrinks the patch**:

| Tier | What happens |
| ---- | ------------ |
| **absent** | A field/element/key that did not change is omitted entirely (no op byte). |
| **`opReplace`** | A scalar change, a presence change (nil ↔ non-nil), or a nil-vs-empty transition ships the whole new value with the normal value codec. |
| **`opMerge`** | A nested struct, or a both-non-nil pointer / map / slice, recurses: the patch carries a sub-patch describing only *its* changed locations. |

---

## Wire format

A patch is **not** a normal qdf blob. It carries its own magic so a patch can
never be mistaken for a full value or vice versa:

```
+-----+-----+-----+-----+-----+--------- 8 ---------+----- 8 (optional) -----+
| 'Q' | 'D' | 'P' | ver |flags|       schemaFP      |   baseFP (if flag set) |
+-----+-----+-----+-----+-----+---------------------+------------------------+
| body — the root patch (rANS-framed iff the rANS flag is set)              |
+--------------------------------------------------------------------------+
```

- 3-byte magic `'Q' 'D' 'P'` + a 1-byte version (`0x01`).
- 1 flag byte: *dense* (field names interned in the body), *rANS* (body is
  rANS-compressed), *baseFP present* (default on).
- An 8-byte little-endian **schemaFP** (always present).
- An optional 8-byte little-endian **baseFP** (present by default).

**Ops.** Every changed location is one op byte plus a payload:

| Op           | Payload |
| ------------ | ------- |
| *absent*     | (nothing — the location is simply not listed) |
| `opReplace`  | the whole new value, encoded with the normal qdf value codec |
| `opMerge`    | a recursive sub-patch (one of the three container-patch tags below) |

**Container-patch tags** describe a changed composite:

| Tag              | Layout |
| ---------------- | ------ |
| `tagStructPatch` | `varuint(nChanged)`, then `nChanged ×` `{varuint(fieldIdx), op}` — sparse: only changed fields. |
| `tagSlicePatch`  | `varuint(newLen)`, `varuint(nEntries)`, then `nEntries ×` `{varuint(idx), op}` — positional. `newLen` resizes the slice (grow/shrink). |
| `tagMapPatch`    | `varuint(nUpdate)`, then `nUpdate ×` `{key, op}` updates, then `varuint(nDelete)`, then `nDelete ×` `{key}` tombstones. |

---

## Semantics

The core rule is **absent = unchanged**: if a location is not listed in the
patch, `Apply` leaves the corresponding location in `base` exactly as it is.

| Transition (`old` → `new`)                              | Patch emits |
| ------------------------------------------------------- | ----------- |
| value unchanged                                         | **absent** (nothing) |
| scalar / string / `[]byte` / `[N]byte` changed          | `opReplace` (whole new value) |
| field/element/key set to its **zero / nil** value       | `opReplace` (the zero/nil is the new value) |
| **presence change** — nil ↔ non-nil pointer, or nil ↔ non-nil container | `opReplace` |
| **nil ↔ empty-non-nil** slice / map / `[]byte`          | `opReplace` (the nil-vs-empty distinction is preserved, exactly as the plain encoder preserves it) |
| nested **struct** changed                               | `opMerge` (recurse) |
| **both-non-nil** pointer / map / slice changed          | `opMerge` (recurse) |

The nil-vs-empty rule matters: a transition between `nil` and an empty
non-nil slice/map/`[]byte` (`json` `null` vs `[]`) is a real change and is
shipped as a whole-value replace, never silently collapsed.

---

## Deletion

| Container | How a removed element is expressed |
| --------- | ---------------------------------- |
| **map**   | A **tombstone** in the delete set of `tagMapPatch`; `Apply` deletes the key from `base`'s map. |
| **slice** | A **positional shrink**: `tagSlicePatch` carries the new (shorter) length and `Apply` truncates. |
| **array** | Arrays are fixed-length — there is no deletion; a removed value is a zero/nil `opReplace` at its index. |
| **struct field** | Struct shape is fixed — a "removed" value is just a zero/nil `opReplace`. |

> **Middle-deletion caveat (slices).** Slice diff is **positional** in Phase 1.
> Deleting an element from the middle is expressed as positional shifts of the
> tail plus a truncate — value-correct, but it **reships the shifted tail**
> because every index after the deletion now holds a different value. A
> keyed-slice diff (match by a declared key field, so a middle insert/delete
> does not reship the tail) is [Phase 2](#limitations--roadmap).

---

## Safety

`Apply` will not silently corrupt a value applied against the wrong type or the
wrong base.

- **`schemaFP`** (always present) is a hash of the type's *shape* — kind, field
  names, and recursive field/element kinds. `Apply` recomputes it from the
  target type and rejects a patch built for a different type with
  `ErrPatchSchemaMismatch`. A patch can never be mis-parsed as if it were for
  another struct.
- **`baseFP`** (on by default) is an **order-independent** hash of `old`
  (map-bearing values fingerprint deterministically regardless of map
  iteration order). `Apply` recomputes it over the supplied base and rejects a
  base that is not the original `old` with `ErrPatchBaseMismatch`. This is why
  applying a patch onto a *divergent* base fails loudly instead of producing a
  corrupt merge.

**Diff-before-rANS.** Under `OptCompression` the optional order-0 rANS entropy
pass runs **after** the structural diff, over the already-minimal patch body,
and only when it shrinks (never larger). The diff does the structural work; rANS
squeezes the residual byte entropy of what is left.

**Fuzz-tested.** The delta path is covered by a round-trip oracle fuzzer
(`Apply(base, Diff(old, new)) == new`, 1.5 M+ executions) and a hostile-patch
fuzzer that feeds malformed patch bytes to `Apply` (3.6 M+ executions). It
never panics and never OOMs on hostile input — every length-prefixed read is
bounded by the input.

---

## Performance

Measured on `BenchmarkDiffApply` (Intel i7-9750H, Go 1.26): a 1000-record
slice with a **single changed field**.

| Operation         | time     | B/op    | allocs/op | output size |
| ----------------- | -------: | ------: | --------: | ----------: |
| **`Diff`**        | ~0.69 ms | 24.5 KB |    ~3 010 | **42 B**    |
| **`Apply`**       | ~0.26 ms |  82 KB  |         5 | —           |
| full `Marshal`    | ~0.21 ms |  27 KB  |         3 | **26 677 B**|

The headline: the **patch is 42 bytes against the 26 677-byte full Marshal —
~635× smaller** when one field of a 1000-record batch changed. You pay diff CPU
to compute it, but you ship a fraction of the bytes — exactly the trade a
state-sync / CDC workload wants.

---

## Limitations & roadmap

**Phase 1 (shipped).** Diff/Apply for structs (incl. nested), scalars / string
/ `[]byte` / `[N]byte`, **positional** slices and arrays, per-key maps with
tombstones; schema + base fingerprints; optional rANS post-pass.

**Slices are positional only** in Phase 1 — a reorder or a middle insert/delete
reships the shifted tail (see [Deletion](#deletion)).

**Phase 2 (not built).**

- **Columnar column-level diff for `[]struct`** — when a slice of flat structs
  is columnar, ship only the columns that changed (1 of 20 columns changed →
  ship 1 column's delta), leaning on qdf's existing columnar transpose.
- **Keyed slice diff** — match elements by a declared key field so a reorder or
  middle insert/delete updates only the affected elements instead of reshipping
  the positional tail.
- **Content-addressed baseline registry** — address the baseline by a content
  hash / id instead of the caller holding `old`, so a patch can be applied
  against a baseline fetched from a store.
</content>
</invoke>
