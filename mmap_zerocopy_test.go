//go:build unix

package qdf_test

import (
	"os"
	"testing"
	"unsafe"

	"github.com/alex60217101990/qdf"
	"golang.org/x/sys/unix"
)

// TestMmapZeroCopyDecode exercises the documented mmap + WithNoCopy pattern
// (docs/DECODE-PERF.md): decode a store mapped read-only straight from the page
// cache, with the decoded string/[]byte values aliasing the mapped pages.
func TestMmapZeroCopyDecode(t *testing.T) {
	type Record struct {
		ID   string
		Tags []string
	}
	rows := make([]Record, 256)
	for i := range rows {
		rows[i] = Record{ID: "doc-id-aliases-the-mapping", Tags: []string{"alpha", "beta"}}
	}
	blob, err := qdf.Marshal(rows, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(t.TempDir(), "store-*.qdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(blob); err != nil {
		t.Fatal(err)
	}
	f.Close()

	rf, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer rf.Close()
	st, _ := rf.Stat()
	data, err := unix.Mmap(int(rf.Fd()), 0, int(st.Size()), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		t.Skipf("mmap unavailable: %v", err) // some CI sandboxes disallow MAP_SHARED
	}
	defer unix.Munmap(data) // must outlive `out` (the WithNoCopy lifetime contract)

	var out []Record
	if err := qdf.Unmarshal(data, &out, qdf.WithNoCopy()); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(rows) {
		t.Fatalf("len %d want %d", len(out), len(rows))
	}
	for i := range rows {
		if out[i].ID != rows[i].ID || len(out[i].Tags) != 2 {
			t.Fatalf("row %d round-trip mismatch", i)
		}
	}

	// Confirm the decoded string actually aliases the mapped region (zero-copy),
	// not a heap copy: its data pointer lies within the mmap'd byte range.
	base := uintptr(unsafe.Pointer(&data[0]))
	end := base + uintptr(len(data))
	sp := uintptr(unsafe.Pointer(unsafe.StringData(out[0].ID)))
	if sp < base || sp >= end {
		t.Fatalf("WithNoCopy string did not alias the mmap (ptr %x not in [%x,%x))", sp, base, end)
	}
}
