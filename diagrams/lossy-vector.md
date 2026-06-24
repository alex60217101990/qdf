# Lossy vector codec (`OptLossyVec`, tag `0xFD`)

The lossy vector codec compresses `[]float32` / `[]float64` embedding fields by
trading a bounded, caller-chosen amount of fidelity for a much smaller wire.
It runs a rotation → lattice-quantize → entropy-code pipeline, picks the
smaller of two quantizers per column, and never exceeds the lossless size.

## Encode pipeline

<img src="svg/lossy-vector-1.svg" alt="Lossy vector encode pipeline flowchart">

<details><summary>Mermaid source</summary>

```mermaid
flowchart TD
    A["[]float32 / []float64 field\n(>= 32 elems, OptLossyVec set)"] --> EX["collect NaN/Inf\ninto exception list\nzero them for the pipeline"]
    EX --> ROT["randomized Hadamard rotation\nR = (1/sqrt(n))*H*D, seed-driven\nspreads outliers -> ~Gaussian"]
    ROT --> DELTA["budget -> step delta\nMSE ~= delta^2 * G_lattice\nverify-loop tightens to meet budget"]
    DELTA --> SC["scalar quantizer\nround each coord (Z lattice)"]
    DELTA --> E8GATE{"pdim >= 16\nAND relTarget <= 0.04?"}
    E8GATE -->|yes| E8["E8 lattice quantizer\nnearest of D8 and D8+1/2\n8-D blocks + 1 coset bit/block"]
    E8GATE -->|no| SKIP["skip E8\n(coset overhead can't pay off)"]
    SC --> RANS1["rANS entropy-code\nzigzag-varint coords"]
    E8 --> RANS2["rANS entropy-code\nzigzag-varint coords"]
    RANS1 --> PICK{"keep smaller\nthat meets budget"}
    RANS2 --> PICK
    PICK --> NW{"smaller than\nlossless float encode?"}
    NW -->|yes| OUT["0xFD block\n(variant + coords + cosets? + exceptions)"]
    NW -->|no| LOSSLESS["fall back to\nlossless float codec"]
```

</details>

Two structural edges over GPU KV-cache quantizers (TurboQuant et al.), which are
forced into fixed-width scalar codebooks: qdf is a CPU serializer, so it can
(1) **entropy-code** the quantization indices — after the rotation they are
near-Gaussian, so an order-0 rANS pass recovers the bits a fixed-width code
wastes — and (2) use a **lattice** (E8) whose Voronoi region is rounder than the
scalar cube. The `delta` step is chosen from the budget by a closed-form
distortion model and then verified on the data so the achieved error meets the
guarantee.

## Variant selection and decode

<img src="svg/lossy-vector-2.svg" alt="Lossy vector wire format and decode flowchart">

<details><summary>Mermaid source</summary>

```mermaid
flowchart LR
    subgraph WIRE["0xFD wire block"]
        T["tag 0xFD"] --> F["flags u8\nbit0 elemF32\nbits1-2 variant"]
        F --> H["dim, count\nseed u64, delta f64"]
        H --> C["coords\n(rANS varint ints)"]
        C --> CO["cosets\n(E8 only,\n1 bit / 8-D block)"]
        CO --> E["exceptions\nnExc x (idx, idx, bits)"]
    end
    E --> D{"variant\nin flags?"}
    D -->|scalar| DS["rANS decode -> dequantize\n-> inverse Hadamard"]
    D -->|E8| DE["rANS decode -> E8 reconstruct\n(+0.5 per coset bit)\n-> inverse Hadamard"]
    DS --> R["restore NaN/Inf\nfrom exception list"]
    DE --> R
    R --> V["[]float32 / []float64\n(approximate)"]
```

</details>

The chosen quantizer is recorded in `flags` bits 1-2, so the decoder needs no
side channel. Decode bounds every allocation by `dim`/`count` read from the
wire, validates the coset stream length exactly, and rejects an unknown variant.
