package qdf

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// updateGolden regenerates the testdata/golden/*.bin fixtures when set.
// Use:  go test -run='TestGolden_' -update
var updateGolden = flag.Bool("update", false, "rewrite golden testdata/golden/*.bin fixtures")

// Golden-file tests pin the on-wire byte sequence for a representative
// payload set. They guard the format against accidental breakage: if a
// later change unintentionally shifts a tag or a length encoding, the
// stored bytes no longer match what the encoder produces and the test
// fails loudly. Regenerate intentionally via `go test -update` after a
// wire-format bump.
//
// Each fixture lives under testdata/golden/<name>.<dialect>.bin. A
// matching decode round-trip is asserted from the same fixture so a
// stale-but-still-valid wire is not silently accepted.

type goldenCase struct {
	name  string
	value any // encoded source; decoder gets a fresh pointer of the same type
	zero  func() any
	// nonDeterministic marks cases (notably Go maps) whose encoded byte
	// sequence depends on Go's randomised iteration order. The decode-
	// then-compare half of the round-trip still runs; only the byte-
	// pin half is skipped.
	nonDeterministic bool
}

func goldenCases() []goldenCase {
	type smallStruct struct {
		ID   int    `qdf:"id"`
		Name string `qdf:"name"`
	}
	type nested struct {
		A int         `qdf:"a"`
		B smallStruct `qdf:"b"`
	}
	type bigBatch struct {
		Bools []bool    `qdf:"bools"`
		IDs   []uint64  `qdf:"ids"`
		Vec   []float64 `qdf:"vec"`
		Tags  []string  `qdf:"tags"`
	}

	mkBigBatch := func() bigBatch {
		return bigBatch{
			Bools: []bool{true, false, true, false, true, false, true, false},
			IDs:   []uint64{1_700_000_000, 1_700_000_001, 1_700_000_002, 1_700_000_003},
			Vec:   []float64{0.5, 1.0, 1.5, 2.0, 2.5},
			Tags:  []string{"prod", "prod", "stage", "prod"},
		}
	}

	return []goldenCase{
		{
			name:  "primitives",
			value: smallStruct{ID: 42, Name: "alice"},
			zero:  func() any { return &smallStruct{} },
		},
		{
			name:  "nested",
			value: nested{A: 7, B: smallStruct{ID: 99, Name: "bob"}},
			zero:  func() any { return &nested{} },
		},
		{
			name:  "bigbatch",
			value: mkBigBatch(),
			zero:  func() any { return &bigBatch{} },
		},
		{
			name:  "empty_slice",
			value: []int{},
			zero:  func() any { v := []int{}; return &v },
		},
		{
			name:  "string_slice_repeated",
			value: []string{"eu-west-1", "eu-west-1", "eu-west-1", "eu-west-1"},
			zero:  func() any { v := []string{}; return &v },
		},
		{
			name:  "monotonic_u64",
			value: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			zero:  func() any { v := []uint64{}; return &v },
		},
		{
			name:  "bool_slice",
			value: []bool{true, true, false, true, false, false, true, false, true},
			zero:  func() any { v := []bool{}; return &v },
		},
		{
			name:             "map_string_int",
			value:            map[string]int{"a": 1, "b": 2, "c": 3},
			zero:             func() any { v := map[string]int{}; return &v },
			nonDeterministic: true,
		},
	}
}

func goldenPath(name, dialect string) string {
	return filepath.Join("testdata", "golden", name+"."+dialect+".bin")
}

func TestGolden_Fast(t *testing.T) {
	runGolden(t, "fast", OptSpeed)
}

func TestGolden_QPack(t *testing.T) {
	runGolden(t, "qpack", OptQPack)
}

func TestGolden_Dense(t *testing.T) {
	runGolden(t, "dense", OptBalanced)
}

// TestGolden_ColIndex pins the on-wire bytes of a columnar []struct slice
// encoded with the column-length index (OptColumnIndex). The shared
// goldenCases table applies one Options value uniformly to every case and
// has no per-case override, so the indexed wire — which only materialises
// when OptColumnIndex is set on a columnar struct slice — gets its own
// fixture here. Same mechanism as the other golden tests: byte-pin plus a
// decode round-trip read back from disk, regenerated via `go test -update`.
func TestGolden_ColIndex(t *testing.T) {
	if err := os.MkdirAll(filepath.Join("testdata", "golden"), 0o755); err != nil {
		t.Fatal(err)
	}
	rows := mkSelFull(8)
	got, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	path := goldenPath("colindex_struct_slice", "colindex")
	if *updateGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d bytes -> %s", len(got), path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing golden %s — run with -update", path)
		}
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("wire mismatch:\n want=%x\n  got=%x", want, got)
	}
	var out []selFull
	if err := Unmarshal(want, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(rows, out) {
		t.Fatalf("decode mismatch:\n want=%+v\n  got=%+v", rows, out)
	}
}

func runGolden(t *testing.T, dialect string, opts Options) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join("testdata", "golden"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range goldenCases() {
		t.Run(c.name, func(t *testing.T) {
			got, err := Marshal(c.value, opts)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			path := goldenPath(c.name, dialect)
			if *updateGolden {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("wrote %d bytes -> %s", len(got), path)
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					t.Fatalf("missing golden %s — run with -update", path)
				}
				t.Fatal(err)
			}
			if !c.nonDeterministic && !bytes.Equal(want, got) {
				t.Fatalf("wire mismatch:\n want=%x\n  got=%x", want, got)
			}
			// Decode from the on-disk bytes (not the just-encoded ones)
			// so a stale wire never round-trips through the in-memory
			// representation alone.
			outPtr := c.zero()
			if err := Unmarshal(want, outPtr); err != nil {
				t.Fatalf("decode: %v", err)
			}
			outVal := reflect.ValueOf(outPtr).Elem().Interface()
			if !reflect.DeepEqual(c.value, outVal) {
				t.Fatalf("decode mismatch:\n want=%+v\n  got=%+v", c.value, outVal)
			}
		})
	}
}
