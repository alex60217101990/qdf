package qdf

import (
	"strconv"
	"testing"
)

// The per-field delta base is the one piece of state that must advance
// identically on both sides. When it does not, nothing errors: the wire is
// still well-formed, the types still line up, and a field simply decodes to a
// string built from the wrong prefix. Selected round-trip tests cannot find
// that — the drift needs a field to be skipped, or a form to alternate, in a
// combination nobody wrote down.
//
// These targets generate the combination instead. The fuzzer picks how many
// fields each side declares, which values repeat, and how long the shared
// prefixes are, so the encode and decode shapes disagree in ways the test
// author did not choose.

// fuzzWide is the widest shape; the narrower targets below drop fields from it,
// which is what forces the decoder to skip — and to advance the base of a field
// it is throwing away.
type fuzzWide struct {
	A string `qdf:"a"`
	B string `qdf:"b"`
	C string `qdf:"c"`
	D int64  `qdf:"d"`
	E string `qdf:"e"`
}

type fuzzDropA struct {
	B string `qdf:"b"`
	C string `qdf:"c"`
	D int64  `qdf:"d"`
	E string `qdf:"e"`
}

type fuzzDropMiddle struct {
	A string `qdf:"a"`
	E string `qdf:"e"`
}

type fuzzOnlyLast struct {
	E string `qdf:"e"`
}

// fuzzMakeRows builds rows whose string fields exercise every form the string
// path can choose between: a fresh value with a long shared prefix (delta
// wins), a repeat of the previous value (a one-byte state-ref wins), a value
// with no prefix in common (the plain intern form wins), a well-known alphabet
// (packed with no table), a restricted alphabet that has to be learned and
// declared, a value that falls just outside such a table, and a value too wide
// to pack. The seed decides which case each field takes on each row, so the
// forms interleave — which is the point. A declared table is per-field state
// carried across values, exactly like the delta base, and it desyncs the same
// way if a form in between fails to maintain it.
func fuzzMakeRows(n int, seed uint32, prefixLen int) []fuzzWide {
	if n < 1 {
		n = 1
	}
	if n > 512 {
		n = 512
	}
	if prefixLen < 0 {
		prefixLen = 0
	}
	if prefixLen > 64 {
		prefixLen = 64
	}
	pfx := ""
	for len(pfx) < prefixLen {
		pfx += "abcdefghij"
	}
	pfx = pfx[:prefixLen]

	rows := make([]fuzzWide, n)
	s := seed | 1
	next := func() uint32 {
		s ^= s << 13
		s ^= s >> 17
		s ^= s << 5
		return s
	}
	const hexDigits = "0123456789abcdef"
	val := func(field string, i int) string {
		switch next() % 7 {
		case 0: // shared prefix, fresh tail — the delta's territory
			return pfx + field + strconv.Itoa(i)
		case 1: // repeat of a small pool — a state-ref should win
			return pfx + field + strconv.Itoa(i%3)
		case 2: // no shared prefix
			return strconv.Itoa(int(next())) + field
		case 3: // a well-known alphabet, packed with no table at all
			b := make([]byte, 32)
			for j := range b {
				b[j] = hexDigits[next()%16]
			}
			return string(b)
		case 4: // a restricted alphabet that is nobody's well-known set, so the
			// field has to learn a table and declare it
			b := make([]byte, 20)
			for j := range b {
				b[j] = byte('a' + next()%26)
			}
			return string(b)
		case 5: // one byte outside that learned set — the value that must NOT
			// be allowed to widen a table already on the wire
			b := make([]byte, 20)
			for j := range b {
				b[j] = byte('a' + next()%26)
			}
			b[next()%20] = byte('!' + next()%14)
			return string(b)
		default: // too wide to pack at all
			b := make([]byte, 24)
			for j := range b {
				b[j] = byte(32 + next()%95)
			}
			return string(b)
		}
	}
	for i := range rows {
		rows[i] = fuzzWide{
			A: val("a", i),
			B: val("b", i),
			C: val("c", i),
			D: int64(i),
			E: val("e", i),
		}
	}
	return rows
}

// FuzzStrDeltaSchemaEvolution encodes the wide shape and decodes it into
// narrower ones. Every dropped field is skipped by the decoder, and every
// skipped value still has to advance that field's state — its delta base and,
// for a field that declared one, its alphabet table. A table announced inside a
// value the decoder throws away is the sharpest case: the reader never returns
// the string, but it must still have built the mapping, or the next reference
// to that field decodes against a table it does not have.
func FuzzStrDeltaSchemaEvolution(f *testing.F) {
	f.Add(64, uint32(1), 24)
	f.Add(3, uint32(7), 0)
	f.Add(512, uint32(0xdeadbeef), 64)
	f.Add(17, uint32(12345), 1)

	f.Fuzz(func(t *testing.T, n int, seed uint32, prefixLen int) {
		rows := fuzzMakeRows(n, seed, prefixLen)
		for _, opts := range []Options{
			OptBalanced,
			OptBalanced | OptCanonical,
			OptBalanced &^ OptMTF,
			OptBalanced &^ OptPairPred,
			OptCompression,
		} {
			b, err := Marshal(rows, opts)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			// Full shape: every value must come back exactly.
			var full []fuzzWide
			if err := Unmarshal(b, &full); err != nil {
				t.Fatalf("full decode: %v", err)
			}
			if len(full) != len(rows) {
				t.Fatalf("full decode: %d rows, want %d", len(full), len(rows))
			}
			for i := range rows {
				if full[i] != rows[i] {
					t.Fatalf("full decode row %d:\n got %+v\nwant %+v", i, full[i], rows[i])
				}
			}

			// Narrowed shapes: the fields that survive must be untouched by the
			// fields that were skipped.
			var dropA []fuzzDropA
			if err := Unmarshal(b, &dropA); err != nil {
				t.Fatalf("dropA decode: %v", err)
			}
			for i := range rows {
				if dropA[i].B != rows[i].B || dropA[i].C != rows[i].C || dropA[i].E != rows[i].E {
					t.Fatalf("dropA row %d: got %+v, want B=%q C=%q E=%q",
						i, dropA[i], rows[i].B, rows[i].C, rows[i].E)
				}
			}

			var dropMid []fuzzDropMiddle
			if err := Unmarshal(b, &dropMid); err != nil {
				t.Fatalf("dropMiddle decode: %v", err)
			}
			for i := range rows {
				if dropMid[i].A != rows[i].A || dropMid[i].E != rows[i].E {
					t.Fatalf("dropMiddle row %d: got %+v, want A=%q E=%q",
						i, dropMid[i], rows[i].A, rows[i].E)
				}
			}

			var last []fuzzOnlyLast
			if err := Unmarshal(b, &last); err != nil {
				t.Fatalf("onlyLast decode: %v", err)
			}
			for i := range rows {
				if last[i].E != rows[i].E {
					t.Fatalf("onlyLast row %d: got %q want %q", i, last[i].E, rows[i].E)
				}
			}

			// Dynamic decode walks the same values without a target struct.
			var any0 any
			if err := Unmarshal(b, &any0); err != nil {
				t.Fatalf("dynamic decode: %v", err)
			}
		}
	})
}

// FuzzStrDeltaWireIsSelfConsistent re-encodes what it decoded and compares the
// bytes under OptCanonical. A base that drifted still produces a decoded value
// that looks plausible field by field; re-encoding walks the same per-field
// state again and the bytes diverge, which is the only cheap way to see it.
func FuzzStrDeltaWireIsSelfConsistent(f *testing.F) {
	f.Add(64, uint32(1), 24)
	f.Add(200, uint32(99), 40)

	f.Fuzz(func(t *testing.T, n int, seed uint32, prefixLen int) {
		rows := fuzzMakeRows(n, seed, prefixLen)
		opts := OptBalanced | OptStringAlphabet | OptCanonical
		b, err := Marshal(rows, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out []fuzzWide
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		b2, err := Marshal(out, opts)
		if err != nil {
			t.Fatalf("re-marshal: %v", err)
		}
		if len(b) != len(b2) {
			t.Fatalf("re-encoded wire is %d bytes, first was %d — a per-field base drifted",
				len(b2), len(b))
		}
		for i := range b {
			if b[i] != b2[i] {
				t.Fatalf("re-encoded wire differs at byte %d (0x%02x vs 0x%02x) — a per-field base drifted",
					i, b[i], b2[i])
			}
		}
	})
}
