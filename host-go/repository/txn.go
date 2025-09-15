// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package repository

// TxnSource represents an object capable of constructing the transactions that
// implicit-transaction registries need internally.
type TxnSource interface {
	NewTxn(bool) (Txn, error)
}

// Txn represents a lens repository transaction.
//
// The repository should function correctly with any sensible implementation of this interface,
// although undefined behaviour will occur if the value returned from `ID` is not unique.
type Txn interface {
	// ID returns the unique immutable identifier for this transaction.
	ID() uint64

	// Commit applies all changes made via this [Txn] to the underlying
	// [Store].
	//
	// This signature should match that of `Commit` on `corekv.Txn`.
	Commit() error

	// Discard discards all changes made via this object so far, returning
	// it to the state it was at at time of construction.
	//
	// This signature should match that of `Discard` on `corekv.Txn`.
	Discard()

	// OnSuccess registers a function to be called when the transaction is committed.
	OnSuccess(fn func())

	// OnError registers a function to be called when the transaction is rolled back.
	OnError(fn func())

	// OnDiscard registers a function to be called when the transaction is discarded.
	OnDiscard(fn func())
}
