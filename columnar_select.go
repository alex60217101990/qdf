package qdf

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
