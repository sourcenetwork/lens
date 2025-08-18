// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"testing"

	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/integration"
)

func TestList(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.List{},
		},
	}

	test.Execute(t)
}
