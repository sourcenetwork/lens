// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import "github.com/stretchr/testify/require"

// TxnCommit executes the `client txn commit` command using the txn id at the given
// state-index.
type TxnCommit struct {
	Nodeful

	TxnIndex int
}

var _ Action = (*TxnCommit)(nil)

func (a *TxnCommit) Execute() {
	for _, n := range a.Nodes() {
		txn := n.Txns[a.TxnIndex]

		err := txn.Commit()
		require.NoError(a.s.T, err)
	}
}
