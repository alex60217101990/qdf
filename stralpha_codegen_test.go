package qdf

import (
	"fmt"
	"testing"
)

type cgSpan struct {
	TraceID string `qdf:"trace_id"`
	SpanID  string `qdf:"span_id"`
	Name    string `qdf:"name"`
}

type cgRec struct {
	Seq   int64    `qdf:"seq"`
	Spans []cgSpan `qdf:"spans"`
}

// decodeLikeQdfgen walks a struct exactly as generated code does: read the
// header, then EnterField / LeaveField around every field, string or not.
//
// This is the sixth reader of struct values, and the only one whose behavior
// was argued from the shape of the API rather than exercised. A generated
// decoder that failed to bind the field context would not error — it would
// return a value built from another field's alphabet, or fail on a reference
// to a table it never recorded.
func decodeLikeQdfgen(d *Decoder, dst *cgSpan) error {
	names, plainN, shaped, err := d.ReadStructHeader()
	if err != nil {
		return err
	}
	read := func(name string) error {
		v, err := d.ReadString()
		if err != nil {
			return err
		}
		switch name {
		case "trace_id":
			dst.TraceID = v
		case "span_id":
			dst.SpanID = v
		case "name":
			dst.Name = v
		default:
			return fmt.Errorf("unexpected field %q", name)
		}
		return nil
	}
	if shaped {
		shapeID := d.ShapeID()
		for i, name := range names {
			d.EnterField(shapeID, len(names), i)
			err := read(name)
			d.LeaveField()
			if err != nil {
				return err
			}
		}
		return nil
	}
	for range plainN {
		kb, err := d.ReadStringBytes()
		if err != nil {
			return err
		}
		if err := read(string(kb)); err != nil {
			return err
		}
	}
	return nil
}

// A wire carrying packed values must decode through the generated-decoder API.
// Generated ENCODERS do not emit the form — they call WriteString, which has
// neither this codec nor the string delta — but a generated decoder has to read
// what a reflect-encoded producer wrote, or the two stop interoperating.
func TestStrAlphaDecodesThroughTheGeneratedDecoderAPI(t *testing.T) {
	const hexDigits = "0123456789abcdef"
	seed := uint64(0x9E3779B97F4A7C15)
	hex := func(w int) string {
		b := make([]byte, w)
		for j := range b {
			seed = seed*6364136223846793005 + 1442695040888963407
			b[j] = hexDigits[(seed>>33)%16]
		}
		return string(b)
	}
	// span_id is lowercase-only: a restricted set that is nobody's well-known
	// alphabet, so the field must LEARN a table, declare it, and reference it
	// afterwards. That is what makes this test about the field context at all —
	// a well-known form carries its alphabet in the selector byte and decodes
	// with no recollection of the field it came from, so a wire of nothing but
	// hex ids would pass even with the binding wrong.
	lower := func(w int) string {
		b := make([]byte, w)
		for j := range b {
			seed = seed*6364136223846793005 + 1442695040888963407
			b[j] = byte('a' + (seed>>33)%26)
		}
		return string(b)
	}
	// name gets its own restricted set, deliberately DIFFERENT from span_id's.
	// Two fields each declaring a table is what makes a wrong field binding
	// detectable: with only one declaring field, a decoder that bound every
	// field to the same slot would still be self-consistent and the test would
	// pass while proving nothing.
	upper := func(w int) string {
		b := make([]byte, w)
		for j := range b {
			seed = seed*6364136223846793005 + 1442695040888963407
			b[j] = byte('A' + (seed>>33)%26)
		}
		return string(b)
	}
	recs := make([]cgRec, 400)
	for i := range recs {
		recs[i].Seq = int64(i)
		recs[i].Spans = []cgSpan{
			{hex(32), lower(20), upper(20)},
			{hex(32), lower(20), upper(20)},
		}
	}

	bw, bd, br := strAlphaEmittedWK.Load(), strAlphaEmittedDecl.Load(), strAlphaEmittedRef.Load()
	b, err := Marshal(recs, OptBalanced|OptStringAlphabet)
	if err != nil {
		t.Fatal(err)
	}
	wk := strAlphaEmittedWK.Load() - bw
	decl, ref := strAlphaEmittedDecl.Load()-bd, strAlphaEmittedRef.Load()-br
	if decl < 2 {
		t.Fatalf("only %d table declared; the test needs two colliding ones", decl)
	}
	if wk == 0 || decl == 0 || ref == 0 {
		t.Fatalf("the wire lacks a form this test needs: wk=%d decl=%d ref=%d", wk, decl, ref)
	}

	// Decode the whole batch with reflect, then re-read every span through the
	// generated-decoder API and compare.
	var got []cgRec
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for i := range recs {
		for k := range recs[i].Spans {
			if got[i].Spans[k] != recs[i].Spans[k] {
				t.Fatalf("reflect decode, rec %d span %d: got %+v want %+v",
					i, k, got[i].Spans[k], recs[i].Spans[k])
			}
		}
	}

	// Now the generated path, on a wire of bare spans so the walk is the one
	// generated code performs.
	spans := make([]cgSpan, 0, len(recs)*2)
	for i := range recs {
		spans = append(spans, recs[i].Spans...)
	}
	sb, err := Marshal(spans, OptBalanced|OptStringAlphabet)
	if err != nil {
		t.Fatal(err)
	}
	d := NewDecoderOnBuf(sb)
	n, err := d.ReadArrayHeader()
	if err != nil {
		t.Fatal(err)
	}
	if n != len(spans) {
		t.Fatalf("array header says %d, wrote %d", n, len(spans))
	}
	for i := range spans {
		var out cgSpan
		if err := decodeLikeQdfgen(d, &out); err != nil {
			t.Fatalf("generated-style decode of span %d: %v", i, err)
		}
		if out != spans[i] {
			t.Fatalf("generated-style decode, span %d: got %+v want %+v", i, out, spans[i])
		}
	}
}
