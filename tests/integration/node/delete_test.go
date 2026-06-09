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

// TestDeleteUnknownIsIdempotent asserts that deleting a syntactically valid but
// unknown lens id returns no error and leaves the store empty.
func TestDeleteUnknownIsIdempotent(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Delete{
				LensID: "bafyreihrdqmhlej6xidpkbwe6ltn2cw34oqvdji4sdh7xbrjciggxzk3ue",
			},
			&action.List{
				Expected: map[string]model.Lens{},
			},
		},
	}

	test.Execute(t)
}

// TestDeleteAlreadyRemovedIsIdempotent asserts that deleting the same lens twice
// does not error on the second, already-removed, call.
func TestDeleteAlreadyRemovedIsIdempotent(t *testing.T) {
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
			&action.Delete{
				LensID: "{{.LensIDs0}}",
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

// TestDelete_TxnDiscard asserts that a Delete staged in a transaction does not take
// effect if the transaction is discarded - the tombstone is transaction-scoped and
// never reaches the shared index or repository pools.
func TestDelete_TxnDiscard(t *testing.T) {
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
			&action.TxnCreate{},
			&action.TxnAction[*action.Delete]{
				Action: &action.Delete{
					LensID: "{{.LensIDs0}}",
				},
			},
			&action.TxnDiscard{},
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
		},
	}

	test.Execute(t)
}

// TestDelete_TxnCommit asserts that a Delete staged in a transaction takes effect once
// the transaction is committed.
func TestDelete_TxnCommit(t *testing.T) {
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
			&action.TxnCreate{},
			&action.TxnAction[*action.Delete]{
				Action: &action.Delete{
					LensID: "{{.LensIDs0}}",
				},
			},
			&action.TxnCommit{},
			&action.List{
				Expected: map[string]model.Lens{},
			},
		},
	}

	test.Execute(t)
}

// TestDeleteOneOfMany asserts that deleting one lens leaves the others intact.
func TestDeleteOneOfMany(t *testing.T) {
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
			&action.Add{
				Config: model.Lens{
					Lenses: []model.LensModule{
						{
							Path: modules.WasmPath2,
						},
					},
				},
			},
			&action.Delete{
				LensID: "{{.LensIDs0}}",
			},
			&action.List{
				Expected: map[string]model.Lens{
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
