package bench

import (
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// decodeArenaEqualsPlain decodes src into a fresh *T with and without an arena
// and asserts the results are deeply equal — the arena must not change values,
// only where their bytes live. Exercises reflect struct / map / slice / []byte
// decode paths that an arena threads through.
func decodeArenaEqualsPlain[T any](t *testing.T, src []byte) {
	t.Helper()
	var plain T
	if err := qdf.Unmarshal(src, &plain); err != nil {
		t.Fatalf("plain decode: %v", err)
	}
	a := qdf.NewArena()
	var arena T
	if err := qdf.Unmarshal(src, &arena, qdf.WithArena(a)); err != nil {
		t.Fatalf("arena decode: %v", err)
	}
	if !reflect.DeepEqual(plain, arena) {
		t.Fatalf("arena decode differs from plain decode")
	}
}

func TestArenaRealistic_AD(t *testing.T) {
	users := makeADUsers(200)
	src, err := qdf.Marshal(&users, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	decodeArenaEqualsPlain[[]ADUser](t, src)
}

func TestArenaRealistic_IoT(t *testing.T) {
	v := mkIoTBatch(32, 64)
	src, err := qdf.Marshal(&v, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	decodeArenaEqualsPlain[IoTBatch](t, src)
}

func TestArenaRealistic_Event(t *testing.T) {
	v := mkEventBatch(500)
	src, err := qdf.Marshal(&v, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	decodeArenaEqualsPlain[EventBatch](t, src)
}

func TestArenaRealistic_Log(t *testing.T) {
	v := MakeLogBatch(500)
	src, err := qdf.Marshal(&v, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	decodeArenaEqualsPlain[LogBatch](t, src)
}

// benchArenaDataset runs an arena on/off epoch loop over a marshaled dataset.
func benchArenaDataset[T any](b *testing.B, src []byte, withArena bool) {
	b.ReportAllocs()
	if withArena {
		a := qdf.NewArena()
		for b.Loop() {
			a.Reset()
			var v T
			if err := qdf.Unmarshal(src, &v, qdf.WithArena(a)); err != nil {
				b.Fatal(err)
			}
		}
		return
	}
	for b.Loop() {
		var v T
		if err := qdf.Unmarshal(src, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func adBytes() []byte    { u := makeADUsers(200); b, _ := qdf.Marshal(&u, qdf.OptSpeed); return b }
func iotBytes() []byte   { v := mkIoTBatch(32, 64); b, _ := qdf.Marshal(&v, qdf.OptSpeed); return b }
func eventBytes() []byte { v := mkEventBatch(500); b, _ := qdf.Marshal(&v, qdf.OptSpeed); return b }
func logBytes() []byte   { v := MakeLogBatch(500); b, _ := qdf.Marshal(&v, qdf.OptSpeed); return b }

func BenchmarkArenaReal_AD_Off(b *testing.B)  { benchArenaDataset[[]ADUser](b, adBytes(), false) }
func BenchmarkArenaReal_AD_On(b *testing.B)   { benchArenaDataset[[]ADUser](b, adBytes(), true) }
func BenchmarkArenaReal_IoT_Off(b *testing.B) { benchArenaDataset[IoTBatch](b, iotBytes(), false) }
func BenchmarkArenaReal_IoT_On(b *testing.B)  { benchArenaDataset[IoTBatch](b, iotBytes(), true) }
func BenchmarkArenaReal_Event_Off(b *testing.B) {
	benchArenaDataset[EventBatch](b, eventBytes(), false)
}

func BenchmarkArenaReal_Event_On(b *testing.B) { benchArenaDataset[EventBatch](b, eventBytes(), true) }
func BenchmarkArenaReal_Log_Off(b *testing.B)  { benchArenaDataset[LogBatch](b, logBytes(), false) }
func BenchmarkArenaReal_Log_On(b *testing.B)   { benchArenaDataset[LogBatch](b, logBytes(), true) }

// Dense-tier arena coverage. The Speed-mode benches above exercise the plain
// string(b) copy path; these encode Balanced (Dense, intern table) so the
// arena is exercised through the intern materialisation in decState.getString.
// Before that path consulted the arena, a Dense decode of a high-cardinality
// string column got no arena benefit at all (every first-sight intern record
// heap-allocated); now it amortises the same way the Speed path does.
func adBytesDense() []byte { u := makeADUsers(200); b, _ := qdf.Marshal(&u, qdf.OptBalanced); return b }

func logBytesDense() []byte {
	v := MakeLogBatch(500)
	b, _ := qdf.Marshal(&v, qdf.OptBalanced)
	return b
}

func iotBytesDense() []byte {
	v := mkIoTBatch(32, 64)
	b, _ := qdf.Marshal(&v, qdf.OptBalanced)
	return b
}

func BenchmarkArenaDense_AD_Off(b *testing.B) { benchArenaDataset[[]ADUser](b, adBytesDense(), false) }

func BenchmarkArenaDense_AD_On(b *testing.B) { benchArenaDataset[[]ADUser](b, adBytesDense(), true) }

func BenchmarkArenaDense_Log_Off(b *testing.B) {
	benchArenaDataset[LogBatch](b, logBytesDense(), false)
}

func BenchmarkArenaDense_Log_On(b *testing.B) { benchArenaDataset[LogBatch](b, logBytesDense(), true) }

func BenchmarkArenaDense_IoT_Off(b *testing.B) {
	benchArenaDataset[IoTBatch](b, iotBytesDense(), false)
}

func BenchmarkArenaDense_IoT_On(b *testing.B) { benchArenaDataset[IoTBatch](b, iotBytesDense(), true) }

// TestArenaDense_EqualsPlain guards that an arena decode of Dense/Compression
// wire is byte-for-byte equal to a plain heap decode — the arena only changes
// where the interned string copies live, never their contents.
func TestArenaDense_EqualsPlain(t *testing.T) {
	for _, n := range []int{1, 7, 50, 200, 1000} {
		u := makeADUsers(n)
		for _, opt := range []qdf.Options{qdf.OptBalanced, qdf.OptCompression} {
			src, err := qdf.Marshal(&u, opt)
			if err != nil {
				t.Fatal(err)
			}
			var plain, arena []ADUser
			if err := qdf.Unmarshal(src, &plain); err != nil {
				t.Fatal(err)
			}
			a := qdf.NewArena()
			if err := qdf.Unmarshal(src, &arena, qdf.WithArena(a)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(plain, arena) {
				t.Fatalf("n=%d opt=%v: arena decode diverges from plain decode", n, opt)
			}
		}
	}
}

// TestArenaDense_ResetReuse exercises the epoch-reuse pattern: a SINGLE arena
// is reused across many distinct messages with Reset between them (the pattern
// the benchmarks use for zero per-message block allocation). Each message is
// compared to a plain decode BEFORE the next Reset, since Reset invalidates the
// strings of the previous epoch. This guards that interned-string
// materialisation stays correct across Reset cycles — the cached stringValues
// table is per-decode (fresh decState), so a Reset of the shared arena must not
// resurrect or corrupt a prior epoch's bytes.
func TestArenaDense_ResetReuse(t *testing.T) {
	for _, opt := range []qdf.Options{qdf.OptBalanced, qdf.OptCompression} {
		a := qdf.NewArena()
		// Vary size per epoch so block sizes/offsets differ across Resets and a
		// stale read from a longer prior epoch would surface as a mismatch.
		for epoch, n := range []int{200, 3, 500, 1, 50} {
			u := makeADUsers(n)
			src, err := qdf.Marshal(&u, opt)
			if err != nil {
				t.Fatal(err)
			}
			var plain []ADUser
			if err := qdf.Unmarshal(src, &plain); err != nil {
				t.Fatal(err)
			}
			a.Reset() // reuse the arena's block(s) from the previous epoch
			var arena []ADUser
			if err := qdf.Unmarshal(src, &arena, qdf.WithArena(a)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(plain, arena) {
				t.Fatalf("epoch=%d n=%d opt=%v: reset-reused arena decode diverges from plain decode", epoch, n, opt)
			}
		}
	}
}
