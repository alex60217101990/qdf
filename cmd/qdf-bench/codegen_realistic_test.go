package main

import (
	"fmt"
	"testing"

	"github.com/alex60217101990/qdf"
)

// mkRealistic is the shape the string codecs were designed for and every other
// fixture in this package deliberately is not: consecutive rows that RESEMBLE
// each other. Access logs, request paths, service identifiers — the values a
// front delta was measured at -11.4% on.
//
// mkServices varies the middle of every field so a stale base cannot decode
// correctly, which is the right call for a correctness fixture and the wrong
// one for pricing a codec. Judging the delta only there is judging it on data
// built to defeat it.
func mkRealistic(n int) []Service {
	out := make([]Service, n)
	for k := range out {
		// Sorted-ish identifiers: long shared prefixes between neighbors, the
		// way real request paths and service names arrive.
		id := fmt.Sprintf("%06d", k)
		out[k] = Service{
			RegistryOwner:        "NT SERVICE\\TrustedInstaller",
			RegistryDACL:         "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)",
			Name:                 "com.acme.platform.worker.service." + id,
			DisplayName:          "Acme Platform Worker Service " + id,
			Description:          "long-running background worker for shard " + id,
			ImagePath:            "/opt/acme/platform/bin/worker --shard=" + id,
			ImageExecutable:      "/opt/acme/platform/bin/worker",
			ImageExecutableOwner: "root",
			ImageExecutableDACL:  "D:(A;;FA;;;SY)(A;;FA;;;BA)",
			Account:              "svc-acme-platform-worker-" + id,
		}
	}
	return out
}

func benchRealistic(b *testing.B, n int, opts qdf.Options) {
	b.Helper()
	plain := mkRealistic(n)
	gen := make([]GenService, n)
	for k := range plain {
		gen[k] = GenService(plain[k])
	}
	wire, err := qdf.Marshal(gen, opts)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		for b.Loop() {
			if _, err := qdf.Marshal(gen, opts); err != nil {
				b.Fatal(err)
			}
		}
	})
	// Decode was never measured for this feature, and it is the side that has
	// to materialize every delta and unpack every alphabet.
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out []GenService
			if err := qdf.Unmarshal(wire, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRealistic64(b *testing.B)  { benchRealistic(b, 64, qdf.OptBalanced) }
func BenchmarkRealistic512(b *testing.B) { benchRealistic(b, 512, qdf.OptBalanced) }
func BenchmarkRealistic64Alpha(b *testing.B) {
	benchRealistic(b, 64, qdf.OptBalanced|qdf.OptStringAlphabet)
}

// Decode of the hostile fixture too, so the two are comparable.
func BenchmarkHostile64Decode(b *testing.B) {
	plain := mkServices(64)
	gen := make([]GenService, 64)
	for k := range plain {
		gen[k] = GenService(plain[k])
	}
	wire, err := qdf.Marshal(gen, qdf.OptBalanced)
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

// The wire each fixture actually produces, so a CPU number is never read
// without the byte number beside it.
func TestRealisticWire(t *testing.T) {
	for _, n := range []int{64, 512} {
		plain := mkRealistic(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range []struct {
			n string
			o qdf.Options
		}{{"balanced", qdf.OptBalanced}, {"bal+alpha", qdf.OptBalanced | qdf.OptStringAlphabet}} {
			b, err := qdf.Marshal(gen, o.o)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("REAL n=%-4d %-10s %d", n, o.n, len(b))
		}
	}
	for _, n := range []int{15, 64} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range []struct {
			n string
			o qdf.Options
		}{{"balanced", qdf.OptBalanced}, {"bal+alpha", qdf.OptBalanced | qdf.OptStringAlphabet}} {
			b, err := qdf.Marshal(gen, o.o)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("HOST n=%-4d %-10s %d", n, o.n, len(b))
		}
	}
}
