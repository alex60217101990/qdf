# qdfgen

Code generator that emits reflection-free `MarshalQDF` / `UnmarshalQDF`
methods for user struct types.

## Install

```bash
go install github.com/alex60217101990/qdf/cmd/qdfgen@latest
```

## Usage

From a Go source file:

```go
//go:generate qdfgen -type Event,User .
type Event struct {
    ID     int    `qdf:"id"`
    Source string `qdf:"source"`
}
```

Then `go generate ./...`.

CLI flags:

| Flag       | Default                                      | Description                                       |
| ---------- | -------------------------------------------- | ------------------------------------------------- |
| `-type`    | required                                     | Comma-separated struct types to generate for.     |
| `-output`  | `<pkgname>_qdf.go` in the source directory   | Override the output file path.                    |
| `-outdir`  | source package directory                     | Override the output directory (when -output has no path component). |
| `-v`       | off                                          | Verbose progress log to stderr.                   |

Positional argument is the package pattern (defaults to `.`):

```bash
qdfgen -type Event,User ./internal/events
```

## What it generates

For every requested type the tool produces methods on `*T`:

```go
func (v *Event) MarshalQDF(dst []byte) ([]byte, error)
func (v *Event) UnmarshalQDF(src []byte) (int, error)
```

`MarshalQDF` appends to `dst` (pass `nil` for a fresh encoding).
`UnmarshalQDF` returns the number of bytes consumed.

The generated code uses the public `qdf` API only:
`NewEncoderOnBuf`, `NewDecoderOnBuf`, `Decoder.InternKey`,
`Decoder.CheckLength`, pre-encoded field-name headers, etc. There is no
`reflect.*` on the hot path.

## Columnar `[]struct` fields

A `[]NamedStruct` field whose element is columnar-eligible is emitted as a
**monomorphized columnar transpose** instead of a row-major per-element loop:
the struct slice is transposed to per-column arrays at compile-time-known field
offsets and each column is written through `qdf`'s adaptive codecs (FOR / Delta /
RLE / dictionary / PFOR for numbers; dict / FSST / raw-slab / constant for
strings) — zero reflection, recovering the columnar wire win a `Marshaler`
element otherwise loses (a generated type can't use the reflect columnar path).

- **Eligible columns**: scalar `int*`/`uint*`/`float*`/`bool`, `string`,
  `time.Time`, `[]byte`, and pointers to scalar/string (`*T` → a **nullable**
  column with a presence bitmap; `nil` vs empty stays distinct).
- **Pure** frame (`tagColStruct`, `0xEF`) when every field is eligible;
  **hybrid** frame (`tagHybridColStruct`, `0xF7`) when some fields (nested
  struct, map, non-byte slice, `*struct`, interface) stay as a per-row residual
  tail.
- **Gated**: fires only at `len >= 16`; shorter/`nil` slices stay row-major. A
  string-only element runs a cardinality probe so high-cardinality strings stay
  row-major (columnar would not shrink them). The probe scores all string
  columns together (one aggregate decision), matching the reflect `columnarProbe`.
- **Custom codecs stay row-major**: an element type with a hand-written
  `MarshalQDF`/`UnmarshalQDF` is *never* columnar-transposed — the transpose
  would replay the struct fields and bypass the custom codec. It is emitted
  row-major through that codec, matching the reflect path. Only plain structs and
  types `qdfgen` itself generates (whose emitted codec *is* the columnar layout)
  are transposed.
- The wire layout is byte-identical to the reflect columnar path, so a generated
  type cross-decodes with reflect `Unmarshal` (which delegates to the generated
  method anyway).

## Supported types

- All primitives (`bool`, every `int*`/`uint*`, `float32`, `float64`,
  `string`, `[]byte`).
- Slices, fixed arrays, maps, pointers, nested structs.
- `time.Time` (encoded as a 64-bit Unix-nanoseconds timestamp).
- Tags: `qdf:"name"`, falling back to `json:"name"`, falling back to
  the Go field name. `qdf:"-"` skips a field.

Cycles through value-typed fields (`type T struct { Next T }`) are
rejected; cycles through pointer-typed fields are fine.

## Build prerequisites

Go 1.26 (uses `golang.org/x/tools/go/packages`). The generator lives in
its own module so its dependencies do not leak into the `qdf` package.
