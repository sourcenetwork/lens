// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package integration

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/modules"
)

func TestSimple(t *testing.T) {
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
		},
	)
}

func TestSimpleWithNoModules(t *testing.T) {
	type Input struct {
		Name string
		Age  int
	}

	executeTest(
		t,
		TestCase[Input, Input]{
			LensFile: `
			{
				"lenses": []
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
			ExpectedOutput: []Input{
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
		},
	)
}

func TestSimpleWithEmptyItem(t *testing.T) {
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
					}
				]
			}`,
			Input: []Input{
				{
					Name: "John",
					Age:  3,
				},
				{},
				{
					Name: "Orpheus",
					Age:  7,
				},
			},
			ExpectedOutput: []Output{
				{
					FullName: "John",
					Age:      3,
				},
				{},
				{
					FullName: "Orpheus",
					Age:      7,
				},
			},
		},
	)
}

func TestSimpleWithNilItem(t *testing.T) {
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
		TestCase[*Input, *Output]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath1 + `"
					}
				]
			}`,
			Input: []*Input{
				{
					Name: "John",
					Age:  3,
				},
				nil,
				{
					Name: "Orpheus",
					Age:  7,
				},
			},
			ExpectedOutput: []*Output{
				{
					FullName: "John",
					Age:      3,
				},
				nil,
				{
					FullName: "Orpheus",
					Age:      7,
				},
			},
		},
	)
}
