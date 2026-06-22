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
	// LogTo defaults to os.Stderr when nil.
	LogTo io.Writer
	// Verbose logs progress to LogTo. (1-byte tail, last to avoid padding
	// before the string/interface fields above.)
	Verbose bool
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
		g.targets = want
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

	// emitted tracks struct types already emitted in this file.
	emitted map[string]bool

	// path tracks the chain of types currently being expanded; used to
	// detect cycles through value (non-pointer) fields.
	path []string

	// targets is the set of type names qdfgen is generating in this run. A
	// []struct field whose element type is a target gets a faithful generated
	// structural codec, so columnar transposition is safe; an element with a
	// codec NOT in this set is hand-written (or a stale prior generation) and
	// must keep its custom codec (see columnarElemPlan's guard).
	targets map[string]bool

	// uniqCounter ensures generated identifiers stay unique. (Pointer-free, kept
	// last so the GC pointer-scan range covers only the fields above.)
	uniqCounter int
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
	ident := "qdfFieldHdr_" + sanitizeIdent(name) + "_" + strconv.Itoa(g.uniqCounter)
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
	Field   *types.Var // single pointer word first to tighten the GC scan range
	GoName  string     // exported Go field name (for diagnostics)
	Access  string     // Go access path from the receiver, e.g. "X" or "Base.X"
	WireKey string     // string used as the map key on the wire
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
	shapeTok := "qdfShapeTok_" + typeName
	hdrsVar := "qdfFieldHdrs_" + typeName
	fmt.Fprintf(w, "var %s byte\n", shapeTok)
	fmt.Fprintf(w, "var %s = [][]byte{", hdrsVar)
	for i, f := range fields {
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(g.fieldNameVar(f.WireKey))
	}
	w.WriteString("}\n\n")

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
	// A named type with its own MarshalQDF (hand-written codec) must route through
	// it, not be descended structurally — mirrors reflect fillDesc, which checks
	// Marshaler before the Kind switch. Target struct types have no method yet at
	// generate time, so they fall through to the *types.Struct case (also
	// EncodeNested); this catches the named non-struct codec types the Kind switch
	// would otherwise emit as a bare scalar/slice/map.
	if hasMethod(n, "MarshalQDF") {
		fmt.Fprintf(w, "%sif err := qdf.EncodeNested(e, &%s); err != nil {\n%s\treturn err\n%s}\n", indent, expr, indent, indent)
		return nil
	}
	switch ut := n.Underlying().(type) {
	case *types.Basic:
		return g.emitEncodeBasic(w, expr, ut, indent)
	case *types.Struct:
		// Nested struct: thread the parent's encoder via qdf.EncodeNested (no new
		// encoder allocated). expr is always addressable (a struct field,
		// slice/array element, map range var, or pointer deref), so take its
		// address directly — EncodeQDF only reads the receiver.
		if !g.canEncodeNested(n) {
			return g.nestedStructErr(n, "encode")
		}
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
			if !g.canEncodeNested(named) {
				return g.nestedStructErr(named, "encode")
			}
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
	GoType   string // rendered field type for scatter narrowing (non-nullable: the field; nullable: the pointed-to elem)
	ColAPI   string // "Int"|"Uint"|"Float64"|"Float32"|"Bool"|"String"|"Time"|"Bytes"
	// 1-byte tails last so they don't force a padding word between the strings.
	KindByte byte // colKind wire byte: int0 uint1 f64 2 bool3 str4 time5 f32 6; |0x80 nullable
	Nullable bool // *T field: presence bitmap + dense column of present values
}

type colResidual struct {
	Type   types.Type // field type, for the row-major encode/decode emitters
	Access string     // Go field selector from the element
}

type colElemPlan struct {
	ElemType         string // rendered element type, e.g. "GenMetric"
	Columns          []colColumn
	Residual         []colResidual // non-columnar fields, declaration order
	HybridNames      []string      // ALL fields, declaration order (hybrid shape)
	HybridKinds      []byte        // colKind per field; 0xFF (residualKind) for residual
	HasNumericOrTime bool          // int/uint/float/bool/time column present
	HasString        bool          // string column present
}

// hybrid reports whether the element has non-columnar (residual) fields, so it
// encodes as a tagHybridColStruct frame rather than a pure tagColStruct.
func (p *colElemPlan) hybrid() bool { return len(p.Residual) > 0 }

// stringOnly reports whether the only columnar benefit is from string columns,
// so the encoder must run a cardinality probe (numeric/time columns always make
// columnar worthwhile, strings only when low-cardinality).
func (p *colElemPlan) stringOnly() bool { return p.HasString && !p.HasNumericOrTime }

// columnarElemPlan returns a plan when elem is a named struct with at least one
// columnar-eligible field (scalar int/uint/float/bool, string, or time.Time).
// Non-eligible fields (nested struct, map, slice, []byte, pointer, array,
// interface) become residual fields carried row-major in a hybrid frame. ok is
// false (→ full row-major) only when no field is eligible.
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
	// A custom-codec element must NOT be columnar-transposed: the transpose
	// replays the struct's field layout and bypasses MarshalQDF/UnmarshalQDF,
	// diverging from the reflect path (which routes through the codec) and
	// corrupting any element whose codec is non-structural. Mirrors the reflect
	// guard (commit 9c6f524). A type qdfgen is generating in THIS run keeps
	// columnar (its emitted structural codec faithfully matches the transpose),
	// so the guard fires only for hand-written codecs (or a stale prior
	// generation no longer in -type), matching reflect's Marshaler skip.
	if !g.targets[named.Obj().Name()] && hasQDFCodecMethod(elem) {
		return colElemPlan{}, false
	}
	// Reuse collectFields so embedded-struct flattening and the qdf>json>name
	// tag precedence match the reflect columnar path's column names exactly.
	fields := collectFields(st)
	if len(fields) == 0 {
		return colElemPlan{}, false
	}
	plan := colElemPlan{ElemType: g.typeExprFromType(elem)}
	for i := range fields {
		fi := &fields[i]
		if col, isCol := g.classifyColField(fi.WireKey, fi.Access, fi.Field.Type()); isCol {
			plan.Columns = append(plan.Columns, col)
			plan.HybridNames = append(plan.HybridNames, fi.WireKey)
			plan.HybridKinds = append(plan.HybridKinds, col.KindByte)
			// A plain (non-nullable) string column is the only kind whose
			// columnar benefit is data-dependent → gates the string-only probe.
			// []byte (raw-slab decode win) and nullable columns always benefit.
			if col.ColAPI == "String" && !col.Nullable {
				plan.HasString = true
			} else {
				plan.HasNumericOrTime = true
			}
			continue
		}
		plan.Residual = append(plan.Residual, colResidual{Access: fi.Access, Type: fi.Field.Type()})
		plan.HybridNames = append(plan.HybridNames, fi.WireKey)
		plan.HybridKinds = append(plan.HybridKinds, 0xFF) // residualKind
	}
	if len(plan.Columns) == 0 {
		return colElemPlan{}, false // no columnar benefit — full row-major
	}
	return plan, true
}

// hasQDFCodecMethod reports whether t (or *t) declares a MarshalQDF or
// UnmarshalQDF method in its source method set. A type qdfgen is about to
// generate does not yet have these methods in its loaded source, so a positive
// result identifies a hand-written custom codec (or a stale prior generation).
func hasQDFCodecMethod(t types.Type) bool {
	ms := types.NewMethodSet(types.NewPointer(t))
	for method := range ms.Methods() {
		switch method.Obj().Name() {
		case "MarshalQDF", "UnmarshalQDF":
			return true
		}
	}
	return false
}

// hasMethod reports whether *t's method set contains name. Used to route a field
// whose named type carries its OWN hand-written codec through qdf.EncodeNested /
// DecodeNested instead of descending it structurally (which would bypass the
// codec). Direction-specific (MarshalQDF vs UnmarshalQDF) so the asymmetric case
// — a type implementing only one side — encodes/decodes structurally on the other,
// mirroring the reflect fillDesc behaviour.
func hasMethod(t types.Type, name string) bool {
	ms := types.NewMethodSet(types.NewPointer(t))
	for method := range ms.Methods() {
		if method.Obj().Name() == name {
			return true
		}
	}
	return false
}

// nestedStructErr reports a nested struct that qdfgen would emit a
// qdf.EncodeNested/DecodeNested call against but that implements neither codec
// side: it is not a -type target (so no codec is generated for it) and carries
// no hand-written MarshalQDF/UnmarshalQDF. Without this check the generated file
// fails to compile with an opaque "does not implement qdf.Marshaler" error far
// from its cause; here we name the missing -type entry instead.
func (g *gen) nestedStructErr(n *types.Named, dir string) error {
	return fmt.Errorf("%s: nested struct type %q is neither a -type target nor a type with a hand-written %s codec; add %q to the -type list (or give it a codec)",
		dir, n.Obj().Name(), map[string]string{"encode": "MarshalQDF", "decode": "UnmarshalQDF"}[dir], n.Obj().Name())
}

// canEncodeNested reports whether a nested named struct will satisfy qdf.Marshaler
// in the generated code — either qdfgen generates its codec (it is a target) or it
// already has a hand-written one.
func (g *gen) canEncodeNested(n *types.Named) bool {
	return g.targets[n.Obj().Name()] || hasMethod(n, "MarshalQDF")
}

// canDecodeNested is canEncodeNested's decode-side mirror (qdf.Unmarshaler).
func (g *gen) canDecodeNested(n *types.Named) bool {
	return g.targets[n.Obj().Name()] || hasMethod(n, "UnmarshalQDF")
}

// classifyColField maps a struct field to its columnar column descriptor, or
// returns ok=false for a residual (non-columnar) field. Eligible: scalar basics,
// string, time.Time, []byte (str column via the byte view), and pointers to any
// of the scalar/string kinds (nullable column: presence bitmap + dense values).
// Residual: *time.Time, *struct, nested struct, map, non-byte slice, array,
// interface.
func (g *gen) classifyColField(wireName, access string, ft types.Type) (colColumn, bool) {
	// A field whose named type carries its own codec must stay residual (row-major)
	// so emitEncode/DecodeValue routes it through qdf.EncodeNested/DecodeNested —
	// a columnar string/scalar column would bypass the codec. Mirrors reflect
	// classifyColKind, which rejects marshalerKind != 0 before the kind switch.
	if hasQDFCodecMethod(ft) {
		return colColumn{}, false
	}
	if isTimeTime(ft) {
		return colColumn{WireName: wireName, Access: access, KindByte: 5, GoType: "time.Time", ColAPI: "Time"}, true
	}
	// []byte (slice of uint8) → a NULLABLE string column over an unsafe byte
	// view: nil is a distinct state from empty for a []byte, so it travels as
	// the presence bit (nil → absent, empty/non-nil → present "") to preserve
	// the nil-vs-empty distinction (a plain string column would collapse them).
	if sl, ok := ft.Underlying().(*types.Slice); ok {
		// Require the element to be *exactly* byte/uint8 — match elem directly,
		// not its Underlying. A defined byte-element slice (`type B byte; []B`)
		// cannot be expressed as []byte: the columnar Bytes emit would generate
		// `unsafe.SliceData([]B)` (*B, not *byte) and `[]B([]byte(...))` (an
		// illegal slice conversion), neither of which compiles. The row-major
		// path (emitEncodeSlice) already gates on `elem.(*types.Basic)` for the
		// same reason and falls through to the generic per-element encoder, which
		// handles defined byte elements correctly; mirror that here.
		if eb, ok := sl.Elem().(*types.Basic); ok && eb.Kind() == types.Uint8 {
			return colColumn{WireName: wireName, Access: access, KindByte: 4 | 0x80, GoType: g.typeExprFromType(ft), ColAPI: "Bytes", Nullable: true}, true
		}
		return colColumn{}, false
	}
	// Pointer to a scalar/string → nullable column.
	if ptr, ok := ft.Underlying().(*types.Pointer); ok {
		if c, isCol := g.classifyNullableElem(wireName, access, ptr.Elem()); isCol {
			return c, true
		}
		return colColumn{}, false
	}
	b, ok := ft.Underlying().(*types.Basic)
	if !ok {
		return colColumn{}, false
	}
	if b.Kind() == types.String {
		return colColumn{WireName: wireName, Access: access, KindByte: 4, GoType: g.typeExprFromType(ft), ColAPI: "String"}, true
	}
	kb, api, ok := basicColKind(b.Kind())
	if !ok {
		return colColumn{}, false
	}
	return colColumn{WireName: wireName, Access: access, KindByte: kb, GoType: g.typeExprFromType(ft), ColAPI: api}, true
}

// classifyNullableElem maps the pointed-to type of a *T field to a nullable
// column (KindByte | 0x80 = colKindNullable). Eligible elem kinds: scalar
// basics and string. *time.Time / *struct / others stay residual. GoType is the
// element type (the value the scatter allocates and points at).
func (g *gen) classifyNullableElem(wireName, access string, elem types.Type) (colColumn, bool) {
	if isTimeTime(elem) {
		return colColumn{}, false // *time.Time → residual (scope)
	}
	b, ok := elem.Underlying().(*types.Basic)
	if !ok {
		return colColumn{}, false
	}
	if b.Kind() == types.String {
		return colColumn{WireName: wireName, Access: access, KindByte: 4 | 0x80, GoType: g.typeExprFromType(elem), ColAPI: "String", Nullable: true}, true
	}
	kb, api, ok := basicColKind(b.Kind())
	if !ok {
		return colColumn{}, false
	}
	return colColumn{WireName: wireName, Access: access, KindByte: kb | 0x80, GoType: g.typeExprFromType(elem), ColAPI: api, Nullable: true}, true
}

// basicColKind maps a numeric/bool basic kind to its colKind wire byte + the
// WriteX/ReadX API suffix. Must match classifyColKind: int*->0, uint*->1,
// float64->2, bool->3, float32->6. (String/time handled in classifyColField.)
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

// colNamesVar emits (once per ident) a file-level []string and returns its
// ident. Appends straight to the var buffer (no intermediate parts slice +
// strings.Join + fmt format-string parse).
func (g *gen) colNamesVar(ident string, names []string) string {
	if !g.colVarSeen[ident] {
		g.colVarSeen[ident] = true
		b := &g.colVars
		b.WriteString("var ")
		b.WriteString(ident)
		b.WriteString(" = []string{")
		for i, nm := range names {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strconv.Quote(nm))
		}
		b.WriteString("}\n")
	}
	return ident
}

// colKindsVar emits (once per ident) a file-level []byte and returns its ident.
// Appends each kind as a 0xNN literal directly (no fmt.Sprintf per element).
func (g *gen) colKindsVar(ident string, kinds []byte) string {
	if !g.colVarSeen[ident] {
		g.colVarSeen[ident] = true
		b := &g.colVars
		b.WriteString("var ")
		b.WriteString(ident)
		b.WriteString(" = []byte{")
		for i, k := range kinds {
			if i > 0 {
				b.WriteString(", ")
			}
			writeHexByteLiteral(b, k)
		}
		b.WriteString("}\n")
	}
	return ident
}

// writeHexByteLiteral appends a Go 0xNN byte literal for k.
func writeHexByteLiteral(b *bytes.Buffer, k byte) {
	const hexd = "0123456789abcdef"
	b.WriteByte('0')
	b.WriteByte('x')
	b.WriteByte(hexd[k>>4])
	b.WriteByte(hexd[k&0xf])
}

// eligNamesKinds returns the eligible columns' wire names + kind bytes (the
// pure tagColStruct shape).
func (p *colElemPlan) eligNamesKinds() ([]string, []byte) {
	names := make([]string, len(p.Columns))
	kinds := make([]byte, len(p.Columns))
	for i, c := range p.Columns {
		names[i] = c.WireName
		kinds[i] = c.KindByte
	}
	return names, kinds
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

// emitEncodeColumnarSlice emits a compile-time transpose of a []struct into the
// columnar frame (pure tagColStruct, or tagHybridColStruct when residual fields
// are present), with a runtime length gate (>= columnarMinElems). A nil slice
// emits WriteNil; short slices fall back to row-major. A string-only element
// runs a cardinality probe so high-cardinality strings stay row-major (matching
// the reflect columnarProbe), while any numeric/time column makes columnar
// unconditional.
func (g *gen) emitEncodeColumnarSlice(w io.Writer, expr string, elem types.Type, plan colElemPlan, indent string) error {
	s := g.fresh("col")
	fmt.Fprintf(w, "%sif %s == nil {\n", indent, expr)
	fmt.Fprintf(w, "%s\te.WriteNil()\n", indent)
	fmt.Fprintf(w, "%s} else if len(%s) >= 16 { // columnarMinElems\n", indent, expr)
	fmt.Fprintf(w, "%s\t%s := %s\n", indent, s, expr)
	if plan.stringOnly() {
		// Probe a sample of ALL string columns and let StringColumnsBeneficial make
		// ONE aggregate decision (sum of colBytes vs rowBytes), matching the reflect
		// columnarProbe. A per-column OR would over-trigger columnar (any one column
		// beneficial) and diverge from reflect. Each column gets its own small probe
		// buffer (<=32 strings) so they can be passed together.
		ben := g.fresh("ben")
		pbs := make([]string, len(plan.Columns))
		for ci, c := range plan.Columns {
			pb := g.fresh("pb")
			pbs[ci] = pb
			fmt.Fprintf(w, "%s\t%s := make([]string, min(len(%s), 32))\n", indent, pb, s)
			fmt.Fprintf(w, "%s\tfor i := range %s { %s[i] = string(%s[i].%s) }\n", indent, pb, pb, s, c.Access)
		}
		fmt.Fprintf(w, "%s\t%s := qdf.StringColumnsBeneficial(%s)\n", indent, ben, strings.Join(pbs, ", "))
		fmt.Fprintf(w, "%s\tif %s {\n", indent, ben)
		if err := g.emitEncodeColumnarFrame(w, s, plan, indent+"\t\t"); err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t} else {\n", indent)
		if err := g.emitEncodeSliceRowMajorBody(w, s, elem, indent+"\t\t"); err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t}\n", indent)
	} else {
		if err := g.emitEncodeColumnarFrame(w, s, plan, indent+"\t"); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "%s} else {\n", indent) // row-major fallback for short slices
	if err := g.emitEncodeSliceRowMajorBody(w, expr, elem, indent+"\t"); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s}\n", indent)
	return nil
}

// emitEncodeColumnarFrame emits the header + eligible-column transpose (+ the
// hybrid residual block). s is a non-nil slice local of length >= 16.
func (g *gen) emitEncodeColumnarFrame(w io.Writer, s string, plan colElemPlan, indent string) error {
	id := sanitizeIdent(plan.ElemType)
	if plan.hybrid() {
		nv := g.colNamesVar("qdfHybNames_"+id, plan.HybridNames)
		kv := g.colKindsVar("qdfHybKinds_"+id, plan.HybridKinds)
		fmt.Fprintf(w, "%se.WriteHybridColStructHeader(len(%s), %s, %s)\n", indent, s, nv, kv)
	} else {
		names, kinds := plan.eligNamesKinds()
		nv := g.colNamesVar("qdfColNames_"+id, names)
		kv := g.colKindsVar("qdfColKinds_"+id, kinds)
		fmt.Fprintf(w, "%se.WriteColStructHeader(len(%s), %s, %s)\n", indent, s, nv, kv)
	}
	if err := g.emitEncodeColumnGathers(w, s, plan, indent); err != nil {
		return err
	}
	if plan.hybrid() {
		ri := g.fresh("i")
		fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, ri, s)
		for _, rf := range plan.Residual {
			if err := g.emitEncodeValue(w, s+"["+ri+"]."+rf.Access, rf.Type, indent+"\t"); err != nil {
				return err
			}
		}
		fmt.Fprintf(w, "%s}\n", indent)
	}
	return nil
}

// emitEncodeColumnGathers emits the per-eligible-column gather-into-scratch +
// WriteXColumn for each column in declaration order. Shared by the pure and
// hybrid frames.
func (g *gen) emitEncodeColumnGathers(w io.Writer, s string, plan colElemPlan, indent string) error {
	for _, c := range plan.Columns {
		if c.Nullable {
			g.emitEncodeNullableColumn(w, s, c, indent)
			continue
		}
		col := g.fresh("c")
		switch c.ColAPI {
		case "Int":
			fmt.Fprintf(w, "%s%s := e.ScratchInt(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s[i] = int64(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%sif err := e.WriteIntColumn(%s); err != nil { return err }\n", indent, col)
		case "Uint":
			fmt.Fprintf(w, "%s%s := e.ScratchUint(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s[i] = uint64(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%sif err := e.WriteUintColumn(%s); err != nil { return err }\n", indent, col)
		case "Float64":
			fmt.Fprintf(w, "%s%s := e.ScratchFloat64(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s[i] = float64(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%sif err := e.WriteFloat64Column(%s); err != nil { return err }\n", indent, col)
		case "Float32":
			fmt.Fprintf(w, "%s%s := e.ScratchFloat32(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s[i] = float32(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%sif err := e.WriteFloat32Column(%s); err != nil { return err }\n", indent, col)
		case "Bool":
			fmt.Fprintf(w, "%s%s := e.ScratchBool(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s[i] = %s[i].%s }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%sif err := e.WriteBoolColumn(%s); err != nil { return err }\n", indent, col)
		case "String":
			fmt.Fprintf(w, "%s%s := e.ScratchString(len(%s))\n", indent, col, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s[i] = string(%s[i].%s) }\n", indent, s, col, s, c.Access)
			fmt.Fprintf(w, "%se.WriteStringColumn(%s)\n", indent, col)
		case "Time":
			sec := g.fresh("sec")
			ns := g.fresh("ns")
			tv := g.fresh("t")
			fmt.Fprintf(w, "%s%s := e.ScratchInt(len(%s))\n", indent, sec, s)
			fmt.Fprintf(w, "%s%s := e.ScratchUint(len(%s))\n", indent, ns, s)
			fmt.Fprintf(w, "%sfor i := range %s { %s := %s[i].%s.UTC(); %s[i] = %s.Unix(); %s[i] = uint64(%s.Nanosecond()) }\n",
				indent, s, tv, s, c.Access, sec, tv, ns, tv)
			fmt.Fprintf(w, "%sif err := e.WriteTimeColumn(%s, %s); err != nil { return err }\n", indent, sec, ns)
		}
	}
	return nil
}

// emitEncodeNullableColumn emits a *T field's nullable column: a presence
// bitmap followed by the dense column of present (non-nil) values, matching
// encodeNullableColumn's wire layout (KindByte carries the 0x80 nullable bit).
func (g *gen) emitEncodeNullableColumn(w io.Writer, s string, c colColumn, indent string) {
	mask := g.fresh("mask")
	di := g.fresh("di")
	fmt.Fprintf(w, "%s%s := e.ScratchMask(len(%s))\n", indent, mask, s)
	switch c.ColAPI {
	case "Int", "Uint", "Float64", "Float32", "Bool":
		col := g.fresh("c")
		var scratchFn, conv string
		switch c.ColAPI {
		case "Int":
			scratchFn, conv = "ScratchInt", "int64"
		case "Uint":
			scratchFn, conv = "ScratchUint", "uint64"
		case "Float64":
			scratchFn, conv = "ScratchFloat64", "float64"
		case "Float32":
			scratchFn, conv = "ScratchFloat32", "float32"
		case "Bool":
			scratchFn, conv = "ScratchBool", ""
		}
		fmt.Fprintf(w, "%s%s := e.%s(len(%s))\n", indent, col, scratchFn, s)
		fmt.Fprintf(w, "%s%s := 0\n", indent, di)
		fmt.Fprintf(w, "%sfor i := range %s {\n", indent, s)
		fmt.Fprintf(w, "%s\tif %s[i].%s != nil {\n", indent, s, c.Access)
		fmt.Fprintf(w, "%s\t\t%s[i>>3] |= 1 << uint(i&7)\n", indent, mask)
		if conv == "" {
			fmt.Fprintf(w, "%s\t\t%s[%s] = *%s[i].%s\n", indent, col, di, s, c.Access)
		} else {
			fmt.Fprintf(w, "%s\t\t%s[%s] = %s(*%s[i].%s)\n", indent, col, di, conv, s, c.Access)
		}
		fmt.Fprintf(w, "%s\t\t%s++\n%s\t}\n%s}\n", indent, di, indent, indent)
		fmt.Fprintf(w, "%se.WriteColNullMask(%s)\n", indent, mask)
		fmt.Fprintf(w, "%sif err := e.Write%sColumn(%s[:%s]); err != nil { return err }\n", indent, c.ColAPI, col, di)
	case "String":
		col := g.fresh("c")
		fmt.Fprintf(w, "%s%s := e.ScratchString(len(%s))\n", indent, col, s)
		fmt.Fprintf(w, "%s%s := 0\n", indent, di)
		fmt.Fprintf(w, "%sfor i := range %s {\n", indent, s)
		fmt.Fprintf(w, "%s\tif %s[i].%s != nil {\n", indent, s, c.Access)
		fmt.Fprintf(w, "%s\t\t%s[i>>3] |= 1 << uint(i&7)\n", indent, mask)
		fmt.Fprintf(w, "%s\t\t%s[%s] = string(*%s[i].%s)\n", indent, col, di, s, c.Access)
		fmt.Fprintf(w, "%s\t\t%s++\n%s\t}\n%s}\n", indent, di, indent, indent)
		fmt.Fprintf(w, "%se.WriteColNullMask(%s)\n", indent, mask)
		fmt.Fprintf(w, "%se.WriteStringColumn(%s[:%s])\n", indent, col, di)
	case "Bytes":
		// []byte presence = (field != nil); present values travel as a string
		// column over a zero-copy byte view. nil stays absent (decodes to nil),
		// empty []byte{} is present (decodes to a non-nil empty slice).
		g.imports["unsafe"] = ""
		col := g.fresh("c")
		fmt.Fprintf(w, "%s%s := e.ScratchString(len(%s))\n", indent, col, s)
		fmt.Fprintf(w, "%s%s := 0\n", indent, di)
		fmt.Fprintf(w, "%sfor i := range %s {\n", indent, s)
		fmt.Fprintf(w, "%s\tif %s[i].%s != nil {\n", indent, s, c.Access)
		fmt.Fprintf(w, "%s\t\t%s[i>>3] |= 1 << uint(i&7)\n", indent, mask)
		fmt.Fprintf(w, "%s\t\t%s[%s] = unsafe.String(unsafe.SliceData(%s[i].%s), len(%s[i].%s))\n", indent, col, di, s, c.Access, s, c.Access)
		fmt.Fprintf(w, "%s\t\t%s++\n%s\t}\n%s}\n", indent, di, indent, indent)
		fmt.Fprintf(w, "%se.WriteColNullMask(%s)\n", indent, mask)
		fmt.Fprintf(w, "%se.WriteStringColumn(%s[:%s])\n", indent, col, di)
	}
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
	// Symmetric to emitEncodeNamed: a named type with its own UnmarshalQDF routes
	// through DecodeNested instead of a structural read that bypasses the codec.
	if hasMethod(n, "UnmarshalQDF") {
		fmt.Fprintf(w, "%sif err := qdf.DecodeNested(d, &%s); err != nil {\n%s\treturn err\n%s}\n", indent, lhs, indent, indent)
		return nil
	}
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
		if !g.canDecodeNested(n) {
			return g.nestedStructErr(n, "decode")
		}
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
			if !g.canDecodeNested(named) {
				return g.nestedStructErr(named, "decode")
			}
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
		if plan.hybrid() {
			fmt.Fprintf(w, "%s\t} else if d.PeekHybridColStruct() {\n", indent)
			if err := g.emitDecodeHybridBody(w, lhs, elem, plan, indent+"\t\t"); err != nil {
				return err
			}
		} else {
			fmt.Fprintf(w, "%s\t} else if d.PeekColStruct() {\n", indent)
			if err := g.emitDecodeColumnarBody(w, lhs, elem, plan, indent+"\t\t"); err != nil {
				return err
			}
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
	g.imports["unsafe"] = ""
	fmt.Fprintf(w, "%sif err := qdf.CheckColumnarBytes(%s, unsafe.Sizeof(*new(%s))); err != nil {\n%s\treturn err\n%s}\n",
		indent, nVar, g.typeExprFromType(elem), indent, indent)
	fmt.Fprintf(w, "%s%s = make([]%s, %s)\n", indent, lhs, g.typeExprFromType(elem), nVar)
	fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, idxVar, namesVar)
	fmt.Fprintf(w, "%s\tswitch %s[%s] {\n", indent, namesVar, idxVar)
	for _, c := range plan.Columns {
		fmt.Fprintf(w, "%s\tcase %s:\n", indent, strconv.Quote(c.WireName))
		// Reject a wire column whose name matches but whose kind differs from the
		// generated field's — mirrors decodeColumnar's `sh.kinds[c] != col.kind`
		// guard so a mismatched/corrupt frame fails cleanly instead of decoding
		// the wrong column reader over the body.
		fmt.Fprintf(w, "%s\t\tif %s[%s] != 0x%02x {\n%s\t\t\treturn qdf.ErrTypeMismatch\n%s\t\t}\n",
			indent, kindsVar, idxVar, c.KindByte, indent, indent)
		g.emitDecodeColumnScatter(w, lhs, nVar, c, indent+"\t\t")
	}
	fmt.Fprintf(w, "%s\tdefault:\n%s\t\treturn qdf.ErrTypeMismatch\n", indent, indent)
	fmt.Fprintf(w, "%s\t}\n", indent) // switch
	fmt.Fprintf(w, "%s}\n", indent)   // for
	// Clear the per-column length bound set by ReadColStructHeader so a sibling
	// slice/map/column decoded afterward on this (shared, threaded) decoder is
	// not wrongly bounded by n — mirrors decodeColumnar's deferred reset and the
	// hybrid path's ClearColMaxLen.
	fmt.Fprintf(w, "%sd.ClearColMaxLen()\n", indent)
	return nil
}

// emitDecodeColumnScatter emits a single column's read + scatter into lhs[i].
// Used by the pure name-switch decode (inside a case) and the hybrid positional
// decode. lhs is the output slice; nVar the row count local.
func (g *gen) emitDecodeColumnScatter(w io.Writer, lhs, nVar string, c colColumn, indent string) {
	if c.Nullable {
		g.emitDecodeNullableColumn(w, lhs, nVar, c, indent)
		return
	}
	colv := g.fresh("col")
	switch c.ColAPI {
	case "String":
		fmt.Fprintf(w, "%s%s, err := d.ReadStringColumn(%s)\n", indent, colv, nVar)
		fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
		fmt.Fprintf(w, "%sfor i := range %s { %s[i].%s = %s(%s[i]) }\n", indent, colv, lhs, c.Access, c.GoType, colv)
	case "Bytes":
		fmt.Fprintf(w, "%s%s, err := d.ReadStringColumn(%s)\n", indent, colv, nVar)
		fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
		fmt.Fprintf(w, "%sfor i := range %s { %s[i].%s = %s([]byte(%s[i])) }\n", indent, colv, lhs, c.Access, c.GoType, colv)
	case "Time":
		g.imports["time"] = ""
		secv := g.fresh("sec")
		nsv := g.fresh("ns")
		fmt.Fprintf(w, "%s%s, %s, err := d.ReadTimeColumn(%s)\n", indent, secv, nsv, nVar)
		fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
		fmt.Fprintf(w, "%sfor i := range %s { %s[i].%s = time.Unix(%s[i], int64(%s[i])).UTC() }\n", indent, secv, lhs, c.Access, secv, nsv)
	case "Bool":
		fmt.Fprintf(w, "%s%s, err := d.ReadBoolColumn(%s)\n", indent, colv, nVar)
		fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
		fmt.Fprintf(w, "%sfor i := range %s { %s[i].%s = %s[i] }\n", indent, colv, lhs, c.Access, colv)
	default:
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
		}
		fmt.Fprintf(w, "%s%s, err := d.%s(%s)\n", indent, colv, readFn, nVar)
		fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
		fmt.Fprintf(w, "%sfor i := range %s { %s[i].%s = %s(%s[i]) }\n", indent, colv, lhs, c.Access, c.GoType, colv)
	}
}

// emitDecodeNullableColumn emits decode for a *T nullable column: read the
// presence bitmap + dense column of present values, then scatter — a set bit
// allocates a fresh element and points the field at it, a clear bit leaves nil.
func (g *gen) emitDecodeNullableColumn(w io.Writer, lhs, nVar string, c colColumn, indent string) {
	mask := g.fresh("mask")
	present := g.fresh("present")
	colv := g.fresh("col")
	di := g.fresh("di")
	v := g.fresh("v")
	fmt.Fprintf(w, "%s%s, %s, err := d.ReadColNullMask(%s)\n", indent, mask, present, nVar)
	fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
	readFn := map[string]string{
		"Int": "ReadIntColumn", "Uint": "ReadUintColumn", "Float64": "ReadFloat64Column",
		"Float32": "ReadFloat32Column", "Bool": "ReadBoolColumn", "String": "ReadStringColumn",
		"Bytes": "ReadStringColumn",
	}[c.ColAPI]
	fmt.Fprintf(w, "%s%s, err := d.%s(%s)\n", indent, colv, readFn, present)
	fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
	fmt.Fprintf(w, "%s%s := 0\n", indent, di)
	fmt.Fprintf(w, "%sfor i := range %s {\n", indent, lhs)
	fmt.Fprintf(w, "%s\tif %s[i>>3]&(1<<uint(i&7)) != 0 {\n", indent, mask)
	switch c.ColAPI {
	case "Bytes":
		// present []byte: a fresh owned copy (incl. a non-nil empty slice for "").
		fmt.Fprintf(w, "%s\t\t%s[i].%s = %s([]byte(%s[%s]))\n", indent, lhs, c.Access, c.GoType, colv, di)
	case "Bool":
		fmt.Fprintf(w, "%s\t\t%s := %s[%s]\n", indent, v, colv, di)
		fmt.Fprintf(w, "%s\t\t%s[i].%s = &%s\n", indent, lhs, c.Access, v)
	default:
		fmt.Fprintf(w, "%s\t\t%s := %s(%s[%s])\n", indent, v, c.GoType, colv, di)
		fmt.Fprintf(w, "%s\t\t%s[i].%s = &%s\n", indent, lhs, c.Access, v)
	}
	fmt.Fprintf(w, "%s\t\t%s++\n", indent, di)
	fmt.Fprintf(w, "%s\t} else {\n%s\t\t%s[i].%s = nil\n%s\t}\n%s}\n", indent, indent, lhs, c.Access, indent, indent)
}

// emitDecodeHybridBody emits decode for the tagHybridColStruct frame: read the
// header, allocate the slice, decode the eligible columns positionally (wire
// order == declaration order == plan.Columns order), then the residual block
// row-major per element. Mirrors decodeHybridColumnar.
func (g *gen) emitDecodeHybridBody(w io.Writer, lhs string, elem types.Type, plan colElemPlan, indent string) error {
	nVar := g.fresh("n")
	namesVar := g.fresh("names")
	kindsVar := g.fresh("kinds")
	fmt.Fprintf(w, "%s%s, %s, %s, err := d.ReadHybridColStructHeader()\n", indent, nVar, namesVar, kindsVar)
	fmt.Fprintf(w, "%sif err != nil {\n%s\treturn err\n%s}\n", indent, indent, indent)
	// The hybrid body is decoded POSITIONALLY (wire order == declaration order),
	// so validate the wire shape matches this type's generated layout exactly
	// before scattering. A frame from a different/evolved schema (added, removed,
	// reordered, or kind-changed columns) fails cleanly with ErrTypeMismatch
	// instead of mis-scattering silently. Schema evolution is the reflect path's
	// job; the generated decoder requires the exact schema it was generated for.
	id := sanitizeIdent(plan.ElemType)
	nv := g.colNamesVar("qdfHybNames_"+id, plan.HybridNames)
	kv := g.colKindsVar("qdfHybKinds_"+id, plan.HybridKinds)
	g.imports["slices"] = ""
	fmt.Fprintf(w, "%sif !slices.Equal(%s, %s) || !slices.Equal(%s, %s) {\n%s\treturn qdf.ErrTypeMismatch\n%s}\n",
		indent, namesVar, nv, kindsVar, kv, indent, indent)
	g.imports["unsafe"] = ""
	fmt.Fprintf(w, "%sif err := qdf.CheckColumnarBytes(%s, unsafe.Sizeof(*new(%s))); err != nil {\n%s\treturn err\n%s}\n",
		indent, nVar, g.typeExprFromType(elem), indent, indent)
	fmt.Fprintf(w, "%s%s = make([]%s, %s)\n", indent, lhs, g.typeExprFromType(elem), nVar)
	for _, c := range plan.Columns {
		g.emitDecodeColumnScatter(w, lhs, nVar, c, indent)
	}
	// Residual block: clear the per-column length bound, then row-major decode
	// each residual field per element.
	fmt.Fprintf(w, "%sd.ClearColMaxLen()\n", indent)
	ri := g.fresh("i")
	fmt.Fprintf(w, "%sfor %s := range %s {\n", indent, ri, lhs)
	for _, rf := range plan.Residual {
		if err := g.emitDecodeValue(w, lhs+"["+ri+"]."+rf.Access, rf.Type, indent+"\t"); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "%s}\n", indent)
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

// fresh returns a unique identifier starting with the given prefix. Hot (one
// call per emitted temporary), so it concatenates directly instead of going
// through fmt's format-string parsing + reflection.
func (g *gen) fresh(prefix string) string {
	g.uniqCounter++
	return prefix + strconv.Itoa(g.uniqCounter)
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
