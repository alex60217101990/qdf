package bench

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	msgpack "github.com/vmihailenco/msgpack/v5"

	qdf "github.com/alex60217101990/qdf"
)

// Differential testing: qdf must agree with msgpack and encoding/json
// on the *meaning* of a round-tripped payload. Wire bytes differ —
// qdf is its own format — but the decoded Go value should match what
// msgpack/json produce for the same input. Catches semantic drift
// that single-codec tests cannot.

type diffPayload struct {
	ID     int               `json:"id"     msgpack:"id"     qdf:"id"`
	Name   string            `json:"name"   msgpack:"name"   qdf:"name"`
	Score  float64           `json:"score"  msgpack:"score"  qdf:"score"`
	Tags   []string          `json:"tags"   msgpack:"tags"   qdf:"tags"`
	Vec    []float64         `json:"vec"    msgpack:"vec"    qdf:"vec"`
	Active bool              `json:"active" msgpack:"active" qdf:"active"`
	Attrs  map[string]string `json:"attrs"  msgpack:"attrs"  qdf:"attrs"`
}

func mkDiffPayload() diffPayload {
	return diffPayload{
		ID:     12345,
		Name:   "differential-fixture",
		Score:  3.14159,
		Tags:   []string{"prod", "eu-west-1", "v3"},
		Vec:    []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0},
		Active: true,
		Attrs:  map[string]string{"version": "v3.42.1", "host": "node-001"},
	}
}

func payloadEqualSemantic(a, b diffPayload) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Active != b.Active {
		return false
	}
	if math.Float64bits(a.Score) != math.Float64bits(b.Score) {
		return false
	}
	if !reflect.DeepEqual(a.Tags, b.Tags) {
		return false
	}
	if !reflect.DeepEqual(a.Vec, b.Vec) {
		return false
	}
	if !reflect.DeepEqual(a.Attrs, b.Attrs) {
		return false
	}
	return true
}

func TestDiff_QDFAgreesWithJSON(t *testing.T) {
	in := mkDiffPayload()
	jb, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var jOut diffPayload
	if err := json.Unmarshal(jb, &jOut); err != nil {
		t.Fatal(err)
	}
	qb, err := qdf.Marshal(in, qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var qOut diffPayload
	if err := qdf.Unmarshal(qb, &qOut); err != nil {
		t.Fatal(err)
	}
	if !payloadEqualSemantic(jOut, qOut) {
		t.Fatalf("qdf differs from json:\n json=%+v\n  qdf=%+v", jOut, qOut)
	}
}

func TestDiff_QDFAgreesWithMsgpack(t *testing.T) {
	in := mkDiffPayload()
	mb, err := msgpack.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var mOut diffPayload
	if err := msgpack.Unmarshal(mb, &mOut); err != nil {
		t.Fatal(err)
	}
	qb, err := qdf.Marshal(in, qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var qOut diffPayload
	if err := qdf.Unmarshal(qb, &qOut); err != nil {
		t.Fatal(err)
	}
	if !payloadEqualSemantic(mOut, qOut) {
		t.Fatalf("qdf differs from msgpack:\n msgpack=%+v\n     qdf=%+v", mOut, qOut)
	}
}

func TestDiff_AllThreeAgree(t *testing.T) {
	in := mkDiffPayload()
	jb, _ := json.Marshal(in)
	mb, _ := msgpack.Marshal(in)
	for _, opts := range []qdf.Options{qdf.OptSpeed, qdf.OptQPack, qdf.OptBalanced} {
		qb, err := qdf.Marshal(in, opts)
		if err != nil {
			t.Fatal(err)
		}
		var jOut, mOut, qOut diffPayload
		if err := json.Unmarshal(jb, &jOut); err != nil {
			t.Fatal(err)
		}
		if err := msgpack.Unmarshal(mb, &mOut); err != nil {
			t.Fatal(err)
		}
		if err := qdf.Unmarshal(qb, &qOut); err != nil {
			t.Fatal(err)
		}
		if !payloadEqualSemantic(jOut, qOut) || !payloadEqualSemantic(mOut, qOut) {
			t.Fatalf("three-way disagreement:\n json=%+v\n msgpack=%+v\n qdf=%+v", jOut, mOut, qOut)
		}
	}
}

// Edge cases: NaN, ±Inf, +0/-0 should be preserved on the qdf side
// per its lossless float guarantee. JSON does not encode NaN/Inf;
// msgpack does. We only assert qdf-vs-msgpack here for those.
func TestDiff_QDFAgreesWithMsgpack_FloatEdgeCases(t *testing.T) {
	type fEdge struct {
		Vals []float64 `msgpack:"vals" qdf:"vals"`
	}
	in := fEdge{Vals: []float64{0, math.Copysign(0, -1), math.NaN(), math.Inf(1), math.Inf(-1), 1.5, -2.25}}
	mb, err := msgpack.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	qb, err := qdf.Marshal(in, qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var mOut, qOut fEdge
	if err := msgpack.Unmarshal(mb, &mOut); err != nil {
		t.Fatal(err)
	}
	if err := qdf.Unmarshal(qb, &qOut); err != nil {
		t.Fatal(err)
	}
	if len(mOut.Vals) != len(qOut.Vals) {
		t.Fatalf("len differs: %d vs %d", len(mOut.Vals), len(qOut.Vals))
	}
	for i, v := range in.Vals {
		mv := mOut.Vals[i]
		qv := qOut.Vals[i]
		if math.IsNaN(v) {
			if !math.IsNaN(qv) {
				t.Fatalf("[%d] qdf lost NaN: %v", i, qv)
			}
			if !math.IsNaN(mv) {
				t.Fatalf("[%d] msgpack lost NaN: %v", i, mv)
			}
			continue
		}
		if math.Float64bits(qv) != math.Float64bits(v) {
			t.Fatalf("[%d] qdf bit-pattern drift: orig=%x qdf=%x", i, math.Float64bits(v), math.Float64bits(qv))
		}
	}
}

// Integer boundary cases. JSON spec restricts to IEEE-754 double
// precision, msgpack supports the full int64/uint64 range. Limit
// the comparison to values JSON can represent precisely.
func TestDiff_IntegerBoundaries(t *testing.T) {
	type intHolder struct {
		I32 int32  `json:"i32"  msgpack:"i32"  qdf:"i32"`
		I64 int64  `json:"i64"  msgpack:"i64"  qdf:"i64"`
		U32 uint32 `json:"u32"  msgpack:"u32"  qdf:"u32"`
		U53 uint64 `json:"u53"  msgpack:"u53"  qdf:"u53"`
	}
	in := intHolder{
		I32: math.MaxInt32,
		I64: 1 << 52, // within IEEE double precision
		U32: math.MaxUint32,
		U53: 1 << 53,
	}
	jb, _ := json.Marshal(in)
	mb, _ := msgpack.Marshal(in)
	qb, _ := qdf.Marshal(in, qdf.OptSpeed)
	var jO, mO, qO intHolder
	if err := json.Unmarshal(jb, &jO); err != nil {
		t.Fatal(err)
	}
	if err := msgpack.Unmarshal(mb, &mO); err != nil {
		t.Fatal(err)
	}
	if err := qdf.Unmarshal(qb, &qO); err != nil {
		t.Fatal(err)
	}
	if jO != mO || jO != qO {
		t.Fatalf("integer disagreement:\n json=%+v\n msgpack=%+v\n qdf=%+v", jO, mO, qO)
	}
}
