// Package gen implements the qdfgen code generator. The cmd/qdfgen
// binary wraps this package as a CLI; tests call Generate directly.
package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Options configures Generate.
type Options struct {
	// Types names the struct types to generate methods for. Each name
	// must resolve in the loaded package(s).
	Types []string
	// OutFile overrides the default output path. When the value has no
	// path separator it is joined with the package directory (or OutDir
	// if set).
	OutFile string
	// OutDir overrides the package directory when OutFile has no
	// directory component.
	OutDir string
	// Verbose logs progress to LogTo.
	Verbose bool
	// LogTo defaults to os.Stderr when nil.
	LogTo io.Writer
}

// Generate loads pkgPatterns, walks each requested type, and writes one
// file per package containing MarshalQDF / UnmarshalQDF methods.
func Generate(pkgPatterns []string, opts Options) error {
	if opts.LogTo == nil {
		opts.LogTo = os.Stderr
	}
	if len(opts.Types) == 0 {
		return fmt.Errorf("no types requested")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedTypesSizes |
			packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, pkgPatterns...)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return fmt.Errorf("package load errors")
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages matched %v", pkgPatterns)
	}

	want := make(map[string]bool, len(opts.Types))
	for _, t := range opts.Types {
		want[t] = true
	}

	wroteAny := false
	for _, pkg := range pkgs {
		var found []*types.Named
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if !want[name] {
				continue
			}
			obj := scope.Lookup(name)
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := tn.Type().(*types.Named)
			if !ok {
				return fmt.Errorf("%s: not a named type", name)
			}
			if _, ok := named.Underlying().(*types.Struct); !ok {
				return fmt.Errorf("%s: only struct types are supported", name)
			}
			found = append(found, named)
		}
		if len(found) == 0 {
			continue
		}
		sort.Slice(found, func(i, j int) bool {
			return found[i].Obj().Name() < found[j].Obj().Name()
		})

		g := newGen(pkg)
		for _, n := range found {
			if err := g.emitType(n); err != nil {
				return err
			}
		}
		src, err := g.bytes()
		if err != nil {
			return fmt.Errorf("%s: %w", pkg.PkgPath, err)
		}

		out := resolveOut(pkg, opts)
		if opts.Verbose {
			fmt.Fprintf(opts.LogTo, "qdfgen: writing %s (%d bytes)\n", out, len(src))
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, src, 0o644); err != nil {
			return err
		}
		wroteAny = true
	}

	// Validate that every requested type was found in at least one package.
	for _, t := range opts.Types {
		seen := false
		for _, pkg := range pkgs {
			if obj := pkg.Types.Scope().Lookup(t); obj != nil {
				seen = true
				break
			}
		}
		if !seen {
			return fmt.Errorf("type %q not found in %v", t, pkgPatterns)
		}
	}
	if !wroteAny {
		return fmt.Errorf("no output written")
	}
	return nil
}

func resolveOut(pkg *packages.Package, opts Options) string {
	if opts.OutFile != "" {
		if filepath.IsAbs(opts.OutFile) || strings.ContainsRune(opts.OutFile, filepath.Separator) {
			return opts.OutFile
		}
		dir := pkgDir(pkg)
		if opts.OutDir != "" {
			dir = opts.OutDir
		}
		return filepath.Join(dir, opts.OutFile)
	}
	dir := pkgDir(pkg)
	if opts.OutDir != "" {
		dir = opts.OutDir
	}
	return filepath.Join(dir, pkg.Name+"_qdf.go")
}

func pkgDir(pkg *packages.Package) string {
	if len(pkg.CompiledGoFiles) > 0 {
		return filepath.Dir(pkg.CompiledGoFiles[0])
	}
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	return "."
}

// ---------------------------------------------------------------------------

// gen accumulates one output file for one package.
type gen struct {
	pkg *packages.Package

	imports map[string]string // import path -> local name (empty = default)
	header  bytes.Buffer      // var-block of pre-encoded field-name headers
	body    bytes.Buffer      // function bodies

	headerNames map[string]string // field-name -> generated var ident

	colVars    bytes.Buffer    // var-block of columnar shape (names/kinds) decls
	colVarSeen map[string]bool // dedupe columnar shape vars by ident

	// uniqCounter ensures generated identifiers stay unique.
	uniqCounter int

	// emitted tracks struct types already emitted in this file.
	emitted map[string]bool

	// path tracks the chain of types currently being expanded; used to
	// detect cycles through value (non-pointer) fields.
	path []string
}

const maxNestingDepth = 64

func newGen(pkg *packages.Package) *gen {
	return &gen{
		pkg:         pkg,
		imports:     map[string]string{"github.com/alex60217101990/qdf": ""},
		headerNames: map[string]string{},
		emitted:     map[string]bool{},
		colVarSeen:  map[string]bool{},
	}
}

// bytes returns the formatted output file contents.
func (g *gen) bytes() ([]byte, error) {
	var out bytes.Buffer
	fmt.Fprintln(&out, "// Code generated by qdfgen; DO NOT EDIT.")
	fmt.Fprintln(&out)
	fmt.Fprintf(&out, "package %s\n\n", g.pkg.Name)

	// Emit imports.
	paths := make([]string, 0, len(g.imports))
	for p := range g.imports {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		fmt.Fprintln(&out, "import (")
		for _, p := range paths {
			if name := g.imports[p]; name != "" {
				fmt.Fprintf(&out, "\t%s %q\n", name, p)
			} else {
				fmt.Fprintf(&out, "\t%q\n", p)
			}
		}
		fmt.Fprintln(&out, ")")
		fmt.Fprintln(&out)
	}

	// Emit pre-encoded field-name headers.
	if g.header.Len() > 0 {
		out.WriteString("// Pre-encoded field-name headers (fixstr / strN). Lets the hot path\n")
		out.WriteString("// emit a name with a single append, no per-call sizing.\n")
		out.WriteString("var (\n")
		out.Write(g.header.Bytes())
		out.WriteString(")\n\n")
	}

	// Emit columnar shape vars (column names + kind bytes) for scalar []struct
	// fields encoded via the columnar transpose path.
	if g.colVars.Len() > 0 {
		out.WriteString("// Columnar shape descriptors: per element-type column names and kind\n")
		out.WriteString("// bytes for the monomorphized tagColStruct transpose path.\n")
		out.Write(g.colVars.Bytes())
		out.WriteString("\n")
	}

	// Function bodies.
	out.Write(g.body.Bytes())

	src, err := format.Source(out.Bytes())
	if err != nil {
		// Return unformatted with the error so callers can see what was
		// produced.
		return out.Bytes(), fmt.Errorf("format: %w\n--- begin source ---\n%s\n--- end source ---", err, out.String())
	}
	return src, nil
}

// importAlias adds an import for the given path (if not already present) and
// returns the local name used to qualify identifiers from it.
func (g *gen) importAlias(path string) string {
	if path == g.pkg.PkgPath {
		return ""
	}
	if _, ok := g.imports[path]; !ok {
		g.imports[path] = ""
	}
	return filepath.Base(path)
}

// fieldNameVar returns (or creates) a unique var holding the pre-encoded
// fixstr / strN header bytes for the given field name.
func (g *gen) fieldNameVar(name string) string {
	if v, ok := g.headerNames[name]; ok {
		return v
	}
	g.uniqCounter++
	ident := fmt.Sprintf("qdfFieldHdr_%s_%d", sanitizeIdent(name), g.uniqCounter)
	g.headerNames[name] = ident
	g.header.WriteString("\t")
	g.header.WriteString(ident)
	g.header.WriteString(" = ")
	g.header.WriteString(byteLiteral(precomputeFixstrHeader(name)))
	g.header.WriteString("\n")
	return ident
}

func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "anon"
	}
	return b.String()
}

func byteLiteral(b []byte) string {
	var sb strings.Builder
	sb.WriteString("[]byte{")
	for i, x := range b {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("0x")
		sb.WriteString(strconv.FormatUint(uint64(x), 16))
	}
	sb.WriteByte('}')
	return sb.String()
}

// precomputeFixstrHeader mirrors qdf.precomputeFixstrHeader without needing
// access to the unexported helper. Returns the tag + length bytes + the name
// bytes.
func precomputeFixstrHeader(name string) []byte {
	const (
		tagFixstr     = 0x80
		tagFixstrMask = 0x1F
		tagStr8       = 0xCD
		tagStr16      = 0xCE
		tagStr32      = 0xCF
	)
	n := len(name)
	switch {
	case n <= tagFixstrMask:
		out := make([]byte, 1+n)
		out[0] = tagFixstr | byte(n)
		copy(out[1:], name)
		return out
	case n <= 0xFF:
		out := make([]byte, 2+n)
		out[0] = tagStr8
		out[1] = byte(n)
		copy(out[2:], name)
		return out
	case n <= 0xFFFF:
		out := make([]byte, 3+n)
		out[0] = tagStr16
		out[1] = byte(n)
		out[2] = byte(n >> 8)
		copy(out[3:], name)
		return out
	default:
		out := make([]byte, 5+n)
		out[0] = tagStr32
		out[1] = byte(n)
		out[2] = byte(n >> 8)
		out[3] = byte(n >> 16)
		out[4] = byte(n >> 24)
		copy(out[5:], name)
		return out
	}
}

// ---------------------------------------------------------------------------
// Type emission

// fieldInfo is the per-emitted-field summary.
type fieldInfo struct {
	GoName  string // exported Go field name (for diagnostics)
	Access  string // Go access path from the receiver, e.g. "X" or "Base.X"
	WireKey string // string used as the map key on the wire
	Field   *types.Var
	Tag     string // raw struct tag, for diagnostics
}

func collectFields(s *types.Struct) []fieldInfo {
	return appendFields(make([]fieldInfo, 0, s.NumFields()), s, "")
}

// appendFields walks s in declaration order, flattening anonymous embedded
// value-struct fields into the parent's wire layout exactly as the reflect
// path (appendStructFields in reflect_desc.go) does: the inner fields appear
// at the parent level, a "-" tag on the embedded field opts the whole nested
// layout out, and a pointer-typed embedded field falls through to the regular
// path (encoded as a pointer-to-struct). prefix is the Go access path to s
// from the receiver ("" at top level, "Base." for fields promoted out of an
// embedded Base).
func appendFields(out []fieldInfo, s *types.Struct, prefix string) []fieldInfo {
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		tag := s.Tag(i)
		if f.Embedded() {
			if st, ok := f.Type().Underlying().(*types.Struct); ok {
				// Honor a "-" tag on the embedded field itself.
				if _, skip := wireKey(f.Name(), tag); skip {
					continue
				}
				out = appendFields(out, st, prefix+f.Name()+".")
				continue
			}
			// Embedded pointer/interface/etc.: regular field path below.
		}
		if !f.Exported() {
			continue
		}
		key, skip := wireKey(f.Name(), tag)
		if skip {
			continue
		}
		out = append(out, fieldInfo{
			GoName:  f.Name(),
			Access:  prefix + f.Name(),
			WireKey: key,
			Field:   f,
			Tag:     tag,
		})
	}
	return out
}

// wireKey mirrors the precedence: qdf tag > json tag > field name. Returns
// (key, skip).
func wireKey(name, tag string) (string, bool) {
	if t, ok := lookupTag(tag, "qdf"); ok {
		if t == "-" {
			return "", true
		}
		parts := strings.Split(t, ",")
		if parts[0] != "" {
			return parts[0], false
		}
		return name, false
	}
	if t, ok := lookupTag(tag, "json"); ok {
		parts := strings.Split(t, ",")
		if parts[0] == "-" {
			return "", true
		}
		if parts[0] != "" {
			return parts[0], false
		}
	}
	return name, false
}

// lookupTag is a tiny replacement for reflect.StructTag.Lookup operating on
// the raw tag string returned by go/types.
func lookupTag(tag, key string) (string, bool) {
	for tag != "" {
		// Skip leading spaces.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}
		// Scan key.
		i = 0
		for i < len(tag) && tag[i] != ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		k := tag[:i]
		tag = tag[i+1:]
		// Scan quoted value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qv := tag[:i+1]
		tag = tag[i+1:]
		if k == key {
			v, err := strconv.Unquote(qv)
			if err != nil {
				return "", false
			}
			return v, true
		}
	}
	return "", false
}

// pushPath / popPath track recursion through value types so we can detect
// cycles like struct A { B B } / struct B { A A }.
func (g *gen) pushPath(name string) error {
	if len(g.path) >= maxNestingDepth {
		return fmt.Errorf("nesting depth %d exceeds limit (cycle through value types?)", len(g.path))
	}
	if slices.Contains(g.path, name) {
		return fmt.Errorf("cycle detected: %s -> ... -> %s through value-typed fields; use a pointer", strings.Join(g.path, " -> "), name)
	}
	g.path = append(g.path, name)
	return nil
}

func (g *gen) popPath() {
	if n := len(g.path); n > 0 {
		g.path = g.path[:n-1]
	}
}

// emitType writes MarshalQDF and UnmarshalQDF for the named struct type.
func (g *gen) emitType(named *types.Named) error {
	typeName := named.Obj().Name()
	if g.emitted[typeName] {
		return nil
	}
	g.emitted[typeName] = true

	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("%s: not a struct", typeName)
	}
	fields := collectFields(st)

	if err := g.pushPath(typeName); err != nil {
		return err
	}
	defer g.popPath()

	if err := g.emitMarshal(typeName, fields); err != nil {
		return err
	}
	if err := g.emitUnmarshal(typeName, fields); err != nil {
		return err
	}
	return nil
}

func (g *gen) emitMarshal(typeName string, fields []fieldInfo) error {
	w := &g.body

	// Public buffer-based entry point — a thin wrapper that sets up ONE encoder
	// and delegates the field writing to EncodeQDF. Nested values share this
	// encoder (via qdf.EncodeNested) instead of each allocating their own on the
	// parent's bytes.
	fmt.Fprintf(w, "// MarshalQDF appends a qdf-encoded representation of v to dst and returns\n")
	fmt.Fprintf(w, "// the extended slice.\n")
	fmt.Fprintf(w, "func (v *%s) MarshalQDF(dst []byte) ([]byte, error) {\n", typeName)
	// Decide whether dst already contains a QDF stream (nested call) or is
	// a fresh buffer (top-level call). We use the magic bytes to detect.
	fmt.Fprintf(w, "\thadHeader := len(dst) >= 5 && dst[0] == qdf.Magic0 && dst[1] == qdf.Magic1 && dst[2] == qdf.Magic2\n")
	fmt.Fprintf(w, "\te := qdf.NewEncoderOnBuf(dst, qdf.Fast)\n")
	fmt.Fprintf(w, "\tif hadHeader {\n\t\te.MarkHeaderWritten()\n\t} else {\n\t\te.EnsureHeader()\n\t}\n")
	fmt.Fprintf(w, "\tif err := v.EncodeQDF(e); err != nil {\n\t\treturn nil, err\n\t}\n")
	fmt.Fprintf(w, "\treturn e.Bytes(), nil\n")
	fmt.Fprintf(w, "}\n\n")

	// Shape-interning state for this type: a stable per-type token the encoder
	// keys the field-name shape on, and the field-name headers in field order.
	// EncodeQDF declares the shape once per encoder (StructShape) so a slice of
	// this struct threaded through one encoder writes the names once.
	shapeTok := fmt.Sprintf("qdfShapeTok_%s", typeName)
	hdrsVar := fmt.Sprintf("qdfFieldHdrs_%s", typeName)
	hdrNames := make([]string, len(fields))
	for i, f := range fields {
		hdrNames[i] = g.fieldNameVar(f.WireKey)
	}
	fmt.Fprintf(w, "var %s byte\n", shapeTok)
	fmt.Fprintf(w, "var %s = [][]byte{%s}\n\n", hdrsVar, strings.Join(hdrNames, ", "))

	// Encoder-based body — writes v's fields into a shared encoder. A parent
	// threads one encoder through nested values via qdf.EncodeNested, avoiding
	// the per-nested-value *Encoder allocation the buffer-based path costs.
	fmt.Fprintf(w, "// EncodeQDF writes v's fields into e. It lets a parent thread one encoder\n")
	fmt.Fprintf(w, "// through nested values instead of allocating an encoder per value.\n")
	fmt.Fprintf(w, "func (v *%s) EncodeQDF(e *qdf.Encoder) error {\n", typeName)
	fmt.Fprintf(w, "\te.StructShape(&%s, %s)\n", shapeTok, hdrsVar)
	for _, f := range fields {
		expr := "v." + f.Access
		if err := g.emitEncodeValue(w, expr, f.Field.Type(), "\t"); err != nil {
			return fmt.Errorf("%s.%s: %w", typeName, f.GoName, err)
		}
	}
	fmt.Fprintf(w, "\treturn nil\n")
	fmt.Fprintf(w, "}\n\n")
	return nil
}

func (g *gen) emitUnmarshal(typeName string, fields []fieldInfo) error {
	w := &g.body

	// DecodeQDF holds the field-reading body and operates on a SHARED decoder,
	// advancing it. It lets a parent thread one decoder through nested values
	// (via qdf.DecodeNested) instead of opening a fresh decoder per nested value.
	// noCopy / arena live on the shared decoder, so nested decodes inherit them.
	fmt.Fprintf(w, "// DecodeQDF reads v's fields from the shared decoder d, advancing it. It lets\n")
	fmt.Fprintf(w, "// a parent thread one decoder through nested values (see qdf.DecodeNested).\n")
	fmt.Fprintf(w, "func (v *%s) DecodeQDF(d *qdf.Decoder) error {\n", typeName)
	fmt.Fprintf(w, "\tnames, plainN, shaped, err := d.ReadStructHeader()\n")
	fmt.Fprintf(w, "\tif err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(w, "\tif shaped {\n")
	fmt.Fprintf(w, "\t\tfor _, name := range names {\n")
	fmt.Fprintf(w, "\t\t\tif err := v.decodeQDFField(d, name); err != nil {\n\t\t\t\treturn err\n\t\t\t}\n")
	fmt.Fprintf(w, "\t\t}\n")
	fmt.Fprintf(w, "\t\treturn nil\n")
	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "\tfor range plainN {\n")
	fmt.Fprintf(w, "\t\tkb, err := d.ReadStringBytes()\n")
	fmt.Fprintf(w, "\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
	fmt.Fprintf(w, "\t\tif err := v.decodeQDFField(d, string(kb)); err != nil {\n\t\t\treturn err\n\t\t}\n")
	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "\treturn nil\n")
	fmt.Fprintf(w, "}\n\n")

	// decodeQDFField decodes one field by wire name from the shared decoder. The
	// switch is shared by both DecodeQDF loops (shape-interned and plain-map), so
	// the field-reading code is emitted once.
	fmt.Fprintf(w, "func (v *%s) decodeQDFField(d *qdf.Decoder, name string) error {\n", typeName)
	fmt.Fprintf(w, "\tswitch name {\n")
	for _, f := range fields {
		fmt.Fprintf(w, "\tcase %q:\n", f.WireKey)
		if err := g.emitDecodeValue(w, "v."+f.Access, f.Field.Type(), "\t\t"); err != nil {
			return fmt.Errorf("%s.%s: %w", typeName, f.GoName, err)
		}
	}
	fmt.Fprintf(w, "\tdefault:\n")
	fmt.Fprintf(w, "\t\tif err := d.Skip(); err != nil {\n\t\t\treturn err\n\t\t}\n")
	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "\treturn nil\n")
	fmt.Fprintf(w, "}\n\n")

	// Wrappers: open one decoder, set flags, consume the stream header, delegate
	// the field reading to DecodeQDF, and report the bytes consumed.
	fmt.Fprintf(w, "// UnmarshalQDF decodes a qdf payload into v and returns the number of\n")
	fmt.Fprintf(w, "// bytes consumed.\n")
	fmt.Fprintf(w, "func (v *%s) UnmarshalQDF(src []byte) (int, error) {\n", typeName)
	fmt.Fprintf(w, "\treturn v.UnmarshalQDFArena(src, false, nil)\n")
	fmt.Fprintf(w, "}\n\n")
	fmt.Fprintf(w, "// UnmarshalQDFOpts decodes like UnmarshalQDF; when noCopy is true the decoded\n")
	fmt.Fprintf(w, "// string and []byte fields alias src instead of copying. The aliases are valid\n")
	fmt.Fprintf(w, "// only while src stays alive and is not modified (see qdf.WithNoCopy).\n")
	fmt.Fprintf(w, "func (v *%s) UnmarshalQDFOpts(src []byte, noCopy bool) (int, error) {\n", typeName)
	fmt.Fprintf(w, "\treturn v.UnmarshalQDFArena(src, noCopy, nil)\n")
	fmt.Fprintf(w, "}\n\n")
	fmt.Fprintf(w, "// UnmarshalQDFArena decodes like UnmarshalQDFOpts; when a is non-nil the copied\n")
	fmt.Fprintf(w, "// string fields are packed into the arena instead of one allocation each (see\n")
	fmt.Fprintf(w, "// qdf.WithArena). The decoded strings then alias the arena's memory.\n")
	fmt.Fprintf(w, "func (v *%s) UnmarshalQDFArena(src []byte, noCopy bool, a *qdf.Arena) (int, error) {\n", typeName)
	fmt.Fprintf(w, "\td := qdf.NewDecoderOnBuf(src)\n")
	fmt.Fprintf(w, "\tif noCopy {\n\t\td.SetNoCopy(true)\n\t}\n")
	fmt.Fprintf(w, "\tif a != nil {\n\t\td.SetArena(a)\n\t}\n")
	// If src starts with QDF magic this is a top-level decode (the first read
	// consumes the header); otherwise a nested call from a parent decoder that
	// already consumed the header.
	fmt.Fprintf(w, "\thasHeader := len(src) >= 5 && src[0] == qdf.Magic0 && src[1] == qdf.Magic1 && src[2] == qdf.Magic2\n")
	fmt.Fprintf(w, "\tif !hasHeader {\n")
	fmt.Fprintf(w, "\t\td.MarkHeaderRead()\n")
	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "\tif err := v.DecodeQDF(d); err != nil {\n\t\treturn 0, err\n\t}\n")
	fmt.Fprintf(w, "\treturn d.Pos(), nil\n")
	fmt.Fprintf(w, "}\n\n")
	return nil
}

// ---------------------------------------------------------------------------
// Value emission

// emitEncodeValue writes statements that encode the value expressed by
// `expr` (which already evaluates to a value of type t) into the encoder e.
func (g *gen) emitEncodeValue(w io.Writer, expr string, t types.Type, indent string) error {
	t = types.Unalias(t) // `any` is an alias for interface{}; resolve it
	if isTimeTime(t) {
		fmt.Fprintf(w, "%s{ _t := (%s).UTC(); e.WriteTimestamp(_t.Unix(), uint32(_t.Nanosecond())) }\n", indent, expr)
		return nil
	}

	switch tt := t.(type) {
	case *types.Basic:
		return g.emitEncodeBasic(w, expr, tt, indent)
	case *types.Named:
		return g.emitEncodeNamed(w, expr, tt, indent)
	case *types.Pointer:
		return g.emitEncodePointer(w, expr, tt, indent)
	case *types.Slice:
		return g.emitEncodeSlice(w, expr, tt, indent)
	case *types.Array:
		return g.emitEncodeArray(w, expr, tt, indent)
	case *types.Map:
		return g.emitEncodeMap(w, expr, tt, indent)
	case *types.Interface:
		// An empty interface (any) carries fully dynamic data; defer to the
		// runtime reflect encoder for that one field. A method interface cannot
		// be reconstructed on decode, so it is still rejected.
		if tt.NumMethods() == 0 {
			fmt.Fprintf(w, "%sif err := e.EncodeValue(%s); err != nil {\n%s\treturn err\n%s}\n", indent, expr, indent, indent)
			return nil
		}
		return fmt.Errorf("non-empty interface fields are not supported by qdfgen")
	default:
		return fmt.Errorf("unsupported type %s (%T)", t, t)
	}
}

func (g *gen) emitEncodeBasic(w io.Writer, expr string, b *types.Basic, indent string) error {
	switch k := b.Kind(); k {
	case types.Bool:
		fmt.Fprintf(w, "%se.WriteBool(bool(%s))\n", indent, expr)
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
		fmt.Fprintf(w, "%se.WriteInt(int64(%s))\n", indent, expr)
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		fmt.Fprintf(w, "%se.WriteUint(uint64(%s))\n", indent, expr)
	case types.Float32:
		fmt.Fprintf(w, "%se.WriteFloat32(float32(%s))\n", indent, expr)
	case types.Float64:
		fmt.Fprintf(w, "%se.WriteFloat64(float64(%s))\n", indent, expr)
	case types.String:
		fmt.Fprintf(w, "%se.WriteString(string(%s))\n", indent, expr)
	default:
		return fmt.Errorf("unsupported basic kind %v", k)
	}
	return nil
}

func (g *gen) emitEncodeNamed(w io.Writer, expr string, n *types.Named, indent string) error {
	switch ut := n.Underlying().(type) {
	case *types.Basic:
		return g.emitEncodeBasic(w, expr, ut, indent)
	case *types.Struct:
		// Nested struct: thread the parent's encoder via qdf.EncodeNested (no new
		// encoder allocated). expr is always addressable (a struct field,
		// slice/array element, map range var, or pointer deref), so take its
		// address directly — EncodeQDF only reads the receiver.
		fmt.Fprintf(w, "%sif err := qdf.EncodeNested(e, &%s); err != nil {\n%s\treturn err\n%s}\n", indent, expr, indent, indent)
		return nil
	case *types.Slice, *types.Array, *types.Map, *types.Pointer:
		return g.emitEncodeValue(w, expr, ut, indent)
	default:
		return fmt.Errorf("unsupported named underlying %T", ut)
	}
}

func (g *gen) emitEncodePointer(w io.Writer, expr string, p *types.Pointer, indent string) error {
	fmt.Fprintf(w, "%sif %s == nil {\n", indent, expr)
	fmt.Fprintf(w, "%s\te.WriteNil()\n", indent)
	fmt.Fprintf(w, "%s} else {\n", indent)

	elem := p.Elem()
	if isTimeTime(elem) {
		fmt.Fprintf(w, "%s\t{ _t := (*%s).UTC(); e.WriteTimestamp(_t.Unix(), uint32(_t.Nanosecond())) }\n", indent, expr)
	} else if named, ok := elem.(*types.Named); ok {
		if _, isStruct := named.Underlying().(*types.Struct); isStruct {
			fmt.Fprintf(w, "%s\tif err := qdf.EncodeNested(e, %s); err != nil {\n%s\t\treturn err\n%s\t}\n", indent, expr, indent, indent)
		} else {
			if err := g.emitEncodeValue(w, "(*"+expr+")", named.Underlying(), indent+"\t"); err != nil {
				return err
			}
		}
	} else {
		if err := g.emitEncodeValue(w, "(*"+expr+")", elem, indent+"\t"); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// colColumn / colElemPlan describe an all-scalar slice element eligible for
// monomorphized columnar transpose. Built without reflection from go/types at
// generate time; the emitted code itself never uses reflection.
type colColumn struct {
	WireName string // column name on the wire (field WireKey: qdf>json>name)
	Access   string // Go field selector from the element, e.g. "TS" or "Base.X"
	KindByte byte   // colKind wire byte: 0/1/2/3/6
	GoType   string // rendered field type, e.g. "int32", for scatter narrowing
	ColAPI   string // "Int"|"Uint"|"Float64"|"Float32"|"Bool"
}

type colElemPlan struct {
	ElemType string // rendered element type, e.g. "GenMetric"
	Columns  []colColumn
}

// columnarElemPlan returns a plan when elem is a named struct whose every
// (flattened) field is a scalar basic type — the v1 columnar-eligibility set.
// Otherwise ok=false and the caller emits the row-major path. Any String,
// time.Time, nested struct, slice, map, pointer, array, or interface field
// disqualifies the element (those need string/hybrid columns, out of scope).
func (g *gen) columnarElemPlan(elem types.Type) (colElemPlan, bool) {
	named, isNamed := types.Unalias(elem).(*types.Named)
	if !isNamed {
		return colElemPlan{}, false
	}
	if isTimeTime(elem) {
		return colElemPlan{}, false
	}
	st, isStruct := named.Underlying().(*types.Struct)
	if !isStruct {
		return colElemPlan{}, false
	}
	// Reuse collectFields so embedded-struct flattening and the qdf>json>name
	// tag precedence match the reflect columnar path's column names exactly.
	fields := collectFields(st)
	if len(fields) == 0 {
		return colElemPlan{}, false
	}
	cols := make([]colColumn, 0, len(fields))
	for i := range fields {
		fi := &fields[i]
		b, isBasic := fi.Field.Type().Underlying().(*types.Basic)
		if !isBasic {
			return colElemPlan{}, false
		}
		kb, api, ok := basicColKind(b.Kind())
		if !ok {
			return colElemPlan{}, false
		}
		cols = append(cols, colColumn{
			WireName: fi.WireKey,
			Access:   fi.Access,
			KindByte: kb,
			GoType:   g.typeExprFromType(fi.Field.Type()),
			ColAPI:   api,
		})
	}
	return colElemPlan{ElemType: g.typeExprFromType(elem), Columns: cols}, true
}

// basicColKind maps a basic kind to its colKind wire byte + the WriteX/ReadX
// API suffix. Must match classifyColKind: int*->0, uint*->1, float64->2,
// bool->3, float32->6.
func basicColKind(k types.BasicKind) (byte, string, bool) {
	switch k {
	case types.Bool:
		return 3, "Bool", true
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
		return 0, "Int", true
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return 1, "Uint", true
	case types.Float32:
		return 6, "Float32", true
	case types.Float64:
		return 2, "Float64", true
	default:
		return 0, "", false
	}
}

// colNamesVar emits (once per element type) a file-level []string of column
// names and returns its ident.
func (g *gen) colNamesVar(plan colElemPlan) string {
	name := "qdfColNames_" + sanitizeIdent(plan.ElemType)
	if !g.colVarSeen[name] {
		g.colVarSeen[name] = true
		parts := make([]string, len(plan.Columns))
		for i, c := range plan.Columns {
			parts[i] = strconv.Quote(c.WireName)
		}
		fmt.Fprintf(&g.colVars, "var %s = []string{%s}\n", name, strings.Join(parts, ", "))
	}
	return name
}

// colKindsVar emits (once per element type) a file-level []byte of column kind
// bytes and returns its ident.
func (g *gen) colKindsVar(plan colElemPlan) string {
	name := "qdfColKinds_" + sanitizeIdent(plan.ElemType)
	if !g.colVarSeen[name] {
		g.colVarSeen[name] = true
		parts := make([]string, len(plan.Columns))
		for i, c := range plan.Columns {
			parts[i] = fmt.Sprintf("0x%02x", c.KindByte)
		}
		fmt.Fprintf(&g.colVars, "var %s = []byte{%s}\n", name, strings.Join(parts, ", "))
	}
	return name
}

func (g *gen) emitEncodeSlice(w io.Writer, expr string, s *types.Slice, indent string) error {
	elem := s.Elem()
	if b, ok := elem.(*types.Basic); ok && b.Kind() == types.Uint8 {
		fmt.Fprintf(w, "%sif %s == nil {\n%s\te.WriteNil()\n%s} else {\n", indent, expr, indent, indent)
		fmt.Fprintf(w, "%s\te.WriteBytes([]byte(%s))\n", indent, expr)
		fmt.Fprintf(w, "%s}\n", indent)
		return nil
	}
	if plan, ok := g.columnarElemPlan(elem); ok {
		return g.emitEncodeColumnarSlice(w, expr, elem, plan, indent)
	}
	fmt.Fprintf(w, "%sif %s == nil {\n%s\te.WriteNil()\n%s} else {\n", indent, expr, indent, indent)
	if err := g.emitEncodeSliceRowMajorBody(w, expr, elem, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// emitEncodeSliceRowMajorBody emits the WriteArrayHeader + per-element loop for
// a non-nil slice (the caller writes the nil guard). Shared by the row-major
// path and the columnar emitter's short-slice fallback.
func (g *gen) emitEncodeSliceRowMajorBody(w io.Writer, expr string, elem types.Type, indent string) error {
	fmt.Fprintf(w, "%se.WriteArrayHeader(len(%s))\n", indent, expr)
	loopVar := g.fresh("i")
	fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, loopVar, expr)
	if err := g.emitEncodeValue(w, expr+"["+loopVar+"]", elem, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// emitEncodeColumnarSlice emits a compile-time transpose of a scalar []struct
// into the tagColStruct frame, with a runtime length gate: slices shorter than
// columnarMinElems (16) stay row-major (frame overhead not worth it), and a nil
// slice emits WriteNil to preserve the nil/empty distinction.
func (g *gen) emitEncodeColumnarSlice(w io.Writer, expr string, elem types.Type, plan colElemPlan, indent string) error {
	namesVar := g.colNamesVar(plan)
	kindsVar := g.colKindsVar(plan)
	s := g.fresh("col")
	fmt.Fprintf(w, "%sif %s == nil {\n", indent, expr)
	fmt.Fprintf(w, "%s\te.WriteNil()\n", indent)
	fmt.Fprintf(w, "%s} else if len(%s) >= 16 { // columnarMinElems\n", indent, expr)
	fmt.Fprintf(w, "%s\t%s := %s\n", indent, s, expr)
	fmt.Fprintf(w, "%s\te.WriteColStructHeader(len(%s), %s, %s)\n", indent, s, namesVar, kindsVar)
	for _, c := range plan.Columns {
		col := g.fresh("c")
		switch c.ColAPI {
		case "Int":
			fmt.Fprintf(w, "%s\t%s := e.ScratchInt(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%s\tfor i := range %s { %s[i] = int64(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%s\tif err := e.WriteIntColumn(%s); err != nil { return err }\n", indent, col)
		case "Uint":
			fmt.Fprintf(w, "%s\t%s := e.ScratchUint(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%s\tfor i := range %s { %s[i] = uint64(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%s\tif err := e.WriteUintColumn(%s); err != nil { return err }\n", indent, col)
		case "Float64":
			fmt.Fprintf(w, "%s\t%s := e.ScratchFloat64(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%s\tfor i := range %s { %s[i] = float64(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%s\tif err := e.WriteFloat64Column(%s); err != nil { return err }\n", indent, col)
		case "Float32":
			fmt.Fprintf(w, "%s\t%s := e.ScratchFloat32(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%s\tfor i := range %s { %s[i] = float32(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%s\tif err := e.WriteFloat32Column(%s); err != nil { return err }\n", indent, col)
		case "Bool":
			fmt.Fprintf(w, "%s\t%s := e.ScratchBool(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%s\tfor i := range %s { %s[i] = %s[i].%s }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%s\tif err := e.WriteBoolColumn(%s); err != nil { return err }\n", indent, col)
		}
	}
	fmt.Fprintf(w, "%s} else {\n", indent) // row-major fallback for short slices
	if err := g.emitEncodeSliceRowMajorBody(w, expr, elem, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

func (g *gen) emitEncodeArray(w io.Writer, expr string, a *types.Array, indent string) error {
	elem := a.Elem()
	// [N]byte fast path: one flat binary blob (matches the reflect encoder), not
	// N tagged elements. Smaller wire for real byte data, single memcpy.
	if b, ok := elem.Underlying().(*types.Basic); ok && b.Kind() == types.Uint8 {
		fmt.Fprintf(w, "%se.WriteBytes(%s[:])\n", indent, expr)
		return nil
	}
	fmt.Fprintf(w, "%se.WriteArrayHeader(%d)\n", indent, a.Len())
	loopVar := g.fresh("i")
	fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, loopVar, expr)
	if err := g.emitEncodeValue(w, expr+"["+loopVar+"]", elem, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

func (g *gen) emitEncodeMap(w io.Writer, expr string, m *types.Map, indent string) error {
	fmt.Fprintf(w, "%sif %s == nil {\n%s\te.WriteNil()\n%s} else {\n", indent, expr, indent, indent)
	fmt.Fprintf(w, "%s\te.WriteMapHeader(len(%s))\n", indent, expr)
	kVar := g.fresh("k")
	vVar := g.fresh("vv")
	fmt.Fprintf(w, "%s\tfor %s, %s := range %s {\n", indent, kVar, vVar, expr)
	if err := g.emitEncodeValue(w, kVar, m.Key(), indent+"\t\t"); err != nil {
		return err
	}
	if err := g.emitEncodeValue(w, vVar, m.Elem(), indent+"\t\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\t}\n", indent)
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// ---------------------------------------------------------------------------
// Decode side

func (g *gen) emitDecodeValue(w io.Writer, lhs string, t types.Type, indent string) error {
	t = types.Unalias(t) // `any` is an alias for interface{}; resolve it
	if isTimeTime(t) {
		secTmp := g.fresh("sec")
		nsecTmp := g.fresh("nsec")
		fmt.Fprintf(w, "%s{\n", indent)
		fmt.Fprintf(w, "%s\t%s, %s, err := d.ReadTimestamp()\n", indent, secTmp, nsecTmp)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		g.imports["time"] = ""
		fmt.Fprintf(w, "%s\t%s = time.Unix(%s, int64(%s)).UTC()\n", indent, lhs, secTmp, nsecTmp)
		fmt.Fprintf(w, "%s}\n", indent)
		return nil
	}

	switch tt := t.(type) {
	case *types.Basic:
		return g.emitDecodeBasic(w, lhs, tt, indent)
	case *types.Named:
		return g.emitDecodeNamed(w, lhs, tt, indent)
	case *types.Pointer:
		return g.emitDecodePointer(w, lhs, tt, indent)
	case *types.Slice:
		return g.emitDecodeSlice(w, lhs, tt, indent)
	case *types.Array:
		return g.emitDecodeArray(w, lhs, tt, indent)
	case *types.Map:
		return g.emitDecodeMap(w, lhs, tt, indent)
	case *types.Interface:
		// Empty interface (any): decode the dynamic value via the runtime,
		// mirroring the e.EncodeValue emitted on the encode side.
		if tt.NumMethods() == 0 {
			fmt.Fprintf(w, "%sif err := d.DecodeValue(&%s); err != nil {\n%s\treturn err\n%s}\n", indent, lhs, indent, indent)
			return nil
		}
		return fmt.Errorf("non-empty interface fields are not supported by qdfgen")
	default:
		return fmt.Errorf("unsupported type %s (%T)", t, t)
	}
}

func (g *gen) emitDecodeBasic(w io.Writer, lhs string, b *types.Basic, indent string) error {
	tmp := g.fresh("rv")
	switch k := b.Kind(); k {
	case types.Bool:
		fmt.Fprintf(w, "%s{\n%s\t%s, err := d.ReadBool()\n", indent, indent, tmp)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\t%s = %s\n%s}\n", indent, lhs, tmp, indent)
	case types.Int:
		g.emitDecodeIntoInt(w, lhs, "int", indent, tmp)
	case types.Int8:
		g.emitDecodeIntoInt(w, lhs, "int8", indent, tmp)
	case types.Int16:
		g.emitDecodeIntoInt(w, lhs, "int16", indent, tmp)
	case types.Int32:
		g.emitDecodeIntoInt(w, lhs, "int32", indent, tmp)
	case types.Int64:
		g.emitDecodeIntoInt(w, lhs, "int64", indent, tmp)
	case types.Uint:
		g.emitDecodeIntoUint(w, lhs, "uint", indent, tmp)
	case types.Uint8:
		g.emitDecodeIntoUint(w, lhs, "uint8", indent, tmp)
	case types.Uint16:
		g.emitDecodeIntoUint(w, lhs, "uint16", indent, tmp)
	case types.Uint32:
		g.emitDecodeIntoUint(w, lhs, "uint32", indent, tmp)
	case types.Uint64:
		g.emitDecodeIntoUint(w, lhs, "uint64", indent, tmp)
	case types.Uintptr:
		g.emitDecodeIntoUint(w, lhs, "uintptr", indent, tmp)
	case types.Float32:
		fmt.Fprintf(w, "%s{\n%s\t%s, err := d.ReadFloat32()\n", indent, indent, tmp)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\t%s = %s\n%s}\n", indent, lhs, tmp, indent)
	case types.Float64:
		fmt.Fprintf(w, "%s{\n%s\t%s, err := d.ReadFloat64()\n", indent, indent, tmp)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\t%s = %s\n%s}\n", indent, lhs, tmp, indent)
	case types.String:
		fmt.Fprintf(w, "%s{\n%s\t%s, err := d.ReadString()\n", indent, indent, tmp)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\t%s = %s\n%s}\n", indent, lhs, tmp, indent)
	default:
		return fmt.Errorf("unsupported basic kind %v", k)
	}
	return nil
}

func (g *gen) emitDecodeIntoInt(w io.Writer, lhs, target, indent, tmp string) {
	fmt.Fprintf(w, "%s{\n%s\t%s, err := d.ReadInt()\n", indent, indent, tmp)
	fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s\t%s = %s(%s)\n%s}\n", indent, lhs, target, tmp, indent)
}

func (g *gen) emitDecodeIntoUint(w io.Writer, lhs, target, indent, tmp string) {
	fmt.Fprintf(w, "%s{\n%s\t%s, err := d.ReadUint()\n", indent, indent, tmp)
	fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s\t%s = %s(%s)\n%s}\n", indent, lhs, target, tmp, indent)
}

func (g *gen) emitDecodeNamed(w io.Writer, lhs string, n *types.Named, indent string) error {
	switch ut := n.Underlying().(type) {
	case *types.Basic:
		tmp := g.fresh("nv")
		fmt.Fprintf(w, "%s{\n", indent)
		switch k := ut.Kind(); k {
		case types.Bool:
			fmt.Fprintf(w, "%s\t%s, err := d.ReadBool()\n", indent, tmp)
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			fmt.Fprintf(w, "%s\t%s, err := d.ReadInt()\n", indent, tmp)
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
			fmt.Fprintf(w, "%s\t%s, err := d.ReadUint()\n", indent, tmp)
		case types.Float32:
			fmt.Fprintf(w, "%s\t%s, err := d.ReadFloat32()\n", indent, tmp)
		case types.Float64:
			fmt.Fprintf(w, "%s\t%s, err := d.ReadFloat64()\n", indent, tmp)
		case types.String:
			fmt.Fprintf(w, "%s\t%s, err := d.ReadString()\n", indent, tmp)
		default:
			return fmt.Errorf("unsupported named basic kind %v", k)
		}
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\t%s = %s(%s)\n", indent, lhs, g.typeRef(n), tmp)
		fmt.Fprintf(w, "%s}\n", indent)
		return nil
	case *types.Struct:
		// Nested struct: thread the shared decoder (no new decoder; the nested
		// DecodeQDF inherits d's noCopy / arena).
		fmt.Fprintf(w, "%sif err := qdf.DecodeNested(d, &%s); err != nil {\n%s\treturn err\n%s}\n", indent, lhs, indent, indent)
		return nil
	case *types.Slice, *types.Array, *types.Map, *types.Pointer:
		return g.emitDecodeValue(w, lhs, ut, indent)
	default:
		return fmt.Errorf("unsupported named underlying %T", ut)
	}
}

func (g *gen) emitDecodePointer(w io.Writer, lhs string, p *types.Pointer, indent string) error {
	fmt.Fprintf(w, "%s{\n", indent)
	fmt.Fprintf(w, "%s\tisNil, err := d.IsNil()\n", indent)
	fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s\tif isNil {\n%s\t\t%s = nil\n%s\t} else {\n", indent, indent, lhs, indent)

	elem := p.Elem()
	if isTimeTime(elem) {
		secTmp := g.fresh("sec")
		nsecTmp := g.fresh("nsec")
		fmt.Fprintf(w, "%s\t\t%s, %s, err := d.ReadTimestamp()\n", indent, secTmp, nsecTmp)
		fmt.Fprintf(w, "%s\t\tif err != nil {\n%s\t\t\treturn err\n%s\t\t}\n", indent, indent, indent)
		g.imports["time"] = ""
		fmt.Fprintf(w, "%s\t\tt := time.Unix(%s, int64(%s)).UTC()\n", indent, secTmp, nsecTmp)
		fmt.Fprintf(w, "%s\t\t%s = &t\n", indent, lhs)
	} else if named, ok := elem.(*types.Named); ok {
		if _, isStruct := named.Underlying().(*types.Struct); isStruct {
			fmt.Fprintf(w, "%s\t\t%s = new(%s)\n", indent, lhs, g.typeRef(named))
			fmt.Fprintf(w, "%s\t\tif err := qdf.DecodeNested(d, %s); err != nil {\n%s\t\t\treturn err\n%s\t\t}\n", indent, lhs, indent, indent)
		} else {
			// Named non-struct element (e.g. *Label where type Label string):
			// route through emitDecodeNamed so it emits the Label(tmp) conversion.
			// Passing named.Underlying() here would assign the raw basic value to
			// a *Label lvalue and fail to compile.
			fmt.Fprintf(w, "%s\t\t%s = new(%s)\n", indent, lhs, g.typeRef(named))
			if err := g.emitDecodeNamed(w, "(*"+lhs+")", named, indent+"\t\t"); err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(w, "%s\t\t%s = new(%s)\n", indent, lhs, g.typeExprFromType(elem))
		if err := g.emitDecodeValue(w, "(*"+lhs+")", elem, indent+"\t\t"); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "%s\t}\n", indent)
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

func (g *gen) emitDecodeSlice(w io.Writer, lhs string, s *types.Slice, indent string) error {
	elem := s.Elem()
	if b, ok := elem.(*types.Basic); ok && b.Kind() == types.Uint8 {
		fmt.Fprintf(w, "%s{\n", indent)
		fmt.Fprintf(w, "%s\tisNil, err := d.IsNil()\n", indent)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\tif isNil {\n%s\t\t%s = nil\n%s\t} else {\n", indent, indent, lhs, indent)
		fmt.Fprintf(w, "%s\t\t_v, err := d.ReadBytes()\n", indent)
		fmt.Fprintf(w, "%s\t\tif err != nil {\n%s\t\t\treturn err\n%s\t\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\t\t%s = _v\n", indent, lhs)
		fmt.Fprintf(w, "%s\t}\n", indent)
		fmt.Fprintf(w, "%s}\n", indent)
		return nil
	}
	fmt.Fprintf(w, "%s{\n", indent)
	fmt.Fprintf(w, "%s\tisNil, err := d.IsNil()\n", indent)
	fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
	plan, columnar := g.columnarElemPlan(elem)
	fmt.Fprintf(w, "%s\tif isNil {\n%s\t\t%s = nil\n", indent, indent, lhs)
	if columnar {
		fmt.Fprintf(w, "%s\t} else if d.PeekColStruct() {\n", indent)
		if err := g.emitDecodeColumnarBody(w, lhs, elem, plan, indent+"\t\t"); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "%s\t} else {\n", indent)
	if err := g.emitDecodeSliceRowMajorBody(w, lhs, elem, indent+"\t\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\t}\n", indent)
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// emitDecodeSliceRowMajorBody emits the ReadArrayHeader + make + per-element
// loop for a non-nil slice (the caller writes the surrounding nil guard).
func (g *gen) emitDecodeSliceRowMajorBody(w io.Writer, lhs string, elem types.Type, indent string) error {
	nVar := g.fresh("n")
	fmt.Fprintf(w, "%s%s, err := d.ReadArrayHeader()\n", indent, nVar)
	fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
	// Bound the allocation by remaining input so a hostile length header
	// can't trigger a multi-GB make before any element is read (matches the
	// reflect decoder's CheckLength gate).
	fmt.Fprintf(w, "%sif err := d.CheckLength(%s, 1); err != nil {\n%s\treturn err\n%s}\n", indent, nVar, indent, indent)
	fmt.Fprintf(w, "%s%s = make([]%s, %s)\n", indent, lhs, g.typeExprFromType(elem), nVar)
	loopVar := g.fresh("i")
	fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, loopVar, nVar)
	if err := g.emitDecodeValue(w, lhs+"["+loopVar+"]", elem, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// emitDecodeColumnarBody emits decode for the tagColStruct frame: read the
// header (row count + column shape), allocate the slice, then decode each
// declared column by name and scatter it into the matching struct field
// (narrowing per field type). The slice is bounded by maxColumnarElems inside
// ReadColStructHeader (the reflect path's cap), so columns may be compressed
// below n bytes without a false length rejection. Unknown columns are not
// tolerated in v1 (the generated decoder requires its exact declared set).
func (g *gen) emitDecodeColumnarBody(w io.Writer, lhs string, elem types.Type, plan colElemPlan, indent string) error {
	nVar := g.fresh("n")
	namesVar := g.fresh("names")
	kindsVar := g.fresh("kinds")
	idxVar := g.fresh("ci")
	fmt.Fprintf(w, "%s%s, %s, %s, err := d.ReadColStructHeader()\n", indent, nVar, namesVar, kindsVar)
	fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s_ = %s\n", indent, kindsVar)
	fmt.Fprintf(w, "%s%s = make([]%s, %s)\n", indent, lhs, g.typeExprFromType(elem), nVar)
	fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, idxVar, namesVar)
	fmt.Fprintf(w, "%s\tswitch %s[%s] {\n", indent, namesVar, idxVar)
	for _, c := range plan.Columns {
		colv := g.fresh("col")
		fmt.Fprintf(w, "%s\tcase %s:\n", indent, strconv.Quote(c.WireName))
		var readFn string
		switch c.ColAPI {
		case "Int":
			readFn = "ReadIntColumn"
		case "Uint":
			readFn = "ReadUintColumn"
		case "Float64":
			readFn = "ReadFloat64Column"
		case "Float32":
			readFn = "ReadFloat32Column"
		case "Bool":
			readFn = "ReadBoolColumn"
		}
		fmt.Fprintf(w, "%s\t\t%s, err := d.%s(%s)\n", indent, colv, readFn, nVar)
		fmt.Fprintf(w, "%s\t\tif err != nil {\n%s\t\t\treturn err\n%s\t\t}\n", indent, indent, indent)
		if c.ColAPI == "Bool" {
			fmt.Fprintf(w, "%s\t\tfor i := range %s {\n%s\t\t\t%s[i].%s = %s[i]\n%s\t\t}\n", indent, colv, indent, lhs, c.Access, colv, indent)
		} else {
			fmt.Fprintf(w, "%s\t\tfor i := range %s {\n%s\t\t\t%s[i].%s = %s(%s[i])\n%s\t\t}\n", indent, colv, indent, lhs, c.Access, c.GoType, colv, indent)
		}
	}
	fmt.Fprintf(w, "%s\tdefault:\n%s\t\treturn qdf.ErrTypeMismatch\n", indent, indent)
	fmt.Fprintf(w, "%s\t}\n", indent) // switch
	fmt.Fprintf(w, "%s}\n", indent)   // for
	return nil
}

func (g *gen) emitDecodeArray(w io.Writer, lhs string, a *types.Array, indent string) error {
	elem := a.Elem()
	// [N]byte fast path: read the flat blob straight into the inline array via
	// one length-checked memcpy (matches the reflect decoder), zero allocation.
	if b, ok := elem.Underlying().(*types.Basic); ok && b.Kind() == types.Uint8 {
		bv := g.fresh("ab")
		fmt.Fprintf(w, "%s{\n", indent)
		fmt.Fprintf(w, "%s\t%s, err := d.ReadStringBytes()\n", indent, bv)
		fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
		fmt.Fprintf(w, "%s\tif len(%s) != %d {\n%s\t\treturn qdf.ErrTypeMismatch\n%s\t}\n", indent, bv, a.Len(), indent, indent)
		fmt.Fprintf(w, "%s\tcopy(%s[:], %s)\n", indent, lhs, bv)
		fmt.Fprintf(w, "%s}\n", indent)
		return nil
	}
	fmt.Fprintf(w, "%s{\n", indent)
	nVar := g.fresh("n")
	fmt.Fprintf(w, "%s\t%s, err := d.ReadArrayHeader()\n", indent, nVar)
	fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s\tif %s != %d {\n%s\t\treturn qdf.ErrTypeMismatch\n%s\t}\n", indent, nVar, a.Len(), indent, indent)
	loopVar := g.fresh("i")
	fmt.Fprintf(w, "%s\tfor %s := range %d {\n", indent, loopVar, a.Len())
	if err := g.emitDecodeValue(w, lhs+"["+loopVar+"]", elem, indent+"\t\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\t}\n", indent)
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

func (g *gen) emitDecodeMap(w io.Writer, lhs string, m *types.Map, indent string) error {
	fmt.Fprintf(w, "%s{\n", indent)
	fmt.Fprintf(w, "%s\tisNil, err := d.IsNil()\n", indent)
	fmt.Fprintf(w, "%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s\tif isNil {\n%s\t\t%s = nil\n%s\t} else {\n", indent, indent, lhs, indent)
	nVar := g.fresh("n")
	fmt.Fprintf(w, "%s\t\t%s, err := d.ReadMapHeader()\n", indent, nVar)
	fmt.Fprintf(w, "%s\t\tif err != nil {\n%s\t\t\treturn err\n%s\t\t}\n", indent, indent, indent)
	keyExpr := g.typeExprFromType(m.Key())
	valExpr := g.typeExprFromType(m.Elem())
	// Each map entry is at least two wire bytes (key + value); CheckLength(n,1)
	// conservatively bounds the alloc by remaining input against a hostile count.
	fmt.Fprintf(w, "%s\t\tif err := d.CheckLength(%s, 1); err != nil {\n%s\t\t\treturn err\n%s\t\t}\n", indent, nVar, indent, indent)
	fmt.Fprintf(w, "%s\t\t%s = make(map[%s]%s, %s)\n", indent, lhs, keyExpr, valExpr, nVar)
	fmt.Fprintf(w, "%s\t\tfor range %s {\n", indent, nVar)
	kVar := g.fresh("k")
	vVar := g.fresh("vv")
	fmt.Fprintf(w, "%s\t\t\tvar %s %s\n", indent, kVar, keyExpr)
	fmt.Fprintf(w, "%s\t\t\tvar %s %s\n", indent, vVar, valExpr)
	// Intern string keys to dedupe across the map (and across the whole
	// stream when the Decoder is reused via a pool).
	if b, ok := m.Key().Underlying().(*types.Basic); ok && b.Kind() == types.String {
		kbVar := g.fresh("kb")
		fmt.Fprintf(w, "%s\t\t\t%s, err := d.ReadStringBytes()\n", indent, kbVar)
		fmt.Fprintf(w, "%s\t\t\tif err != nil { return err }\n", indent)
		// InternKey returns string; a defined string key type (e.g. type K string)
		// needs an explicit conversion. keyExpr is "string" for a plain key, so
		// the conversion is a harmless no-op there.
		fmt.Fprintf(w, "%s\t\t\t%s = %s(d.InternKey(%s))\n", indent, kVar, keyExpr, kbVar)
	} else {
		if err := g.emitDecodeValue(w, kVar, m.Key(), indent+"\t\t\t"); err != nil {
			return err
		}
	}
	if err := g.emitDecodeValue(w, vVar, m.Elem(), indent+"\t\t\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\t\t\t%s[%s] = %s\n", indent, lhs, kVar, vVar)
	fmt.Fprintf(w, "%s\t\t}\n", indent)
	fmt.Fprintf(w, "%s\t}\n", indent)
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// ---------------------------------------------------------------------------

// typeRef returns the qualified source-level identifier for a named type
// (e.g. "pkgname.TypeName" or just "TypeName" if it's in our package).
func (g *gen) typeRef(n *types.Named) string {
	obj := n.Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() == g.pkg.PkgPath {
		return obj.Name()
	}
	alias := g.importAlias(obj.Pkg().Path())
	if alias == "" {
		return obj.Name()
	}
	return alias + "." + obj.Name()
}

// typeExprFromType returns a Go source-level expression for the type. Mirrors
// types.TypeString but routes the import qualifier through our import map.
func (g *gen) typeExprFromType(t types.Type) string {
	qf := func(p *types.Package) string {
		if p == nil || p.Path() == g.pkg.PkgPath {
			return ""
		}
		return g.importAlias(p.Path())
	}
	return types.TypeString(t, qf)
}

// fresh returns a unique identifier starting with the given prefix.
func (g *gen) fresh(prefix string) string {
	g.uniqCounter++
	return fmt.Sprintf("%s%d", prefix, g.uniqCounter)
}

// isTimeTime reports whether t resolves to the standard library's time.Time.
func isTimeTime(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "time" && obj.Name() == "Time"
}
