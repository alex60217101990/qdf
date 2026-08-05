package tans

// Regression probes adopted from the adversarial correctness review
// (2026-08-03): reference-formula equivalence for the strength-reduced
// decode table, worst-case 12-bit-per-symbol streams, and buildFreqs
// normalization invariants.

import (
	"bytes"
	"math/bits"
	"math/rand"
	"testing"
)

// naiveDecTable is the per-entry reference formula for buildDecTable.
func naiveDecTable(freq *[256]uint32, dec *[TableSize]DecEntry) {
	var cumul [256]uint32
	var c uint32
	for s := range 256 {
		cumul[s] = c
		c += freq[s]
	}
	for s := range 256 {
		f := freq[s]
		for j := range f {
			y := f + j
			nb := uint32(TableLog+1) - uint32(bits.Len32(y))
			dec[cumul[s]+j] = DecEntry(s) | DecEntry(nb)<<8 | DecEntry(y<<nb)<<16
		}
	}
}

func randomFreqTable(rng *rand.Rand) [256]uint32 {
	var freq [256]uint32
	k := 1 + rng.Intn(256)
	perm := rng.Perm(256)[:k]
	for _, s := range perm {
		freq[s] = 1
	}
	for total := uint32(k); total < TableSize; total++ {
		freq[perm[rng.Intn(k)]]++
	}
	return freq
}

func TestDecTableMatchesNaive(t *testing.T) {
	checkTable := func(freq *[256]uint32, tag string) {
		var got, want [TableSize]DecEntry
		buildDecTable(freq, &got)
		naiveDecTable(freq, &want)
		if got != want {
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("%s: entry %d got %08x want %08x", tag, i, uint32(got[i]), uint32(want[i]))
				}
			}
		}
	}
	// All f in 1..4096 as symbol 7, remainder on symbol 200.
	for f := uint32(1); f <= TableSize; f++ {
		var freq [256]uint32
		freq[7] = f
		if f < TableSize {
			freq[200] = TableSize - f
		}
		checkTable(&freq, "two-symbol f="+itoa(int(f)))
	}
	rng := rand.New(rand.NewSource(1))
	for iter := range 200 {
		freq := randomFreqTable(rng)
		checkTable(&freq, "random iter="+itoa(iter))
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

// checkAlgebra exhaustively verifies encode step vs decode entry for every
// state in [TableSize, 2*TableSize) and every present symbol.
func checkAlgebra(t *testing.T, freq *[256]uint32, tag string) {
	t.Helper()
	var enc [256]EncSymbol
	var dec [TableSize]DecEntry
	buildEncTable(freq, &enc)
	buildDecTable(freq, &dec)
	for s := range 256 {
		f := freq[s]
		if f == 0 {
			continue
		}
		e := enc[s]
		for st := uint32(TableSize); st < 2*TableSize; st++ {
			nb := (st + e.DeltaNbBits) >> 16
			if nb > TableLog {
				t.Fatalf("%s: s=%d f=%d st=%d nb=%d > TableLog", tag, s, f, st, nb)
			}
			top := st >> nb
			if top < f || top >= 2*f {
				t.Fatalf("%s: s=%d f=%d st=%d nb=%d top=%d not in [f,2f)", tag, s, f, st, nb, top)
			}
			next := uint32(int32(top) + e.DeltaFindState + int32(TableSize))
			if next < TableSize || next >= 2*TableSize {
				t.Fatalf("%s: s=%d f=%d st=%d next=%d out of range", tag, s, f, st, next)
			}
			d := dec[next&(TableSize-1)]
			if d.Symbol() != byte(s) {
				t.Fatalf("%s: s=%d f=%d st=%d decoded sym=%d", tag, s, f, st, d.Symbol())
			}
			if uint32(d.NbBits()) != nb {
				t.Fatalf("%s: s=%d f=%d st=%d enc nb=%d dec nb=%d", tag, s, f, st, nb, d.NbBits())
			}
			if d.NewBase()+(st&(1<<nb-1)) != st {
				t.Fatalf("%s: s=%d f=%d st=%d newBase=%d low=%d != st", tag, s, f, st, d.NewBase(), st&(1<<nb-1))
			}
		}
	}
}

// (Encode never produces this table for this src, but the codec must handle it).
func TestMaxBitsStreams(t *testing.T) {
	var freq [256]uint32
	freq['A'] = 1
	freq['B'] = 4095

	for _, n := range []int{1, 2, 3, 4, 5, 7, 8, 100, 4094, 4095} {
		src := bytes.Repeat([]byte{'A'}, n)
		blob := encodeStream(nil, src, &freq)
		got, err := decodeStream(blob, &freq, n)
		if err != nil {
			t.Fatalf("single n=%d: %v", n, err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("single n=%d: mismatch", n)
		}
	}

	for _, n := range []int{4096, 4097, 4098, 4099, 4100, 8191, 8192, 8193} {
		src := bytes.Repeat([]byte{'A'}, n)
		blob := appendInterleaved4(nil, src, &freq)
		got, err := decodeInterleaved4(blob, &freq, n)
		if err != nil {
			t.Fatalf("inter4 n=%d: %v", n, err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("inter4 n=%d: mismatch", n)
		}
	}
}

func TestBuildFreqsSum(t *testing.T) {
	sum := func(f [256]uint32) uint32 {
		var s uint32
		for _, v := range f {
			s += v
		}
		return s
	}
	inputs := map[string][]byte{
		"one-byte": {42},
		"distinct256": func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}(),
		"skew-1M": func() []byte {
			b := bytes.Repeat([]byte{0xEE}, 1000000)
			b[0] = 1
			return b
		}(),
		"255-rare": func() []byte {
			b := bytes.Repeat([]byte{0}, 1000000)
			for i := range 255 {
				b[i+1] = byte(i + 1)
			}
			return b
		}(),
	}
	rng := rand.New(rand.NewSource(5))
	for i := range 50 {
		b := make([]byte, 1+rng.Intn(100000))
		rng.Read(b)
		inputs["rand"+itoa(i)] = b
	}
	for tag, src := range inputs {
		f := buildFreqs(src)
		if got := sum(f); got != TableSize {
			t.Fatalf("%s: freq sum = %d, want %d", tag, got, TableSize)
		}
		for s, v := range f {
			if bytes.IndexByte(src, byte(s)) >= 0 && v == 0 {
				t.Fatalf("%s: occurring symbol %d has freq 0", tag, s)
			}
		}
	}
}

// TestEncodeDecodeAlgebra runs the exhaustive encode/decode-step check over
// boundary and random normalized tables.
func TestEncodeDecodeAlgebra(t *testing.T) {
	mk := func(pairs ...uint32) *[256]uint32 { // symbol,freq pairs
		var f [256]uint32
		for i := 0; i+1 < len(pairs); i += 2 {
			f[pairs[i]] = pairs[i+1]
		}
		return &f
	}
	checkAlgebra(t, mk('a', 1, 'b', 4095), "f=1/4095")
	checkAlgebra(t, mk('a', 2, 'b', 4094), "f=2/4094")
	checkAlgebra(t, mk('x', 4096), "single=4096")
	checkAlgebra(t, mk('a', 4095, 'b', 1), "f=4095/1")
	checkAlgebra(t, mk('a', 2048, 'b', 2048), "2048x2")
	checkAlgebra(t, mk('a', 3, 'b', 5, 'c', 4088), "3/5/4088")
	var many [256]uint32
	for s := range 240 {
		many[s] = 1
	}
	many[255] = 4096 - 240
	checkAlgebra(t, &many, "240xf=1")
	rng := rand.New(rand.NewSource(31))
	for range 10 {
		src := make([]byte, 4096)
		for i := range src {
			src[i] = byte(rng.Intn(1 + rng.Intn(256)))
		}
		freq := buildFreqs(src)
		checkAlgebra(t, &freq, "random")
	}
}
