package bufpool

import "testing"

// TestGetHonorsCapHint pins the documented "cap >= hint" contract. Put files a
// buffer by its cap, so a sub-class buffer returned by a caller lands in a pool
// a later Get of that class draws from; Get must not hand back an undersized
// buffer (it drops it and allocates a right-sized one).
func TestGetHonorsCapHint(t *testing.T) {
	small := make([]byte, 0, 8) // below classSmall
	Put(&small)
	// Pull enough times to cycle the round-robin shards and surface the buffer.
	for range 200 {
		b := Get(classSmall)
		if cap(*b) < classSmall {
			t.Fatalf("Get(%d) returned cap=%d < hint", classSmall, cap(*b))
		}
		*b = (*b)[:cap(*b)] // a caller would index up to hint; must not panic
	}
}

// TestGetClassesAndPutRoundTrip sanity-checks each size class returns a buffer
// with cap >= hint and that Put/Get round-trips without panic.
func TestGetClassesAndPutRoundTrip(t *testing.T) {
	for _, hint := range []int{0, 1, classSmall, classMedium, classLarge, classHuge, classHuge + 1} {
		b := Get(hint)
		if cap(*b) < hint {
			t.Fatalf("Get(%d): cap=%d < hint", hint, cap(*b))
		}
		if len(*b) != 0 {
			t.Fatalf("Get(%d): len=%d, want 0", hint, len(*b))
		}
		Put(b)
	}
}
