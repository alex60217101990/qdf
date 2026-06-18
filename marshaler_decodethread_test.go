package qdf

import "testing"

// threadProbe records the decoder identity DecodeQDF saw, to prove threading.
type threadProbe struct {
	V    int64
	seen *Decoder
}

func (t *threadProbe) MarshalQDF(dst []byte) ([]byte, error) {
	hadHeader := len(dst) >= 5 && dst[0] == Magic0 && dst[1] == Magic1 && dst[2] == Magic2
	e := NewEncoderOnBuf(dst, Fast)
	if hadHeader {
		e.MarkHeaderWritten()
	} else {
		e.EnsureHeader()
	}
	e.WriteMapHeader(1)
	e.WriteString("V")
	e.WriteInt(t.V)
	return e.Bytes(), nil
}

func (t *threadProbe) UnmarshalQDF(src []byte) (int, error) {
	d := NewDecoderOnBuf(src)
	hasMagic := len(src) >= 5 && src[0] == Magic0 && src[1] == Magic1 && src[2] == Magic2
	if !hasMagic {
		d.MarkHeaderRead()
	}
	if err := t.DecodeQDF(d); err != nil {
		return 0, err
	}
	return d.Pos(), nil
}

func (t *threadProbe) DecodeQDF(d *Decoder) error {
	t.seen = d
	n, err := d.ReadMapHeader()
	if err != nil {
		return err
	}
	for range n {
		kb, err := d.ReadStringBytes()
		if err != nil {
			return err
		}
		switch string(kb) {
		case "V":
			v, err := d.ReadInt()
			if err != nil {
				return err
			}
			t.V = v
		default:
			if err := d.Skip(); err != nil {
				return err
			}
		}
	}
	return nil
}

// TestDecodeNestedThreadsDecoder proves DecodeNested reads through the SAME
// decoder it is handed (no fresh decoder per nested value) and decodes
// back-to-back values correctly.
func TestDecodeNestedThreadsDecoder(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	if err := EncodeNested(e, &threadProbe{V: 7}); err != nil {
		t.Fatal(err)
	}
	if err := EncodeNested(e, &threadProbe{V: 9}); err != nil {
		t.Fatal(err)
	}
	buf := e.Bytes()

	d := NewDecoderOnBuf(buf) // buf has the stream header; first read consumes it
	var a, b threadProbe
	if err := DecodeNested(d, &a); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := DecodeNested(d, &b); err != nil {
		t.Fatalf("second: %v", err)
	}
	if a.V != 7 || b.V != 9 {
		t.Fatalf("values: a=%d b=%d", a.V, b.V)
	}
	if a.seen != d || b.seen != d {
		t.Fatal("DecodeNested did not thread the shared decoder")
	}
}

// plainOnly implements only Unmarshaler (no DecodeQDF) → exercises the
// buffer-based fallback path of DecodeNested.
type plainOnly struct{ V int64 }

func (p *plainOnly) MarshalQDF(dst []byte) ([]byte, error) {
	hadHeader := len(dst) >= 5 && dst[0] == Magic0 && dst[1] == Magic1 && dst[2] == Magic2
	e := NewEncoderOnBuf(dst, Fast)
	if hadHeader {
		e.MarkHeaderWritten()
	} else {
		e.EnsureHeader()
	}
	e.WriteMapHeader(1)
	e.WriteString("V")
	e.WriteInt(p.V)
	return e.Bytes(), nil
}

func (p *plainOnly) UnmarshalQDF(src []byte) (int, error) {
	d := NewDecoderOnBuf(src)
	hasMagic := len(src) >= 5 && src[0] == Magic0 && src[1] == Magic1 && src[2] == Magic2
	if !hasMagic {
		d.MarkHeaderRead()
	}
	n, err := d.ReadMapHeader()
	if err != nil {
		return 0, err
	}
	for range n {
		kb, err := d.ReadStringBytes()
		if err != nil {
			return 0, err
		}
		if string(kb) == "V" {
			v, err := d.ReadInt()
			if err != nil {
				return 0, err
			}
			p.V = v
		} else if err := d.Skip(); err != nil {
			return 0, err
		}
	}
	return d.Pos(), nil
}

func TestDecodeNestedFallback(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	if err := EncodeNested(e, &plainOnly{V: 5}); err != nil {
		t.Fatal(err)
	}
	if err := EncodeNested(e, &plainOnly{V: 6}); err != nil {
		t.Fatal(err)
	}
	buf := e.Bytes()

	d := NewDecoderOnBuf(buf) // buf has the stream header; first read consumes it
	var a, b plainOnly
	if err := DecodeNested(d, &a); err != nil {
		t.Fatal(err)
	}
	if err := DecodeNested(d, &b); err != nil {
		t.Fatal(err)
	}
	if a.V != 5 || b.V != 6 {
		t.Fatalf("fallback values a=%d b=%d", a.V, b.V)
	}
}
