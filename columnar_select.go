package qdf

import "slices"

// wantedColumns maps each WIRE column index to the target plan column to
// decode it into, or nil to skip. Matched by field name. Wire columns whose
// name is absent from the target are skipped; target fields absent from the
// wire are left zero.
func wantedColumns(plan *columnarPlan, names []string) []*colColumn {
	want := make([]*colColumn, len(names))
	for wi, wn := range names {
		for ti := range plan.cols {
			if plan.cols[ti].name == wn {
				want[wi] = &plan.cols[ti]
				break
			}
		}
	}
	return want
}

// UnmarshalColumns decodes only the named columns of a columnar []struct
// payload into out, leaving the rest undecoded. out may point at a typed
// subset slice (e.g. *[]Subset whose fields are the wanted columns, matched
// by name like Unmarshal) or at the dynamic *[]map[string]any form, where
// only the named columns are stored. When the payload carries the
// column-length index (OptColumnIndex), unrequested columns are skipped
// without decoding; otherwise they are decoded but dropped.
//
// fields names the wire columns to keep. With no fields it behaves like
// Unmarshal.
func UnmarshalColumns(data []byte, out any, fields ...string) error {
	return unmarshal(data, out, fields, false, nil)
}

// wantField reports whether name is in the active selectFields filter. A nil
// filter wants every column (no filtering).
func (d *Decoder) wantField(name string) bool {
	if len(d.selectFields) == 0 {
		return true
	}
	return slices.Contains(d.selectFields, name)
}

// keyFilter is the consumed form of an UnmarshalKeys projection. The zero
// value wants every key, so an absent filter costs one nil check.
type keyFilter struct {
	set  map[string]struct{} // built only for large lists, see takeKeyFilter
	list []string
}

// keyFilterSetMin is where linear scanning stops paying. Map entry counts are
// data-driven (an attacker picks them), so a large key list must not turn a
// projection into O(keys x entries).
const keyFilterSetMin = 16

// takeKeyFilter consumes the pending UnmarshalKeys projection. It is one-shot:
// the root map's decode loop takes it, and every nested decode — values, Skip,
// nested maps, columnar containers — then runs unfiltered. Sharing one filter
// field with the columnar column names, and clearing it per value rather than
// once per map, produced three separate silent-data-loss bugs; this shape has
// neither hazard.
func (d *Decoder) takeKeyFilter() keyFilter {
	keys := d.selectKeys
	if len(keys) == 0 {
		return keyFilter{}
	}
	d.selectKeys = nil
	f := keyFilter{list: keys}
	if len(keys) >= keyFilterSetMin {
		f.set = make(map[string]struct{}, len(keys))
		for _, k := range keys {
			f.set[k] = struct{}{}
		}
	}
	return f
}

// want reports whether name survives the filter. The zero filter wants all.
func (f keyFilter) want(name string) bool {
	if f.list == nil {
		return true
	}
	if f.set != nil {
		_, ok := f.set[name]
		return ok
	}
	return slices.Contains(f.list, name)
}

// UnmarshalKeys decodes only the named keys of a map-rooted payload into out,
// skipping every other entry's value without decoding it. out must point at a
// string-keyed map (or at an interface, for the dynamic form) — the projection
// is defined on the ROOT map only, and values decode in full, so a nested map
// inside a selected value keeps all of its keys.
//
// This is the projection path for payloads whose member names are dynamic — a
// model checkpoint keyed by tensor name, a metrics blob keyed by series —
// where a typed subset struct (which Unmarshal already projects through Skip)
// cannot be written ahead of time. Skipping is far cheaper than decoding: on a
// 64-tensor checkpoint, taking two tensors runs ~27x faster than a full decode
// under OptBalanced. Under OptRANS the gain is much smaller (~2x): the entropy
// pass has to inflate the whole message before any entry can be skipped.
//
// With no keys it behaves like Unmarshal.
func UnmarshalKeys(data []byte, out any, keys ...string) error {
	if len(keys) == 0 {
		return Unmarshal(data, out)
	}
	return unmarshalKeys(data, out, keys)
}
