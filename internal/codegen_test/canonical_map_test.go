package cgsample

import (
	"bytes"
	"os"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// TestGenerated_CanonicalMapDeterministic guards that generated EncodeQDF emits
// map entries in sorted key order under OptCanonical. Go map iteration is
// randomized, so before the emitEncodeMap canonical-sort fix two encodes of the
// same map produced different byte order under OptCanonical (breaking the
// canonical byte-identical guarantee). With the fix every encode is identical.
func TestGenerated_CanonicalMapDeterministic(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	in := Sample{
		Name: "x",
		Meta: map[string]string{
			"zeta": "1", "alpha": "2", "mike": "3", "bravo": "4",
			"yankee": "5", "charlie": "6", "november": "7", "delta": "8",
			"oscar": "9", "kilo": "10", "echo": "11", "papa": "12",
		},
	}
	var first []byte
	for i := range 40 {
		e := qdf.NewEncoderWith(qdf.OptCanonical)
		if err := (&in).EncodeQDF(e); err != nil {
			t.Fatalf("EncodeQDF: %v", err)
		}
		b := append([]byte(nil), e.Bytes()...)
		if i == 0 {
			first = b
			continue
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical generated map output not deterministic at run %d", i)
		}
	}
}
