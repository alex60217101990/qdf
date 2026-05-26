package qdf

// Typed map encode / decode fast paths bypass the reflect.Value
// per-element materialisation that dominates generic map encode time.
// Each pair (K, V) gets a hand-shaped encode/decode function that
// inlines the matching WriteX / ReadX calls, keeping the hot loop on
// the inlined codepath.
//
// Pair selection and the dispatch table live in
// maps_fast_generated.go. Add or remove pairs by editing the table
// in internal/mapsgen/main.go and regenerating:
//
//	go generate ./...
//
// The generator emits maps_fast_generated.go (committed). Do not edit
// that file by hand.

//go:generate go run ./internal/mapsgen
