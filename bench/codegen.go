package bench

// Regeneration commands for the protobuf and flatbuffers competitor schemas.
//
// The generated code (pb/bench.pb.go, fbs/bench_generated.go) is committed so
// CI and `go test` need no protoc/flatc. The `codegen-drift` CI job verifies
// the committed code is up to date by regenerating and diffing; the tool
// versions it pins MUST match the ones below:
//
//	protoc        v34.1   (header reports "protoc v7.34.1")
//	protoc-gen-go v1.36.11
//	flatc         v25.12.19
//
// flatc --gen-onefile emits an empty `package` line for a namespaced schema,
// so the sed fixup restores `package benchfbs` deterministically (BSD/GNU sed
// compatible via -i.bak).
//
//go:generate protoc --go_out=. --go_opt=paths=source_relative pb/bench.proto
//go:generate sh -c "flatc --go --gen-onefile -o fbs fbs/bench.fbs && sed -i.bak 's/^package[[:space:]]*$/package benchfbs/' fbs/bench_generated.go && rm -f fbs/bench_generated.go.bak"
