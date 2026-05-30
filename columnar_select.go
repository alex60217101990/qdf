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
	return unmarshal(data, out, fields)
}

// sliceContains reports whether ss contains s.
func sliceContains(ss []string, s string) bool { return slices.Contains(ss, s) }

// wantField reports whether name is in the active selectFields filter. A nil
// filter wants every column (no filtering).
func (d *Decoder) wantField(name string) bool {
	if d.selectFields == nil {
		return true
	}
	return slices.Contains(d.selectFields, name)
}
