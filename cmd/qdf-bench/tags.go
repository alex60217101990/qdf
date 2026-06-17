package main

import "strings"

// activeTags is populated by the per-build-tag files (tags_reflect2.go,
// tags_simd.go) via init(), so the binary can report which qdf build tags it was
// compiled with. A single binary cannot switch build tags at runtime — run.sh
// builds one binary per tag combination.
var activeTags []string

func buildTagLabel() string {
	if len(activeTags) == 0 {
		return "none"
	}
	return strings.Join(activeTags, "+")
}
