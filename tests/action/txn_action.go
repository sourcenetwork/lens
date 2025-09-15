// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/sourcenetwork/lens/tests/state"
)

// TxnAction wraps an action within the context of a transaction.
//
// Executing this TxnAction will execute the given action within the scope
// of the given transaction.
type TxnAction[T Action] struct {
	stateful

	TxnIndex int
	Action   T
}

var _ Action = (*TxnAction[Action])(nil)

// WithTxn wraps the given action within the scope of the default (ID: 0) transaction.
//
// If a transaction with the default ID (0) has not been created by the time this action
// executes, executing the transaction will panic.
func WithTxn[T Action](action T) *TxnAction[T] {
	return &TxnAction[T]{
		Action: action,
	}
}

// WithTxn wraps the given action within the scope of given transaction.
//
// If a transaction with the given ID has not been created by the time this action
// executes, executing the transaction will panic.
func WithTxnI[T Action](action T, txnIndex int) *TxnAction[T] {
	return &TxnAction[T]{
		TxnIndex: txnIndex,
		Action:   action,
	}
}

func (a *TxnAction[T]) SetState(s *state.State) {
	a.s = s
	if inner, ok := any(a.Action).(Stateful); ok {
		inner.SetState(s)
	}
}

func (a *TxnAction[T]) Execute() {
	originalStore := a.s.Store
	defer func() {
		// Make sure the original store is restored after executing otherwise
		// subsequent actions will erroneously act on the transaction.
		a.s.Store = originalStore
	}()

	// Replace the active store with the transaction, allowing the inner action
	// to act on the transaction without needing to be aware of it.
	a.s.Store = a.s.Node.Store.WithTxn(a.s.Txns[a.TxnIndex])

	a.Action.Execute()
}
