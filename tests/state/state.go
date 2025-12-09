// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package state

import (
	"context"
	"testing"

	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/node"
	"github.com/sourcenetwork/lens/host-go/store"
)

type State struct {
	Ctx context.Context
	T   testing.TB

	Nodes []*NodeInfo

	LensIDs   []string
	WasmBytes [][]byte
}

type NodeInfo struct {
	Node *node.Node
	Path string

	Source     corekv.TxnReaderWriter
	Store      store.Store
	Txns       []corekv.Txn
	Transforms []enumerable.Enumerable[store.Document]
}
