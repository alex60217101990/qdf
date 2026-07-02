package qdf

import "testing"

// cid: named int64 carrying ONLY the decode-half codec (UnmarshalQDF).
// Documented asymmetric case: encode structurally, decode via UnmarshalQDF.
type cid int64

func (c *cid) UnmarshalQDF(src []byte) (int, error) {
	// consume the structural int64 value, then post-process (sentinel +1000).
	d := NewDecoderOnBuf(src)
	d.MarkHeaderRead()
	v, err := d.ReadInt()
	if err != nil {
		return 0, err
	}
	*c = cid(v + 1000)
	return d.Pos(), nil
}

type asymRow struct {
	ID cid
	N  int64
}

func TestAsymmetricDecodeFieldColumnar(t *testing.T) {
	small := make([]asymRow, 4)  // row-major
	large := make([]asymRow, 32) // columnar-eligible
	for i := range small {
		small[i] = asymRow{ID: cid(i + 1), N: int64(i)}
	}
	for i := range large {
		large[i] = asymRow{ID: cid(i + 1), N: int64(i)}
	}

	bs, _ := Marshal(small, OptBalanced)
	bl, _ := Marshal(large, OptBalanced)

	var outS []asymRow
	if err := Unmarshal(bs, &outS); err != nil {
		t.Fatalf("small: %v", err)
	}
	var outL []asymRow
	if err := Unmarshal(bl, &outL); err != nil {
		t.Fatalf("large: %v", err)
	}

	t.Logf("row-major ID[0]=%d   columnar ID[0]=%d   (in=1; +1000 if UnmarshalQDF ran)", outS[0].ID, outL[0].ID)
	if (outS[0].ID >= 1000) != (outL[0].ID >= 1000) {
		t.Errorf("DIVERGENCE: row-major=%d columnar=%d — UnmarshalQDF skipped on one path", outS[0].ID, outL[0].ID)
	}
}
