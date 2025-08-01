module github.com/sourcenetwork/lens/tests/integration

go 1.23

require (
	github.com/sourcenetwork/lens/tests/modules v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/sourcenetwork/lens/tests/modules => ../modules
