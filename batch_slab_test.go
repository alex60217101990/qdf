package qdf

import "testing"

func TestBatchSlabAppendResolve(t *testing.T) {
	s := newBatchSlab()
	off1, ln1 := s.append([]byte("hello"))
	off2, ln2 := s.append([]byte("world!"))
	if got := s.str(Str{off: off1, len: ln1}); got != "hello" {
		t.Fatalf("str1 = %q", got)
	}
	if got := s.str(Str{off: off2, len: ln2}); got != "world!" {
		t.Fatalf("str2 = %q", got)
	}
	// Growth must preserve earlier offsets (grow-copy keeps offsets stable).
	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = byte(i)
	}
	s.append(big)
	if got := s.str(Str{off: off1, len: ln1}); got != "hello" {
		t.Fatalf("str1 after grow = %q", got)
	}
	s.release()
	s2 := newBatchSlab()
	if s2 == nil {
		t.Fatal("pool returned nil")
	}
	s2.release()
}

func TestBatchSlabEmptyStr(t *testing.T) {
	s := newBatchSlab()
	defer s.release()
	if got := s.str(Str{}); got != "" {
		t.Fatalf("zero handle = %q, want empty", got)
	}
}

func TestBatchSlabBytesAndGrow(t *testing.T) {
	s := newBatchSlab()
	off, ln := s.append([]byte{1, 2, 3})
	if got := s.bytes(Bytes{off: off, len: ln}); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("bytes = %v", got)
	}
	if got := s.bytes(Bytes{}); got != nil {
		t.Fatalf("zero Bytes handle = %v, want nil", got)
	}
	// grow reserves capacity; earlier offsets stay valid afterwards.
	s.grow(1 << 16)
	if got := s.bytes(Bytes{off: off, len: ln}); len(got) != 3 || got[1] != 2 {
		t.Fatalf("bytes after grow = %v", got)
	}
	s.release()
}
