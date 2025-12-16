// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/stretchr/testify/require"
)

type Sync struct {
	Nodeful

	LensID string
}

var _ Action = (*Sync)(nil)
var _ Stateful = (*Sync)(nil)

func (a *Sync) Execute() {
	lensID := replace(a.s, a.LensID)

	for _, n := range a.Nodes() {
		err := n.P2P.Value().SyncLens(a.s.Ctx, lensID)
		require.NoError(a.s.T, err)
	}
}
