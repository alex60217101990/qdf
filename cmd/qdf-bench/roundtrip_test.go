package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// sampleFiles is the set of localmachine JSON dumps the tests run against,
// resolved once by TestMain. Empty means no data was available (offline with no
// local clone) and the tests skip.
var sampleFiles []string

// TestMain makes the suite fully self-contained: with no override it downloads
// the adalanche-sampledata dumps into a temp dir, runs the tests, then removes
// the temp tree — `go test ./cmd/qdf-bench` needs nothing cloned by hand.
//
// Overrides (both skip the download, for offline / repeated runs):
//   - QDF_BENCH_DATAPATH=<clone>  — a local adalanche-sampledata clone.
//   - QDF_BENCH_SAMPLE=<file>     — a single localmachine JSON file.
func TestMain(m *testing.M) {
	os.Exit(resolveSamplesAndRun(m))
}

// resolveSamplesAndRun populates sampleFiles and runs the suite. It is split out
// so the temp-dir cleanup defer fires before os.Exit in TestMain.
func resolveSamplesAndRun(m *testing.M) int {
	if dp := os.Getenv("QDF_BENCH_DATAPATH"); dp != "" {
		sampleFiles, _ = filepath.Glob(filepath.Join(dp, "goad", "localmachine", "*.json"))
	}
	if len(sampleFiles) == 0 {
		if p := os.Getenv("QDF_BENCH_SAMPLE"); p != "" {
			if _, err := os.Stat(p); err == nil {
				sampleFiles = []string{p}
			}
		}
	}
	if len(sampleFiles) == 0 {
		dir, cleanup, err := fetchSampleData()
		if err != nil {
			// No network / no clone: leave sampleFiles empty so tests skip
			// rather than fail.
			fmt.Fprintf(os.Stderr, "qdf-bench test: sample data unavailable (%v); tests will skip\n", err)
		} else {
			defer cleanup()
			sampleFiles, _ = filepath.Glob(filepath.Join(dir, "*.json"))
		}
	}
	return m.Run()
}

// TestCodegenAnyField guards the qdfgen interface-field support: GenTask is a
// code-generated type (MarshalQDF / UnmarshalQDF) carrying a map[string]any
// Definition, so its round trip exercises the EncodeValue / DecodeValue fallback
// that lets generated code carry fully dynamic data. A regression that drops
// that support would either fail to generate or corrupt the dynamic field here.
func TestCodegenAnyField(t *testing.T) {
	in := GenTask{
		Name:    "Defrag",
		Path:    `\Microsoft\Windows\Defrag`,
		Enabled: true,
		State:   "Ready",
		Definition: map[string]any{
			"Triggers": []any{
				map[string]any{"Type": "Daily", "Interval": float64(1), "Enabled": true},
			},
			"Actions": []any{map[string]any{"Exec": "defrag.exe", "Args": "-c"}},
			"Author":  "Microsoft",
			"Nested":  map[string]any{"a": float64(2), "b": []any{"x", "y"}},
		},
	}
	// GenTask implements Marshaler/Unmarshaler, so Marshal/Unmarshal route through
	// the generated methods.
	b, err := qdf.Marshal(in, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got GenTask
	if err := qdf.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("codegen any-field round-trip mismatch:\n in=%#v\ngot=%#v", in, got)
	}
}

// TestRoundtripMatrix round-trips every sample file across the full encode
// option matrix (and, for the map representation, every decode mode), asserting
// a lossless DeepEqual on each combination.
func TestRoundtripMatrix(t *testing.T) {
	if len(sampleFiles) == 0 {
		t.Skip("no sample data (set QDF_BENCH_DATAPATH or QDF_BENCH_SAMPLE, or run online)")
	}
	for _, f := range sampleFiles {
		name := filepath.Base(f)
		info, err := loadTyped(f)
		if err != nil {
			t.Fatalf("loadTyped %s: %v", f, err)
		}
		m, err := loadMap(f)
		if err != nil {
			t.Fatalf("loadMap %s: %v", f, err)
		}

		for _, b := range bundles {
			b := b
			t.Run(fmt.Sprintf("%s/typed/%s", name, b.name), func(t *testing.T) {
				buf, err := qdf.MarshalT(*info, b.opts)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				var got Info
				if err := qdf.UnmarshalT(buf, &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if !reflect.DeepEqual(*info, got) {
					t.Fatal("typed roundtrip mismatch")
				}
			})

			for _, dm := range mapDecModes {
				dm := dm
				t.Run(fmt.Sprintf("%s/map/%s/%s", name, b.name, dm.name), func(t *testing.T) {
					buf, err := qdf.Marshal(m, b.opts)
					if err != nil {
						t.Fatalf("marshal: %v", err)
					}
					var got map[string]any
					if dm.build == nil {
						err = qdf.Unmarshal(buf, &got)
					} else {
						var arena *qdf.Arena
						if dm.usesArena {
							arena = qdf.NewArena()
						}
						err = qdf.Unmarshal(buf, &got, dm.build(arena))
					}
					if err != nil {
						t.Fatalf("unmarshal: %v", err)
					}
					if !reflect.DeepEqual(m, got) {
						t.Fatal("map roundtrip mismatch")
					}
				})
			}
		}
	}
}
