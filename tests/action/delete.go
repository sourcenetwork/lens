// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/stretchr/testify/require"
)

type Delete struct {
	Nodeful

	LensID string
}

var _ Action = (*Delete)(nil)
var _ Stateful = (*Delete)(nil)

func (a *Delete) Execute() {
	lensID := replace(a.s, a.LensID)

	for _, n := range a.Nodes() {
		err := n.Store.Delete(a.s.Ctx, lensID)
		require.NoError(a.s.T, err)
	}
}
