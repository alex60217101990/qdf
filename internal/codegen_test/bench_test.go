package cgsample

import (
	"encoding/json"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
)

func sampleFixture() Sample {
	return Sample{
		Name:   "alice",
		Age:    33,
		Active: true,
		Score:  98.6,
		Tags:   []string{"a", "b", "c"},
		Meta:   map[string]string{"k1": "v1", "k2": "v2"},
		Inner:  Inner{X: 7, Y: 1.5},
		When:   time.Unix(1700000000, 0),
		Buf:    []byte{1, 2, 3, 4, 5},
		OptPtr: &Inner{X: 99, Y: -2.0},
		Counts: [3]int32{10, 20, 30},
	}
}

func BenchmarkEncode_GenVsReflect(b *testing.B) {
	v := sampleFixture()
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := json.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_reflect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_codegen", func(b *testing.B) {
		b.ReportAllocs()
		vp := &v
		for b.Loop() {
			if _, err := vp.MarshalQDF(nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecode_GenVsReflect(b *testing.B) {
	v := sampleFixture()
	jsonBytes, _ := json.Marshal(v)
	qdfBytes, _ := qdf.Marshal(v)
	qdfGenBytes, _ := (&v).MarshalQDF(nil)

	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Sample
			if err := json.Unmarshal(jsonBytes, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_reflect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Sample
			if err := qdf.Unmarshal(qdfBytes, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_codegen", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Sample
			if _, err := out.UnmarshalQDF(qdfGenBytes); err != nil {
				b.Fatal(err)
			}
		}
	})
}
