// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

// TxCreate executes the `client tx create` command and appends the returned transaction id
// to state.Txns.
type TxnCreate struct {
	stateful
}

var _ Action = (*TxnCreate)(nil)

func (a *TxnCreate) Execute() {
	panic("todo")
}
