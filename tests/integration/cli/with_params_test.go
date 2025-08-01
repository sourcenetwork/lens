// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package integration

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/modules"
)

func TestWithParams(t *testing.T) {
	type Input struct {
		Name string
		Age  int
	}

	type Output struct {
		MiddleName string
		Age        int
	}

	executeTest(
		t,
		TestCase[Input, Output]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath4 + `",
						"arguments": {
							"src": "Name",
							"dst": "MiddleName"
						}
					}
				]
			}`,
			Input: []Input{
				{
					Name: "John",
					Age:  3,
				},
				{
					Name: "Shahzad",
					Age:  9,
				},
				{
					Name: "Pavneet",
					Age:  11,
				},
			},
			ExpectedOutput: []Output{
				{
					MiddleName: "John",
					Age:        3,
				},
				{
					MiddleName: "Shahzad",
					Age:        9,
				},
				{
					MiddleName: "Pavneet",
					Age:        11,
				},
			},
		},
	)
}

func TestWithParamsReturnsErrorGivenBadParam(t *testing.T) {
	type Input struct {
		Name string
		Age  int
	}

	type Output struct {
		MiddleName string
		Age        int
	}

	executeTest(
		t,
		TestCase[Input, Output]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath4 + `",
						"arguments": {
							"src": "NotAField",
							"dst": "MiddleName"
						}
					}
				]
			}`,
			Input: []Input{
				{
					Name: "John",
					Age:  3,
				},
				{
					Name: "Shahzad",
					Age:  9,
				},
				{
					Name: "Pavneet",
					Age:  11,
				},
			},
			ExpectedError: "The requested property was not found. Requested: NotAField",
		},
	)
}

func TestWithParamsReturnsErrorGivenNilParam(t *testing.T) {
	type Input struct {
		Name string
		Age  int
	}

	type Output struct {
		MiddleName string
		Age        int
	}

	executeTest(
		t,
		TestCase[Input, Output]{
			LensFile: `
			{
				"lenses": [
					{
						"path": "` + modules.WasmPath4 + `",
						"arguments": null
					}
				]
			}`,
			Input: []Input{
				{
					Name: "John",
					Age:  3,
				},
				{
					Name: "Shahzad",
					Age:  9,
				},
				{
					Name: "Pavneet",
					Age:  11,
				},
			},
			ExpectedError: "Parameters have not been set.",
		},
	)
}
