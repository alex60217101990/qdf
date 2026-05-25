package qdf

import (
	"reflect"
	"testing"
)

// Compares two map-iteration strategies in the reflect path:
// the original *MapIter (rv.MapRange + iter.Next loop) versus the
// Go 1.26 rv.Seq2 range-over-func. Same body, same accumulators —
// the only difference is the iterator machinery.

func BenchmarkMapIter_MapRangeOriginal(b *testing.B) {
	m := make(map[string]int, 64)
	for i := range 64 {
		m[string(rune('a'+i%26))+string(rune('A'+i/26))] = i
	}
	rv := reflect.ValueOf(m)
	b.ReportAllocs()
	for b.Loop() {
		var sum int
		iter := rv.MapRange()
		for iter.Next() {
			sum += int(iter.Value().Int())
		}
		_ = sum
	}
}

func BenchmarkMapIter_Seq2(b *testing.B) {
	m := make(map[string]int, 64)
	for i := range 64 {
		m[string(rune('a'+i%26))+string(rune('A'+i/26))] = i
	}
	rv := reflect.ValueOf(m)
	b.ReportAllocs()
	for b.Loop() {
		var sum int
		for _, v := range rv.Seq2() {
			sum += int(v.Int())
		}
		_ = sum
	}
}
