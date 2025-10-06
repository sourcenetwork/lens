// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/stretchr/testify/require"
)

type Connect struct {
	stateful

	FromNodeIndex int
	ToNodeIndex   int
}

var _ Action = (*Connect)(nil)
var _ Stateful = (*Connect)(nil)

func (a *Connect) Execute() {
	toP2PAddresses, err := a.s.Nodes[a.ToNodeIndex].Node.P2P.Value().Host.Addresses()
	require.NoError(a.s.T, err)

	err = a.s.Nodes[a.FromNodeIndex].Node.P2P.Value().Host.Connect(a.s.Ctx, toP2PAddresses)
	require.NoError(a.s.T, err)
}
