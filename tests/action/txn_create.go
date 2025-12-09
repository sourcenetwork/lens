// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import "github.com/sourcenetwork/corekv"

// TxCreate executes the `client tx create` command and appends the returned transaction id
// to state.Txns.
type TxnCreate struct {
	Nodeful

	TxnIndex int
	ReadOnly bool
}

var _ Action = (*TxnCreate)(nil)

func NewTxn() *TxnCreate {
	return &TxnCreate{}
}

func (a *TxnCreate) Execute() {
	for _, n := range a.Nodes() {
		txn := n.Source.NewTxn(a.ReadOnly)

		if a.TxnIndex >= len(n.Txns) {
			// Expand the slice if needed.
			n.Txns = append(n.Txns, make([]corekv.Txn, a.TxnIndex+1)...)
		}

		n.Txns[a.TxnIndex] = txn
	}
}
