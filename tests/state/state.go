// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package state

import (
	"context"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/sourcenetwork/lens/host-go/node"
	"github.com/sourcenetwork/lens/host-go/store"
)

type State struct {
	Ctx context.Context
	T   testing.TB

	Nodes []*NodeInfo

	LensIDs   []cid.Cid
	WasmBytes [][]byte
}

type NodeInfo struct {
	Node *node.Node
	Path string

	Store store.Store
	Txns  []store.Txn
}
