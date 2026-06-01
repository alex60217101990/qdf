# Wire Format

## What this shows

The exact byte layout of a qdf buffer: the 5-byte header, the flags bit map,
and the complete tag space. Every qdf buffer is self-describing — the decoder
reads mode, codec hints, and compression state entirely from the header and
in-body tags; no out-of-band schema is required.

## Buffer layout

```
[ byte 0 ] [ byte 1 ] [ byte 2 ] [ byte 3  ] [ byte 4    ] [ byte 5 .. N ]
  0x51 'Q'   0x44 'D'   0x46 'F'   0x01 ver   flags byte    tagged body
```

```mermaid
flowchart LR
    B0["byte 0\n0x51 'Q'\nMagic0"]
    B1["byte 1\n0x44 'D'\nMagic1"]
    B2["byte 2\n0x46 'F'\nMagic2"]
    B3["byte 3\n0x01\nVersion1"]
    B4["byte 4\nflags\n(see bit map)"]
    Body["bytes 5..N\ntagged body\n(or rANS stream\nif FlagRANS set)"]
    B0 --> B1 --> B2 --> B3 --> B4 --> Body
```

Byte positions 0–4 are the fixed header. Everything from byte 5 onward is
the tagged body (or, if `FlagRANS` is set: `varuint(origLen)` + 256-entry
frequency table + rANS stream).

## Flags byte (byte 4) bit map

```
bit:  7   6   5   4      3            2          1          0
      ---reserved---   FlagColIndex  FlagRANS  FlagQPack  FlagDense
                         0x08         0x04       0x02       0x01
```

| Flag | Hex | Meaning |
|------|-----|---------|
| `FlagDense` | `0x01` | Body uses Dense intern dialect; state-ref tags (`0xE0`–`0xEA`, `0xEC`) are present |
| `FlagQPack` | `0x02` | Body may carry QPack codec tags (`0xE3`–`0xEF`, `0xF4`–`0xF5`); early hint so readers can reject unsupported codecs before parsing |
| `FlagRANS` | `0x04` | Body is rANS-compressed: `varuint(origLen)` + 256-entry freq table + rANS stream. Decoder decompresses before reading tags. Set only when rANS form is strictly smaller. |
| `FlagColIndex` | `0x08` | A `tagColStruct` payload carries a fixed-width column-length index (K × `uint32` LE) right after the shape declaration. Backpatched by the encoder; never set on non-columnar payloads. |

## Tag space

**Scalar / structural tags** (present in both Fast and Dense modes):

| Range | Name | Description |
|-------|------|-------------|
| `0x00–0x7F` | fixint | Positive integer; value is the tag byte |
| `0x80–0x9F` | fixstr | String; length 0..31 packed into low 5 bits |
| `0xA0–0xBF` | fixarr | Array; length 0..31 packed into low 5 bits |
| `0xC0` | nil | Null value |
| `0xC1, 0xC2` | false, true | Boolean |
| `0xC3–0xC6` | uint8..uint64 | Unsigned integers |
| `0xC7–0xCA` | int8..int64 | Signed integers |
| `0xCB, 0xCC` | float32, float64 | IEEE-754, raw LE bits |
| `0xCD–0xCF` | str8, str16, str32 | Longer strings |
| `0xD0–0xD2` | bin8, bin16, bin32 | Byte slices |
| `0xD3, 0xD4` | arr16, arr32 | Longer arrays |
| `0xD5–0xD7` | map8, map16, map32 | Maps |
| `0xD8–0xDF` | negfixint | Negative integer -1 .. -8 |

**Dense-mode tags** (present when `FlagDense` set):

| Tag | Hex | Wire payload |
|-----|-----|--------------|
| `tagInternStr` | `0xE0` | First occurrence of interned string: `varuint(len)` + bytes |
| `tagStateRef` | `0xE1` | Reference to intern ID: `varuint(id)` |
| `tagInternBin` | `0xE2` | First occurrence of interned `[]byte`: `varuint(len)` + bytes |
| `tagStateRepeat` | `0xE8` | Markov-0: id == lastID; 1 byte total, no payload |
| `tagStateMTF` | `0xE9` | MTF rank: `varuint(rank)` |
| `tagStatePair` | `0xEA` | Markov-1 hit: `varuint(rank=0)` (top-1, rank always 0) |
| `tagMapShape` | `0xEC` | Struct shape reference: `varuint(shapeID)` |

**QPack codec tags** (present when `FlagQPack` set):

| Tag | Hex | Description |
|-----|-----|-------------|
| `tagPackBool` | `0xE3` | Bitpacked `[]bool`, 1 bit/elem |
| `tagPackRaw` | `0xE4` | Raw little-endian numeric slice |
| `tagPackFor` | `0xE5` | Frame-of-Reference bitpacked integer slice |
| `tagPackDeltaFor` | `0xE6` | Delta + zigzag + FOR integer slice |
| `tagPackGorilla` | `0xE7` | Gorilla XOR-coded `[]float64` / `[]float32` |
| `tagPackRLE` | `0xEB` | Run-length encoded integer slice |
| `tagPackDict` | `0xED` | Dictionary-coded integer slice (≤64 distinct) |
| `tagPackPFor` | `0xEE` | Patched FOR: narrow body + outlier exception list |
| `tagColStruct` | `0xEF` | Columnar container for `[]struct` |
| `tagPackALP` | `0xF4` | ALP decimal-coded `[]float64` |
| `tagColStrDict` | `0xF5` | Dictionary-coded string column (inside `tagColStruct`) |

**Other**:

| Tag | Hex | Description |
|-----|-----|-------------|
| ext8/16/32 | `0xF0–0xF2` | User extension envelope |
| `tagTimestamp` | `0xF3` | `zigzag(sec)` + `varuint(nsec)` |

## tagColStruct body layout

When `tagColStruct` (0xEF) is present, the body following it is:

```mermaid
flowchart LR
    A["0xEF\ntagColStruct"] --> B["varuint(M)\nrow count"]
    B --> C{"shapeID?"}
    C -->|"0 = new shape"| D["varuint(K)\ncolumn count\nK × name+kind pairs"]
    C -->|"N > 0 = cached"| E["(shape already known)"]
    D --> F
    E --> F
    F{"FlagColIndex\nin header?"}
    F -->|yes| G["K × uint32 LE\ncolumn byte lengths"]
    F -->|no| H["(no index)"]
    G --> I["column 0 body\n(FOR/bitpack/dict/…)"]
    H --> I
    I --> J["column 1 body"]
    J --> K["…\ncolumn K-1 body"]
```

The column-length index appears only when `FlagColIndex` is set, enabling
O(1) skip of unwanted columns during selective decode.
