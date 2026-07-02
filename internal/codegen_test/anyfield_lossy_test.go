package cgsample

import (
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

type vecRow32 struct {
	Emb []float32 `qdf:"emb"`
}

// TestGenAnyFieldLossyRoundTrip guards the codegen any-field path under
// OptLossyVec. The generator emits e.EncodeAny (not e.EncodeValue) for an any
// field so the value is encoded in a schemaless context (ifaceDepth>0); a
// []float32/[]float64 or batchable []struct held in the any field must NOT emit
// a tagColVecLossy (0xFD) / tagVecBatchStruct (0xFE) block, because the field is
// read back via decodeAny which cannot decode them. Before the fix the
// generated code called EncodeValue (ifaceDepth==0), emitted the lossy block,
// and decode failed with ErrBadTag ("unknown tag").
func TestGenAnyFieldLossyRoundTrip(t *testing.T) {
	vec := make([]float32, 64)
	for i := range vec {
		vec[i] = float32(i) * 0.01
	}
	rows := make([]vecRow32, 32)
	for i := range rows {
		v := make([]float32, 128)
		for j := range v {
			v[j] = float32(i*128+j) * 0.001
		}
		rows[i] = vecRow32{Emb: v}
	}

	cases := []struct {
		name string
		val  any
	}{
		{"bare_f32_vector", vec},
		{"batchable_struct_slice", rows},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := &GenAnyBox{ID: 7, Val: c.val}

			// Generated encode with an OptLossyVec-configured encoder.
			enc := qdf.NewEncoderWith(qdf.OptBalanced | qdf.OptLossyVec)
			enc.SetVectorBudget(qdf.MinCosine(0.99))
			enc.EnsureHeader()
			if err := in.EncodeQDF(enc); err != nil {
				t.Fatalf("EncodeQDF: %v", err)
			}
			b := enc.Bytes()

			// Decode via the generated path.
			var out GenAnyBox
			if _, err := out.UnmarshalQDF(b); err != nil {
				t.Fatalf("generated UnmarshalQDF (any-field lossy must decode): %v", err)
			}
			if out.ID != 7 {
				t.Fatalf("ID mismatch: %d", out.ID)
			}

			// Decode via the public reflect path.
			var out2 GenAnyBox
			if err := qdf.Unmarshal(b, &out2); err != nil {
				t.Fatalf("qdf.Unmarshal (any-field lossy must decode): %v", err)
			}
		})
	}
}
