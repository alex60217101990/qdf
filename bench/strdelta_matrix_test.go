package bench

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// The delta form (tagStrDelta) codes a struct's string field against the
// previous value of that same field, so encoder and decoder each carry a
// per-field base. Correctness rests on those bases advancing in lockstep — and
// a drift is silent, because the types still line up and nothing errors.
//
// Selected round-trip tests cannot establish that: the risk is a decode path
// nobody enumerated. This matrix runs every payload in this package through
// every option set and through every decode entry point that exists, and
// asserts the strongest invariant available for each.
func strDeltaPayloads() []struct {
	name string
	val  any
} {
	return []struct {
		name string
		val  any
	}{
		{"access", mkAccessBatch(2048)},
		{"iot", mkIoTBatch(64, 32)},
		{"otlp", mkOTLPBatch(16, 8)},
		{"log", mkLogBatch(2048)},
		{"event", mkEventBatch(2048)},
		{"rtb", mkRTBBatch(512)},
		{"alpha", mkAlphaBatch(1024)},
	}
}

func strDeltaOptionSets() []struct {
	name string
	opts qdf.Options
} {
	return []struct {
		name string
		opts qdf.Options
	}{
		{"speed", qdf.OptSpeed},
		{"balanced", qdf.OptBalanced},
		{"qpack", qdf.OptQPack},
		{"compression", qdf.OptCompression},
		{"speed+canonical", qdf.OptSpeed | qdf.OptCanonical},
		{"balanced+canonical", qdf.OptBalanced | qdf.OptCanonical},
		{"qpack+canonical", qdf.OptQPack | qdf.OptCanonical},
		{"compression+canonical", qdf.OptCompression | qdf.OptCanonical},
		{"balanced+canonical-shape", qdf.OptBalanced&^qdf.OptShapeIntern | qdf.OptCanonical},
		{"balanced+canonical-dense", qdf.OptBalanced&^qdf.OptDense | qdf.OptCanonical},
		{"balanced+colindex", qdf.OptBalanced | qdf.OptColumnIndex},
		{"balanced+fsst", qdf.OptBalanced | qdf.OptFSST},
		{"balanced-dense", qdf.OptBalanced &^ qdf.OptDense},
		{"balanced-shape", qdf.OptBalanced &^ qdf.OptShapeIntern},
		{"balanced-mtf", qdf.OptBalanced &^ qdf.OptMTF},
		{"balanced-pair", qdf.OptBalanced &^ qdf.OptPairPred},
	}
}

// alphaRow is built for the alphabet packer rather than the delta: fields with
// a restricted character set and no resemblance to the row above.
//
// The strings live in a SHORT NESTED SLICE, which is not decoration. A flat
// batch of these rows is taken by the columnar container before the row-major
// writer ever sees it, and the packer never runs — the first version of this
// payload added nothing but the appearance of coverage. A handful of elements
// per row is below what the columnar probe accepts, so those values go
// row-major, which is the only path this codec is on. Same reason RTB is the
// payload where it fires.
type alphaSpan struct {
	TraceID string `qdf:"trace_id"`
	SpanID  string `qdf:"span_id"`
	Token   string `qdf:"token"`
	Escapee string `qdf:"escapee"`
	Free    string `qdf:"free"`
}

type alphaRow struct {
	Seq   int64             `qdf:"seq"`
	Spans []alphaSpan       `qdf:"spans"`
	Attrs map[string]string `qdf:"attrs"`
}

// mkAlphaBatch produces every form the packer can emit and, deliberately, the
// values that must make it back off: a field that declares a table and then
// meets a byte outside it, and a field too wide to pack at all. Running this
// through the matrix puts each of those forms in front of all four decode
// entry points under all sixteen option sets.
func mkAlphaBatch(n int) []alphaRow {
	const hexDigits = "0123456789abcdef"
	rows := make([]alphaRow, n)
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
	lower := func(w int) []byte {
		b := make([]byte, w)
		for j := range b {
			b[j] = byte('a' + next()%26)
		}
		return b
	}
	for i := range rows {
		spans := make([]alphaSpan, 1+next()%3)
		for k := range spans {
			esc := lower(20)
			// Every so often a byte outside the table this field has declared.
			if (i*4+k)%37 == 36 {
				esc[next()%20] = byte('!' + next()%14)
			}
			free := make([]byte, 24)
			for j := range free {
				free[j] = byte(32 + next()%95)
			}
			spans[k] = alphaSpan{
				TraceID: hex(32),
				SpanID:  hex(16),
				Token:   string(lower(20)),
				Escapee: string(esc),
				Free:    string(free),
			}
		}
		rows[i] = alphaRow{Seq: int64(i), Spans: spans, Attrs: map[string]string{"k": hex(8)}}
	}
	return rows
}

// TestStrDeltaMatrixTypedRoundTrip: decode into the original type and compare
// the values, then RE-ENCODE and compare the wires byte for byte.
//
// The re-encode is the part that matters. A base that drifted produces a
// decoded value that still looks plausible field by field, but re-encoding it
// walks the same per-field state again and the bytes diverge. Comparing only
// the decoded value would miss a drift that happens to reconstruct something
// self-consistent.
func TestStrDeltaMatrixTypedRoundTrip(t *testing.T) {
	for _, p := range strDeltaPayloads() {
		for _, o := range strDeltaOptionSets() {
			t.Run(fmt.Sprintf("%s/%s", p.name, o.name), func(t *testing.T) {
				b, err := qdf.Marshal(p.val, o.opts)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				out := reflect.New(reflect.TypeOf(p.val))
				if err := qdf.Unmarshal(b, out.Interface()); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				got := out.Elem().Interface()
				if !reflect.DeepEqual(got, p.val) {
					t.Fatalf("value mismatch after round-trip")
				}
				// The byte comparison only means something when map iteration
				// is pinned: several payloads carry maps, and without
				// OptCanonical Go's randomised order changes the bytes on every
				// encode regardless of anything this feature does.
				if o.opts.Has(qdf.OptCanonical) {
					b2, err := qdf.Marshal(got, o.opts)
					if err != nil {
						t.Fatalf("re-marshal: %v", err)
					}
					if !bytes.Equal(b, b2) {
						t.Fatalf("re-encoded wire differs: %d bytes vs %d — a per-field base drifted",
							len(b), len(b2))
					}
				}
			})
		}
	}
}

// TestStrDeltaMatrixDynamicDecode: decode into any.
//
// The dynamic reader is a separate entry point from the typed one; it walks
// values without a target struct, so a form it does not know surfaces as an
// error here and nowhere else.
func TestStrDeltaMatrixDynamicDecode(t *testing.T) {
	for _, p := range strDeltaPayloads() {
		for _, o := range strDeltaOptionSets() {
			t.Run(fmt.Sprintf("%s/%s", p.name, o.name), func(t *testing.T) {
				b, err := qdf.Marshal(p.val, o.opts)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				var out any
				if err := qdf.Unmarshal(b, &out); err != nil {
					t.Fatalf("dynamic unmarshal: %v", err)
				}
				if out == nil {
					t.Fatal("dynamic decode produced nil")
				}
			})
		}
	}
}

// TestStrDeltaMatrixSkipEverything: decode into a target that shares no fields
// with the payload, so every wire field routes through Skip.
//
// Skip must advance the per-field bases it walks past. It cannot reconstruct
// values it does not keep, but it must leave the state exactly where a full
// decode would — otherwise a payload that is partly skipped and partly decoded
// desynchronises.
func TestStrDeltaMatrixSkipEverything(t *testing.T) {
	type unrelated struct {
		Nothing string `qdf:"__nothing_matches_this__"`
	}
	for _, p := range strDeltaPayloads() {
		if reflect.TypeOf(p.val).Kind() != reflect.Struct {
			continue // a slice root cannot decode into a struct
		}
		for _, o := range strDeltaOptionSets() {
			t.Run(fmt.Sprintf("%s/%s", p.name, o.name), func(t *testing.T) {
				b, err := qdf.Marshal(p.val, o.opts)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				var out unrelated
				if err := qdf.Unmarshal(b, &out); err != nil {
					t.Fatalf("skip-everything unmarshal: %v", err)
				}
				if out.Nothing != "" {
					t.Fatalf("a field that does not exist decoded as %q", out.Nothing)
				}
			})
		}
	}
}

// TestStrDeltaMatrixStreamed: encode several messages through one encoder and
// decode them through one decoder, so the per-field bases cross message
// boundaries on both sides — or on neither.
func TestStrDeltaMatrixStreamed(t *testing.T) {
	for _, o := range strDeltaOptionSets() {
		t.Run(o.name, func(t *testing.T) {
			msgs := []AccessBatch{mkAccessBatch(256), mkAccessBatch(256), mkAccessBatch(256)}
			var wires [][]byte
			for _, m := range msgs {
				b, err := qdf.Marshal(m, o.opts)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				wires = append(wires, b)
			}
			for i, b := range wires {
				var out AccessBatch
				if err := qdf.Unmarshal(b, &out); err != nil {
					t.Fatalf("message %d: %v", i, err)
				}
				if !reflect.DeepEqual(out, msgs[i]) {
					t.Fatalf("message %d: value mismatch", i)
				}
			}
		})
	}
}
