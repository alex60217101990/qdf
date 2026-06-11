// internal/mapsgen — code generator for typed map encode/decode fast
// paths. Reflect-driven maps are 3–4× slower than typed paths because
// of per-element reflect.Value materialisation. Each generated pair
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
type kind struct {
	// suffix is the camel-case identifier used in the generated
	// function names and dispatch table entries.
	suffix string
	// goType is the in-memory Go type. Always a single token so it
	// fits into `map[K]V` without parentheses.
	goType string
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
	suffix:     "Bytes",
	goType:     "[]byte",
	writeBlock: func(v string) string { return fmt.Sprintf("e.WriteBytes(%s)", v) },
	readBlock:  mkReadValue("d.ReadBytes()", "[]byte"),
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
		return fmt.Sprintf(`e.WriteArrayHeader(len(%s))
for _, sv := range %s {
e.WriteString(sv)
}`, v, v)
	},
	readBlock: func(v string) string {
		return fmt.Sprintf(`hdrN, err := d.ReadArrayHeader()
if err != nil {
return err
}
if err := d.CheckLength(hdrN, 1); err != nil {
return err
}
%s := make([]string, hdrN)
for i := range hdrN {
sv, err := d.ReadString()
if err != nil {
return err
}
%s[i] = sv
}`, v, v)
	},
}

// kindAny — generic any value. Dispatches through the reflect path
// so heterogeneous values keep round-trip parity with the existing
// reflect encoder.
var kindAny = kind{
	suffix: "Any",
	goType: "any",
	writeBlock: func(v string) string {
		return fmt.Sprintf("if err := encodeReflect(e, %s); err != nil {\nreturn err\n}", v)
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
		fmt.Fprintf(buf, "\tif len(m) > 0 && e.state != nil && e.opts.Has(OptMapShape) && e.opts.Has(OptDense) {\n")
		fmt.Fprintf(buf, "\t\tfor _, k := range mapStringShapeOrder(e, m) {\n")
		fmt.Fprintf(buf, "\t\t\tv := m[k]\n")
		fmt.Fprintf(buf, "\t\t\t%s\n", p.V.writeBlock("v"))
		fmt.Fprintf(buf, "\t\t}\n\t\treturn nil\n\t}\n")
	}
	fmt.Fprintf(buf, "\te.WriteMapHeader(len(m))\n")
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
		fmt.Fprintf(buf, "\t\tm := reuseOrMakeMap[%s, %s](p, len(names))\n", p.K.goType, p.V.goType)
		fmt.Fprintf(buf, "\t\tfor _, k := range names {\n")
		fmt.Fprintf(buf, "\t\t\t%s\n", p.V.readBlock("v"))
		fmt.Fprintf(buf, "\t\t\tm[k] = v\n")
		fmt.Fprintf(buf, "\t\t}\n\t\t*(*%s)(p) = m\n\t\treturn nil\n\t}\n", mapTy)
	}
	fmt.Fprintf(buf, "\tn, err := d.ReadMapHeader()\n\tif err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(buf, "\tif err := d.CheckLength(n, 2); err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(buf, "\tm := reuseOrMakeMap[%s, %s](p, n)\n", p.K.goType, p.V.goType)
	fmt.Fprintf(buf, "\tfor range n {\n")
	fmt.Fprintf(buf, "\t\t%s\n", indent(p.K.readKey("k")))
	fmt.Fprintf(buf, "\t\t%s\n", indent(p.V.readBlock("v")))
	fmt.Fprintf(buf, "\t\tm[k] = v\n")
	fmt.Fprintf(buf, "\t}\n\t*(*%s)(p) = m\n\treturn nil\n}\n\n", mapTy)
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
	fmt.Fprintf(&buf, "import (\n\t\"reflect\"\n\t\"unsafe\"\n)\n\n")
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
