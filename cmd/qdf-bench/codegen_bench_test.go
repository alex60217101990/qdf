package main

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

// Generated encoders exist for speed, so the wire win has to be priced. Both
// sides of every pair encode the same data: "codegen" through the generated
// methods, "reflect" through the plain twin, so a run shows what the codecs
// cost AND what they cost relative to the path that already paid for them.
func benchCodegenTopLevel(b *testing.B, n int, opts qdf.Options) {
	plain := mkServices(n)
	gen := make([]GenService, n)
	for k := range plain {
		gen[k] = GenService(plain[k])
	}
	b.Run("codegen", func(b *testing.B) {
		e := qdf.NewEncoderWith(opts)
		b.ResetTimer()
		for b.Loop() {
			e.Reset()
			if err := e.EncodeValue(gen); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("reflect", func(b *testing.B) {
		e := qdf.NewEncoderWith(opts)
		b.ResetTimer()
		for b.Loop() {
			e.Reset()
			if err := e.EncodeValue(plain); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// The nested shape is where the generator's own loop runs, with the push and
// pop it emits around it. The top-level shape above is driven by reflect's
// slice encoder instead, so the two measure different code.
func benchCodegenNested(b *testing.B, n int, opts qdf.Options) {
	svc := mkServices(n)
	gsvc := make([]GenService, n)
	for k := range svc {
		gsvc[k] = GenService(svc[k])
	}
	tsk := mkTasks(n)
	gtsk := make([]GenTask, n)
	for k := range tsk {
		gtsk[k] = GenTask(tsk[k])
	}
	gh := GenHost{Services: gsvc, Tasks: gtsk}
	b.Run("codegen", func(b *testing.B) {
		e := qdf.NewEncoderWith(opts)
		b.ResetTimer()
		for b.Loop() {
			e.Reset()
			if err := e.EncodeValue(gh); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCodegenTop15(b *testing.B) { benchCodegenTopLevel(b, 15, qdf.OptBalanced) }
func BenchmarkCodegenTop15Alpha(b *testing.B) {
	benchCodegenTopLevel(b, 15, qdf.OptBalanced|qdf.OptStringAlphabet)
}
func BenchmarkCodegenTop64(b *testing.B)    { benchCodegenTopLevel(b, 64, qdf.OptBalanced) }
func BenchmarkCodegenNested15(b *testing.B) { benchCodegenNested(b, 15, qdf.OptBalanced) }
func BenchmarkCodegenNested15Alpha(b *testing.B) {
	benchCodegenNested(b, 15, qdf.OptBalanced|qdf.OptStringAlphabet)
}
