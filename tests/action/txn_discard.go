// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

type TxnDiscard struct {
	Nodeful

	TxnIndex int
}

var _ Action = (*TxnDiscard)(nil)

func DiscardTxn() *TxnDiscard {
	return &TxnDiscard{}
}

func (a *TxnDiscard) Execute() {
	for _, n := range a.Nodes() {
		txn := n.Txns[a.TxnIndex]

		txn.Discard()
	}
}
