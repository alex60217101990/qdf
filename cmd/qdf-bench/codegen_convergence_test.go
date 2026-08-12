package main

import (
	"fmt"
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
//     comes out 29.4% SMALLER than reflect at OptSpeed with five elements: it
//     applies an optimisation the options switched off.
//   - Generated code emits field names from pre-built fixstr headers
//     (qdfFieldHdrs_*) where reflect interns them (tagInternStr). The
//     generator bakes those headers precisely so the hot path emits a name
//     with one append and no per-call sizing.
//
// Forcing the bytes to match would mean giving both up. So the invariant is
// the one that matters: generated code must never write MORE than reflect for
// the same value and options, and where it can write less it may.

// convergenceOptions lists the option sets that produce DISTINCT encodings for
// these fixtures. There are four, not seven.
//
// An adversarial review of an earlier version of this file found four of its
// seven rows were byte-identical duplicates, which made 13 real measurements
// look like 42. OptCanonical changes nothing here — no maps, no floats needing
// normalisation — and OptStringAlphabet under OptCompression is a no-op BY
// CONSTRUCTION: the encoder disables the alphabet whenever rANS or FSST is on,
// because packing to five bits destroys the byte skew those coders feed on.
// Rows that cannot differ are not coverage; they are noise that hides how
// little is being tested.
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
		{"balanced+alpha", qdf.OptBalanced | qdf.OptStringAlphabet},
		{"compression", qdf.OptCompression},
	}
}

// mkServices builds the fixture for the parity assertions.
//
// Every varying field varies in the MIDDLE, not at the end, and that is not
// cosmetic. With a constant prefix and a one-character tail — what this fixture
// used to have — a delta rebuilds correctly against ANY earlier value of the
// field, so a per-field base one row out of step still decodes to the right
// answer. The review that caught it measured 0 of 13 fields detecting an
// off-by-one base. A varying middle makes a stale base produce a visibly wrong
// string.
func mkServices(n int) []Service {
	out := make([]Service, n)
	for k := range out {
		// Two independent varying segments, so neither a stale base nor a
		// swapped field can reconstruct a plausible value.
		mid := fmt.Sprintf("%04d", k*7919%10000)
		tail := fmt.Sprintf("%03d", k*31%1000)
		out[k] = Service{
			RegistryOwner:        "NT SERVICE\\TrustedInstaller",
			RegistryDACL:         "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)",
			Name:                 "com.acme." + mid + ".worker.service." + tail,
			DisplayName:          "Acme " + mid + " Worker Service " + tail,
			Description:          "long-running " + mid + " background worker, shard " + tail,
			ImagePath:            "/opt/acme/" + mid + "/bin/worker --shard=" + tail,
			ImageExecutable:      "/opt/acme/bin/worker",
			ImageExecutableOwner: "root",
			ImageExecutableDACL:  "D:(A;;FA;;;SY)(A;;FA;;;BA)",
			Account:              "svc-account-" + tail,
		}
	}
	return out
}

// mkTasks is the alphabet fixture: fields whose character set is restricted,
// which is what tagStrAlpha exists for and what mkServices cannot exercise —
// every one of its strings carries a backslash, a slash, a colon or a space.
func mkTasks(n int) []Task {
	const hexDigits = "0123456789abcdef"
	seed := uint64(0x9E3779B97F4A7C15)
	next := func() uint64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		return seed >> 33
	}
	hex := func(w int) string {
		b := make([]byte, w)
		for j := range b {
			b[j] = hexDigits[next()%16]
		}
		return string(b)
	}
	out := make([]Task, n)
	for k := range out {
		mid := fmt.Sprintf("%04d", k*7919%10000)
		out[k] = Task{
			Name:        "acme." + mid + ".scheduled.task",
			Path:        "\\Acme\\" + mid + "\\Scheduled",
			Enabled:     k%2 == 0,
			State:       hex(32), // trace-id shaped: the alphabet's territory
			MissedRuns:  k % 5,
			NextRunTime: hex(16),
			LastRunTime: hex(16),
		}
	}
	return out
}

// The reflect side must keep the codecs it has, or the parity assertions below
// are satisfiable by degrading reflect instead of improving codegen.
//
// This was the most serious finding in review. With the reflect delta
// short-circuited to a plain WriteString and the generated encoder untouched,
// 28 of the 42 violations went green and settled at a comfortable-looking -13
// bytes. A parity test with no absolute anchor cannot tell "codegen caught up"
// from "reflect fell back".
//
// The anchor is self-referential rather than a golden byte count, so a genuine
// improvement on either side does not break it. OptSpeed encodes the same data
// with no codecs at all — OptBalanced&^OptDense measures byte-identical to it —
// so that ratio is exactly what the codecs are worth.
//
// Measured on this fixture: 2.02x at five elements, 2.49x at fifteen. Muting the
// delta collapses the ratio towards 1. The bar is 1.6: below the measured floor
// with margin for the fixture drifting, far above where a muted delta lands.
//
// The ratio is lower than it would be on values sharing long prefixes, and
// deliberately so — mkServices varies the middle of each field precisely so a
// stale base cannot decode correctly, which also leaves the delta less to work
// with. Detectability was worth more than a comfortable-looking number.
func TestReflectStillHasItsStringCodecs(t *testing.T) {
	const minCodecWorth = 1.6
	for _, n := range []int{5, 15} {
		v := mkServices(n)
		bare, err := qdf.Marshal(v, qdf.OptSpeed)
		if err != nil {
			t.Fatal(err)
		}
		coded, err := qdf.Marshal(v, qdf.OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		if worth := float64(len(bare)) / float64(len(coded)); worth < minCodecWorth {
			t.Fatalf("n=%d: reflect wrote %d bytes with codecs against %d without, only %.2fx — "+
				"the delta looks muted, and the parity assertions would then pass by "+
				"meeting a degraded reflect rather than by fixing codegen",
				n, len(coded), len(bare), worth)
		}
	}
}

// Alphabet packing has to fire somewhere in this package's fixtures, or every
// parity row naming it is decoration.
//
// Asserted by its effect rather than by a counter: the counters live in the qdf
// package and are unreachable from here. A restricted-alphabet payload must come
// out strictly smaller with the bit than without it.
func TestAlphabetPackingFiresOnHexFields(t *testing.T) {
	// Fifteen, not sixty-four. A flat slice of sixteen or more takes the
	// columnar container, where a different alphabet codec applies and this one
	// never runs — the trap that made an earlier fixture in this repository
	// report wk=0 decl=0 ref=0 while looking like coverage. Measured here:
	// -21.0% at fifteen elements, 0.0% at sixty-four.
	v := mkTasks(15)
	without, err := qdf.Marshal(v, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	with, err := qdf.Marshal(v, qdf.OptBalanced|qdf.OptStringAlphabet)
	if err != nil {
		t.Fatal(err)
	}
	if len(with) >= len(without) {
		t.Fatalf("hex-id fields: %d bytes with the alphabet bit against %d without — "+
			"the codec never fired", len(with), len(without))
	}
}

// codegenParity is the shared body: same data through both encoders, generated
// output may not be larger.
func codegenParity[P any, G any](t *testing.T, name string, lens []int,
	mk func(int) []P, conv func(P) G,
) {
	t.Helper()
	for _, n := range lens {
		plain := mk(n)
		gen := make([]G, n)
		for k := range plain {
			gen[k] = conv(plain[k])
		}
		for _, o := range convergenceOptions() {
			rb, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatalf("%s n=%d %s reflect: %v", name, n, o.name, err)
			}
			gb, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("%s n=%d %s codegen: %v", name, n, o.name, err)
			}
			if n == 1 {
				// One element has nobody to amortise the shape declaration
				// against: generated code always declares a shape, so under
				// OptSpeed — where reflect declares none — it pays for it. The
				// measured excess is ONE byte. The allowance is two, not the
				// eight an earlier version used, which was wide enough to hide
				// a whole extra header on a field.
				const shapeDeclAllowance = 2
				if len(gb) > len(rb)+shapeDeclAllowance {
					t.Errorf("%s n=1 %-16s reflect=%d codegen=%d — past the %d-byte allowance",
						name, o.name, len(rb), len(gb), shapeDeclAllowance)
				}
				continue
			}
			if len(gb) > len(rb) {
				t.Errorf("%s n=%-3d %-16s reflect=%d codegen=%d (%+.1f%%)",
					name, n, o.name, len(rb), len(gb),
					float64(len(gb)-len(rb))/float64(len(rb))*100)
			}
		}
	}
}

// Lengths outside the columnar window, where the row-major string codecs are
// the whole difference. See TestCodegenColumnarWindowIsAKnownGap for 16..29.
func TestCodegenIsNeverLargerThanReflect(t *testing.T) {
	codegenParity(t, "service", []int{1, 2, 5, 12, 15, 30, 64},
		mkServices, func(s Service) GenService { return GenService(s) })
	codegenParity(t, "task", []int{1, 2, 5, 12, 15, 30, 64},
		mkTasks, func(x Task) GenTask { return GenTask(x) })
}

// Between 16 and 29 elements reflect takes the columnar container and generated
// code does not: the generator decides with a static len >= 16 where reflect
// probes the data. Giving generated encoders the string codecs cannot close
// this — the reflect side is not using them there either.
//
// Asserted rather than skipped, so the gap stays visible and cannot quietly
// widen. Closing it is separate work on the container decision, recorded as a
// known open problem in the design note.
func TestCodegenColumnarWindowIsAKnownGap(t *testing.T) {
	const knownWorstRatio = 3.5
	for _, n := range []int{16, 17, 29} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		rb, err := qdf.Marshal(plain, qdf.OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		gb, err := qdf.Marshal(gen, qdf.OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		ratio := float64(len(gb)) / float64(len(rb))
		if ratio > knownWorstRatio {
			t.Errorf("n=%d: codegen %d against reflect %d is %.2fx, past the %.1fx measured for this gap",
				n, len(gb), len(rb), ratio, knownWorstRatio)
		}
		t.Logf("n=%-3d columnar gap: reflect=%d codegen=%d (%.2fx)", n, len(rb), len(gb), ratio)
	}
}

// Byte counts are worthless if the bytes are wrong. Both fixtures, both
// encoders, every option set, including the boundary lengths the size
// assertions treat specially.
func TestCodegenWireRoundTrips(t *testing.T) {
	for _, n := range []int{1, 2, 15, 16, 17, 64} {
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
		for _, o := range convergenceOptions() {
			gb, err := qdf.Marshal(gsvc, o.opts)
			if err != nil {
				t.Fatalf("service n=%d %s: %v", n, o.name, err)
			}
			var got []GenService
			if err := qdf.Unmarshal(gb, &got); err != nil {
				t.Fatalf("service n=%d %s decode: %v", n, o.name, err)
			}
			if !reflect.DeepEqual(got, gsvc) {
				t.Fatalf("service n=%d %s: decoded value differs", n, o.name)
			}

			tb, err := qdf.Marshal(gtsk, o.opts)
			if err != nil {
				t.Fatalf("task n=%d %s: %v", n, o.name, err)
			}
			var gotT []GenTask
			if err := qdf.Unmarshal(tb, &gotT); err != nil {
				t.Fatalf("task n=%d %s decode: %v", n, o.name, err)
			}
			if !reflect.DeepEqual(gotT, gtsk) {
				t.Fatalf("task n=%d %s: decoded value differs", n, o.name)
			}
		}
	}
}

// The two encoders and the two decoders have to interoperate, and only one of
// the two directions currently does.
//
// TestCodegenWireRoundTrips covers codegen->codegen only: it decodes into
// []GenService, and GenService carries a DecodeQDF, so Unmarshal dispatches to
// generated code on both ends. The off-diagonal combinations are the ones that
// get used in anger — a producer built with qdfgen and a consumer that only has
// the plain struct, or the reverse across a version skew — and they were
// untested.
//
// Measured, and the same on merged main (23a3b18), so this is not the string
// codecs and not this branch:
//
//   - A wire written by generated code decodes into the plain type at every
//     length and every option set. Asserted strictly below.
//   - A wire written by reflect FAILS to decode into the generated type from
//     sixteen elements under OptBalanced and OptCompression, with "type
//     mismatch on decode". Reflect picks the hybrid columnar container
//     (tagHybridColStruct, 0xF7) and generated decoders do not read that form.
//     Under OptSpeed reflect never goes columnar, so it works there.
//
// The broken direction is asserted as what it is: it must fail CLEANLY. An
// error is recoverable and visible; silent corruption is neither, and that is
// the property worth pinning while the container decision remains split
// between a static threshold on one side and a data probe on the other.
func TestCodegenAndReflectWiresAreInterchangeable(t *testing.T) {
	for _, n := range []int{1, 2, 15, 16, 64} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range convergenceOptions() {
			rb, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s reflect encode: %v", n, o.name, err)
			}
			gb, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s codegen encode: %v", n, o.name, err)
			}

			// Generated wire into the plain type: must always work.
			var intoPlain []Service
			if err := qdf.Unmarshal(gb, &intoPlain); err != nil {
				t.Fatalf("n=%d %s codegen wire into the plain type: %v", n, o.name, err)
			}
			if !reflect.DeepEqual(intoPlain, plain) {
				t.Fatalf("n=%d %s: codegen wire decoded into the plain type differs", n, o.name)
			}

			// Reflect wire into the generated type. The property pinned here is
			// the one that matters and holds unconditionally: never silent
			// corruption. Either the value comes back intact, or the decode
			// fails with an error a caller can see and handle.
			//
			// Which lengths fail is deliberately NOT asserted from the root
			// tag: under OptCompression the container is wrapped and the tag
			// varies (0xBD, 0xEF, 0xD7 were all observed), so a byte test there
			// would be guessing. TestReflectWireIntoGeneratedTypeBoundary pins
			// the boundary separately, where it can be measured rather than
			// inferred.
			var intoGen []GenService
			if err := qdf.Unmarshal(rb, &intoGen); err == nil {
				if !reflect.DeepEqual(intoGen, gen) {
					t.Fatalf("n=%d %s: reflect wire decoded into the generated type "+
						"WITHOUT error but with the wrong value — silent corruption",
						n, o.name)
				}
			}
		}
	}
}

// Where the reverse direction stops working, pinned so that fixing it or
// breaking it further are both noticed.
//
// A wire written by reflect decodes into the generated type until reflect
// switches to the columnar container; generated decoders do not read that form.
// Measured here and identically on merged main (23a3b18), so it is neither the
// string codecs nor this branch — it is the same split that leaves the
// generator deciding its container with a static len >= 16 while reflect probes
// the data.
func TestReflectWireIntoGeneratedTypeBoundary(t *testing.T) {
	type result struct {
		n  int
		ok bool
	}
	var got []result
	for _, n := range []int{2, 15, 16, 17, 64} {
		plain := mkServices(n)
		rb, err := qdf.Marshal(plain, qdf.OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		var into []GenService
		got = append(got, result{n, qdf.Unmarshal(rb, &into) == nil})
	}
	want := []result{{2, true}, {15, true}, {16, false}, {17, false}, {64, false}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("n=%d: reflect wire into the generated type decodes=%v, was %v — "+
				"the boundary moved; if this is a fix, update the expectation",
				got[i].n, got[i].ok, want[i].ok)
		}
	}
}
