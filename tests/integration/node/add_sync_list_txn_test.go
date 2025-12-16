// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"testing"

	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/integration"
	"github.com/sourcenetwork/lens/tests/modules"
)

func TestAddSyncList_SyncMultiple_TxnCommit(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.NewNode{},
			&action.NewNode{},
			&action.Connect{
				FromNodeIndex: 1,
				ToNodeIndex:   0,
			},
			&action.Add{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(0),
				},
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath1_Release,
						},
					},
				},
			},
			&action.Add{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(0),
				},
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath1,
						},
					},
				},
			},
			&action.TxnCreate{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
			},
			&action.TxnAction[*action.Sync]{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
				Action: &action.Sync{
					LensID: "{{.LensIDs0}}",
				},
			},
			&action.TxnAction[*action.Sync]{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
				Action: &action.Sync{
					LensID: "{{.LensIDs1}}",
				},
			},
			&action.TxnCommit{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
			},
			&action.List{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
				Expected: map[string]model.Lens{
					"{{.LensIDs0}}": {
						Lenses: []model.LensModule{
							{
								Path: "{{.WasmBytes0}}",
							},
						},
					},
					"{{.LensIDs1}}": {
						Lenses: []model.LensModule{
							{
								Path: "{{.WasmBytes1}}",
							},
						},
					},
				},
			},
		},
	}

	test.Execute(t)
}

func TestAddSyncList_SyncMultiple_TxnDiscard(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.NewNode{},
			&action.NewNode{},
			&action.Connect{
				FromNodeIndex: 1,
				ToNodeIndex:   0,
			},
			&action.Add{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(0),
				},
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath1_Release,
						},
					},
				},
			},
			&action.Add{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(0),
				},
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath1,
						},
					},
				},
			},
			&action.TxnCreate{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
			},
			&action.TxnAction[*action.Sync]{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
				Action: &action.Sync{
					LensID: "{{.LensIDs0}}",
				},
			},
			&action.TxnAction[*action.Sync]{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
				Action: &action.Sync{
					LensID: "{{.LensIDs1}}",
				},
			},
			&action.TxnDiscard{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
			},
			&action.List{
				Nodeful: action.Nodeful{
					NodeIndex: immutable.Some(1),
				},
				Expected: map[string]model.Lens{},
			},
		},
	}

	test.Execute(t)
}
