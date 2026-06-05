module github.com/alex60217101990/qdf/internal/codegen_test

go 1.26

require (
	github.com/alex60217101990/qdf v0.0.0
	github.com/alex60217101990/qdf/cmd/qdfgen v0.0.0
)

require (
	github.com/modern-go/reflect2 v1.0.2 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
)

replace (
	github.com/alex60217101990/qdf => ../../
	github.com/alex60217101990/qdf/cmd/qdfgen => ../../cmd/qdfgen
)
