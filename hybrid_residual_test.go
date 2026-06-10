package qdf

import (
	"reflect"
	"testing"
)

// Regression: hybrid columnar left the per-column length bound (colMaxLen = n)
// active while decoding the row-major residual fields, so a residual slice/map
// longer than the batch row count n was wrongly rejected with ErrInvalidLength.
// Residual fields are row-major and may hold any number of elements.

type hybResidualRow struct {
	Level  string         `qdf:"level"`  // low-card → FSST → hybrid fires
	Region string         `qdf:"region"` // low-card
	Seq    int64          `qdf:"seq"`
	Nums   []int          `qdf:"nums"`  // residual slice
	Attrs  map[string]int `qdf:"attrs"` // residual map
}

func TestHybridResidualLongerThanRowCount(t *testing.T) {
	const n = 200 // row count; residual collections are deliberately longer
	levels := []string{"INFO", "WARN", "ERROR"}
	regions := []string{"us", "eu", "ap"}
	in := make([]hybResidualRow, n)
	for i := range in {
		nums := make([]int, 256) // > n
		for j := range nums {
			nums[j] = j * 3
		}
		attrs := make(map[string]int, 300) // > n
		for j := 0; j < 300; j++ {
			attrs[string(rune('a'+j%26))+itoaSmall(j)] = j
		}
		in[i] = hybResidualRow{
			Level: levels[i%3], Region: regions[i%3], Seq: int64(i),
			Nums: nums, Attrs: attrs,
		}
	}

	b, err := Marshal(in, OptCompression)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Typed decode path (decodeHybridColumnar).
	var out []hybResidualRow
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("typed hybrid decode failed on residual longer than n: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("typed hybrid round-trip mismatch")
	}

	// Schemaless decode path (decodeHybridColumnarAny).
	var anyOut any
	if err := Unmarshal(b, &anyOut); err != nil {
		t.Fatalf("any hybrid decode failed on residual longer than n: %v", err)
	}
	rows, ok := anyOut.([]any)
	if !ok || len(rows) != n {
		t.Fatalf("any hybrid shape: %T", anyOut)
	}
	row0 := rows[0].(map[string]any)
	// A homogeneous int slice decodes schemalessly to []int64 (packed-slice path).
	if nums, ok := row0["nums"].([]int64); !ok || len(nums) != 256 {
		t.Fatalf("any residual slice wrong: %T", row0["nums"])
	}
}

func itoaSmall(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
