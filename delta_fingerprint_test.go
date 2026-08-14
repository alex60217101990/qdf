package qdf

import "testing"

type fpMapRow struct {
	M map[string][]string `qdf:"m"`
}

// The base fingerprint exists to REJECT a patch built from a different base, so
// a collision there is not a lost optimisation — it is a patch applied to a
// value it was not built from, silently.
//
// fpHashReflect used to write strings with no length and slices with no length
// and no nil marker, so {"ab": {}} and {"a": {"b"}} both reduced to the bytes
// 'a','b'. A patch diffed from the second applied cleanly to the first and
// turned it into map[a:[ c] ab:[]] with a nil error.
func TestBaseFingerprintSeparatesRunTogetherValues(t *testing.T) {
	for _, c := range []struct {
		name           string
		base, old, new fpMapRow
	}{
		{
			name: "key_boundary",
			base: fpMapRow{M: map[string][]string{"ab": {}}},
			old:  fpMapRow{M: map[string][]string{"a": {"b"}}},
			new:  fpMapRow{M: map[string][]string{"a": {"b", "c"}}},
		},
		{
			name: "element_boundary",
			base: fpMapRow{M: map[string][]string{"k": {"ab"}}},
			old:  fpMapRow{M: map[string][]string{"k": {"a", "b"}}},
			new:  fpMapRow{M: map[string][]string{"k": {"a", "b", "c"}}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			patch, err := Diff(c.old, c.new, OptBalanced)
			if err != nil {
				t.Fatalf("diff: %v", err)
			}
			target := c.base
			if err := Apply(&target, patch); err == nil {
				t.Fatalf("a patch built from %+v applied cleanly to %+v and left it %+v — "+
					"the fingerprint did not separate them", c.old.M, c.base.M, target.M)
			}
		})
	}
}

// And a patch built from the RIGHT base must still apply, or the fix above has
// simply broken the feature.
func TestBaseFingerprintStillAcceptsItsOwnBase(t *testing.T) {
	old := fpMapRow{M: map[string][]string{"a": {"b"}}}
	newv := fpMapRow{M: map[string][]string{"a": {"b", "c"}}}
	patch, err := Diff(old, newv, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	target := old
	if err := Apply(&target, patch); err != nil {
		t.Fatalf("a patch rejected its own base: %v", err)
	}
	if len(target.M["a"]) != 2 || target.M["a"][1] != "c" {
		t.Fatalf("applied but wrong: %+v", target.M)
	}
}
