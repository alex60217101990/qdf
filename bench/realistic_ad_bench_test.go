package bench

import (
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// Focused encode/decode benchmarks on the realistic AD dataset, for pprof.

func benchADEncode(b *testing.B, opt qdf.Options) {
	users := makeADUsers(5000)
	b.ReportAllocs()
	b.ResetTimer()
	var sink []byte
	for b.Loop() {
		blob, err := qdf.Marshal(users, opt)
		if err != nil {
			b.Fatal(err)
		}
		sink = blob
	}
	_ = sink
}

func benchADDecode(b *testing.B, opt qdf.Options) {
	users := makeADUsers(5000)
	blob, err := qdf.Marshal(users, opt)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(blob)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var out []ADUser
		if err := qdf.Unmarshal(blob, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRealisticAD_Encode_Balanced(b *testing.B)    { benchADEncode(b, qdf.OptBalanced) }
func BenchmarkRealisticAD_Decode_Balanced(b *testing.B)    { benchADDecode(b, qdf.OptBalanced) }
func BenchmarkRealisticAD_Encode_Compression(b *testing.B) { benchADEncode(b, qdf.OptCompression) }
func BenchmarkRealisticAD_Decode_Compression(b *testing.B) { benchADDecode(b, qdf.OptCompression) }
