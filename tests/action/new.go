// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	sourceP2P "github.com/sourcenetwork/go-p2p"
	"github.com/sourcenetwork/lens/host-go/node"
	"github.com/sourcenetwork/lens/tests/state"
	"github.com/stretchr/testify/require"
)

type NewNode struct {
	stateful

	Options []node.Option
}

var _ Action = (*NewNode)(nil)
var _ Stateful = (*NewNode)(nil)

func New() *NewNode {
	return &NewNode{}
}

func (a *NewNode) Execute() {
	path := a.s.T.TempDir()

	n, err := node.New(
		a.s.Ctx,
		append(
			[]node.Option{
				// Add the defaults before the options declared by the test to make sure
				// they are overwritten if need be.
				node.WithPath(path),
				node.WithPoolSize(1),
				node.WithP2Poptions(
					sourceP2P.WithListenAddresses("/ip4/127.0.0.1/tcp/0"),
				),
			},
			a.Options...,
		)...,
	)
	if err != nil {
		require.NoError(a.s.T, err)
	}

	a.s.Nodes = append(
		a.s.Nodes,
		&state.NodeInfo{
			Node:  n,
			Store: n.Store,
			Path:  path,
		},
	)
}
