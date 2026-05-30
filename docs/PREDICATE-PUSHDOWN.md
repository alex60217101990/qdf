# Predicate pushdown — filter and project a batch inside `Unmarshal`

> **The one thing no other Go serializer does.** With qdf you can read back a
> *filtered, projected subset* of a `[]struct` batch — "give me `ts` and `code`
> for the rows where `level == "ERROR"` and `code >= 500`" — **without first
> decoding the whole batch into memory.** It is a `SELECT … WHERE …` over a
> serialized blob, single-pass, with no database, no index server, no second
> copy, and zero per-value boxing.

This document is the full guide: why it matters, how to use it, the semantics
and error model, performance, and how it works under the hood.

- [Why this is unique](#why-this-is-unique)
- [The API](#the-api)
- [Tutorial](#tutorial)
  - [1. Produce a columnar batch](#1-produce-a-columnar-batch)
  - [2. Filter rows with `Where`](#2-filter-rows-with-where)
  - [3. AND multiple predicates](#3-and-multiple-predicates)
  - [4. Project columns with `Select`](#4-project-columns-with-select)
  - [5. Decode into a map instead of a struct](#5-decode-into-a-map-instead-of-a-struct)
- [Semantics](#semantics)
- [Error model](#error-model)
- [Performance](#performance)
- [How it works](#how-it-works)
- [Limitations and roadmap](#limitations-and-roadmap)

---

## Why this is unique

Every mainstream Go serializer — `encoding/json`, `vmihailenco/msgpack`,
`google.golang.org/protobuf`, `encoding/gob`, `fxamacker/cbor` — has the same
shape on decode: you hand it bytes and a destination, and it reconstructs
**the entire value**. If a 50 000-row telemetry batch has one row you care
about, you decode all 50 000 rows, then loop in Go to throw away 49 999 of
them. You pay the full allocation, the full copy, and the full CPU for data
you immediately discard.

qdf is different because under `OptBalanced` it already stores a slice of flat
structs **columnar** — transposed into one contiguous, codec-compressed body
per field (see [GUIDE.md](GUIDE.md#columnar-struct-array-codec-tagcolstruct-0xef)).
That layout is exactly what a columnar database uses, and it unlocks the same
trick:

- **Filter on a column without touching the others.** To test
  `level == "ERROR"` qdf decodes only the `level` column, never the seven other
  columns of a wide row.
- **Materialize only surviving rows.** Rows that fail the predicate are never
  reconstructed into a struct.
- **Project only selected columns.** Columns you don't ask for are skipped
  entirely (with `OptColumnIndex`, by a direct offset seek).

The net effect is "Parquet-style predicate pushdown" available from a plain
`Unmarshal` call, on data you already serialize for transport or storage. No
other Go format ships this.

---

## The API

Three symbols, all in the root package (`query.go`):

```go
// Unmarshal gains a variadic tail of query options.
func Unmarshal(data []byte, out any, opts ...QueryOption) error

// Where keeps only rows whose `field` column satisfies a typed predicate.
// T is the column's element type. Multiple Where options are AND-ed.
func Where[T Queryable](field string, pred func(T) bool) QueryOption

// Select restricts the output to the named columns (projection).
func Select(fields ...string) QueryOption
```

`Queryable` is the set of column element types a predicate may take:

```go
int, int8, int16, int32, int64,
uint, uint8, uint16, uint32, uint64, uintptr,
float32, float64, string, bool
```

`T` is resolved **once**, when you construct the `Where`, into a native
`func(int64) bool` / `func(uint64) bool` / `func(float64) bool` /
`func(string) bool` / `func(bool) bool`. The predicate is then called per row
with the decoder's native scratch value — there is **no `interface{}` boxing
per value**, which is what keeps a million-row scan cheap.

With **no** query options, `Unmarshal` behaves exactly as before — passing
zero `QueryOption`s is the plain decode path, byte-for-byte.

---

## Tutorial

### 1. Produce a columnar batch

Predicate pushdown only fires on a **columnar payload** — a `[]struct` that the
encoder transposed into the columnar container. That happens automatically
under `OptBalanced` once the batch is large and regular enough for the columnar
probe to commit (a handful of rows is too few; a couple dozen is plenty). Add
`OptColumnIndex` so the reader can *seek* past skipped columns instead of
decoding and discarding them:

```go
type Event struct {
    TS    int64  `qdf:"ts"`
    Level string `qdf:"level"`
    Code  int32  `qdf:"code"`
}

events := loadEvents() // a []Event with many rows
buf, err := qdf.Marshal(events, qdf.OptBalanced|qdf.OptColumnIndex)
```

`OptColumnIndex` is a no-op on non-columnar payloads and leaves the default
columnar wire byte-identical when the index is not emitted — so it is safe to
set on the producer unconditionally.

### 2. Filter rows with `Where`

`Where(field, pred)` keeps only the rows for which `pred` returns true. The
predicate's argument type selects the column kind:

```go
var errs []Event
err := qdf.Unmarshal(buf, &errs,
    qdf.Where("level", func(s string) bool { return s == "ERROR" }))
// errs now holds only the ERROR rows, in input order.
```

The filter field **does not have to appear in the output type**. You can keep
`ts` only while filtering on `level`:

```go
type TS struct {
    TS int64 `qdf:"ts"`
}
var hot []TS
_ = qdf.Unmarshal(buf, &hot,
    qdf.Where("level", func(s string) bool { return s == "ERROR" }))
// hot holds the timestamps of the ERROR rows; level itself is never
// materialized into the output.
```

### 3. AND multiple predicates

Pass more than one `Where`; they are combined with logical **AND**. A row
survives only if every predicate accepts it:

```go
var hot []Event
_ = qdf.Unmarshal(buf, &hot,
    qdf.Where("level", func(s string) bool { return s == "ERROR" }),
    qdf.Where("code", func(c int32) bool { return c >= 500 }))
// rows where level == "ERROR" AND code >= 500
```

Note that the `code` predicate takes `int32` — the same width as the wire
column. Any integer width works; qdf converts the decoded value to your
predicate's type before the call.

### 4. Project columns with `Select`

Without a `Select`, the output columns are the fields of your target struct
(matched by `qdf` tag / field name). When you decode into a wide struct but
want only some columns materialized — or when you decode into a map — use
`Select` to name them explicitly:

```go
type Out struct {
    TS   int64 `qdf:"ts"`
    Code int32 `qdf:"code"`
}
var hot []Out
_ = qdf.Unmarshal(buf, &hot,
    qdf.Where("level", func(s string) bool { return s == "ERROR" }),
    qdf.Select("ts", "code"))
```

### 5. Decode into a map instead of a struct

The destination can be `*[]map[string]any` — handy when the schema is dynamic
or you are forwarding data without a Go type for it. Only the projected keys
are populated:

```go
var rows []map[string]any
_ = qdf.Unmarshal(buf, &rows,
    qdf.Where("level", func(s string) bool { return s == "ERROR" }),
    qdf.Select("ts", "level"))
// each map has only "ts" and "level"; "code" is projected out.
```

---

## Semantics

- **AND, not OR.** Multiple `Where` clauses are conjunctive. (OR is not a
  built-in; express it inside a single predicate, e.g.
  `func(c int32) bool { return c == 500 || c == 503 }`.)
- **Input order preserved.** Surviving rows come out in wire (= input) order.
- **Filter field need not be in the output.** You can filter on a column you
  do not project.
- **Nullable columns: `nil` never matches.** A nullable field
  (`*T` for a scalar/string) is stored as a presence bitmap plus a dense
  present-only column. A `nil` row is not a value, so its predicate is never
  called and the row is excluded — even from a predicate like
  `func(int64) bool { return true }`. A nullable column that is only
  *projected* (not filtered) round-trips its `nil`s normally.
- **Base types only.** `Where` dispatches on the concrete `func(T) bool` type
  once at construction. That matches Go base types (`int32`, `string`, …) but
  **not user-defined named types** (`type Level string`). Write the predicate
  over the base type.
- **Zero query options = plain decode.** `Unmarshal(buf, &out)` with no options
  is unchanged.

---

## Error model

Pushdown reports problems as a `*QueryError`, which wraps a sentinel so you can
both categorize (`errors.Is`) and inspect (`errors.As`):

| Condition | Wrapped sentinel |
| --- | --- |
| Payload is not columnar (single struct, or the columnar probe declined) | `ErrUnsupported` |
| A `Where`/`Select` names a field that is not on the wire | `ErrFieldNotFound` |
| A predicate's `T` does not match the column's kind | `ErrTypeMismatch` |

```go
var out []Event
err := qdf.Unmarshal(buf, &out,
    qdf.Where("level", func(v int) bool { return v > 0 })) // level is a string

if errors.Is(err, qdf.ErrTypeMismatch) {
    var qe *qdf.QueryError
    if errors.As(err, &qe) {
        // qe.Field == "level", qe.Want and qe.Got hold the kinds
    }
}
```

Because a non-columnar payload returns `ErrUnsupported`, a defensive caller can
fall back to a full decode + manual filter on that error if it must handle both
columnar and non-columnar inputs.

---

## Performance

On a wide batch — 8 `int64` columns + 1 `string` column, 2000 rows — comparing
pushdown against a full decode followed by a manual filter loop (Intel
i7-9750H, `OptBalanced|OptColumnIndex`, 1% of rows match):

| | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| **pushdown** (`Where` + project 2 cols) | 77,477 | 75,801 | 2022 |
| full decode + manual filter | 161,458 | 309,222 | 2021 |

≈ **2.1× faster and ≈4× fewer bytes moved** — pushdown never materializes the
six columns it neither filters nor projects, and never reconstructs the 99% of
rows that fail the predicate.

Selectivity sweep (same batch, varying the fraction of matching rows):

| match rate | ns/op | allocs/op |
| --- | ---: | ---: |
| 1%  | 70,932 | 2022 |
| 50% | 56,996 | 33 |
| 100% | 98,320 | 2030 |

(Allocations track the number of surviving rows that must be materialized; the
byte and CPU savings hold across the range.) Reproduce with:

```bash
go test -C bench -bench BenchmarkQuery -benchmem -run=^$ .
```

---

## How it works

Pushdown reuses the `tagColStruct` columnar layout and the `OptColumnIndex`
column-length index (see
[GUIDE.md](GUIDE.md#column-length-index--selective-decode-optcolumnindex-flagcolindex)).
The decode is **whole-column**:

1. **Decode each predicate column in full.** With the column index the decoder
   seeks straight to the column body; without it, it decodes-and-discards the
   columns in front of it.
2. **Evaluate the predicate into a row bitmask** — one bit per row, set where
   `func(T) bool` returns true. The call is a direct typed call against the
   decoder's native scratch value (no per-value boxing).
3. **AND the per-predicate bitmasks.** A nullable column clears the `nil`
   rows' bits before the AND, so they never survive.
4. **Compact and project.** Walk the surviving set bits once; for each selected
   column, copy only those rows into the output `[]struct` or
   `[]map[string]any`. Wire order is preserved.

The seek is **column-granular, not element-granular**: a column body is a
single codec payload (Frame-of-Reference / bit-pack / dictionary) with no
per-row byte offsets, so skipping a column is a direct offset add, but skipping
*rows within a column* still requires decoding that column whole. The win comes
from (a) never touching the columns a row is filtered out on, and (b)
materializing only the surviving rows of the projected columns.

---

## Limitations and roadmap

- **Element-addressable seek (deferred).** Today every predicate column is
  decoded in full before any row is tested. A finer scheme — decode only the
  rows that survived earlier predicates — would need per-row addressing inside
  the codec payloads. It is on the roadmap, not yet implemented.
- **Single message only.** Like the column index, pushdown applies to a
  single `Unmarshal` of a columnar `[]struct`. It is not a streaming-mode
  feature (the column index is not emitted in streaming mode).
- **No OR / cross-column predicates as a built-in.** Combine conditions inside
  one predicate, or post-filter the (already much smaller) result.
- **Base types only** for `Where[T]` — named types are not matched.

See also: [USAGE.md](USAGE.md), [CHOOSING.md](CHOOSING.md), and the columnar
deep-dive in [GUIDE.md](GUIDE.md).
