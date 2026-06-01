# Architecture: Marshal / Unmarshal End-to-End

## What this shows

The full encode/decode pipeline: how a caller-supplied value enters the
`Marshal` entry point, is routed through the pool and type-descriptor cache,
dispatched to Fast or Dense mode, optionally columnar-transposed, and finally
rANS-compressed. The decode mirror image is shown separately. Key design
choices that appear here: reflect-once (cached `typeDesc`), pooled encoders
(`encPool`), lazy Dense-state allocation, and the rANS pass as a final stage
that never grows the buffer.

## Encode path

```mermaid
sequenceDiagram
    participant Caller
    participant Pool as encPool (sync.Pool)
    participant Enc as "*Encoder"
    participant Cache as "typeCache (sync.Map)"
    participant Wire as "[]byte wire"

    Caller->>Pool: acquire *Encoder
    Pool-->>Enc: pooled encoder (buf reused)
    Caller->>Enc: applyOpts(opts)\nsets mode/qpack/rans/colIndex
    Enc->>Enc: writeHeader()\n'Q','D','F',0x01,flags byte
    Caller->>Enc: encodeReflect(v)
    Enc->>Cache: descOf(reflect.TypeOf(v))
    Cache-->>Enc: *typeDesc (hit) or build+store (miss)
    Note over Cache: typeDesc holds precompiled\nencode/decode closures\nover unsafe field offsets\n(reflect runs once per type)
    Enc->>Enc: td.encode(enc, unsafe.Pointer(&v))
    alt []struct with ≥16 flat-field elems (columnarProbe says yes)
        Enc->>Enc: encodeColumnar()\ntranspose rows→columns\ntagColStruct (0xEF)
        opt OptColumnIndex set
            Enc->>Enc: backpatch FlagColIndex\nwrite K×uint32 column-length table
        end
    else row-major
        Enc->>Enc: tag-stream field by field\n(Fast: inline) or (Dense: intern+predict)
    end
    opt OptRANS set
        Enc->>Enc: maybeApplyRANS()\norder-0 rANS over body\nonly if strictly smaller\nset FlagRANS in header
    end
    Enc-->>Wire: e.buf (encoded bytes)
    Enc->>Pool: release *Encoder
```

## Decode path

```mermaid
sequenceDiagram
    participant Caller
    participant Pool as decPool (sync.Pool)
    participant Dec as "*Decoder"
    participant Cache as "typeCache (sync.Map)"

    Caller->>Pool: acquire *Decoder
    Pool-->>Dec: pooled decoder
    Dec->>Dec: readHeader()\ncheck magic 'Q','D','F'\nread version + flags byte
    Note over Dec: Decoder is header-driven:\nno opts arg on Unmarshal.\nFlags auto-detect mode.
    opt FlagRANS set
        Dec->>Dec: rANS decompress body\nrestore plain tag stream
    end
    Dec->>Cache: descOf(reflect.TypeOf(out))
    Cache-->>Dec: *typeDesc closures
    alt FlagColIndex + tagColStruct in body
        Dec->>Dec: readColShape()\nread K×uint32 column-length index
        Dec->>Dec: decodeColumnar()\nscatter columns → []struct\n(selective: skip via index)
    else row-major
        Dec->>Dec: td.decode(dec, unsafe.Pointer(out))
    end
    Dec->>Pool: release *Decoder
```

## typeDesc cache detail

```mermaid
flowchart LR
    A["reflect.TypeOf(v)"] --> B{"typeCache.Load(rType)"}
    B -->|hit| C["*typeDesc\n(cached closures)"]
    B -->|miss| D["buildDesc(rType)\n- walk struct fields\n- record field offsets\n- compile encode/decode\n  closures per field kind\n- store in typeCache"]
    D --> C
    C --> E["encode closure:\nfunc(*Encoder, unsafe.Pointer)\naccess fields via unsafe offset\nno per-call reflect"]
```

The cache is populated once per unique Go type across the process lifetime.
After the first call, encoding a `MyEvent` touches only the cached closure
array — no reflection, no type assertions on the hot path.

## Pool lifecycle

```mermaid
stateDiagram-v2
    [*] --> Idle : sync.Pool.New()
    Idle --> Encoding : Marshal acquires
    Encoding --> ResetCheck : encode done
    ResetCheck --> Idle : cap within limits\n(buf ≤ 16 MiB,\nintern ≤ 4096 IDs)\nreleased to pool
    ResetCheck --> GC : cap exceeded\nbacking arrays dropped\nencoder discarded
    GC --> [*]
```

Dense-mode intern state (`encState`) is allocated lazily on the first Dense
call and reused across pool recycles. `Reset()` shrinks the backing arrays
when they exceed soft caps (`maxRetainedIDs = 4096`, `maxPooledBuf = 16 MiB`)
to prevent a single large payload from permanently inflating every pooled
encoder.
