// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/lens/host-go/config/model"
)

// AddSchema is an action that will add the given GQL schema to the Defra nodes.
type List struct {
	stateful

	Expected map[string]model.Lens
}

var _ Action = (*List)(nil)
var _ Stateful = (*List)(nil)

func (a *List) Execute() {
	result, err := a.s.Store.List(a.s.Ctx)
	require.NoError(a.s.T, err)

	expected := a.Expected
	if expected == nil {
		expected = map[string]model.Lens{}
	}

	// Convert the result keys to strings for greater test readibilty - defining tests
	// with `cid.Cid`s is cumbersome, and the error on failure is not as nice.
	resultStringified := make(map[string]model.Lens, len(result))
	for cid, lens := range result {
		resultStringified[cid.String()] = lens
	}

	require.Equal(a.s.T, expected, resultStringified)
}
