// internal/mapsgen — code generator for typed map encode/decode fast
// paths. Reflect-driven maps are 3–4× slower than typed paths because
// of per-element reflect.Value materialization. Each generated pair
// inlines WriteX / ReadX calls directly, keeping the encoder inliner
// happy while covering the common map shapes Go programs reach for.
//
// Generated output: ../../maps_fast_generated.go. Re-run via
//
//	go generate ./...
//
// from the repository root.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// kind captures everything the generator needs to emit a write / read
// block for a single Go type.
//
// The three func fields (single pointer words) lead and the two strings
// (each a pointer word + a non-pointer length word) trail, keeping the GC
// pointer-scan range tight (48 bytes vs 56 for the source order).
type kind struct {
	// writeBlock returns a Go statement that emits the value bound
	// to `varName`. Multi-line is fine; the generator wraps each
	// statement inside the map loop body.
	writeBlock func(varName string) string
	// readBlock returns a Go statement that declares a fresh local
	// of type goType, named varName, reading it from the decoder.
	// Block must `return err` on failure. Used for map values.
	readBlock func(varName string) string
	// readKeyBlock is the readBlock variant for the map KEY position.
	// String kinds dedupe via d.keyCache.Make so high-cardinality
	// repeats stay zero-alloc; other kinds delegate to readBlock.
	readKeyBlock func(varName string) string
	// suffix is the camel-case identifier used in the generated
	// function names and dispatch table entries.
	suffix string
	// goType is the in-memory Go type. Always a single token so it
	// fits into `map[K]V` without parentheses.
	goType string
}

func mkReadValue(call, goType string) func(string) string {
	return func(v string) string {
		// ReadInt / ReadUint return int64 / uint64. When the requested
		// goType matches the read's natural width we keep the source
		// shape minimal (`v, err := <call>`); otherwise emit a
		// `<v>Wide` temp and narrow it. The cast site stays uniform
		// (`m[k] = v`) across all pairs so the dispatch table doesn't
		// have to special-case the cast position.
		if goType == "int64" || goType == "uint64" || goType == "string" ||
			goType == "bool" || goType == "float32" || goType == "float64" ||
			goType == "[]byte" {
			return fmt.Sprintf("%s, err := %s\nif err != nil {\nreturn err\n}", v, call)
		}
		return fmt.Sprintf("%sWide, err := %s\nif err != nil {\nreturn err\n}\n%s := %s(%sWide)",
			v, call, v, goType, v)
	}
}

func mkWriteScalar(call, castTo string) func(string) string {
	return func(v string) string {
		if castTo == "" {
			return fmt.Sprintf("%s(%s)", call, v)
		}
		return fmt.Sprintf("%s(%s(%s))", call, castTo, v)
	}
}

// kindString — string. Keys dedupe through d.keyCache so repeated
// map keys (the common case for telemetry-style payloads) stay
// allocation-free across many calls. Values use ReadString which
// allocates a Go string per call.
var kindString = kind{
	suffix:     "String",
	goType:     "string",
	writeBlock: func(v string) string { return fmt.Sprintf("e.WriteString(%s)", v) },
	readBlock:  mkReadValue("d.ReadString()", "string"),
	readKeyBlock: func(v string) string {
		return fmt.Sprintf("kb, err := d.readStringBytes()\nif err != nil {\nreturn err\n}\n%s := d.keyCache.Make(kb)", v)
	},
}

var kindBool = kind{
	suffix:     "Bool",
	goType:     "bool",
	writeBlock: func(v string) string { return fmt.Sprintf("e.WriteBool(%s)", v) },
	readBlock:  mkReadValue("d.ReadBool()", "bool"),
}

var kindBytes = kind{
	suffix: "Bytes",
	goType: "[]byte",
	// A nil []byte value travels as tagNil, distinct from an empty []byte —
	// matching the reflect encoder so the nil-vs-empty distinction round-trips.
	writeBlock: func(v string) string {
		return fmt.Sprintf(`if %[1]s == nil {
e.WriteNil()
} else {
e.WriteBytes(%[1]s)
}`, v)
	},
	readBlock: func(v string) string {
		return fmt.Sprintf(`var %[1]s []byte
%[1]sTag, err := d.peekTag()
if err != nil {
return err
}
if %[1]sTag != tagNil {
%[1]s, err = d.ReadBytes()
if err != nil {
return err
}
} else {
d.i++
}`, v)
	},
}

var kindFloat32 = kind{
	suffix:     "Float32",
	goType:     "float32",
	writeBlock: func(v string) string { return fmt.Sprintf("e.WriteFloat32(%s)", v) },
	readBlock:  mkReadValue("d.ReadFloat32()", "float32"),
}

var kindFloat64 = kind{
	suffix:     "Float64",
	goType:     "float64",
	writeBlock: func(v string) string { return fmt.Sprintf("e.WriteFloat64(%s)", v) },
	readBlock:  mkReadValue("d.ReadFloat64()", "float64"),
}

// Integer / unsigned kinds. WriteInt / WriteUint take int64 / uint64;
// in-memory operand width drives the cast on both sides.
func makeIntKind(suffix, goType string) kind {
	return kind{
		suffix:     suffix,
		goType:     goType,
		writeBlock: mkWriteScalar("e.WriteInt", "int64"),
		readBlock:  mkReadValue("d.ReadInt()", goType),
	}
}

func makeUintKind(suffix, goType string) kind {
	return kind{
		suffix:     suffix,
		goType:     goType,
		writeBlock: mkWriteScalar("e.WriteUint", "uint64"),
		readBlock:  mkReadValue("d.ReadUint()", goType),
	}
}

var (
	kindInt8   = makeIntKind("Int8", "int8")
	kindInt16  = makeIntKind("Int16", "int16")
	kindInt32  = makeIntKind("Int32", "int32")
	kindInt    = makeIntKind("Int", "int")
	kindInt64  = makeIntKind("Int64", "int64")
	kindUint8  = makeUintKind("Uint8", "uint8")
	kindUint16 = makeUintKind("Uint16", "uint16")
	kindUint32 = makeUintKind("Uint32", "uint32")
	kindUint   = makeUintKind("Uint", "uint")
	kindUint64 = makeUintKind("Uint64", "uint64")
)

// kindStringSlice — []string value. Encoded as ArrayHeader + each
// WriteString. Used by tag-style payloads (e.g. labels, categories).
var kindStringSlice = kind{
	suffix: "StringSlice",
	goType: "[]string",
	writeBlock: func(v string) string {
		// A nil slice value travels as tagNil (WriteNil), distinct from an empty
		// slice (header 0) — matching the reflect encoder's encodeNilSlice so the
		// fast path and reflect path emit identical wire and the nil-vs-empty
		// distinction survives the round-trip.
		return fmt.Sprintf(`if %[1]s == nil {
e.WriteNil()
} else {
e.WriteArrayHeader(len(%[1]s))
for _, sv := range %[1]s {
e.WriteString(sv)
}
}`, v)
	},
	readBlock: func(v string) string {
		return fmt.Sprintf(`var %[1]s []string
%[1]sTag, err := d.peekTag()
if err != nil {
return err
}
if %[1]sTag != tagNil {
hdrN, err := d.ReadArrayHeader()
if err != nil {
return err
}
if err := d.CheckLength(hdrN, 1); err != nil {
return err
}
%[1]s = make([]string, hdrN)
for i := range hdrN {
sv, err := d.ReadString()
if err != nil {
return err
}
%[1]s[i] = sv
}
} else {
d.i++
}`, v)
	},
}

// kindAny — generic any value. Dispatches through encodeIface (not
// encodeReflect) so the value is encoded in a dynamic-dispatch context:
// encodeIface raises e.ifaceDepth, which suppresses the lossy-vector /
// batched-struct codecs (gated on ifaceDepth==0). A map[K]any value is always
// read back via decodeAny, which cannot decode a tagColVecLossy (0xFD) or
// tagVecBatchStruct (0xFE) block, so those tags must never be emitted here.
// This matches the []any element path (encodeSliceAny → encodeIface).
var kindAny = kind{
	suffix: "Any",
	goType: "any",
	writeBlock: func(v string) string {
		return fmt.Sprintf("if err := encodeIface(e, unsafe.Pointer(&%s)); err != nil {\nreturn err\n}", v)
	},
	readBlock: func(v string) string {
		return fmt.Sprintf("%s, err := decodeAny(d)\nif err != nil {\nreturn err\n}", v)
	},
}

// readKey: returns the readBlock variant when no specialised key
// read exists. Wraps readBlock for non-string kinds.
func (k kind) readKey(varName string) string {
	if k.readKeyBlock != nil {
		return k.readKeyBlock(varName)
	}
	return k.readBlock(varName)
}

type pair struct {
	K, V kind
}

// pairs is the generated set. Keep in sync with the dispatch table
// emitted at the bottom of the file.
//
// Selection: string keys × all common scalar / slice / any values,
// integer keys × the lookup-table variants that actually appear in
// real Go code (enum → label, id → metadata).
var pairs = []pair{
	// string × *
	{kindString, kindString},
	{kindString, kindBool},
	{kindString, kindInt8},
	{kindString, kindInt16},
	{kindString, kindInt32},
	{kindString, kindInt},
	{kindString, kindInt64},
	{kindString, kindUint8},
	{kindString, kindUint16},
	{kindString, kindUint32},
	{kindString, kindUint},
	{kindString, kindUint64},
	{kindString, kindFloat32},
	{kindString, kindFloat64},
	{kindString, kindBytes},
	{kindString, kindStringSlice},
	{kindString, kindAny},

	// int × {string, int, int64, any}
	{kindInt, kindString},
	{kindInt, kindInt},
	{kindInt, kindInt64},
	{kindInt, kindAny},

	// int64 × {string, int64, any}
	{kindInt64, kindString},
	{kindInt64, kindInt64},
	{kindInt64, kindAny},

	// uint64 × {string, uint64, any}
	{kindUint64, kindString},
	{kindUint64, kindUint64},
	{kindUint64, kindAny},
}

func emitPair(buf *bytes.Buffer, p pair) {
	mapTy := fmt.Sprintf("map[%s]%s", p.K.goType, p.V.goType)
	fnName := fmt.Sprintf("Map%s%s", p.K.suffix, p.V.suffix)

	// encode
	fmt.Fprintf(buf, "// ----- %s -----\n\n", mapTy)
	fmt.Fprintf(buf, "func encode%s(e *Encoder, p unsafe.Pointer) error {\n", fnName)
	fmt.Fprintf(buf, "\tm := *(*%s)(p)\n", mapTy)
	fmt.Fprintf(buf, "\tif m == nil {\n\t\te.WriteNil()\n\t\treturn nil\n\t}\n")
	if p.K.suffix == "String" {
		// OptMapShape fast path: recurring key-sets emit a shape header +
		// values in canonical order via the shared generic helper.
		//
		// !OptCanonical, matching the reflect encoder, which documents that the
		// canonical emit takes precedence over this branch "regardless of the
		// other shape bits". Both forms are deterministic on their own, but they
		// are DIFFERENT BYTES, and OptCanonical promises that one logical value
		// has one encoding — bytes callers are invited to hash, sign and
		// content-address. Letting the branch depend on whether a (K,V) pair
		// happens to have a generated fast path breaks exactly that.
		fmt.Fprintf(buf, "\tif len(m) > 0 && e.state != nil && !e.opts.Has(OptCanonical) && e.opts.Has(OptMapShape) && e.opts.Has(OptDense) {\n")
		fmt.Fprintf(buf, "\t\tfor _, k := range mapStringShapeOrder(e, m) {\n")
		fmt.Fprintf(buf, "\t\t\tv := m[k]\n")
		fmt.Fprintf(buf, "\t\t\t%s\n", p.V.writeBlock("v"))
		fmt.Fprintf(buf, "\t\t}\n\t\treturn nil\n\t}\n")
	}
	fmt.Fprintf(buf, "\te.WriteMapHeader(len(m))\n")
	// Canonical fast path: emit keys in sorted order so logically-equal maps
	// serialize byte-identically. The key type is concrete, so slices.Sort is
	// directly monomorphized (no reflect). Reuse the pooled canonKeys* scratch
	// when state is available; fall back to a local slice otherwise. Canonical
	// now takes precedence over the OptMapShape/OptDense shape branch above,
	// which is gated on !OptCanonical for the reason stated there.
	emitCanonicalEncode(buf, p)
	fmt.Fprintf(buf, "\tfor k, v := range m {\n")
	fmt.Fprintf(buf, "\t\t%s\n", indent(p.K.writeBlock("k")))
	fmt.Fprintf(buf, "\t\t%s\n", indent(p.V.writeBlock("v")))
	fmt.Fprintf(buf, "\t}\n\treturn nil\n}\n\n")

	// decode
	fmt.Fprintf(buf, "func decode%s(d *Decoder, p unsafe.Pointer) error {\n", fnName)
	fmt.Fprintf(buf, "\tt, err := d.peekTag()\n\tif err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(buf, "\tif t == tagNil {\n\t\td.i++\n\t\t*(*%s)(p) = nil\n\t\treturn nil\n\t}\n", mapTy)
	if p.K.suffix == "String" {
		// OptMapShape decode: a tagMapShape header carries the ordered keys;
		// read len(names) values in that order.
		fmt.Fprintf(buf, "\tif t == tagMapShape {\n")
		fmt.Fprintf(buf, "\t\tnames, err := decodeMapStringShapeHeader(d)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
		fmt.Fprintf(buf, "\t\tm := reuseOrMakeMap[%s, %s](d, p, len(names))\n", p.K.goType, p.V.goType)
		// UnmarshalKeys projection: consume the root filter ONCE, before the
		// loop, so values and Skip below run unfiltered.
		fmt.Fprintf(buf, "\t\tkf := d.takeKeyFilter()\n")
		fmt.Fprintf(buf, "\t\tfor _, k := range names {\n")
		fmt.Fprintf(buf, "\t\t\tif !kf.want(k) {\n\t\t\t\tif err := d.Skip(); err != nil {\n\t\t\t\t\treturn err\n\t\t\t\t}\n\t\t\t\tcontinue\n\t\t\t}\n")
		fmt.Fprintf(buf, "\t\t\t%s\n", p.V.readBlock("v"))
		fmt.Fprintf(buf, "\t\t\tm[k] = v\n")
		fmt.Fprintf(buf, "\t\t}\n\t\t*(*%s)(p) = m\n\t\treturn nil\n\t}\n", mapTy)
	}
	fmt.Fprintf(buf, "\tn, err := d.ReadMapHeader()\n\tif err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(buf, "\tif err := d.CheckLength(n, 2); err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(buf, "\tm := reuseOrMakeMap[%s, %s](d, p, n)\n", p.K.goType, p.V.goType)
	if p.K.suffix == "String" {
		// UnmarshalKeys projection: consume the root filter ONCE, before the
		// loop. Only string-keyed maps participate — the filter is name-based.
		fmt.Fprintf(buf, "\tkf := d.takeKeyFilter()\n")
	}
	fmt.Fprintf(buf, "\tfor range n {\n")
	fmt.Fprintf(buf, "\t\t%s\n", indent(p.K.readKey("k")))
	if p.K.suffix == "String" {
		fmt.Fprintf(buf, "\t\tif !kf.want(k) {\n\t\t\tif err := d.Skip(); err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tcontinue\n\t\t}\n")
	}
	fmt.Fprintf(buf, "\t\t%s\n", indent(p.V.readBlock("v")))
	fmt.Fprintf(buf, "\t\tm[k] = v\n")
	fmt.Fprintf(buf, "\t}\n\t*(*%s)(p) = m\n\treturn nil\n}\n\n", mapTy)
}

// emitCanonicalEncode writes the OptCanonical sorted-key branch for a map
// encoder. The key type is concrete, so the canonical key slice is built with a
// monomorphized slices.Sort. The pooled canonKeys* scratch is reused when the
// encoder carries an encState; otherwise a local slice is allocated.
//
// scratchField / scratchElem map the concrete key type onto the encState scratch
// (string→canonKeysStr, signed int kinds→canonKeysI64, unsigned→canonKeysU64).
// For int (machine width) the scratch element is int64, so the gather casts and
// the map lookup casts back — the sort order is identical (int values fit int64).
func emitCanonicalEncode(buf *bytes.Buffer, p pair) {
	var scratchField, scratchElem string
	switch p.K.goType {
	case "string":
		scratchField, scratchElem = "canonKeysStr", "string"
	case "int", "int64", "int8", "int16", "int32":
		scratchField, scratchElem = "canonKeysI64", "int64"
	case "uint", "uint64", "uint8", "uint16", "uint32", "uintptr":
		scratchField, scratchElem = "canonKeysU64", "uint64"
	default:
		// No canonical fast path for this key kind; the range loop below stays
		// the only emit (such keys are not in the generated pair set today).
		return
	}
	needCast := p.K.goType != scratchElem

	// Re-entrancy: the pooled canonKeys* scratch is shared with the reflect map
	// encoder and the other generated encoders. A map nested inside another map's
	// value (or a reflect map whose value reaches this generated encoder) would
	// clobber the outer map's sorted-key slice while it is still being iterated.
	// canonKeysBusy guards it: when already busy, this encoder allocates a fresh
	// local slice; otherwise it borrows the pool and sets the guard for the
	// duration of the emit loop (the value writes may recurse into another map).
	fmt.Fprintf(buf, "\tif e.opts.Has(OptCanonical) {\n")
	fmt.Fprintf(buf, "\t\tvar keys []%s\n", scratchElem)
	fmt.Fprintf(buf, "\t\tcanonPooled := false\n")
	fmt.Fprintf(buf, "\t\tif e.state != nil && !e.state.canonKeysBusy {\n")
	fmt.Fprintf(buf, "\t\t\tkeys = e.state.%s[:0]\n", scratchField)
	fmt.Fprintf(buf, "\t\t\tcanonPooled = true\n")
	fmt.Fprintf(buf, "\t\t} else {\n")
	fmt.Fprintf(buf, "\t\t\tkeys = make([]%s, 0, len(m))\n", scratchElem)
	fmt.Fprintf(buf, "\t\t}\n")
	if needCast {
		fmt.Fprintf(buf, "\t\tfor k := range m {\n\t\t\tkeys = append(keys, %s(k))\n\t\t}\n", scratchElem)
	} else {
		fmt.Fprintf(buf, "\t\tfor k := range m {\n\t\t\tkeys = append(keys, k)\n\t\t}\n")
	}
	fmt.Fprintf(buf, "\t\tslices.Sort(keys)\n")
	// Release the re-entrancy latch via defer so it clears on EVERY exit —
	// including a mid-loop `return err` from a value write (the *Any pairs).
	// An inline post-loop release leaks busy=true on the error path, pinning a
	// pooled encoder to the fresh-allocation fallback forever. Mirrors the
	// reflect path's `defer e.canonKeysRelease(pooled)`.
	fmt.Fprintf(buf, "\t\tif canonPooled {\n\t\t\te.state.%s = keys\n\t\t\te.state.canonKeysBusy = true\n\t\t\tdefer func() { e.state.canonKeysBusy = false }()\n\t\t}\n", scratchField)
	fmt.Fprintf(buf, "\t\tfor _, sk := range keys {\n")
	// sk is already a per-iteration copy of the range value. Only the narrow-key
	// variants need a real conversion (k := T(sk)); the string/wide-key variants
	// use sk directly — a `k := sk` rename would be dead generated code.
	keyVar := "sk"
	if needCast {
		fmt.Fprintf(buf, "\t\t\tk := %s(sk)\n", p.K.goType)
		keyVar = "k"
	}
	fmt.Fprintf(buf, "\t\t\tv := m[%s]\n", keyVar)
	fmt.Fprintf(buf, "\t\t\t%s\n", indent(indent(p.K.writeBlock(keyVar))))
	fmt.Fprintf(buf, "\t\t\t%s\n", indent(indent(p.V.writeBlock("v"))))
	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\treturn nil\n\t}\n")
}

func emitDispatch(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "// installMapFastPath returns (encode, decode, true) when t matches\n")
	fmt.Fprintf(buf, "// one of the generated typed-map shapes; otherwise (_, _, false). The\n")
	fmt.Fprintf(buf, "// table is generated; edit internal/mapsgen/main.go to add or remove\n")
	fmt.Fprintf(buf, "// pairs.\n")
	fmt.Fprintf(buf, "func installMapFastPath(t reflect.Type) (\n")
	fmt.Fprintf(buf, "\tenc func(*Encoder, unsafe.Pointer) error,\n")
	fmt.Fprintf(buf, "\tdec func(*Decoder, unsafe.Pointer) error,\n")
	fmt.Fprintf(buf, "\tok bool,\n) {\n")
	fmt.Fprintf(buf, "\tswitch t {\n")
	for _, p := range pairs {
		fnName := fmt.Sprintf("Map%s%s", p.K.suffix, p.V.suffix)
		mapTy := fmt.Sprintf("map[%s]%s", p.K.goType, p.V.goType)
		fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", mapTy)
		fmt.Fprintf(buf, "\t\treturn encode%s, decode%s, true\n", fnName, fnName)
	}
	fmt.Fprintf(buf, "\t}\n\treturn nil, nil, false\n}\n")
}

// indent reindents a multi-line snippet so the second and later
// lines line up with the two-tab loop-body column. The first-line
// prefix is supplied by the caller via fmt.Fprintf alignment.
func indent(s string) string {
	const prefix = "\t\t"
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func main() {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by internal/mapsgen — DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "// Re-run via `go generate ./...` from the repository root.\n\n")
	fmt.Fprintf(&buf, "package qdf\n\n")
	fmt.Fprintf(&buf, "import (\n\t\"reflect\"\n\t\"slices\"\n\t\"unsafe\"\n)\n\n")
	for _, p := range pairs {
		emitPair(&buf, p)
	}
	emitDispatch(&buf)

	src, err := format.Source(buf.Bytes())
	if err != nil {
		// Emit unformatted for easier debugging then fail.
		_, _ = os.Stderr.Write(buf.Bytes())
		log.Fatalf("\nformat: %v", err)
	}

	// Locate the module root by walking up to the go.mod, so the generator
	// works whether invoked from internal/mapsgen or via `go generate ./...`
	// (which runs it with the repo root as the working directory).
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			log.Fatalf("could not find go.mod above %s", wd)
		}
		root = parent
	}
	out := filepath.Join(root, "maps_fast_generated.go")
	if err := os.WriteFile(out, src, 0o600); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d bytes, %d pairs)\n", out, len(src), len(pairs))
}
