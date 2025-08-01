package integration

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/modules"
)

func TestWithState(t *testing.T) {
	type Value struct {
		Id   int
		Name string
	}

	executeTest(
		t,
		TestCase[Value, Value]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath5 + `"
					}
				]
			}`,
			Input: []Value{
				{
					Name: "John",
				},
				{
					Name: "Fred",
				},
				{
					Name: "Orpheus",
				},
			},
			ExpectedOutput: []Value{
				{
					Id:   1,
					Name: "John",
				},
				{
					Id:   2,
					Name: "Fred",
				},
				{
					Id:   3,
					Name: "Orpheus",
				},
			},
		},
	)
}

// As the configuration will spin up an instance per lens, the state will not be shared between
// them by default and the resultant Ids will be the same as if only a single lens was specified.
func TestWithStateDuplicated(t *testing.T) {
	type Value struct {
		Id   int
		Name string
	}

	executeTest(
		t,
		TestCase[Value, Value]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath5 + `"
					},
					{
						"path": "` + modules.WasmPath5 + `"
					},
					{
						"path": "` + modules.WasmPath5 + `"
					}
				]
			}`,
			Input: []Value{
				{
					Name: "John",
				},
				{
					Name: "Fred",
				},
				{
					Name: "Orpheus",
				},
			},
			ExpectedOutput: []Value{
				{
					Id:   1,
					Name: "John",
				},
				{
					Id:   2,
					Name: "Fred",
				},
				{
					Id:   3,
					Name: "Orpheus",
				},
			},
		},
	)
}

// todo - add tests that split a single record into many
