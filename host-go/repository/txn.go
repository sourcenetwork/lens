// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package repository

// TxnSource represents an object capable of constructing the transactions that
// implicit-transaction registries need internally.
type TxnSource interface {
	NewTxn(bool) (Txn, error)
}

// Txn is a common interface to the BasicTxn struct.
type Txn interface { // todo - document that this takes commit/discard from corekv.txn and adds a few extras
	// ID returns the unique immutable identifier for this transaction.
	ID() uint64

	// Commit applies all changes made via this [Txn] to the underlying
	// [Store].
	Commit() error

	// Discard discards all changes made via this object so far, returning
	// it to the state it was at at time of construction.
	Discard()

	// OnSuccess registers a function to be called when the transaction is committed.
	OnSuccess(fn func())

	// OnError registers a function to be called when the transaction is rolled back.
	OnError(fn func())

	// OnDiscard registers a function to be called when the transaction is discarded.
	OnDiscard(fn func())
}
