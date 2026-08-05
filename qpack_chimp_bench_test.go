package qdf

import (
	"strconv"
	"testing"
)

func BenchmarkChimpVsGorilla(b *testing.B) {
	for _, sz := range []int{1 << 10, 1 << 13, 1 << 16} {
		data := mkSensor(sz, 11)
		b.Run("chimp/encode/n"+strconv.Itoa(sz), func(b *testing.B) {
			b.SetBytes(int64(sz * 8))
			enc := NewEncoder(Fast)
			for i := 0; i < b.N; i++ {
				enc.Reset()
				enc.writePackedChimpFloat64Slice(data)
			}
		})
		b.Run("gorilla/encode/n"+strconv.Itoa(sz), func(b *testing.B) {
			b.SetBytes(int64(sz * 8))
			enc := NewEncoder(Fast)
			for i := 0; i < b.N; i++ {
				enc.Reset()
				enc.writePackedGorillaFloat64Slice(data)
			}
		})
		encC := NewEncoder(Fast)
		encC.writePackedChimpFloat64Slice(data)
		blobC := append([]byte(nil), encC.buf...)
		b.Run("chimp/decode/n"+strconv.Itoa(sz), func(b *testing.B) {
			b.SetBytes(int64(sz * 8))
			for i := 0; i < b.N; i++ {
				dec := NewDecoderOnBuf(blobC)
				if _, err := dec.peekTag(); err != nil {
					b.Fatal(err)
				}
				dec.i++
				if _, err := dec.readPackedChimpFloat64Slice(); err != nil {
					b.Fatal(err)
				}
			}
		})
		encG := NewEncoder(Fast)
		encG.writePackedGorillaFloat64Slice(data)
		blobG := append([]byte(nil), encG.buf...)
		b.Run("gorilla/decode/n"+strconv.Itoa(sz), func(b *testing.B) {
			b.SetBytes(int64(sz * 8))
			for i := 0; i < b.N; i++ {
				dec := NewDecoderOnBuf(blobG)
				if _, err := dec.peekTag(); err != nil {
					b.Fatal(err)
				}
				dec.i++
				if _, err := dec.readPackedGorillaFloat64Slice(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
