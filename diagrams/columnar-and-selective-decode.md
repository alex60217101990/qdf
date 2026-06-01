# Columnar Encoding and Selective Decode

## What this shows

How qdf transparently transposes a `[]struct` into a column-per-field layout
(`tagColStruct`), including the special handling for `time.Time` (split into
two sub-columns) and nullable fields (`*T` columns with presence bitmap).
Then: how `OptColumnIndex` adds a fixed-width length table enabling O(1) column
skipping, and how `Select`/`Where` predicate-pushdown with SQL 3-valued logic
filters rows inside the decoder.

## []struct → columnar transpose

```mermaid
flowchart TD
    Input["[]MyStruct{row0, row1, ..., rowM-1}\n(M ≥ 16, flat fields, under OptBalanced)"]
    Input --> Probe["columnarProbe:\nsample ≤32 rows\nestimate col vs row wire size\n(string dict + FOR for int cols)"]
    Probe -->|"columnar wins"| Trans["transpose: gather each field\nacross all rows into a typed slice"]
    Probe -->|"row wins or uncertain"| RowMajor["row-major encode\n(tagMapShape stream)"]

    Trans --> ColKind{"classify colKind\nper field"}

    ColKind -->|"int8..int64"| IntCol["colKindInt\ngather → []int64\nQPack picker\n(FOR/Delta/RLE/dict/PFOR)"]
    ColKind -->|"uint8..uint64"| UintCol["colKindUint\ngather → []uint64\nQPack picker"]
    ColKind -->|"float32/64"| FloatCol["colKindFloat\ngather → []float64\nraw-LE or Gorilla (OptGorillaFloat)"]
    ColKind -->|"bool"| BoolCol["colKindBool\ngather → []bool\ntagPackBool (1 bit/row)"]
    ColKind -->|"string / []byte"| StrCol["colKindString\ntry tagColStrDict first\n(distinct table + bitpacked index)\nif dict wins → emit 0xF5;\notherwise per-value + intern"]
    ColKind -->|"time.Time"| TimeCol["colKindTime\nSPLIT into TWO sub-columns:\n  sec  → []int64 (Delta+FOR, monotonic timestamps → near-zero bytes)\n  nsec → []uint64 (QPack)"]
    ColKind -->|"*T (pointer to scalar/string)"| NullCol["colKindNullable (high bit 0x80)\npresence bitmap: 1 bit/row, LSB-first\ndense column of present values only\n→ base colKind codec applied\n(no per-row alloc for absent rows)"]

    IntCol --> Wire["tagColStruct body:\nshapeID + M columns in order"]
    UintCol --> Wire
    FloatCol --> Wire
    BoolCol --> Wire
    StrCol --> Wire
    TimeCol --> Wire
    NullCol --> Wire
```

## Column-length index (OptColumnIndex / FlagColIndex)

```mermaid
flowchart LR
    Enc["Encoder\n(OptColumnIndex set)"]
    Enc --> H["reserve K×4 bytes\nright after shape declaration\n(backpatch during encode)"]
    H --> C0["column 0 body"]
    C0 --> C1["column 1 body"]
    C1 --> Cdots["..."]
    Cdots --> CK["column K-1 body"]

    Idx["column-length index\n[len0, len1, ..., lenK-1]\nK × uint32 LE"]
    H -.->|"backpatch\nafter body written"| Idx

    style Idx fill:#f9f,stroke:#333
```

With the index, the decoder can skip any column body with a single pointer
advance (`d.i += colLen[c]`) instead of parsing the column to advance the
cursor. Cost becomes O(columns read), not O(all columns).

The flag is **backpatched** — set in the header byte only after the encoder
confirms a columnar body was actually written. `OptColumnIndex` on a
non-columnar payload is a true no-op.

## Selective decode: Select + Where

```mermaid
flowchart TD
    Call["Unmarshal(buf, &out,\n  Select('ts','code'),\n  Where('level', func(s string) bool {...}),\n  Where('code',  func(c int32)  bool {...}))"]

    Call --> Read["readColShape()\nread shape + column-length index"]
    Read --> Pass["single forward pass over columns"]

    Pass --> Skip{"column needed\n(projected or\nreferenced by Where)?"}
    Skip -->|no + FlagColIndex| SeekPast["d.i += colLen[c]\nO(1) skip"]
    Skip -->|no, no index| Discard["decode + discard\n(cursor must advance)"]
    Skip -->|yes| Decode["decodeColumnVals(kind, M)"]

    Decode --> Eval["evalDense / evalNullable\nper column with WHERE predicate\ntyped func(T) bool\nno per-value boxing\nbuild per-column bitmask"]

    Eval --> TreeEval["3VL predicate tree evaluation\n(And/Or/Not over per-column bitmasks)\ntwin (T,F) bitmask pairs\nnullable nil never matches"]

    TreeEval --> Compact["compute matched rows:\nwalk set-bits once"]
    Compact --> Scatter["scatter matched rows only\ninto output []struct\nor []map[string]any\n(scatterRow / scatterNullableRowInto)"]
    Scatter --> Result["output: only matched rows\nof projected columns\nzero per-value boxing"]
```

## Nullable column: slab allocation

```mermaid
flowchart LR
    Col["nullable column\n*T field\n(presence bitmap + dense values)"]
    Col --> Slab["reflect.MakeSlice once\n(one slab for all matched rows)"]
    Slab --> Loop["for each matched row:\ncopy value into slab[dst]\npoint *T field at slab[dst]"]
    Loop --> Result["output slice\nno per-row reflect.New\n(one slab per column, not one alloc per row)"]
```

The slab allocator (added in `perf/nullable-scatter-slab`) replaces
per-row `reflect.New` calls with a single slice allocation per nullable
column, cutting allocations and GC pressure on batches with optional fields.

## 3VL predicate tree

```mermaid
flowchart TD
    A["predicate tree root\nAnd / Or / Not / leaf"]
    A --> B{"node type"}
    B -->|"Leaf Where(field, pred)"| C["decode column 'field'\nif nullable: clear absent rows from T-mask\napply pred to each present value\nresult: (T-bitmask, F-bitmask)"]
    B -->|"And(left, right)"| D["T = Tl & Tr\nF = Fl | Fr\nU = ~T & ~F (unknown = null rows)"]
    B -->|"Or(left, right)"| E["T = Tl | Tr\nF = Fl & Fr\nU = ~T & ~F"]
    B -->|"Not(child)"| F["swap T and F masks\nU unchanged"]
    C --> G["matched rows = popcount(T-mask)"]
    D --> G
    E --> G
    F --> G
```

SQL 3-valued logic: a nullable `nil` row never matches a predicate
(it contributes to the Unknown set, which is excluded from results).
Only rows in the True bitmask are materialised in the output.

## Performance impact (confirmed numbers)

| Scenario | Full decode + filter | Selective pushdown |
|----------|---------------------|--------------------|
| 3 of 16 columns, 2000 rows | baseline | ~5.7× faster, ~5.3× fewer bytes |
| 9 columns, 2000 rows, 1% selectivity | baseline | ~2.3× faster, ~4.4× fewer bytes |
