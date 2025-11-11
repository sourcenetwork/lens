// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"testing"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/node"
	"github.com/sourcenetwork/lens/host-go/store"
	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/integration"
	"github.com/sourcenetwork/lens/tests/modules"
)

func TestAddTransformReset_ClearsLensState(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.NewNode{
				Options: []node.Option{
					// Create the node with pool size 1, to guarantee instance re-use
					node.WithPoolSize(1),
				},
			},
			&action.Add{
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath5,
						},
					},
				},
			},
			&action.Transform{
				LensID: "{{.LensIDs0}}",
				Input: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   0,
						},
						{
							"Name": "Addo",
							"Id":   0,
						},
					},
				),
				Expected: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   float64(1),
						},
						{
							"Name": "Addo",
							"Id":   float64(2),
						},
					},
				),
			},
			&action.TransformReset{
				TransformIndex: 0,
			},
			&action.Transform{
				LensID: "{{.LensIDs0}}",
				Input: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   0,
						},
						{
							"Name": "Addo",
							"Id":   0,
						},
					},
				),
				Expected: enumerable.New(
					// This is undesirable, the ids should start from `1`, but instead Lens has preserved
					// state despite the `Reset` call.
					[]store.Document{
						{
							"Name": "John",
							"Id":   float64(3),
						},
						{
							"Name": "Addo",
							"Id":   float64(4),
						},
					},
				),
			},
		},
	}

	test.Execute(t)
}
