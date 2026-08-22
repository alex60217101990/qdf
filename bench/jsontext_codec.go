package bench

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"sort"
)

// Hand-written JSON codecs over encoding/json/jsontext, as the ceiling of what
// JSON can do on these fixtures.
//
// The filename matters: an earlier name for this file was jsontext_arm.go, and
// _arm is a GOARCH suffix — Go silently excluded the whole file from the build
// and every check passed while nothing in here compiled.
//
// What this arm is: JSON with the reflection removed. jsontext is a token
// stream, so the struct walk that json.Marshal performs at run time is written
// out here at compile time, field by field. Nothing is discovered by reflect;
// the encoder appends tokens into a buffer it keeps between calls.
//
// What this arm is NOT: a fair comparison against qdf's *reflect* tiers. Its
// honest counterpart is qdf_codegen, which is also a compile-time struct walk.
// Reading it against qdf_speed or qdf_balanced flatters JSON, and the tables say
// so where they show it.
//
// The output is byte-compatible with encoding/json v1 semantics — map keys
// sorted, nil slices as null — because otherwise the size column would not line
// up with the v1 row it sits beside. Sorting is the one place this codec does
// work v2's defaults skip, and it is deliberate: it is what makes the row
// comparable.

// jsontextEncoder is a reusable encode context. The Encoder and the buffer both
// survive between calls — that reuse is the point, since a fresh Encoder per
// message would put allocation back in exactly the place this arm exists to
// remove.
type jsontextEncoder struct {
	buf  bytes.Buffer
	enc  *jsontext.Encoder
	keys []string // scratch for sorted map keys
}

func newJSONTextEncoder() *jsontextEncoder {
	e := &jsontextEncoder{}
	e.enc = jsontext.NewEncoder(io.Discard)
	return e
}

// reset points the encoder at its own buffer for a fresh message.
func (e *jsontextEncoder) reset() {
	e.buf.Reset()
	e.enc.Reset(&e.buf)
}

// writeStringMapSorted writes a map[string]string with its keys in sorted order,
// matching encoding/json v1. The key scratch is reused across calls.
func (e *jsontextEncoder) writeStringMapSorted(m map[string]string) error {
	if m == nil {
		return e.enc.WriteToken(jsontext.Null)
	}
	if err := e.enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	e.keys = e.keys[:0]
	for k := range m {
		e.keys = append(e.keys, k)
	}
	sort.Strings(e.keys)
	for _, k := range e.keys {
		if err := e.enc.WriteToken(jsontext.String(k)); err != nil {
			return err
		}
		if err := e.enc.WriteToken(jsontext.String(m[k])); err != nil {
			return err
		}
	}
	return e.enc.WriteToken(jsontext.EndObject)
}

func (e *jsontextEncoder) writeInt64Slice(s []int64) error {
	if s == nil {
		return e.enc.WriteToken(jsontext.Null)
	}
	if err := e.enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, v := range s {
		if err := e.enc.WriteToken(jsontext.Int(v)); err != nil {
			return err
		}
	}
	return e.enc.WriteToken(jsontext.EndArray)
}

func (e *jsontextEncoder) writeFloat64Slice(s []float64) error {
	if s == nil {
		return e.enc.WriteToken(jsontext.Null)
	}
	if err := e.enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, v := range s {
		if err := e.enc.WriteToken(jsontext.Float(v)); err != nil {
			return err
		}
	}
	return e.enc.WriteToken(jsontext.EndArray)
}

// marshalIoTBatch writes an IoTBatch as JSON without reflecting over it. The
// returned slice aliases the encoder's buffer and is valid until the next
// marshal on the same encoder — the caller in a benchmark reads its length and
// drops it, which is the usage this is shaped for.
func (e *jsontextEncoder) marshalIoTBatch(v *IoTBatch) ([]byte, error) {
	e.reset()
	if err := e.enc.WriteToken(jsontext.BeginObject); err != nil {
		return nil, err
	}
	if err := e.enc.WriteToken(jsontext.String("devices")); err != nil {
		return nil, err
	}
	if v.Devices == nil {
		if err := e.enc.WriteToken(jsontext.Null); err != nil {
			return nil, err
		}
	} else {
		if err := e.enc.WriteToken(jsontext.BeginArray); err != nil {
			return nil, err
		}
		for i := range v.Devices {
			if err := e.writeDeviceReading(&v.Devices[i]); err != nil {
				return nil, err
			}
		}
		if err := e.enc.WriteToken(jsontext.EndArray); err != nil {
			return nil, err
		}
	}
	if err := e.enc.WriteToken(jsontext.EndObject); err != nil {
		return nil, err
	}
	// jsontext.Encoder is a stream encoder and terminates a top-level value with
	// a newline; json.Marshal does not. Trimming it is what makes the two
	// outputs byte-comparable, and the size column honest — a stray byte per
	// message is invisible in a log and shows up as a real difference at scale.
	return bytes.TrimSuffix(e.buf.Bytes(), []byte("\n")), nil
}

func (e *jsontextEncoder) writeDeviceReading(d *DeviceReading) error {
	if err := e.enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String("device_id")); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String(d.DeviceID)); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String("ts")); err != nil {
		return err
	}
	if err := e.writeInt64Slice(d.Ts); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String("temp")); err != nil {
		return err
	}
	if err := e.writeFloat64Slice(d.Temp); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String("humidity")); err != nil {
		return err
	}
	if err := e.writeFloat64Slice(d.Humidity); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String("tags")); err != nil {
		return err
	}
	if err := e.writeStringMapSorted(d.Tags); err != nil {
		return err
	}
	return e.enc.WriteToken(jsontext.EndObject)
}

// jsontextDecoder mirrors the encoder: one reusable Decoder, no per-message
// allocation of the machinery itself.
type jsontextDecoder struct {
	dec *jsontext.Decoder
	rd  bytes.Reader
}

func newJSONTextDecoder() *jsontextDecoder {
	d := &jsontextDecoder{}
	d.dec = jsontext.NewDecoder(bytes.NewReader(nil))
	return d
}

func (d *jsontextDecoder) reset(data []byte) {
	d.rd.Reset(data)
	d.dec.Reset(&d.rd)
}

// unmarshalIoTBatch reads what marshalIoTBatch wrote. It reads by position
// rather than by name: this decoder's whole purpose is to skip the name lookup
// that reflection cannot avoid, and it is paired with the encoder above, whose
// field order is fixed.
func (d *jsontextDecoder) unmarshalIoTBatch(data []byte, out *IoTBatch) error {
	d.reset(data)
	if _, err := d.dec.ReadToken(); err != nil { // {
		return err
	}
	if _, err := d.dec.ReadToken(); err != nil { // "devices"
		return err
	}
	if d.dec.PeekKind() == 'n' {
		if _, err := d.dec.ReadToken(); err != nil {
			return err
		}
		out.Devices = nil
		return nil
	}
	if _, err := d.dec.ReadToken(); err != nil { // [
		return err
	}
	out.Devices = out.Devices[:0]
	for d.dec.PeekKind() != ']' {
		var dr DeviceReading
		if err := d.readDeviceReading(&dr); err != nil {
			return err
		}
		out.Devices = append(out.Devices, dr)
	}
	if _, err := d.dec.ReadToken(); err != nil { // ]
		return err
	}
	_, err := d.dec.ReadToken() // }
	return err
}

func (d *jsontextDecoder) readDeviceReading(dr *DeviceReading) error {
	if _, err := d.dec.ReadToken(); err != nil { // {
		return err
	}
	for d.dec.PeekKind() != '}' {
		nameTok, err := d.dec.ReadToken()
		if err != nil {
			return err
		}
		switch nameTok.String() {
		case "device_id":
			t, err := d.dec.ReadToken()
			if err != nil {
				return err
			}
			dr.DeviceID = t.String()
		case "ts":
			if dr.Ts, err = d.readInt64Slice(dr.Ts); err != nil {
				return err
			}
		case "temp":
			if dr.Temp, err = d.readFloat64Slice(dr.Temp); err != nil {
				return err
			}
		case "humidity":
			if dr.Humidity, err = d.readFloat64Slice(dr.Humidity); err != nil {
				return err
			}
		case "tags":
			if dr.Tags, err = d.readStringMap(); err != nil {
				return err
			}
		default:
			if err := d.dec.SkipValue(); err != nil {
				return err
			}
		}
	}
	_, err := d.dec.ReadToken() // }
	return err
}

func (d *jsontextDecoder) readInt64Slice(dst []int64) ([]int64, error) {
	if d.dec.PeekKind() == 'n' {
		_, err := d.dec.ReadToken()
		return nil, err
	}
	if _, err := d.dec.ReadToken(); err != nil { // [
		return nil, err
	}
	dst = dst[:0]
	for d.dec.PeekKind() != ']' {
		t, err := d.dec.ReadToken()
		if err != nil {
			return nil, err
		}
		n, err := t.Int()
		if err != nil {
			return nil, err
		}
		dst = append(dst, n)
	}
	_, err := d.dec.ReadToken() // ]
	return dst, err
}

func (d *jsontextDecoder) readFloat64Slice(dst []float64) ([]float64, error) {
	if d.dec.PeekKind() == 'n' {
		_, err := d.dec.ReadToken()
		return nil, err
	}
	if _, err := d.dec.ReadToken(); err != nil { // [
		return nil, err
	}
	dst = dst[:0]
	for d.dec.PeekKind() != ']' {
		t, err := d.dec.ReadToken()
		if err != nil {
			return nil, err
		}
		f, err := t.Float()
		if err != nil {
			return nil, err
		}
		dst = append(dst, f)
	}
	_, err := d.dec.ReadToken() // ]
	return dst, err
}

func (d *jsontextDecoder) readStringMap() (map[string]string, error) {
	if d.dec.PeekKind() == 'n' {
		_, err := d.dec.ReadToken()
		return nil, err
	}
	if _, err := d.dec.ReadToken(); err != nil { // {
		return nil, err
	}
	m := make(map[string]string)
	for d.dec.PeekKind() != '}' {
		kt, err := d.dec.ReadToken()
		if err != nil {
			return nil, err
		}
		// The key must be materialised BEFORE the value is read. A Token aliases
		// the decoder's buffer and is voided by the next Decoder call — reading
		// the value first and then asking the key for its String panics with
		// "invalid jsontext.Token; it has been voided by a subsequent
		// json.Decoder call". That aliasing is what makes the API zero-copy, and
		// it is the one rule a hand-written codec has to keep.
		k := kt.String()
		vt, err := d.dec.ReadToken()
		if err != nil {
			return nil, err
		}
		m[k] = vt.String()
	}
	_, err := d.dec.ReadToken() // }
	return m, err
}

// ---- RTB: nested structs, a string slice, and a map per impression ----
//
// The shape JSON is most competitive on: mostly strings, so there is little for
// a binary format's numeric codecs to exploit. It is also the shape where
// qdf_compression loses encode CPU to json/v2, which makes the ceiling worth
// knowing precisely.

func (e *jsontextEncoder) writeStringSlice(s []string) error {
	if s == nil {
		return e.enc.WriteToken(jsontext.Null)
	}
	if err := e.enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, v := range s {
		if err := e.enc.WriteToken(jsontext.String(v)); err != nil {
			return err
		}
	}
	return e.enc.WriteToken(jsontext.EndArray)
}

// field writes a name token and is the shape every writer below repeats.
func (e *jsontextEncoder) field(name string) error {
	return e.enc.WriteToken(jsontext.String(name))
}

func (e *jsontextEncoder) writeGeo(g *Geo) error {
	for _, step := range []func() error{
		func() error { return e.enc.WriteToken(jsontext.BeginObject) },
		func() error { return e.field("country") },
		func() error { return e.enc.WriteToken(jsontext.String(g.Country)) },
		func() error { return e.field("lat") },
		func() error { return e.enc.WriteToken(jsontext.Float(g.Lat)) },
		func() error { return e.field("lon") },
		func() error { return e.enc.WriteToken(jsontext.Float(g.Lon)) },
		func() error { return e.field("type") },
		func() error { return e.enc.WriteToken(jsontext.Int(int64(g.Type))) },
		func() error { return e.enc.WriteToken(jsontext.EndObject) },
	} {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (e *jsontextEncoder) writeDevice(d *Device) error {
	if err := e.enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := e.field("ua"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String(d.UA)); err != nil {
		return err
	}
	if err := e.field("ip"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String(d.IP)); err != nil {
		return err
	}
	if err := e.field("os"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Int(int64(d.OS))); err != nil {
		return err
	}
	if err := e.field("type"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Int(int64(d.Type))); err != nil {
		return err
	}
	if err := e.field("geo"); err != nil {
		return err
	}
	if err := e.writeGeo(&d.Geo); err != nil {
		return err
	}
	return e.enc.WriteToken(jsontext.EndObject)
}

func (e *jsontextEncoder) writeImpression(im *Impression) error {
	if err := e.enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := e.field("id"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String(im.ID)); err != nil {
		return err
	}
	if err := e.field("bid_floor"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Float(im.BidFloor)); err != nil {
		return err
	}
	if err := e.field("w"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Int(int64(im.W))); err != nil {
		return err
	}
	if err := e.field("h"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Int(int64(im.H))); err != nil {
		return err
	}
	if err := e.field("deal_ids"); err != nil {
		return err
	}
	if err := e.writeStringSlice(im.DealIDs); err != nil {
		return err
	}
	if err := e.field("ext"); err != nil {
		return err
	}
	if err := e.writeStringMapSorted(im.Ext); err != nil {
		return err
	}
	return e.enc.WriteToken(jsontext.EndObject)
}

func (e *jsontextEncoder) writeBidRequest(r *BidRequest) error {
	if err := e.enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := e.field("id"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.String(r.ID)); err != nil {
		return err
	}
	if err := e.field("at"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Int(int64(r.At))); err != nil {
		return err
	}
	if err := e.field("tmax"); err != nil {
		return err
	}
	if err := e.enc.WriteToken(jsontext.Int(int64(r.Tmax))); err != nil {
		return err
	}
	if err := e.field("imp"); err != nil {
		return err
	}
	if r.Imp == nil {
		if err := e.enc.WriteToken(jsontext.Null); err != nil {
			return err
		}
	} else {
		if err := e.enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for i := range r.Imp {
			if err := e.writeImpression(&r.Imp[i]); err != nil {
				return err
			}
		}
		if err := e.enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
	}
	if err := e.field("dev"); err != nil {
		return err
	}
	if err := e.writeDevice(&r.Dev); err != nil {
		return err
	}
	if err := e.field("cur"); err != nil {
		return err
	}
	if err := e.writeStringSlice(r.Cur); err != nil {
		return err
	}
	return e.enc.WriteToken(jsontext.EndObject)
}

// marshalRTBBatch writes a []BidRequest. The returned slice aliases the
// encoder's buffer, like marshalIoTBatch.
func (e *jsontextEncoder) marshalRTBBatch(v []BidRequest) ([]byte, error) {
	e.reset()
	if v == nil {
		if err := e.enc.WriteToken(jsontext.Null); err != nil {
			return nil, err
		}
		return bytes.TrimSuffix(e.buf.Bytes(), []byte("\n")), nil
	}
	if err := e.enc.WriteToken(jsontext.BeginArray); err != nil {
		return nil, err
	}
	for i := range v {
		if err := e.writeBidRequest(&v[i]); err != nil {
			return nil, err
		}
	}
	if err := e.enc.WriteToken(jsontext.EndArray); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(e.buf.Bytes(), []byte("\n")), nil
}
