package main

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

// Generated encoders exist for speed, so the wire win has to be priced. Both
// sides of every pair encode the same data: "codegen" through the generated
// methods, "reflect" through the plain twin, so a run shows what the codecs
// cost AND what they cost relative to the path that already paid for them.
//
// qdf.Marshal per iteration, NOT a reused encoder with Reset(). Reset() sets
// opts back to OptSpeed, so a loop that builds the encoder once and resets it
// measures OptSpeed from the second iteration on — every codec silently off.
// The first version of this file did exactly that and produced a confident
// +144%; the profile gave it away by showing writeStringInline on a path that
// should have been interning. This is the pattern bench/competitor.go uses.
func benchCodegenTopLevel(b *testing.B, n int, opts qdf.Options) {
	plain := mkServices(n)
	gen := make([]GenService, n)
	for k := range plain {
		gen[k] = GenService(plain[k])
	}
	b.Run("codegen", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(gen, opts); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("reflect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(plain, opts); err != nil {
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
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(gh, opts); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// The alphabet only EMITS on the hostile fixture at 64 elements — everywhere
// else it finds nothing to pack and the wire comes out byte-identical with the
// bit and without it (2778/2778 realistic at 64 and 512, 2892/2892 hostile at
// 15). So every other alpha benchmark prices the codec's cost and never its
// win: a regression in the packer or the pack loop would not show up in any of
// them. This one fires — 11288 -> 10003, -11.4% — and has a decode twin,
// because the decode side had no alpha benchmark at all.
func BenchmarkCodegenTop64Alpha(b *testing.B) {
	benchCodegenTopLevel(b, 64, qdf.OptBalanced|qdf.OptStringAlphabet)
}

func BenchmarkCodegenTop64AlphaDecode(b *testing.B) {
	const opts = qdf.OptBalanced | qdf.OptStringAlphabet
	plain := mkServices(64)
	gen := make([]GenService, 64)
	for k := range plain {
		gen[k] = GenService(plain[k])
	}
	wire, err := qdf.Marshal(gen, opts)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		var out []GenService
		if err := qdf.Unmarshal(wire, &out); err != nil {
			b.Fatal(err)
		}
	}
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
