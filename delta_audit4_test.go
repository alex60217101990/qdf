package qdf

import (
	"maps"
	"reflect"
	"slices"
	"testing"
)

// TestColDiffInternLeakAcrossField is the audit-4 RED repro for the columnar
// never-larger intern leak: diffColumnar builds a positional baseline under
// normal Dense interning, then (when the column body wins) discards those bytes
// but leaves the assigned intern ids live in enc.state. A later field that
// reuses one of those string values emits a state-ref to an id whose
// tagInternStr definition lived only in the discarded positional bytes, so the
// decoder hits ErrUnknownStateID — a valid Diff produces an undecodable patch.
//
// The struct places a columnar []struct with a changing string column BEFORE a
// map field whose value reuses one of the column's strings ("green").
func TestColDiffInternLeakAcrossField(t *testing.T) {
	type colE struct {
		N int32
		S string
	}
	type colR struct {
		Cols []colE
		M    map[int32]string
	}
	palette := []string{"red", "green", "blue"}
	const n = 32

	mkOld := func() colR {
		r := colR{Cols: make([]colE, n), M: map[int32]string{1: "x"}}
		for i := range n {
			r.Cols[i] = colE{int32(i), palette[i%3]}
		}
		return r
	}
	mkNew := func() colR {
		r := colR{Cols: make([]colE, n), M: map[int32]string{1: "green"}}
		for i := range n {
			r.Cols[i] = colE{int32(i), palette[(i+1)%3]}
		}
		return r
	}

	for _, opt := range []Options{OptBalanced, OptCompression, OptDense | OptBalanced} {
		old, nw := mkOld(), mkNew()
		patch, err := Diff(old, nw, opt)
		if err != nil {
			t.Fatalf("opt %v: Diff: %v", opt, err)
		}
		base := colR{Cols: slices.Clone(old.Cols), M: maps.Clone(old.M)}
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opt %v: Apply rejected a valid patch: %v", opt, err)
		}
		if !reflect.DeepEqual(base, nw) {
			t.Fatalf("opt %v: round-trip mismatch\n got %+v\nwant %+v", opt, base, nw)
		}
	}
}

// a4InnerTok is the stable per-type shape token (as qdfgen emits per type).
var a4InnerTok byte

// a4Inner is a hand-rolled EncoderMarshaler/DecoderUnmarshaler (the shape a
// qdfgen-generated struct takes) that uses Encoder.StructShape — the one
// wire-state id-assigning site the keyed never-larger trial failed to gate.
type a4Inner struct{ W int64 }

func (n a4Inner) EncodeQDF(e *Encoder) error {
	e.StructShape(&a4InnerTok, [][]byte{append([]byte{tagFixstr | 1}, 'W')})
	e.WriteInt(n.W)
	return nil
}

func (n a4Inner) MarshalQDF(dst []byte) ([]byte, error) {
	e := NewEncoderOnBuf(dst, Fast)
	e.EnsureHeader()
	if err := n.EncodeQDF(e); err != nil {
		return nil, err
	}
	return e.Bytes(), nil
}

func (n *a4Inner) DecodeQDF(d *Decoder) error {
	if _, _, _, err := d.ReadStructHeader(); err != nil {
		return err
	}
	w, err := d.ReadInt()
	if err != nil {
		return err
	}
	n.W = w
	return nil
}

func (n *a4Inner) UnmarshalQDF(src []byte) (int, error) {
	d := NewDecoderOnBuf(src)
	if err := d.readHeader(); err != nil {
		return 0, err
	}
	if err := n.DecodeQDF(d); err != nil {
		return 0, err
	}
	return len(src), nil
}

// TestKeyedStructShapeNoLeak is the audit-4 RED repro for the keyed never-larger
// trial leaking a StructShape id: a keyed-slice element whose Sub field is an
// EncoderMarshaler using StructShape. During diffKeyedSlice's suspended trial,
// the discarded positional candidate binds the shape token; the kept keyed
// candidate then emits an id-only tagMapShape reference whose declaration lived
// only in the discarded bytes, so Apply of a valid Diff hits ErrUnknownStateID.
func TestKeyedStructShapeNoLeak(t *testing.T) {
	type a4Elem struct {
		K   string  `qdf:"id,key"`
		Sub a4Inner `qdf:"sub"`
	}
	type a4Holder struct {
		Items []a4Elem `qdf:"items"`
		// P and Q sit AFTER the keyed slice and share a4Inner's shape token, so
		// the first declares the shape and the second references it by id. If the
		// trial leaves the shape counter mis-based, that post-slice ref desyncs.
		P a4Inner `qdf:"p"`
		Q a4Inner `qdf:"q"`
	}

	// 30 keyed elements rotated by one (so a positional diff re-encodes every
	// row, but a keyed diff only re-encodes the few genuinely changed rows and
	// wins the never-larger trial), with two rows changing their Sub.W. The
	// re-encoded rows drive Encoder.StructShape during the suspended trial.
	const m = 200
	key := func(i int) string { return string(rune('A'+i%26)) + string(rune('0'+i/26)) }
	oldItems := make([]a4Elem, m)
	for i := range m {
		oldItems[i] = a4Elem{key(i), a4Inner{int64(i)}}
	}
	newItems := make([]a4Elem, m)
	for i := range m {
		src := (i + 1) % m // rotate by one
		newItems[i] = oldItems[src]
	}
	newItems[3].Sub.W = 9990
	newItems[7].Sub.W = 9991
	old := a4Holder{Items: oldItems, P: a4Inner{100}, Q: a4Inner{100}}
	nw := a4Holder{Items: newItems, P: a4Inner{200}, Q: a4Inner{200}}

	for _, opt := range []Options{OptBalanced, OptCompression, OptDense | OptBalanced} {
		patch, err := Diff(old, nw, opt)
		if err != nil {
			t.Fatalf("opt %v: Diff: %v", opt, err)
		}
		base := a4Holder{Items: slices.Clone(old.Items), P: old.P, Q: old.Q}
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opt %v: Apply rejected a valid patch: %v", opt, err)
		}
		if !reflect.DeepEqual(base, nw) {
			t.Fatalf("opt %v: round-trip mismatch\n got %+v\nwant %+v", opt, base, nw)
		}
	}
}
