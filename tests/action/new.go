// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	badgerds "github.com/dgraph-io/badger/v4"
	"github.com/sourcenetwork/corekv/badger"
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
	a.s.T.Fatalf("Store type multiplier not specified")
}

type NewBadgerFileNode struct {
	stateful

	Options []node.Option
}

var _ Action = (*NewBadgerFileNode)(nil)
var _ Stateful = (*NewBadgerFileNode)(nil)

func (a *NewBadgerFileNode) Execute() {
	path := a.s.T.TempDir()

	rootstore, err := badger.NewDatastore(path, badgerds.DefaultOptions(path))
	require.NoError(a.s.T, err)

	n, err := node.New(
		a.s.Ctx,
		append(
			[]node.Option{
				// Add the defaults before the options declared by the test to make sure
				// they are overwritten if need be.
				node.WithPath(path),
				node.WithPoolSize(1),
				node.WithRootstore(rootstore),
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

const (
	// 1 MB, this matches the maximum badger-in-memory value size.
	//
	// Nearly at least, badger panics if this is set to it's max for reasons not yet
	// looked into.  Going one byte smaller does not have this issue.
	defaultchunksize = (1 << 20) - 1
)

type NewBadgerMemoryNode struct {
	stateful

	Options []node.Option
}

var _ Action = (*NewBadgerMemoryNode)(nil)
var _ Stateful = (*NewBadgerMemoryNode)(nil)

func (a *NewBadgerMemoryNode) Execute() {
	rootstore, err := badger.NewDatastore("", badgerds.DefaultOptions("").WithInMemory(true))
	require.NoError(a.s.T, err)

	n, err := node.New(
		a.s.Ctx,
		append(
			[]node.Option{
				// Add the defaults before the options declared by the test to make sure
				// they are overwritten if need be.
				node.WithPoolSize(1),
				node.WithRootstore(rootstore),
				// Badger in-memory only allows values of size 1MB or smaller so we must chunk the
				// blockstore.
				//
				// Comment this option out and run the tests to view/verify the problem.
				node.WithBlockstoreChunkSize(defaultchunksize),
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
		},
	)
}
