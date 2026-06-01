# QPack Codecs

## What this shows

The per-slice codec picker that runs when `OptQPack` is set. For every
numeric, bool, or float slice in the payload the encoder estimates the wire
cost of several codecs and emits the smallest one. A never-larger floor
guarantees the QPack path never grows the output vs raw encoding. The SIMD
acceleration layer is noted at the end.

## Codec picker decision tree

```mermaid
flowchart TD
    Start["slice value\n([]bool / []intN / []uintN / []float32 / []float64)"]
    Start --> Bool{"[]bool?"}
    Bool -->|yes| PBool["tagPackBool (0xE3)\n1 bit/elem\nalways smaller than per-tag stream"]

    Bool -->|no| Float{"[]float32\nor []float64?"}
    Float -->|yes| GorillaOpt{"OptGorillaFloat\nset?"}
    GorillaOpt -->|no| Raw["tagPackRaw (0xE4)\nraw little-endian bulk"]
    GorillaOpt -->|yes| GorillaProbe["probe first 32 XOR pairs\n~30 ns"]
    GorillaProbe -->|"projected cost\n< 64 bits/sample"| Gorilla["tagPackGorilla (0xE7)\nXOR-delta, Gorilla coding\n~70% win on smooth series"]
    GorillaProbe -->|"cost ≥ 64 bits"| ALPCheck{"float64 +\ndecimal-like?"}
    ALPCheck -->|yes| ALP["tagPackALP (0xF4)\nAdaptive Lossless\nFloating-Point (CWI 2023)\nzig-zag FOR + exception list"]
    ALPCheck -->|no| Raw

    Float -->|no| IntSlice["integer slice\ncompute min, max, spread"]
    IntSlice --> RawEst["estimate raw-LE cost\n(always available: baseline)"]
    IntSlice --> FOR["estimate FOR cost\nbitsPer = ceil(log2(max-min))\nFOR body = 3 + n*bitsPer/8 bytes"]
    IntSlice --> Delta["monotonic / clustered?\nestimate Delta+FOR\nencode(aᵢ - aᵢ₋₁) then FOR\n→ near-zero bytes for timestamps"]
    IntSlice --> RLE["run-fraction probe\n(first 32 elems)\nif run-heavy: estimate RLE\n(value,count) pairs"]
    IntSlice --> Dict["distinct ≤ 64?\nestimate dict\ntable + ceil(log2 d)*n/8 index bits\nO(1) open-addressed hash probe"]
    IntSlice --> PFor["outlier tail?\nestimate PFOR\nFOR at reduced width\n+ exception list for outliers"]

    RawEst --> Floor
    FOR --> Floor
    Delta --> Floor
    RLE --> Floor
    Dict --> Floor
    PFor --> Floor

    Floor{"never-larger floor:\npick smallest estimate;\nif none beats raw → tagPackRaw"}
    Floor --> Emit["emit winner tag\n+ varuint(n) + payload"]
```

## Codec reference table

| Tag | Hex | Best case | Notes |
|-----|-----|-----------|-------|
| `tagPackBool` | `0xE3` | 8× vs per-tag stream | Always chosen for `[]bool` |
| `tagPackRaw` | `0xE4` | baseline | Raw LE; fallback when no codec wins |
| `tagPackFor` | `0xE5` | spread is small (e.g. IDs in a range) | SIMD-accelerated pack/unpack |
| `tagPackDeltaFor` | `0xE6` | monotonic sequences (timestamps) | 512× on consecutive Unix seconds |
| `tagPackGorilla` | `0xE7` | smooth float time-series | Opt-in: `OptGorillaFloat` |
| `tagPackRLE` | `0xEB` | run-heavy integer columns (status codes, enums) | Probe on first 32 elems |
| `tagPackDict` | `0xED` | ≤64 distinct values, wide spread | O(1) open-addressed hash |
| `tagPackPFor` | `0xEE` | mostly-small integers with rare large outliers | Narrow FOR body + exception list |
| `tagPackALP` | `0xF4` | quantized / decimal float64 | Chosen only when strictly smaller than raw and Gorilla |

## SIMD acceleration (build tag `qdf_simd`)

```mermaid
flowchart LR
    Scalar["Pure-Go\nscalar path\n(default)"]
    SIMD["qdf_simd\nbuild tag"]
    AVX2["amd64 AVX2\nCPUID-gated at runtime"]
    NEON["arm64 NEON\nbaseline, always active"]

    SIMD --> AVX2
    SIMD --> NEON

    AVX2 --> D1["Decode: VPMOVZX\nbyte-aligned widths 8/16/32"]
    AVX2 --> D2["Decode: VPBROADCASTQ+VPSRLVQ\nwidths 1–28"]
    AVX2 --> E1["Encode: VPSHUFB\nbyte-gather for 8/16/32"]
    AVX2 --> E2["Encode: VPSLLVQ\nfor widths 10/12/14/20"]
    AVX2 --> E3["Bool pack: VPSLLW+VPMOVMSKB"]

    NEON --> N1["Decode widths 1–28 + 32\nUSHLL zero-extend widen\n2-lane VLD1R+USHL kernel"]

    Scalar -.->|"fallback when\nno SIMD or\nunsupported width"| Fallback["scalar window\nuint64 shifts"]
```

The SIMD path is a pure speed switch — wire format and output bytes are
**identical** to the scalar path on every architecture. Typical speedup:
3–11× over pure-Go on the accelerated bit widths. Widths outside the
supported set fall back to the scalar window kernel silently.
