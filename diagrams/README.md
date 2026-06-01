# qdf Architecture Diagrams

Visual reference for the qdf serialization format. Each document covers one
layer of the system with a Mermaid diagram (rendered natively by GitHub) and
a short explanatory prose section.

## The concept: "Parquet for messages"

JSON, msgpack, and protobuf encode every record independently. qdf treats a
batch of structured records as a single unit of compression: strings are
interned once across all records, numeric slices are compressed column-by-column,
and struct shapes are declared once and reused by ID. The closest prior art is
Apache Parquet, applied on-the-fly to in-memory message batches with no schema
file and no two-pass encode.

### Overview flowchart

<img src="svg/README-1.svg" alt="qdf encoder dispatch flowchart">

<details><summary>Mermaid source</summary>

```mermaid
flowchart TD
    A["Marshal(v, opts)"] --> B["acquire *Encoder\nfrom encPool"]
    B --> C["applyOpts\n(mode, qpack, rans, colIndex flags)"]
    C --> D["writeHeader\n5 bytes: 'Q','D','F', 0x01, flags"]
    D --> E{"opts mode?"}
    E -->|"OptSpeed (0)"| F["Fast path\ntagged value stream\nno intern, pooled"]
    E -->|"OptDense set"| G["Dense path\ninline intern table\nstate-ref predictors"]
    F --> H{"[]struct ≥16 elems\n+ flat fields?"}
    G --> H
    H -->|yes| I["columnar transpose\nper-column QPack\n(tagColStruct 0xEF)"]
    H -->|no| J["row-major\nfield-by-field encode"]
    I --> K["optional colIndex\n4 B/column length table\n(FlagColIndex)"]
    J --> L["maybeApplyRANS\n(OptRANS: whole-body\norder-0, never larger)"]
    K --> L
    L --> M["release *Encoder\nto encPool"]
    M --> N["wire bytes"]
```

</details>

The encoder path (top-left to bottom-right) is a straight line with two
decision branches: mode (Fast vs Dense) and shape (columnar vs row-major).
The rANS pass is the last stage and fires only when it shrinks the output.

## Diagram index

| File | What it shows |
|------|---------------|
| [architecture.md](architecture.md) | Marshal/Unmarshal end-to-end: pool lifecycle, typeDesc cache, mode dispatch, rANS pass |
| [wire-format.md](wire-format.md) | 5-byte header layout, flags bit map, tag ranges, tagged-body structure |
| [options-and-modes.md](options-and-modes.md) | Options bit positions, bundle compositions (Speed/Balanced/Compression), Fast vs Dense decision tree |
| [qpack-codecs.md](qpack-codecs.md) | QPack codec picker for numeric/bool/float slices, never-larger floor, SIMD kernels |
| [dense-interning.md](dense-interning.md) | Intern table, state-ref emission, Markov-0/MTF/Markov-1/shape-intern predictors |
| [columnar-and-selective-decode.md](columnar-and-selective-decode.md) | []struct columnar transpose, Time split, Nullable column, colIndex, Select/Where predicate-pushdown with 3VL |
| [performance.md](performance.md) | The 10 algorithmic/CPU/memory wins grouped by category |
