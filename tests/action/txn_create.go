// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/sourcenetwork/lens/host-go/store"
	"github.com/stretchr/testify/require"
)

// TxCreate executes the `client tx create` command and appends the returned transaction id
// to state.Txns.
type TxnCreate struct {
	stateful

	TxnIndex int
	ReadOnly bool
}

var _ Action = (*TxnCreate)(nil)

func NewTxn() *TxnCreate {
	return &TxnCreate{}
}

func (a *TxnCreate) Execute() {
	txn, err := a.s.Node.Store.NewTxn(a.ReadOnly)
	require.NoError(a.s.T, err)

	if a.TxnIndex >= len(a.s.Txns) {
		// Expand the slice if needed.
		a.s.Txns = append(a.s.Txns, make([]store.Txn, a.TxnIndex+1)...)
	}

	a.s.Txns[a.TxnIndex] = txn
}
