// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"net/url"
	"os"
	"testing"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/integration"
	"github.com/sourcenetwork/lens/tests/modules"
	"github.com/stretchr/testify/require"
)

func TestAddList(t *testing.T) {
	parsed, err := url.Parse(modules.WasmPath1)
	require.NoError(t, err)

	wasm, err := os.ReadFile(parsed.Path)
	require.NoError(t, err)
	_ = wasm
	test := &integration.Test{
		Actions: []action.Action{
			&action.Add{
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath1, //"data://" + string(wasm), // todo - add substitution magic
						},
					},
				},
			},
			&action.List{
				Expected: map[string]model.Lens{
					"bafyreibzldjobcopchktwpnchjmk7bqoka6s6exyjp6ys62hnfv7mzik4e": {
						Lenses: []model.LensModule{
							{
								Path: "data://" + string(wasm), // todo - add substitution magic
							},
						},
					},
				},
			},
		},
	}

	test.Execute(t)
}
