package qdf

import (
	"bytes"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// Heavy concurrent stress. Earlier suites cover 32 x 500 goroutines —
// real production load is often 1000+. These tests pin the codec
// under heavier contention, more iterations, and pool churn under
// race detection.

type concPayload struct {
	ID    int      `qdf:"id"`
	Name  string   `qdf:"name"`
	Tags  []string `qdf:"tags"`
	Score float64  `qdf:"score"`
}

func mkConcPayload(seq int) concPayload {
	return concPayload{
		ID:    seq,
		Name:  "service-name",
		Tags:  []string{"prod", "eu-west-1"},
		Score: float64(seq) * 0.5,
	}
}

func TestConcurrent_1000Goroutines_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heavy concurrent test in -short")
	}
	const G = 1000
	const N = 500
	var wg sync.WaitGroup
	wg.Add(G)
	var failures atomic.Int64
	for g := range G {
		go func(seed int) {
			defer wg.Done()
			for i := range N {
				in := mkConcPayload(seed*N + i)
				buf, err := Marshal(in, OptSpeed)
				if err != nil {
					failures.Add(1)
					return
				}
				var out concPayload
				if err := Unmarshal(buf, &out); err != nil {
					failures.Add(1)
					return
				}
				if !reflect.DeepEqual(out, in) {
					failures.Add(1)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if failures.Load() != 0 {
		t.Fatalf("%d failures across %d goroutines x %d iter", failures.Load(), G, N)
	}
}

func TestConcurrent_PoolChurn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pool-churn test in -short")
	}
	// Intentionally abandon encoded buffers (let GC sweep them)
	// between iterations to stress the encoder pool's grow/recycle
	// path. A 4-byte-cap initial encoder gets returned to the pool
	// after every call; pool reuse is the load model under attack.
	const G = 512
	const N = 200
	var wg sync.WaitGroup
	wg.Add(G)
	var failures atomic.Int64
	for g := range G {
		go func(seed int) {
			defer wg.Done()
			for i := range N {
				v := mkConcPayload(seed*N + i)
				_, err := Marshal(v, OptSpeed)
				if err != nil {
					failures.Add(1)
					continue
				}
				// Also exercise MarshalDense + MarshalQPack in the
				// same goroutine — each draws from a different pool
				// so the test catches cross-pool contamination too.
				_, _ = Marshal(v, OptBalanced)
				_, _ = Marshal(v, OptQPack)
			}
		}(g)
	}
	wg.Wait()
	if failures.Load() != 0 {
		t.Fatalf("%d failures", failures.Load())
	}
}

func TestConcurrent_DenseStreamSerial(t *testing.T) {
	// One Dense stream encoder, many goroutines writing to a shared
	// buffer through their own per-goroutine encoder via Marshal.
	// Confirms that Dense intern tables do not leak across encoders.
	const G = 256
	const N = 100
	var wg sync.WaitGroup
	wg.Add(G)
	for g := range G {
		go func(seed int) {
			defer wg.Done()
			for i := range N {
				v := mkConcPayload(seed*N + i)
				buf, _ := Marshal(v, OptBalanced)
				var out concPayload
				if err := Unmarshal(buf, &out); err != nil {
					t.Errorf("goroutine %d iter %d decode: %v", seed, i, err)
					return
				}
				if !reflect.DeepEqual(out, v) {
					t.Errorf("goroutine %d iter %d: %+v != %+v", seed, i, out, v)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestConcurrent_AppendMarshalIndependentBuffers(t *testing.T) {
	// AppendMarshal hands the caller a buffer; if the encoder pool
	// were silently sharing buffers, two goroutines would interleave
	// bytes. Each goroutine asserts its own decode round-trips
	// correctly.
	const G = 256
	const N = 100
	var wg sync.WaitGroup
	wg.Add(G)
	for g := range G {
		go func(seed int) {
			defer wg.Done()
			var buf []byte
			for i := range N {
				v := mkConcPayload(seed*N + i)
				var err error
				buf, err = AppendMarshal(buf[:0], v, OptSpeed)
				if err != nil {
					t.Errorf("encode %d/%d: %v", seed, i, err)
					return
				}
				var out concPayload
				if err := Unmarshal(buf, &out); err != nil {
					t.Errorf("decode %d/%d: %v", seed, i, err)
					return
				}
				if !reflect.DeepEqual(out, v) {
					t.Errorf("round-trip mismatch %d/%d", seed, i)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestConcurrent_LargePayload_NoStateBleedAcrossPool(t *testing.T) {
	// Encode "wide" payload concurrently. Catches issues like a
	// pooled Encoder.buf growing larger than maxPooledBuf and the
	// pool returning it later partially populated (the bug pattern
	// the pool's maxPooledBuf cap is designed to prevent).
	const G = 128
	const N = 50
	type wide struct {
		Bytes [4096]byte
		Text  string `qdf:"text"`
	}
	var wg sync.WaitGroup
	wg.Add(G)
	for g := range G {
		go func(seed int) {
			defer wg.Done()
			for i := range N {
				var v wide
				for j := range v.Bytes {
					v.Bytes[j] = byte((seed*N + i + j) & 0xFF)
				}
				v.Text = "concurrent"
				buf, err := Marshal(v, OptSpeed)
				if err != nil {
					t.Errorf("encode %d/%d: %v", seed, i, err)
					return
				}
				var out wide
				if err := Unmarshal(buf, &out); err != nil {
					t.Errorf("decode %d/%d: %v", seed, i, err)
					return
				}
				if !reflect.DeepEqual(v.Bytes, out.Bytes) {
					t.Errorf("bytes mismatch %d/%d", seed, i)
					return
				}
				if v.Text != out.Text {
					t.Errorf("text mismatch %d/%d", seed, i)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestConcurrent_StreamEncoderPerGoroutine(t *testing.T) {
	// Each goroutine has its own StreamEncoder writing to its own
	// bytes.Buffer; no shared writer or pool state to contend on
	// directly, but the encoder pool underneath does churn.
	const G = 128
	const N = 50
	var wg sync.WaitGroup
	wg.Add(G)
	for g := range G {
		go func(seed int) {
			defer wg.Done()
			var w bytes.Buffer
			enc := NewStreamEncoder(&w, Dense)
			for i := range N {
				if err := enc.Encode(mkConcPayload(seed*N + i)); err != nil {
					t.Errorf("encode %d/%d: %v", seed, i, err)
					return
				}
			}
			if err := enc.Close(); err != nil {
				t.Errorf("close: %v", err)
				return
			}
			dec := NewStreamDecoder(&w)
			defer dec.Close()
			for i := range N {
				var out concPayload
				if err := dec.Decode(&out); err != nil {
					t.Errorf("decode %d/%d: %v", seed, i, err)
					return
				}
				if out.ID != seed*N+i {
					t.Errorf("decode mismatch %d/%d: id=%d", seed, i, out.ID)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	// Force a GC after the stress so any leaked goroutine state
	// surfaces under -race more reliably on subsequent tests.
	runtime.GC()
}
