package qdf

import "testing"

// FuzzUnmarshalBatch mutates valid batch wires (row-major, columnar, and a
// columnar+OptCompression wire) and requires UnmarshalBatch[batDoc] to never
// panic: either it returns an error, or it returns a Batch whose handles all
// resolve in-bounds. Under -race/-tags qdfdebug the resolve calls also carry
// the epoch/bounds checks from batch_check_debug.go, so a decode that
// produces a handle pointing outside the slab (rather than just erroring)
// fails loudly here instead of silently reading garbage in production.
func FuzzUnmarshalBatch(f *testing.F) {
	src := mkBatSrc(64)
	seed1, err := Marshal(src, OptBalanced) // row-major-eligible size, no columnar opts
	if err != nil {
		f.Fatal(err)
	}
	seed2, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern) // columnar wire
	if err != nil {
		f.Fatal(err)
	}
	seed3, err := Marshal(src, OptCompression|OptDense|OptShapeIntern) // columnar + rANS
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed1)
	f.Add(seed2)
	f.Add(seed3)
	f.Fuzz(func(t *testing.T, data []byte) {
		b, err := UnmarshalBatch[batDoc](data)
		if err != nil {
			return
		}
		for i := range b.Rows {
			// NOT dead code: resolving every handle is the fuzz oracle — under
			// -race / -tags qdfdebug the slab resolve bounds-checks the handle
			// and panics on any out-of-slab offset a hostile wire produced.
			_ = b.Str(b.Rows[i].Name)
			_ = b.TimeOf(b.Rows[i].At)
		}
		b.Release()
	})
}
