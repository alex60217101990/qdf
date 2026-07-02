// Package cgsample contains the user-defined types used to exercise the
// qdfgen code generator. The actual *_qdf.go file is produced by the
// TestGenerate test which invokes the in-process Generate() entry point.
package cgsample

import (
	"errors"
	"time"
)

var qdfErrShort = errors.New("cgsample: short tag")

type Inner struct {
	X int     `qdf:"x"`
	Y float64 `qdf:"y"`
}

type Sample struct {
	Name   string            `qdf:"name"`
	Age    int               `qdf:"age"`
	Active bool              `qdf:"active"`
	Score  float64           `qdf:"score"`
	Tags   []string          `qdf:"tags"`
	Meta   map[string]string `qdf:"meta"`
	Inner  Inner             `qdf:"inner"`
	When   time.Time         `qdf:"when"`
	Buf    []byte            `qdf:"buf"`
	OptPtr *Inner            `qdf:"opt"`
	Counts [3]int32          `qdf:"counts"`
}

// Label is a defined string type, used both as a map key (the intern fast
// path) and as a pointed-to named scalar — two spots the generator must emit
// an explicit type conversion for.
type Label string

// EmbeddedBase is embedded by value into Edge; its exported fields must be
// flattened into Edge's wire layout, matching the reflect path.
type EmbeddedBase struct {
	BaseA int    `qdf:"base_a"`
	BaseB string `qdf:"base_b"`
}

// Edge exercises three generator paths the reflect path already handles but the
// codegen previously mis-emitted: anonymous embedded value-struct flattening,
// a map keyed by a defined string type, and a pointer to a defined scalar.
type Edge struct {
	EmbeddedBase
	Labels map[Label]int `qdf:"labels"`
	Ptr    *Label        `qdf:"ptr"`
	Tail   int           `qdf:"tail"`
}

// GenMetric is an all-scalar struct: the columnar-eligible element shape that
// triggers the monomorphized transpose path in generated encode/decode.
type GenMetric struct {
	TS    int64   `qdf:"ts"`
	Value float64 `qdf:"value"`
	Count uint32  `qdf:"count"`
	OK    bool    `qdf:"ok"`
	Ratio float32 `qdf:"ratio"`
}

// GenMetricBatch carries a slice of the scalar element — the field that
// triggers columnar codegen.
type GenMetricBatch struct {
	Name    string      `qdf:"name"`
	Metrics []GenMetric `qdf:"metrics"`
}

// --- Phase 2b columnar codegen fixtures: string / time / hybrid columns ---

// GenEvent mixes a time.Time, string, and numeric column — a pure columnar
// element (every field eligible; numeric/time present so no probe).
type GenEvent struct {
	TS    time.Time `qdf:"ts"`
	Level string    `qdf:"level"`
	Code  int32     `qdf:"code"`
	Msg   string    `qdf:"msg"`
}

// GenEventLog wraps an event slice (the columnar-eligible field).
type GenEventLog struct {
	Source string     `qdf:"source"`
	Events []GenEvent `qdf:"events"`
}

// GenRowInner is a nested struct used as a residual (non-columnar) field.
type GenRowInner struct {
	X int    `qdf:"x"`
	Y string `qdf:"y"`
}

// GenRow exercises the HYBRID frame: scalar + string columns plus a residual
// nested struct and a residual map.
type GenRow struct {
	ID    int64          `qdf:"id"`
	Name  string         `qdf:"name"`
	Inner GenRowInner    `qdf:"inner"`
	Tags  map[string]int `qdf:"tags"`
}

// GenRowSet wraps a hybrid-element slice.
type GenRowSet struct {
	Rows []GenRow `qdf:"rows"`
}

// GenName is a string-only element: the cardinality probe decides columnar vs
// row-major at encode time.
type GenName struct {
	First string `qdf:"first"`
	Last  string `qdf:"last"`
}

// GenNameList wraps a string-only-element slice.
type GenNameList struct {
	Names []GenName `qdf:"names"`
}

// --- Phase 2d fixtures: []byte columns + nullable (*T) columns ---

// GenBlobRow exercises a []byte column (routed through the string column codec).
type GenBlobRow struct {
	ID   int64  `qdf:"id"`
	Data []byte `qdf:"data"`
}

// GenBlobSet wraps a []byte-column-bearing slice.
type GenBlobSet struct {
	Rows []GenBlobRow `qdf:"rows"`
}

// GenOpt exercises nullable scalar/string columns (presence bitmap + dense).
type GenOpt struct {
	A *int32   `qdf:"a"`
	B *string  `qdf:"b"`
	C *bool    `qdf:"c"`
	D *float64 `qdf:"d"`
}

// GenOptSet wraps a nullable-column slice.
type GenOptSet struct {
	Rows []GenOpt `qdf:"rows"`
}

// GenTrailed places fields AFTER the columnar []struct field, exercising the
// colMaxLen reset (a sibling decoded on the shared decoder must not inherit the
// columnar length bound).
type GenTrailed struct {
	Rows []GenMetric `qdf:"rows"`
	Note string      `qdf:"note"`
	Tail []int64     `qdf:"tail"`
}

// GenTag is a defined STRING type with a hand-written codec — a field of it must
// route through MarshalQDF/UnmarshalQDF in the generated code, not be emitted as
// a bare string (which bypasses the codec and diverges from the reflect path).
type GenTag string

// Self-delimiting codec: a 'T' marker byte, a 1-byte length, then the raw bytes.
// (EncodeNested/DecodeNested do not length-frame the body, so the codec must
// report exactly how many bytes it consumed.)
func (t GenTag) MarshalQDF(dst []byte) ([]byte, error) {
	dst = append(dst, 'T', byte(len(t)))
	return append(dst, t...), nil
}

func (t *GenTag) UnmarshalQDF(src []byte) (int, error) {
	if len(src) < 2 || src[0] != 'T' {
		return 0, qdfErrShort
	}
	n := int(src[1])
	if len(src) < 2+n {
		return 0, qdfErrShort
	}
	*t = GenTag(src[2 : 2+n])
	return 2 + n, nil
}

// GenNamedCodec uses the named-non-struct codec type as a field.
type GenNamedCodec struct {
	Label GenTag `qdf:"label"`
	N     int64  `qdf:"n"`
}

// GenFloatMap exercises the OptCanonical sorted-emit path for float-keyed maps,
// whose canonical order is by Float64bits and whose values must be carried as
// pairs (a NaN key is unfindable by map index).
type GenFloatMap struct {
	M map[float64]string `qdf:"m"`
}

// GenAnyBox exercises the generated any-field path. The generator routes an
// `any` field through the encoder's dynamic entry point; that entry MUST mark a
// schemaless context (ifaceDepth>0) so a []float32/[]float64 or batchable
// []struct held in Val does not emit an OptLossyVec tagColVecLossy (0xFD) /
// tagVecBatchStruct (0xFE) block — decodeAny (used for any values) cannot read
// those. Regression fixture for the codegen EncodeValue-vs-EncodeAny gap.
type GenAnyBox struct {
	ID  int64 `qdf:"id"`
	Val any   `qdf:"val"`
}

// GenEmbedTime embeds time.Time by value. The generator must NOT flatten it
// (its fields are unexported — flattening yields zero fields and drops the
// value); it must encode as one named "Time" field via the timestamp codec,
// exactly as the reflect path does. Regression for the appendFields gate.
type GenEmbedTime struct {
	time.Time
	N int64 `qdf:"n"`
}
