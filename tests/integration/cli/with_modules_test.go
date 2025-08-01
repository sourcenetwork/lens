// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package integration

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/modules"
)

func TestSimpleWithModules(t *testing.T) {
	type Input struct {
		Name string
		Age  int
	}

	type Output struct {
		FullName string
		Age      int
	}

	executeTest(
		t,
		TestCase[Input, Output]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath1 + `"
					},
					{
						"path": "` + modules.WasmPath2 + `"
					}
				]
			}`,
			Input: []Input{
				{
					Name: "John",
					Age:  3,
				},
				{
					Name: "Fred",
					Age:  5,
				},
				{
					Name: "Orpheus",
					Age:  7,
				},
			},
			ExpectedOutput: []Output{
				{
					FullName: "John",
					Age:      4,
				},
				{
					FullName: "Fred",
					Age:      6,
				},
				{
					FullName: "Orpheus",
					Age:      8,
				},
			},
		},
	)
}

// This test asserts that there are no issues running the same module multiple times
func TestSimpleWithModulesRepeated(t *testing.T) {
	type Value struct {
		FullName string
		Age      int
	}

	executeTest(
		t,
		TestCase[Value, Value]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath2 + `"
					},
					{
						"path": "` + modules.WasmPath2 + `"
					},
					{
						"path": "` + modules.WasmPath2 + `"
					}
				]
			}`,
			Input: []Value{
				{
					FullName: "John",
					Age:      3,
				},
				{
					FullName: "Fred",
					Age:      5,
				},
				{
					FullName: "Orpheus",
					Age:      7,
				},
			},
			ExpectedOutput: []Value{
				{
					FullName: "John",
					Age:      6,
				},
				{
					FullName: "Fred",
					Age:      8,
				},
				{
					FullName: "Orpheus",
					Age:      10,
				},
			},
		},
	)
}
