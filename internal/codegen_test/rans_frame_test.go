package cgsample

import (
	"fmt"
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// ransRows is prefix-rich so rANS has real redundancy to remove — otherwise the
// assertions below would be measuring an entropy coder that had nothing to do
// and would pass whether or not it ran.
func ransRows(n int) []GenRow {
	out := make([]GenRow, n)
	for i := range out {
		id := fmt.Sprintf("%06d", i)
		out[i] = GenRow{
			ID:   int64(i),
			Name: "com.acme.platform.worker.service." + id,
			Inner: GenRowInner{
				X: i,
				Y: "/opt/acme/platform/bin/worker --shard=" + id,
			},
		}
	}
	return out
}

// A top-level value that is a GENERATED STRUCT must be rANS-framed like any
// other, and today it is not.
//
// reflect_encode.go marks any top-level Marshaler `customFramed`, and
// maybeApplyRANS then declines: "a top-level Marshaler forced Fast framing and
// its bytes are opts-invariant by contract". True of a hand-written Marshaler,
// which emits its own Fast body and ignores Options. NOT true of a generated
// EncoderMarshaler, which writes into the shared encoder and honours its mode —
// it calls StructShape and WriteStringField, both of which respect Dense.
//
// The slice arm is the anchor rather than decoration. The same rows as a
// []GenRow ARE framed, so it proves this data compresses; without it, a test
// that only checks the struct would pass the day rANS stops helping for an
// unrelated reason.
func TestTopLevelGeneratedStructIsRANSFramed(t *testing.T) {
	rows := ransRows(64)
	set := GenRowSet{Rows: rows}

	sliceBal, err := qdf.Marshal(rows, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	sliceRANS, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	if len(sliceRANS) >= len(sliceBal) {
		t.Fatalf("anchor: a slice of the same rows went %d -> %d with rANS — this data "+
			"does not compress, so the struct assertion below would prove nothing",
			len(sliceBal), len(sliceRANS))
	}

	structBal, err := qdf.Marshal(set, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	structRANS, err := qdf.Marshal(set, qdf.OptBalanced|qdf.OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	if len(structRANS) >= len(structBal) {
		t.Errorf("a top-level generated struct went %d -> %d with rANS (no change), "+
			"while the same rows as a slice went %d -> %d (%.1f%% off) — the struct "+
			"is not being framed",
			len(structBal), len(structRANS), len(sliceBal), len(sliceRANS),
			float64(len(sliceBal)-len(sliceRANS))/float64(len(sliceBal))*100)
	}

	// Whatever framing it gets, the value must come back.
	for _, o := range []struct {
		name string
		opts qdf.Options
	}{
		{"balanced", qdf.OptBalanced},
		{"balanced+rans", qdf.OptBalanced | qdf.OptRANS},
		{"compression", qdf.OptCompression},
	} {
		b, err := qdf.Marshal(set, o.opts)
		if err != nil {
			t.Fatalf("%s: %v", o.name, err)
		}
		var got GenRowSet
		if err := qdf.Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: decode: %v", o.name, err)
		}
		if len(got.Rows) != len(rows) {
			t.Fatalf("%s: %d rows, want %d", o.name, len(got.Rows), len(rows))
		}
		for i := range rows {
			if !reflect.DeepEqual(got.Rows[i], rows[i]) {
				t.Fatalf("%s row %d:\n got %+v\nwant %+v", o.name, i, got.Rows[i], rows[i])
			}
		}
	}
}

// handBlob is a HAND-WRITTEN Marshaler in the shape this repository already
// uses for one (see GenTag): a self-delimiting body it writes and reads itself,
// with no reference to the encoder's Options at all.
//
// That is precisely the contract customFramed protects. Reframing such a body
// would stamp FlagRANS onto bytes that were never produced under those options,
// and the fix below must not touch it.
type handBlob struct {
	Payload string
}

func (h handBlob) MarshalQDF(dst []byte) ([]byte, error) {
	dst = append(dst, 'B', byte(len(h.Payload)>>8), byte(len(h.Payload)))
	return append(dst, h.Payload...), nil
}

func (h *handBlob) UnmarshalQDF(src []byte) (int, error) {
	if len(src) < 3 || src[0] != 'B' {
		return 0, qdfErrShort
	}
	n := int(src[1])<<8 | int(src[2])
	if len(src) < 3+n {
		return 0, qdfErrShort
	}
	h.Payload = string(src[3 : 3+n])
	return 3 + n, nil
}

// The protection must survive the fix: a hand-written Marshaler stays unframed.
//
// Its body is opts-invariant by contract, so the bytes come out identical with
// rANS on and off — and they must, or the decoder is handed a FlagRANS frame
// around a payload that was never compressed.
func TestHandWrittenMarshalerIsNotReframed(t *testing.T) {
	// Long and entirely uniform, so rANS would certainly take it if allowed to.
	v := handBlob{Payload: string(make([]byte, 4096))}

	plain, err := qdf.Marshal(v, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	framed, err := qdf.Marshal(v, qdf.OptBalanced|qdf.OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != string(framed) {
		t.Fatalf("a hand-written Marshaler's bytes changed with rANS: %d vs %d — its body "+
			"is opts-invariant by contract and must not be reframed",
			len(plain), len(framed))
	}
	var got handBlob
	if err := qdf.Unmarshal(framed, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Payload != v.Payload {
		t.Fatal("hand-written Marshaler did not round-trip")
	}
}
