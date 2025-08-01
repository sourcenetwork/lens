package integration

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/modules"
)

// Rust is very aggressive in cleaning up memory, and we had a bug where lenses would fail if the input
// was particularly large, and/or the transform copied the input too many times.
func TestWithMem(t *testing.T) {
	type Value struct {
		Name string
		Type string `json:"__type"`
		// Inline arrays seem to be particularly problematic, I guess it is due to some lifetime stuff
		// in the serde library.
		Array []string
	}

	executeTest(
		t,
		TestCase[Value, Value]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath_Memory + `"
					}
				]
			}`,
			Input: []Value{
				{
					Name:  "John",
					Type:  "passsssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssss",
					Array: []string{},
				},
			},
			ExpectedOutput: []Value{
				{
					Name:  "John",
					Type:  "passsssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssss",
					Array: []string{},
				},
			},
		},
	)
}
