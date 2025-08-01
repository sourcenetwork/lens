// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package integration

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/modules"
)

func TestWithModulesWithParams(t *testing.T) {
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
							"dst": "LastName"
						}
					},
					{
						"path": "` + modules.WasmPath4 + `",
						"arguments": {
							"src": "LastName",
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
