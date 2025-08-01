module github.com/sourcenetwork/lens/host-go

go 1.23

require (
	github.com/bytecodealliance/wasmtime-go/v35 v35.0.0
	github.com/sourcenetwork/immutable v0.3.0
	github.com/sourcenetwork/lens/tests/modules v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
	github.com/tetratelabs/wazero v1.9.0
	github.com/wasmerio/wasmer-go v1.0.4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/sourcenetwork/lens/tests/modules => ../tests/modules
