// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/integration"
	"github.com/sourcenetwork/lens/tests/modules"
)

func TestDelete(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Add{
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath1,
						},
					},
				},
			},
			&action.List{
				Expected: map[string]model.Lens{
					"{{.LensIDs0}}": {
						Lenses: []model.LensModule{
							{
								Path: "{{.WasmBytes0}}",
							},
						},
					},
				},
			},
			&action.Delete{
				LensID: "{{.LensIDs0}}",
			},
			&action.List{
				Expected: map[string]model.Lens{},
			},
		},
	}

	test.Execute(t)
}
