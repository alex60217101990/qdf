# Lossy Vector Codec

qdf includes a lossy codec for `[]float32` and `[]float64` fields that
contain embedding vectors or other high-dimensional float data. The codec
is **opt-in** (disabled by default) and **lossy** — decoded values are
reconstructed approximations, not bit-exact copies.

## Enabling the codec

Add `OptLossyVec` to the encoder's option set and supply a fidelity budget:

```go
enc := qdf.NewEncoderWith(qdf.OptBalanced | qdf.OptLossyVec)
enc.SetVectorBudget(qdf.MinCosine(0.999))

if err := enc.EncodeValue(rows); err != nil {
    log.Fatal(err)
}
data := enc.Bytes()

var out []EmbedRow
if err := qdf.Unmarshal(data, &out); err != nil {
    log.Fatal(err)
}
```

Decoding requires no special flag — `qdf.Unmarshal` (or any `Decoder`)
recognises the `0xFD` tag and reconstructs the vectors automatically.

## Budget knobs

Three constructors set the fidelity target. All are per-vector guarantees.

| Constructor | Meaning |
|---|---|
| `MaxRelError(eps float64)` | Per-vector relative L2 error ≤ eps (e.g. `0.02` = 2 %). |
| `MinCosine(c float64)` | Cosine similarity ≥ c (e.g. `0.999`). Best for embedding search. |
| `TargetSNR(db float64)` | Signal-to-noise ratio ≥ db decibels (e.g. `40`). |

If no budget is set, the default is `MinCosine(0.999)`.

## When the codec fires

The codec is applied only when all of the following are true:

- `OptLossyVec` is set on the encoder.
- The slice has at least 32 elements (`lossyVecMinElems`).
- The encoded result is not larger than the lossless path (never-worse
  guarantee — if quantization would inflate the wire size the encoder
  falls back to the standard float codec automatically).

## Pipeline

For each column of float vectors, the encoder:

1. **Randomised Hadamard transform** — decorrelates the coordinates using a
   seeded random Hadamard rotation, spreading energy evenly before quantization.
2. **Scalar uniform quantization** — chooses the smallest integer bit-width
   whose reconstruction error meets the budget; the quantised integers are
   stored in the `Coords` byte block.
3. **rANS entropy coding** — compresses the `Coords` block using the same
   interleaved rANS stage used by the rest of qdf.

The decoder reverses the steps: rANS decode → dequantize → inverse Hadamard.

## NaN and Inf handling

Non-finite values (`NaN`, `+Inf`, `-Inf`) cannot be quantized. The codec
detects them before encoding, stores them in an **exception list** appended
to the wire block, and replaces them with `0` for the lossy pipeline. After
decoding, the exception values are written back to the reconstructed vectors
at their original positions.

The exception list is always present in the wire format (zero-length when
there are no non-finite values). This means a payload that contains NaN or
Inf round-trips with the exact bit pattern preserved:

```go
v := []float32{1.0, float32(math.NaN()), float32(math.Inf(1))}
// ... encode with OptLossyVec, decode ...
// out[1] is NaN, out[2] is +Inf — bit-exact
```

## Wire format

```
0xFD                         tag byte
flags (u8)                   bit 0: elemF32 (1=float32, 0=float64)
varuint dim                  vector length
varuint count                number of vectors
u64le seed                   Hadamard rotation seed
f64le delta                  quantization step size
varuint coordsLen            byte length of coords block
[coordsLen]byte coords       rANS-compressed quantised integers
varuint nExc                 number of exceptions (0 if none)
nExc × {
    varuint vecIdx           which vector (0-based)
    varuint coordIdx         which coordinate (0-based)
    u32le / u64le bits       raw float32 or float64 bits (per elemF32 flag)
}
```

## Never-worse guarantee

The encoder measures the output of the lossy path and compares it to the
lossless float encoding. If the lossy output is not smaller, it discards it
and writes the lossless encoding instead. The caller sees no API difference;
`OptLossyVec` is a performance hint, not a hard commitment.

## Worked example

```go
package main

import (
    "fmt"
    "log"
    "math"

    "github.com/alex60217101990/qdf"
)

type Doc struct {
    ID  string
    Emb []float32
}

func main() {
    // Build 100 synthetic embedding vectors of dimension 384.
    docs := make([]Doc, 100)
    for i := range docs {
        v := make([]float32, 384)
        for j := range v {
            v[j] = float32(math.Sin(float64(i*384+j) * 0.007))
        }
        docs[i] = Doc{ID: fmt.Sprintf("doc-%d", i), Emb: v}
    }

    // Encode with lossy compression targeting cosine similarity >= 0.999.
    enc := qdf.NewEncoderWith(qdf.OptBalanced | qdf.OptLossyVec)
    enc.SetVectorBudget(qdf.MinCosine(0.999))
    if err := enc.EncodeValue(docs); err != nil {
        log.Fatal(err)
    }
    data := enc.Bytes()
    fmt.Printf("encoded %d docs, wire size %d bytes (raw f32: %d)\n",
        len(docs), len(data), len(docs)*384*4)

    // Decode — no special flag needed.
    var out []Doc
    if err := qdf.Unmarshal(data, &out); err != nil {
        log.Fatal(err)
    }

    // Verify cosine similarity of the first vector.
    var dot, na, nb float64
    for j := range docs[0].Emb {
        a, b := float64(docs[0].Emb[j]), float64(out[0].Emb[j])
        dot += a * b
        na += a * a
        nb += b * b
    }
    fmt.Printf("cosine similarity of doc-0: %.6f\n", dot/(math.Sqrt(na)*math.Sqrt(nb)))
}
```
