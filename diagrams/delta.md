# Structural Delta: `Diff` / `Apply`

## What this shows

The structural-delta layer: how `Diff(old, new, opts)` walks two values of the
same type and emits a patch carrying only the locations that changed, how
`Apply` merges that patch back onto a base in place, the self-describing patch
wire format (its own `'Q','D','P'` magic and the two fingerprints that guard
it), and the three advanced matchers layered on top — keyed slices, columnar
column-level diff, and the content-addressed baseline registry. Delta reuses the
same `typeDesc` cache and value codecs as `Marshal`, so a patch is as tightly
packed as a fresh encode of the changed cells and never has to be told which
`Options` produced it.

## Diff / Apply pipeline

<img src="svg/delta-1.svg" alt="Diff and Apply pipeline flowchart">

<details><summary>Mermaid source</summary>

```mermaid
flowchart TD
    A["Diff(old, new, opts)"] --> B["descOf(T)\nshared typeDesc cache\n(same as Marshal)"]
    B --> C["writePatchHeader\n'Q','D','P', ver, flags\n+ schemaFP (8 B)\n+ baseFP (8 B, unless opt-out)"]
    C --> D{"root kind?"}
    D -->|struct| E["diffStruct\nfield-by-field\nsparse: only changed fields"]
    D -->|slice / array| F{"elem has ,key tag\n& equal-len columnar?"}
    D -->|map| G["diffMap\nper-key update set\n+ tombstone delete set"]
    D -->|scalar / ptr| H["whole-value replace\n(nil-vs-empty preserved)"]
    F -->|",key"| I["keyed slice match\n(tagKeyedSlicePatch 0x04)"]
    F -->|"pure-columnar ≥16"| J["column-level diff\n(tagColSlicePatch 0x05)\nnever-larger vs positional"]
    F -->|else| K["positional slice diff\n(tagSlicePatch 0x02)"]
    E --> L["op per location:\nopReplace (whole value)\nor opMerge (sub-patch)"]
    G --> L
    H --> L
    I --> L
    J --> L
    K --> L
    L --> M{"OptRANS set\n& body shrinks?"}
    M -->|yes| N["rANS-frame body\nset rANS flag"]
    M -->|no| O["plain body"]
    N --> P["patch bytes"]
    O --> P
```

</details>

`Diff` expresses every change at the finest granularity that still shrinks the
patch: an unchanged field, element, or key costs **zero bytes**; a scalar or
presence change ships the whole new value with the normal codec; a nested
struct / pointer / map / slice recurses and carries a sub-patch of only *its*
changed locations. The apply side is the mirror: **absent means unchanged** —
`Apply` leaves any location the patch does not list exactly as it found it.

## Patch wire format

<img src="svg/delta-2.svg" alt="patch wire format layout">

<details><summary>Mermaid source</summary>

```mermaid
flowchart LR
    subgraph Header
        M["'Q' 'D' 'P'\n3-byte magic"]
        V["ver\n1 byte"]
        F["flags\ndense | rANS | baseFP-present"]
        S["schemaFP\n8 B LE (always)"]
        BFP["baseFP\n8 B LE (optional)"]
    end
    subgraph Body["Body — root patch"]
        SP["tagStructPatch 0x01\nvaruint(nChanged) × {fieldIdx, op}"]
        SL["tagSlicePatch 0x02\nvaruint(newLen), {idx, op}…"]
        MP["tagMapPatch 0x03\nupdates {key, op}… + tombstones {key}…"]
        KS["tagKeyedSlicePatch 0x04\nflags, [order list], {key, op}…"]
        CS["tagColSlicePatch 0x05\nper-column {colIdx, mode, body}"]
    end
    M --> V --> F --> S --> BFP --> Body
    Body --> SP
    Body --> SL
    Body --> MP
    Body --> KS
    Body --> CS
```

</details>

A patch carries its own `'Q','D','P'` magic so it can never be mistaken for a
full value (`'Q','D','F'`) or vice versa. Two fingerprints guard `Apply`: the
**schemaFP** (always present) is a hash of the type's shape and rejects a patch
built for a different type (`ErrPatchSchemaMismatch`); the **baseFP** (a content
hash of `old`, present unless `OptDeltaNoBaseFingerprint`) rejects a patch
applied to the wrong base (`ErrPatchBaseMismatch`) — never silent corruption.
Each op byte is either `opReplace` (whole new value) or `opMerge` (a recursive
sub-patch).

## Keyed slices vs columnar column-diff

<img src="svg/delta-3.svg" alt="slice matcher decision tree">

<details><summary>Mermaid source</summary>

```mermaid
flowchart TD
    A["[]struct change"] --> B{"elem field tagged\n,key (comparable)?"}
    B -->|yes, keys unique| C["keyed match\n- reorder ships key order only\n- insert/delete/move touches\n  just that element\n- value-only edit ships no order"]
    B -->|keys not unique| D["fall back to positional\n(still correct)"]
    B -->|no ,key| E{"equal-len old/new\n& pure-columnar\n& ≥16 elems?"}
    E -->|yes| F["column-level diff\nper changed column, smallest of:"]
    E -->|no| G["positional diff\ntagSlicePatch\n(reships shifted tail)"]
    F --> H["sparse\ngap-encoded changed rows\n+ new cell values"]
    F --> I["arithmetic-delta\nfull-length per-row delta\n(numeric/bool, not float)"]
    F --> J["dense-whole\nentire new column re-shipped"]
    F --> K{"column patch <\npositional patch?"}
    K -->|yes| L["emit tagColSlicePatch 0x05"]
    K -->|no| G
```

</details>

By default slices match **positionally**, so a middle insert/delete or a reorder
reships the shifted tail. Two opt-free matchers fix the common cases: a `,key`
tag matches elements by stable identity (reorder ships only the new key order;
a moved element touches just itself), and an equal-length pure-columnar batch
is diffed **by column** — each changed column packed with the same FOR / delta /
RLE / dictionary codecs as a fresh columnar encode. Both are **never-larger**:
the column patch is emitted only when it beats the positional patch for the same
change, and a non-unique-key slice transparently falls back to positional.

## Baseline registry — state-sync chains

<img src="svg/delta-4.svg" alt="baseline registry state-sync sequence">

<details><summary>Mermaid source</summary>

```mermaid
sequenceDiagram
    participant Prod as Producer
    participant Reg as "BaselineRegistry[T]"
    participant GC as "Go GC"

    Note over Prod: holds prev state, ships ordinary Diff\n(baseFP embedded by default — no wire change)
    Prod->>Reg: Register(s0)  // bootstrap full value
    Note over Reg: stores s0 via weak.Pointer\nkeyed by content fingerprint
    Prod->>Reg: Apply(patch1)
    Reg->>Reg: resolve baseFP → s0\ndeep-clone s0\napply patch1 → s1\nauto-register s1
    Reg-->>Prod: *s1  (keep alive for next step)
    Prod->>Reg: Apply(patch2)
    Reg->>Reg: resolve baseFP → s1 → clone → apply → s2
    Reg-->>Prod: *s2
    Note over GC: any baseline the caller dropped\nis reclaimed (weak ref, zero leak)
    Reg->>Reg: dropped baseline → ErrBaselineEvicted\n(re-sync a full value to recover)
```

</details>

`BaselineRegistry[T]` removes the bookkeeping from a delta *stream*: the consumer
applies a chain of patches without threading `old` by hand. It reuses the
patch's existing `baseFP` as a lookup key (no new tag, flag, or field), holds
each baseline through a **`weak.Pointer`** so the GC reclaims anything the caller
dropped, and deep-clones the resolved baseline before applying so a branch off
the same state stays intact. The `baseFP` check is the backstop that turns any
clone discrepancy into an error, never silent corruption. It is safe for
concurrent `Register` / `Apply` / `Len`.
