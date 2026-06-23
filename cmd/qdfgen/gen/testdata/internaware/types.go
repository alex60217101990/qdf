package internaware

// HybridStrElem is a string-ONLY element with a residual map field (no numeric,
// no time): a HYBRID string-only element. Its columnar gate in generated code
// must be the intern-aware StringColumnsBeneficialHybrid so it flips into the
// columnar form (and alpha-packs a restricted-alphabet ID) exactly when reflect
// does.
type HybridStrElem struct {
	Span string            `qdf:"span"`
	Tags map[string]string `qdf:"tags"`
}

// HybridStrHolder holds a slice of the hybrid element so the columnar transpose
// fires on the field.
type HybridStrHolder struct {
	Items []HybridStrElem `qdf:"items"`
}

// PureStrElem is a string-ONLY element with NO residual: a PURE string-only
// element. Its gate must remain the plain StringColumnsBeneficial.
type PureStrElem struct {
	A string `qdf:"a"`
	B string `qdf:"b"`
}

type PureStrHolder struct {
	Items []PureStrElem `qdf:"items"`
}
