// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/sourcenetwork/lens/host-go/node"
	"github.com/stretchr/testify/require"
)

// AddSchema is an action that will add the given GQL schema to the Defra nodes.
type NewAction struct {
	stateful
}

var _ Action = (*NewAction)(nil)
var _ Stateful = (*NewAction)(nil)

func New() *NewAction {
	return &NewAction{}
}

func (a *NewAction) Execute() {
	n, err := node.New(a.s.Ctx)
	if err != nil {
		require.NoError(a.s.T, err)
	}

	a.s.Node = n
}
