package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// wantIntColumnWireDigest pins the encoding of integer columns across the three
// shapes the codec picker discriminates between. The PFOR planner is gated on a
// cost bound rather than run for every column, and a gate that ever skipped a
// plan PFOR would have won would show up here as a larger — and so different —
// encoding. Verified equal with the gate present and removed.
//
// A change means the picker's choices moved, which changes the bytes on the
// wire. Update it only alongside a note saying why.
const wantIntColumnWireDigest = "ef9d19b45b42a7068f5ddebee93243596daa6f8b38e3fb707ba6be85222556cf"

func mkIntColumns(shape string, n int) struct {
	A []int64 `qdf:"a"`
	B []int64 `qdf:"b"`
} {
	a, b := make([]int64, n), make([]int64, n)
	for i := range a {
		switch shape {
		case "monotone": // delta/FOR territory — PFOR never wins
			a[i], b[i] = int64(i), int64(i*3)
		case "repeating": // run/dict territory — PFOR never wins
			a[i], b[i] = int64(i/100), 42
		default: // spiky: mostly small with rare outliers — PFOR's own ground
			a[i], b[i] = int64(i%8), int64(i%5)
			if i%997 == 0 {
				a[i] = 1 << 40
			}
		}
	}
	return struct {
		A []int64 `qdf:"a"`
		B []int64 `qdf:"b"`
	}{a, b}
}

// mkUintColumns is the unsigned twin — pickU64Codec carries its own copy of the
// planner call, so the gate has to be proven on both paths.
func mkUintColumns(shape string, n int) struct {
	A []uint64 `qdf:"a"`
	B []uint32 `qdf:"b"`
	C []uint16 `qdf:"c"`
} {
	a := make([]uint64, n)
	b := make([]uint32, n)
	c := make([]uint16, n)
	for i := range a {
		switch shape {
		case "monotone":
			a[i], b[i], c[i] = uint64(i), uint32(i*3), uint16(i)
		case "repeating":
			a[i], b[i], c[i] = uint64(i/100), 42, 7
		default:
			a[i], b[i], c[i] = uint64(i%8), uint32(i%5), uint16(i%3)
			if i%997 == 0 {
				a[i] = 1 << 40
				b[i] = 1 << 30
			}
		}
	}
	return struct {
		A []uint64 `qdf:"a"`
		B []uint32 `qdf:"b"`
		C []uint16 `qdf:"c"`
	}{a, b, c}
}

const wantUintColumnWireDigest = "3eb828ab33a1a4aecea3938a0828e67ed46c6cbf0d4ead30eca5998926d44884"

func TestPForGateKeepsUnsignedWireIdentical(t *testing.T) {
	h := sha256.New()
	for _, shape := range []string{"monotone", "repeating", "spiky"} {
		for _, n := range []int{64, 1000, 20000} {
			for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
				blob, err := Marshal(mkUintColumns(shape, n), o)
				if err != nil {
					t.Fatalf("%s n=%d %v: %v", shape, n, o, err)
				}
				h.Write(blob)
			}
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != wantUintColumnWireDigest {
		t.Errorf("unsigned wire digest changed:\n got %s\nwant %s", got, wantUintColumnWireDigest)
	}
}

func TestPForGateKeepsWireIdentical(t *testing.T) {
	h := sha256.New()
	for _, shape := range []string{"monotone", "repeating", "spiky"} {
		for _, n := range []int{64, 1000, 20000} {
			for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
				blob, err := Marshal(mkIntColumns(shape, n), o)
				if err != nil {
					t.Fatalf("%s n=%d %v: %v", shape, n, o, err)
				}
				h.Write(blob)
			}
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != wantIntColumnWireDigest {
		t.Errorf("integer-column wire digest changed:\n got %s\nwant %s\nsee the note on wantIntColumnWireDigest", got, wantIntColumnWireDigest)
	}
}
