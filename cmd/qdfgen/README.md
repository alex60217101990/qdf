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
