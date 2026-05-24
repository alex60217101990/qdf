// Command qdfgen generates MarshalQDF / UnmarshalQDF methods for user
// struct types so the qdf package can encode/decode them without any
// runtime reflection.
//
// Typical usage from a //go:generate directive:
//
//	//go:generate qdfgen -type Foo,Bar
//
// or invoked explicitly against a package import path:
//
//	qdfgen -type Foo,Bar ./pkg/path
//
// The output file is written next to the source as <basename>_qdf.go by
// default and is gofmt-formatted. See README.md for details.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alex60217101990/qdf/cmd/qdfgen/gen"
)

func main() {
	var (
		typesCSV string
		outFile  string
		outDir   string
		verbose  bool
	)
	flag.StringVar(&typesCSV, "type", "", "comma-separated list of struct type names to generate methods for (required)")
	flag.StringVar(&outFile, "output", "", "output file name (default: <pkgdir>/<pkgname>_qdf.go)")
	flag.StringVar(&outDir, "outdir", "", "output directory (default: source package dir)")
	flag.BoolVar(&verbose, "v", false, "verbose log to stderr")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `qdfgen — code generator for qdf MarshalQDF/UnmarshalQDF methods

Usage:
  qdfgen -type Foo,Bar [-output file.go] [package]

If [package] is omitted, the current directory is used. The package
argument may be any pattern understood by 'go list', e.g. "./mypkg".
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if typesCSV == "" {
		fmt.Fprintln(os.Stderr, "qdfgen: -type is required")
		flag.Usage()
		os.Exit(2)
	}
	types := splitCSV(typesCSV)
	if len(types) == 0 {
		fmt.Fprintln(os.Stderr, "qdfgen: -type produced no names")
		os.Exit(2)
	}

	pkgPattern := "."
	if flag.NArg() > 0 {
		pkgPattern = flag.Arg(0)
	}

	opts := gen.Options{
		Types:   types,
		OutFile: outFile,
		OutDir:  outDir,
		Verbose: verbose,
		LogTo:   os.Stderr,
	}
	if err := gen.Generate([]string{pkgPattern}, opts); err != nil {
		fmt.Fprintf(os.Stderr, "qdfgen: %v\n", err)
		os.Exit(1)
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
