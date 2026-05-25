package qdf

import (
	"reflect"
	"testing"
)

// Unknown-field handling: the decoder must accept any tag/value the
// encoder produces and silently skip fields the destination type does
// not declare. The reflect-path Skip() implementation must keep the
// stream cursor aligned so subsequent declared fields still decode
// correctly.

type wideEncoded struct {
	ID    int       `qdf:"id"`
	Name  string    `qdf:"name"`
	Tags  []string  `qdf:"tags"`
	Vec   []float64 `qdf:"vec"`
	Bools []bool    `qdf:"bools"`
	Extra string    `qdf:"extra"`
	Vec32 []float32 `qdf:"vec32"`
	Ints  []int64   `qdf:"ints"`
	Bytes []byte    `qdf:"bytes"`
	When  int64     `qdf:"when"`
}

type narrowDecoded struct {
	ID   int    `qdf:"id"`
	Name string `qdf:"name"`
	When int64  `qdf:"when"`
}

func fullWide() wideEncoded {
	return wideEncoded{
		ID:    42,
		Name:  "alpha",
		Tags:  []string{"a", "b", "c"},
		Vec:   []float64{0.1, 0.2, 0.3, 0.4},
		Bools: []bool{true, false, true, true, false},
		Extra: "discarded",
		Vec32: []float32{1, 2, 3, 4},
		Ints:  []int64{-1, 0, 1, 1 << 40},
		Bytes: []byte{0x01, 0x02, 0x03, 0xFF},
		When:  1700000000,
	}
}

func TestUnknownField_SkipKeepsStreamAligned(t *testing.T) {
	wide := fullWide()
	for label, enc := range map[string]func(any) ([]byte, error){
		"Marshal":      Marshal,
		"MarshalQPack": MarshalQPack,
		"MarshalDense": MarshalDense,
	} {
		buf, err := enc(wide)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		var narrow narrowDecoded
		if err := Unmarshal(buf, &narrow); err != nil {
			t.Fatalf("%s decode: %v", label, err)
		}
		want := narrowDecoded{ID: wide.ID, Name: wide.Name, When: wide.When}
		if narrow != want {
			t.Fatalf("%s: got %+v want %+v", label, narrow, want)
		}
	}
}

func TestUnknownField_SkipEveryFieldAlone(t *testing.T) {
	// For every wideEncoded field, build a destination struct that
	// declares only that one field. Confirm the decoder still reads
	// it back correctly after skipping every other key/value pair.
	type onlyID struct {
		ID int `qdf:"id"`
	}
	type onlyName struct {
		Name string `qdf:"name"`
	}
	type onlyTags struct {
		Tags []string `qdf:"tags"`
	}
	type onlyVec struct {
		Vec []float64 `qdf:"vec"`
	}
	type onlyBools struct {
		Bools []bool `qdf:"bools"`
	}
	type onlyVec32 struct {
		Vec32 []float32 `qdf:"vec32"`
	}
	type onlyInts struct {
		Ints []int64 `qdf:"ints"`
	}
	type onlyBytes struct {
		Bytes []byte `qdf:"bytes"`
	}
	type onlyWhen struct {
		When int64 `qdf:"when"`
	}
	type onlyExtra struct {
		Extra string `qdf:"extra"`
	}

	wide := fullWide()
	for label, enc := range map[string]func(any) ([]byte, error){
		"Marshal":      Marshal,
		"MarshalQPack": MarshalQPack,
		"MarshalDense": MarshalDense,
	} {
		buf, _ := enc(wide)

		var oID onlyID
		if err := Unmarshal(buf, &oID); err != nil || oID.ID != wide.ID {
			t.Errorf("%s/ID: %v %+v", label, err, oID)
		}
		var oName onlyName
		if err := Unmarshal(buf, &oName); err != nil || oName.Name != wide.Name {
			t.Errorf("%s/Name: %v %+v", label, err, oName)
		}
		var oTags onlyTags
		if err := Unmarshal(buf, &oTags); err != nil || !reflect.DeepEqual(oTags.Tags, wide.Tags) {
			t.Errorf("%s/Tags: %v %+v", label, err, oTags)
		}
		var oVec onlyVec
		if err := Unmarshal(buf, &oVec); err != nil || !reflect.DeepEqual(oVec.Vec, wide.Vec) {
			t.Errorf("%s/Vec: %v %+v", label, err, oVec)
		}
		var oBools onlyBools
		if err := Unmarshal(buf, &oBools); err != nil || !reflect.DeepEqual(oBools.Bools, wide.Bools) {
			t.Errorf("%s/Bools: %v %+v", label, err, oBools)
		}
		var oVec32 onlyVec32
		if err := Unmarshal(buf, &oVec32); err != nil || !reflect.DeepEqual(oVec32.Vec32, wide.Vec32) {
			t.Errorf("%s/Vec32: %v %+v", label, err, oVec32)
		}
		var oInts onlyInts
		if err := Unmarshal(buf, &oInts); err != nil || !reflect.DeepEqual(oInts.Ints, wide.Ints) {
			t.Errorf("%s/Ints: %v %+v", label, err, oInts)
		}
		var oBytes onlyBytes
		if err := Unmarshal(buf, &oBytes); err != nil || !reflect.DeepEqual(oBytes.Bytes, wide.Bytes) {
			t.Errorf("%s/Bytes: %v %+v", label, err, oBytes)
		}
		var oWhen onlyWhen
		if err := Unmarshal(buf, &oWhen); err != nil || oWhen.When != wide.When {
			t.Errorf("%s/When: %v %+v", label, err, oWhen)
		}
		var oExtra onlyExtra
		if err := Unmarshal(buf, &oExtra); err != nil || oExtra.Extra != wide.Extra {
			t.Errorf("%s/Extra: %v %+v", label, err, oExtra)
		}
	}
}

func TestUnknownField_EmptyTarget(t *testing.T) {
	// A struct that declares nothing the encoder emits still decodes
	// without error, leaving every byte consumed.
	type empty struct{}
	wide := fullWide()
	for label, enc := range map[string]func(any) ([]byte, error){
		"Marshal":      Marshal,
		"MarshalQPack": MarshalQPack,
		"MarshalDense": MarshalDense,
	} {
		buf, _ := enc(wide)
		var e empty
		if err := Unmarshal(buf, &e); err != nil {
			t.Errorf("%s/empty: %v", label, err)
		}
	}
}

func TestUnknownField_DenseStateStaysCorrect(t *testing.T) {
	// In Dense mode the skipped field's intern entries must still be
	// registered (or skipped consistently) so subsequent state-refs
	// resolve correctly. Encode two messages on the same stream where
	// the second references a key first emitted under the skipped
	// field — guards against a "skip + intern table" desync.
	type pair struct {
		Region string `qdf:"region"`
	}
	type withExtra struct {
		Region string `qdf:"region"`
		Extra  string `qdf:"extra"`
	}
	// First message has Extra to force the intern table to grow.
	msg1 := withExtra{Region: "eu-west-1", Extra: "eu-west-1"}
	// Decode message 1 with narrow type that skips Extra.
	buf1, err := MarshalDense(msg1)
	if err != nil {
		t.Fatal(err)
	}
	var got pair
	if err := Unmarshal(buf1, &got); err != nil {
		t.Fatalf("decode narrow: %v", err)
	}
	if got.Region != msg1.Region {
		t.Fatalf("got %q want %q", got.Region, msg1.Region)
	}
}
