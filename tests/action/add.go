// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/stretchr/testify/require"
)

// AddSchema is an action that will add the given GQL schema to the Defra nodes.
type Add struct {
	stateful

	Config model.Lens

	// NodeID may hold the ID (index) of a node to apply this update to.
	//
	// If a value is not provided the update will be applied to all nodes.
	NodeID immutable.Option[int]
}

var _ Action = (*Add)(nil)
var _ Stateful = (*Add)(nil)

func (a *Add) Execute() {
	_, err := a.s.Node.Add(a.s.Ctx, a.Config) // todo - add cid to state
	require.NoError(a.s.T, err)
}
