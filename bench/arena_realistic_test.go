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

func BenchmarkArenaReal_AD_Off(b *testing.B)    { benchArenaDataset[[]ADUser](b, adBytes(), false) }
func BenchmarkArenaReal_AD_On(b *testing.B)     { benchArenaDataset[[]ADUser](b, adBytes(), true) }
func BenchmarkArenaReal_IoT_Off(b *testing.B)   { benchArenaDataset[IoTBatch](b, iotBytes(), false) }
func BenchmarkArenaReal_IoT_On(b *testing.B)    { benchArenaDataset[IoTBatch](b, iotBytes(), true) }
func BenchmarkArenaReal_Event_Off(b *testing.B) { benchArenaDataset[EventBatch](b, eventBytes(), false) }
func BenchmarkArenaReal_Event_On(b *testing.B)  { benchArenaDataset[EventBatch](b, eventBytes(), true) }
func BenchmarkArenaReal_Log_Off(b *testing.B)   { benchArenaDataset[LogBatch](b, logBytes(), false) }
func BenchmarkArenaReal_Log_On(b *testing.B)    { benchArenaDataset[LogBatch](b, logBytes(), true) }
