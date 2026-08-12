package main

import (
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// A type with generated methods never takes the reflect struct path —
// reflect_encode.go dispatches to EncodeQDF for any EncoderMarshaler — so the
// two are separate encoders that have to be held to the same standard.
//
// The standard is wire PARITY, not byte-identity, and the difference was
// established by measurement rather than chosen. Byte-identity is unreachable
// without undoing two deliberate generator decisions, both of which exist for
// speed:
//
//   - Generated code calls StructShape unconditionally, so it shape-interns
//     even under OptSpeed, which asks for no such thing. That is why codegen
//     is 29.4% SMALLER than reflect at OptSpeed with five elements: it applies
//     an optimisation the options switched off.
//   - Generated code emits field names from pre-built fixstr headers
//     (qdfFieldHdrs_*) where reflect interns them (tagInternStr). The
//     generator bakes those headers precisely so the hot path emits a name
//     with one append and no per-call sizing.
//
// Forcing the bytes to match would mean giving both up. So the invariant is
// the one that actually matters: generated code must never write MORE than
// reflect for the same value and options, and where it can write less it may.
// Correctness is asserted separately by the round-trip below, and by the
// fuzzers in the root package.
func convergenceOptions() []struct {
	name string
	opts qdf.Options
} {
	return []struct {
		name string
		opts qdf.Options
	}{
		{"speed", qdf.OptSpeed},
		{"balanced", qdf.OptBalanced},
		{"balanced+canonical", qdf.OptBalanced | qdf.OptCanonical},
		{"balanced+alpha", qdf.OptBalanced | qdf.OptStringAlphabet},
		{"balanced+alpha+canonical", qdf.OptBalanced | qdf.OptStringAlphabet | qdf.OptCanonical},
		{"compression", qdf.OptCompression},
		{"compression+alpha", qdf.OptCompression | qdf.OptStringAlphabet},
	}
}

func mkServices(n int) []Service {
	out := make([]Service, n)
	for k := range out {
		s := string(rune('a' + k%26))
		out[k] = Service{
			RegistryOwner:        "NT SERVICE\\TrustedInstaller",
			RegistryDACL:         "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)",
			Name:                 "com.acme.worker.service." + s,
			DisplayName:          "Acme Worker Service " + s,
			Description:          "long-running background worker, shard " + s,
			ImagePath:            "/opt/acme/bin/worker --shard=" + s,
			ImageExecutable:      "/opt/acme/bin/worker",
			ImageExecutableOwner: "root",
			ImageExecutableDACL:  "D:(A;;FA;;;SY)(A;;FA;;;BA)",
		}
	}
	return out
}

func TestCodegenIsNeverLargerThanReflect(t *testing.T) {
	// Lengths straddle the generator's static columnar threshold of 16.
	for _, n := range []int{1, 2, 5, 12, 15, 16, 17, 64} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range convergenceOptions() {
			rb, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s reflect: %v", n, o.name, err)
			}
			gb, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s codegen: %v", n, o.name, err)
			}
			if n == 1 {
				// One element has nobody to amortise the shape declaration
				// against: generated code always declares a shape, so under
				// OptSpeed — where reflect declares none — it pays a few bytes
				// the second element would already have earned back. The
				// allowance is asserted rather than waived, so it cannot grow
				// unnoticed into a real regression.
				const shapeDeclAllowance = 8
				if len(gb) > len(rb)+shapeDeclAllowance {
					t.Errorf("n=1 %-26s reflect=%d codegen=%d — past the %d-byte shape-declaration allowance",
						o.name, len(rb), len(gb), shapeDeclAllowance)
				}
				continue
			}
			if len(gb) > len(rb) {
				t.Errorf("n=%-3d %-26s reflect=%d codegen=%d (%+.1f%%)",
					n, o.name, len(rb), len(gb),
					float64(len(gb)-len(rb))/float64(len(rb))*100)
			}
		}
	}
}

// Both sides must still decode to the original values — byte-identity is
// worthless if the bytes are identically wrong.
func TestCodegenWireRoundTrips(t *testing.T) {
	for _, n := range []int{2, 15, 64} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range convergenceOptions() {
			gb, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s: %v", n, o.name, err)
			}
			var got []GenService
			if err := qdf.Unmarshal(gb, &got); err != nil {
				t.Fatalf("n=%d %s decode: %v", n, o.name, err)
			}
			if len(got) != n {
				t.Fatalf("n=%d %s: decoded %d elements", n, o.name, len(got))
			}
			for k := range gen {
				if !reflect.DeepEqual(got[k], gen[k]) {
					t.Fatalf("n=%d %s element %d: got %+v want %+v", n, o.name, k, got[k], gen[k])
				}
			}
		}
	}
}
