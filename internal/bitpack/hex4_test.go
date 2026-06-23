package bitpack

import (
	"math/rand"
	"testing"
)

// hex4Ref is an independent byte-at-a-time reference for DecodeHex4.
func hex4Ref(dst, src []byte, lut *[16]byte) {
	for k := range dst {
		b := src[k>>1]
		if k&1 == 0 {
			dst[k] = lut[b&0x0f]
		} else {
			dst[k] = lut[b>>4]
		}
	}
}

func TestDecodeHex4Parity(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	var lut [16]byte
	copy(lut[:], "0123456789abcdef")
	// also a non-hex LUT to ensure the mapping is honored, not assumed
	var lut2 [16]byte
	for i := range lut2 {
		lut2[i] = byte('A' + i)
	}
	sizes := []int{0, 1, 2, 3, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 1000, 1001}
	for _, n := range sizes { // n = number of output bytes (nibbles)
		srcLen := (n + 1) / 2
		src := make([]byte, srcLen)
		for i := range src {
			src[i] = byte(r.Intn(256))
		}
		for _, l := range []*[16]byte{&lut, &lut2} {
			want := make([]byte, n)
			hex4Ref(want, src, l)
			got := make([]byte, n)
			DecodeHex4(got, src, l)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("n=%d i=%d: got %d want %d", n, i, got[i], want[i])
				}
			}
		}
	}
}
