package gen

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestImportAliasQualifier verifies the qualifier is the package's real name,
// not filepath.Base(path): those differ for versioned/renamed modules, and a
// base like "yaml.v2" or "go-bar" is not a valid identifier — emitting it
// produced code that would not compile.
func TestImportAliasQualifier(t *testing.T) {
	g := newGen(&packages.Package{PkgPath: "example.com/target", Name: "target"})

	cases := []struct {
		path, name, wantQual, wantAlias string
	}{
		{"time", "time", "time", ""},              // base == name: unaliased
		{"gopkg.in/yaml.v2", "yaml", "yaml", ""},  // base "yaml.v2" != name "yaml"
		{"github.com/x/go-bar", "bar", "bar", ""}, // base "go-bar" not an ident
		{"net/url", "url", "url", ""},             // base == name
	}
	for _, c := range cases {
		if got := g.importAlias(c.path, c.name); got != c.wantQual {
			t.Errorf("importAlias(%q,%q) qualifier = %q, want %q", c.path, c.name, got, c.wantQual)
		}
		if got := g.imports[c.path]; got != c.wantAlias {
			t.Errorf("importAlias(%q,%q) alias directive = %q, want %q", c.path, c.name, got, c.wantAlias)
		}
	}
	// Idempotent: a second call returns the same qualifier, no re-register.
	if got := g.importAlias("gopkg.in/yaml.v2", "yaml"); got != "yaml" {
		t.Errorf("re-call qualifier = %q, want yaml", got)
	}
}

// TestImportAliasCollision verifies two distinct paths sharing a package name
// get disambiguated: the first binds the bare name, the second an alias.
func TestImportAliasCollision(t *testing.T) {
	g := newGen(&packages.Package{PkgPath: "example.com/target", Name: "target"})

	if q := g.importAlias("example.com/a/utils", "utils"); q != "utils" {
		t.Fatalf("first utils qualifier = %q, want utils", q)
	}
	if a := g.imports["example.com/a/utils"]; a != "" {
		t.Fatalf("first utils alias = %q, want empty", a)
	}
	q2 := g.importAlias("example.com/b/utils", "utils")
	if q2 != "utils2" {
		t.Fatalf("second utils qualifier = %q, want utils2", q2)
	}
	if a := g.imports["example.com/b/utils"]; a != "utils2" {
		t.Fatalf("second utils alias directive = %q, want utils2", a)
	}
	// The reserved "qdf" qualifier forces a same-named user package to alias.
	if q := g.importAlias("example.com/user/qdf", "qdf"); q != "qdf2" {
		t.Fatalf("user qdf qualifier = %q, want qdf2", q)
	}
}
