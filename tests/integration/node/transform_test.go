// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"testing"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/store"
	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/integration"
)

func TestTransform_EmptyString_Errors(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Transform{
				LensID:        "",
				ExpectedError: "invalid cid: cid too short",
			},
		},
	}

	test.Execute(t)
}

func TestTransform_InverseEmptyString_Errors(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Transform{
				LensID:        "",
				Inverse:       true,
				ExpectedError: "invalid cid: cid too short",
			},
		},
	}

	test.Execute(t)
}

func TestTransform_InvalidString_Errors(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Transform{
				LensID:        "fjndshjbavgc",
				ExpectedError: "invalid cid: encoding",
			},
		},
	}

	test.Execute(t)
}

func TestTransform_InverseInvalidString_Errors(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Transform{
				LensID:        "fjndshjbavgc",
				Inverse:       true,
				ExpectedError: "invalid cid: encoding",
			},
		},
	}

	test.Execute(t)
}

func TestTransform_UnknownString_ReturnsInput(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Transform{
				LensID: "bafyreihmforc6mpkonqenmf7aiek4numdgusqjgdb6phfp2migopmju3fi",
				Input: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   1,
						},
						{
							"Name": "Addo",
							"Id":   2,
						},
					},
				),
				Expected: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   1,
						},
						{
							"Name": "Addo",
							"Id":   2,
						},
					},
				),
			},
		},
	}

	test.Execute(t)
}

func TestTransform_InverseUnknownString_ReturnsInput(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Transform{
				LensID:  "bafyreihmforc6mpkonqenmf7aiek4numdgusqjgdb6phfp2migopmju3fi",
				Inverse: true,
				Input: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   1,
						},
						{
							"Name": "Addo",
							"Id":   2,
						},
					},
				),
				Expected: enumerable.New(
					[]store.Document{
						{
							"Name": "John",
							"Id":   1,
						},
						{
							"Name": "Addo",
							"Id":   2,
						},
					},
				),
			},
		},
	}

	test.Execute(t)
}
