package qdf

import (
	"math"
	"math/bits"
	"slices"
)

// Chimp128 XOR codec for []float64, from "Chimp: Efficient Lossless Floating
// Point Compression" (Liakos, Papakonstantinopoulou, Kotidis; VLDB 2022).
// Successor to the Gorilla XOR codec: instead of XORing only against the
// immediately previous value, each value may reference any of the previous
// 128 values — the candidate is found via a hash on the low bits, and is
// used when the XOR against it has more than `chimpThreshold` trailing
// zeros. Control codes are 2 bits (vs Gorilla's 1/2-bit prefix scheme) and
// leading-zero counts are rounded onto a 3-bit code. NaN and ±Inf survive
// unchanged (pure bit-pattern transform).
//
// Wire: shares tagPackGorilla framing with a distinct kind byte
// (qpackKindChimp64): tag, kind, varuint(n), first u64 LE,
// uvarint(totalBits), body (MSB-first bit stream, same bitWriter/bitReader
// as Gorilla). Skip() handles it unchanged — qpackRawWidthBytes(kind)
// yields 8.
//
// Per value (after the raw 64-bit first value), with ref = chosen previous
// value's ring index (7 bits):
//
//	flag 00: xor == 0            -> 9 bits total: [00][7-bit ref]
//	flag 01: trailingZeros > 13  -> 18 bits [01][7-bit ref][3-bit lzCode]
//	                                [6-bit sigBits] + sigBits of xor>>tz
//	flag 10: lz == storedLZ      -> [10] + (64-lz) bits of xor (prev ref)
//	flag 11: otherwise           -> [11][3-bit lzCode] + (64-lz) bits of xor
//
// storedLZ persists across flag-10/11 values and is invalidated (65) by
// flags 00/01, mirroring the reference implementation (ChimpN.java).
const (
	chimpNLog2     = 7                         // window = 128 previous values
	chimpN         = 1 << chimpNLog2           // ring size
	chimpThreshold = 6 + chimpNLog2            // min trailing zeros to take an indexed ref
	chimpLsbMask   = 1<<(chimpThreshold+1) - 1 // low-bits hash key (14 bits)
)

// chimpLeadingRound rounds a leading-zero count down onto the 8 representable
// values {0,8,12,16,18,20,22,24}; chimpLeadingRep is its 3-bit code.
var chimpLeadingRound = [65]uint8{
	0, 0, 0, 0, 0, 0, 0, 0,
	8, 8, 8, 8, 12, 12, 12, 12,
	16, 16, 18, 18, 20, 20, 22, 22,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24, 24,
}

var chimpLeadingRep = [65]uint8{
	0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 2, 2, 2, 2,
	3, 3, 4, 4, 5, 5, 6, 6,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7,
}

// chimpRepToLZ inverts chimpLeadingRep for the decoder.
var chimpRepToLZ = [8]uint8{0, 8, 12, 16, 18, 20, 22, 24}

// chimpScratch is the encoder-owned hash window for Chimp128. indices[key]
// holds the insertion index of the last value whose low 14 bits were key,
// valid only when stamps[key] == epoch — bumping epoch invalidates the whole
// table in O(1) instead of zeroing 64 KiB per encoded slice. The stored ring
// needs no clearing either: the encoder only dereferences slots gated by a
// current-epoch stamp, so the wire stays deterministic across encoder reuse.
type chimpScratch struct {
	epoch   uint32
	indices [chimpLsbMask + 1]int32
	stamps  [chimpLsbMask + 1]uint32
	stored  [chimpN]uint64
}

// chimpScratchFor returns the encoder's scratch with a fresh epoch.
func (e *Encoder) chimpScratchFor() *chimpScratch {
	if e.chimpScr == nil {
		e.chimpScr = new(chimpScratch)
	}
	cs := e.chimpScr
	cs.epoch++
	if cs.epoch == 0 { // uint32 wrap: hard-reset stamps once per 4G slices
		clear(cs.stamps[:])
		cs.epoch = 1
	}
	return cs
}

// pickChimpF64 cheaply projects Chimp128's bits/value over a 32-transition
// prefix. The dominant cost case (flag 11) spends 5 + (64 - roundedLZ(xor))
// bits; window hits (flags 00/01) only reduce that, so the projection is a
// safe upper bound on smooth data. Chimp is worth attempting when the
// projection sits comfortably below raw's 64 bits/value — a looser gate than
// pickF64Codec's Gorilla projection (mb+14), which over-prices Chimp's
// lz-only trailing layout and rejects slices Chimp compresses by 20%+.
func pickChimpF64(s []float64) bool {
	n := len(s)
	if n < 8 {
		return false // fixed header dominates tiny slices
	}
	probe := min(32, n-1)
	var total uint64
	prev := math.Float64bits(s[0])
	for i := 1; i <= probe; i++ {
		cur := math.Float64bits(s[i])
		x := cur ^ prev
		if x == 0 {
			total += 9 // flag 00
		} else {
			total += 5 + 64 - uint64(chimpLeadingRound[bits.LeadingZeros64(x)])
		}
		prev = cur
	}
	return total/uint64(probe) < 58
}

// writePackedChimpFloat64Slice emits a Chimp128-coded []float64 under
// tagPackGorilla with kind qpackKindChimp64. Framing mirrors
// writePackedGorillaFloat64Slice: n=0 writes only tag/kind/n; n=1 also
// writes the first value and a zero bit count.
func (e *Encoder) writePackedChimpFloat64Slice(s []float64) {
	e.writeHeader()
	n := len(s)
	out := slices.Grow(e.buf, 2+10+8+10+10*n)
	out = append(out, tagPackGorilla, qpackKindChimp64)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	first := math.Float64bits(s[0])
	out = appendU64(out, first)
	if n == 1 {
		out = appendUvarint(out, 0)
		e.buf = out
		return
	}

	cs := e.chimpScratchFor()
	epoch := cs.epoch
	stored := &cs.stored

	bodyStart := len(out)
	bw := bitWriter{buf: out}

	stored[0] = first
	cs.indices[first&chimpLsbMask] = 0
	cs.stamps[first&chimpLsbMask] = epoch
	index := int32(0) // insertion index of the newest value
	current := 0      // ring slot of the newest value
	storedLZ := uint8(65)

	for i := 1; i < n; i++ {
		v := math.Float64bits(s[i])
		key := v & chimpLsbMask

		ref := current
		xor := stored[current] ^ v
		tz := 0
		if cs.stamps[key] == epoch {
			candIdx := cs.indices[key]
			if index-candIdx < chimpN {
				cand := int(candIdx) & (chimpN - 1)
				tempXor := v ^ stored[cand]
				tz = bits.TrailingZeros64(tempXor)
				if tz > chimpThreshold {
					ref = cand
					xor = tempXor
				}
			}
		}

		if xor == 0 {
			// flag 00 + 7-bit ref, packed as one 9-bit write.
			bw.writeBits(uint64(ref), chimpNLog2+2)
			storedLZ = 65
		} else if tz > chimpThreshold {
			// flag 01 + ref + lz code + significant-bit count, one 18-bit write.
			lz := chimpLeadingRound[bits.LeadingZeros64(xor)]
			sig := 64 - int(lz) - tz
			bw.writeBits(
				uint64(chimpN+ref)<<9|uint64(chimpLeadingRep[lz])<<6|uint64(sig),
				chimpNLog2+11)
			bw.writeBits(xor>>tz, uint8(sig))
			storedLZ = 65
		} else {
			lz := chimpLeadingRound[bits.LeadingZeros64(xor)]
			if lz == storedLZ {
				bw.writeBits(2, 2) // flag 10
			} else {
				storedLZ = lz
				bw.writeBits(uint64(24+chimpLeadingRep[lz]), 5) // flag 11 + 3-bit code
			}
			bw.writeBits(xor, 64-lz)
		}

		current = (current + 1) & (chimpN - 1)
		stored[current] = v
		index++
		cs.indices[key] = index
		cs.stamps[key] = epoch
	}
	totalBits := bw.flush()
	e.buf = finishGorillaBody(bw.buf, bodyStart, totalBits)
}

// readPackedChimpFloat64Slice decodes a Chimp128 body. The tag must already
// be consumed and the next byte must be qpackKindChimp64.
func (d *Decoder) readPackedChimpFloat64Slice() ([]float64, error) {
	n, firstU, body, err := d.readPackedGorillaHeader(qpackKindChimp64)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []float64{}, nil
	}
	out := make([]float64, n)
	out[0] = math.Float64frombits(firstU)
	if n == 1 {
		return out, nil
	}

	var stored [chimpN]uint64
	stored[0] = firstU
	current := 0
	storedLZ := uint8(65)

	i := 1

	// Fast path: big-endian 64-bit window over the MSB-first stream, refilled
	// by whole bytes (mirrors the tans decoder's word-wise reader, but reading
	// forward). Each iteration starts with a refill, so consumed <= 7 and up
	// to 57 bits are readable without another refill; the two long-body cases
	// refill once more mid-value, and a 64-bit body (storedLZ == 0) is read as
	// two 32-bit halves. The `pos+32 <= len(body)` entry guard over-covers the
	// worst per-iteration advance (<= 24 bytes), and the remaining tail runs
	// through the byte-wise bitReader below.
	var container uint64
	pos := 0
	consumed := uint(0)
	for ; i < n && pos+32 <= len(body); i++ {
		pos += int(consumed >> 3)
		consumed &= 7
		container = readU64BE(body[pos:])

		var v uint64
		flag := container << consumed >> 62
		consumed += 2
		switch flag {
		case 0:
			ref := container << consumed >> 57
			consumed += chimpNLog2
			v = stored[ref]
			storedLZ = 65
		case 1:
			hdr := container << consumed >> 48
			consumed += chimpNLog2 + 9
			ref := hdr >> 9 & (chimpN - 1)
			lz := chimpRepToLZ[hdr>>6&7]
			sig := uint(hdr & 63)
			tz := 64 - int(lz) - int(sig)
			if sig == 0 || tz <= chimpThreshold {
				return nil, ErrBadTag
			}
			pos += int(consumed >> 3)
			consumed &= 7
			container = readU64BE(body[pos:])
			mb := container << consumed >> (64 - sig)
			consumed += sig
			v = stored[ref] ^ (mb << tz)
			storedLZ = 65
		default: // 2 or 3: xor against previous value
			if flag == 3 {
				storedLZ = chimpRepToLZ[container<<consumed>>61]
				consumed += 3
			} else if storedLZ == 65 {
				return nil, ErrBadTag
			}
			cb := 64 - uint(storedLZ)
			pos += int(consumed >> 3)
			consumed &= 7
			container = readU64BE(body[pos:])
			var mb uint64
			if cb > 57 { // storedLZ == 0: split into two 32-bit halves
				hi := container << consumed >> 32
				consumed += 32
				pos += int(consumed >> 3)
				consumed &= 7
				container = readU64BE(body[pos:])
				lo := container << consumed >> 32
				consumed += 32
				mb = hi<<32 | lo
			} else {
				mb = container << consumed >> (64 - cb)
				consumed += cb
			}
			v = stored[current] ^ mb
		}
		current = (current + 1) & (chimpN - 1)
		stored[current] = v
		out[i] = math.Float64frombits(v)
	}

	// Byte-wise tail (also covers bodies < 32 bytes entirely).
	br := bitReader{buf: body, pos: pos + int(consumed>>3), used: uint8(consumed & 7)}
	for ; i < n; i++ {
		flag, ok := br.readBits(2)
		if !ok {
			return nil, ErrShortBuffer
		}
		var v uint64
		switch flag {
		case 0: // identical to referenced window value
			ref, ok := br.readBits(chimpNLog2)
			if !ok {
				return nil, ErrShortBuffer
			}
			v = stored[ref]
			storedLZ = 65
		case 1: // indexed ref + trimmed xor
			hdr, ok := br.readBits(chimpNLog2 + 9)
			if !ok {
				return nil, ErrShortBuffer
			}
			ref := hdr >> 9 & (chimpN - 1)
			lz := chimpRepToLZ[hdr>>6&7]
			sig := int(hdr & 63)
			tz := 64 - int(lz) - sig
			// A valid encoder emits sig >= 1 and tz > chimpThreshold; anything
			// else is a corrupt or hostile stream.
			if sig == 0 || tz <= chimpThreshold {
				return nil, ErrBadTag
			}
			mb, ok := br.readBits(uint8(sig))
			if !ok {
				return nil, ErrShortBuffer
			}
			v = stored[ref] ^ (mb << tz)
			storedLZ = 65
		case 2: // xor against previous value, reuse stored leading-zero count
			if storedLZ == 65 {
				return nil, ErrBadTag // no valid window: encoder never emits this
			}
			mb, ok := br.readBits(64 - storedLZ)
			if !ok {
				return nil, ErrShortBuffer
			}
			v = stored[current] ^ mb
		default: // 3: xor against previous value, new leading-zero count
			rep, ok := br.readBits(3)
			if !ok {
				return nil, ErrShortBuffer
			}
			storedLZ = chimpRepToLZ[rep]
			mb, ok := br.readBits(64 - storedLZ)
			if !ok {
				return nil, ErrShortBuffer
			}
			v = stored[current] ^ mb
		}
		current = (current + 1) & (chimpN - 1)
		stored[current] = v
		out[i] = math.Float64frombits(v)
	}
	return out, nil
}
