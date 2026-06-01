# Dense Mode: Intern Table and State-Ref Predictors

## What this shows

Dense mode's four-layer compression stack for repeated string and []byte
values. The intern table assigns a numeric ID to each unique value on first
sight; subsequent occurrences emit a compact state-ref tag. Four predictors
layer on top of raw state-refs — Markov-0 (same-as-previous), MTF rank,
Markov-1 pair, and shape interning — each saving bytes only when they cost
fewer bytes than the raw ref, so the wire is never larger than plain state-ref
encoding.

## Intern table write path

```mermaid
flowchart TD
    A["WriteString(s)\nor WriteBytes(b)"]
    A --> B{"OptDense set\nand len(s) >= minIntern\nand internLoad < maxStateEntries?"}
    B -->|no| Inline["writeStringInline(s)\nfixstr / str8 / str16 / str32\n+ raw bytes"]
    B -->|yes| C["lookupOrAssign(s)\nopen-addressed flat hash table\nhash via maphash.String"]
    C -->|"miss (new value)"| D["assign next sequential ID\ncopy key into internarena slab\n(bump-pointer, one alloc/batch)\nwrite tagInternStr + varuint(len) + bytes"]
    C -->|"hit (known ID)"| E["emitStateRef(id)"]

    E --> F{"id == lastID?"}
    F -->|yes| G["tagStateRepeat (0xE8)\n1 byte total\nMarkov-0 hit"]
    F -->|no| H{"OptPairPred set\nand pairLookup(lastID, id)?"}
    H -->|yes| I["tagStatePair (0xEA) + 0x00\n2 bytes total\nMarkov-1 hit (top-1, rank always 0)"]
    H -->|no| J{"OptMTF set\nand mruRank(id) = r\nand uvarintLen(r) < uvarintLen(id)?"}
    J -->|yes| K["tagStateMTF (0xE9) + varuint(r)\nMTF rank coding\n(mruRing: 128-slot side-cache,\nO(1) rank lookup)"]
    J -->|no| L["tagStateRef (0xE1) + varuint(id)\nraw state-ref fallback"]

    G --> M["update lastID, LRU, pairPred, mruRing"]
    I --> M
    K --> M
    L --> M
```

## Predictor stack

```mermaid
flowchart LR
    subgraph "4 predictors, tried in order"
        P0["Markov-0\ntagStateRepeat 0xE8\n1 byte\nid == lastID\nalways on in Dense mode"]
        P1["Markov-1 pair\ntagStatePair 0xEA\n2 bytes (rank always 0)\npairPred lookup:\nprev→succ top-1 table\nrequires OptPairPred"]
        P2["MTF rank\ntagStateMTF 0xE9\n1 + uvarintLen(rank) bytes\n128-slot mruRing\nO(1) rank lookup\nrequires OptMTF"]
        P3["Raw state-ref\ntagStateRef 0xE1\n1 + uvarintLen(id) bytes\nalways available fallback"]
    end
    P0 -->|"miss"| P1
    P1 -->|"miss or OptPairPred off"| P2
    P2 -->|"miss or OptMTF off\nor rank not shorter"| P3
```

Encoder picks the **shortest** of the applicable forms. If MTF rank encodes
as a longer varuint than the raw ID, the raw form is used instead — the wire
never grows from predictor overhead.

## Intern state data structures

```mermaid
classDiagram
    class encState {
        internTable []internSlot
        internLoad int
        lastID uint32
        lruLink []uint32
        mruRing [128]uint16
        mruHead int
        pairPred []uint32
        arena internarena
        colShapeBindings []shapeBinding
    }
    class internarena {
        chunks []slice_of_bytes
        locs []uint64
        off uintptr
        cur int
    }
    note for encState "lruLink packs prev+next as\n(prev<<0)|(next<<16)\nhalving cache-line cost\nvs separate prev/next arrays"
    note for internarena "bump-pointer allocator:\nPut(s) copies into active slab,\nreturns packed uint64 id.\nGet(id) = zero alloc.\nKeeps GC off per-key path."
```

## Shape interning (OptShapeIntern)

```mermaid
sequenceDiagram
    participant Enc as Encoder
    participant ST as encState.shapeBindings
    participant Wire

    Enc->>ST: lookup *typeDesc in shapeBindings
    alt first time this struct type
        ST-->>Enc: not found
        Enc->>Wire: 0xEC 0x00 varuint(N) N×(key_state_ref)
        Note over Wire: shape declaration (keys as state-refs)
        Enc->>ST: record (typeDesc, shapeID)
    else subsequent occurrence
        ST-->>Enc: shapeID
        Enc->>Wire: 0xEC varuint(shapeID) N×value
        Note over Wire: keys NOT re-emitted
    end
```

Per-record wire saving on an array of identical struct types: approximately
`N × 2` bytes (elided key state-refs + map-header tag) per row after the first.
On a 100-row batch with 8-field structs, that is roughly 1 600 bytes saved.

## Streaming vs single-message intern scope

The intern table is scoped to the encoder instance.

| API | Intern table scope |
|-----|--------------------|
| `Marshal(v, opts)` | per call — fresh table each time |
| `StreamEncoder.Encode(v)` | per stream — shared across all `Encode` calls |

`StreamEncoder` is where the intern table amortises best: a 10 000-row log
stream shares one table, so region codes and service names written 10 000
times each ship once.
