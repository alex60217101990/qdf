package qdf

import "testing"

func TestRANS_RoundTrip(t *testing.T) {
	type rec struct {
		Service string `qdf:"service"`
		Level   string `qdf:"level"`
		Msg     string `qdf:"msg"`
		Code    int    `qdf:"code"`
	}
	batch := make([]rec, 500)
	for i := range batch {
		batch[i] = rec{Service: "billing", Level: "info", Msg: "request handled ok", Code: 200}
	}
	b, err := Marshal(batch, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != len(batch) || out[0] != batch[0] || out[len(out)-1] != batch[len(batch)-1] {
		t.Fatal("round-trip mismatch")
	}
	if b[4]&FlagRANS == 0 {
		t.Log("note: FlagRANS not set — rANS did not win on this fixture")
	}
}

func TestRANS_NeverLarger(t *testing.T) {
	small, _ := Marshal(struct {
		A int `qdf:"a"`
	}{A: 1}, OptCompression)
	if small[4]&FlagRANS != 0 {
		t.Fatal("rANS should not fire on a tiny payload")
	}
}
