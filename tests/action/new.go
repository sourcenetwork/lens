// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/sourcenetwork/lens/host-go/node"
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
	a.s.Path = path

	n, err := node.New(
		a.s.Ctx,
		append(
			[]node.Option{
				// Add the defaults before the options declared by the test to make sure
				// they are overwritten if need be.
				node.WithPath(a.s.Path),
				node.WithPoolSize(1),
			},
			a.Options...,
		)...,
	)
	if err != nil {
		require.NoError(a.s.T, err)
	}

	a.s.Node = n
	a.s.Store = n.Store
}
