// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type txnWrapper struct {
	corekv.Txn

	id uint64

	successFns []func()
	errorFns   []func()
	discardFns []func()
}

var _ repository.Txn = (*txnWrapper)(nil)

func (t *txnWrapper) ID() uint64 {
	return t.id
}

// Commit finalizes a transaction, attempting to commit it to the Datastore.
// May return an error if the transaction has gone stale. The presence of an
// error is an indication that the data was not committed to the Datastore.
func (t *txnWrapper) Commit() error {
	var fns []func()

	err := t.Txn.Commit()
	if err != nil {
		fns = t.errorFns
	} else {
		fns = t.successFns
	}

	for _, fn := range fns {
		fn()
	}

	return err
}

// Discard throws away changes recorded in a transaction without committing
// them to the underlying Datastore. Any calls made to Discard after Commit
// has been successfully called will have no effect on the transaction and
// state of the Datastore, making it safe to defer.
func (t *txnWrapper) Discard() {
	t.Txn.Discard()

	for _, fn := range t.discardFns {
		fn()
	}
}

// OnSuccess registers a function to be called when the transaction is committed.
func (t *txnWrapper) OnSuccess(fn func()) {
	t.successFns = append(t.successFns, fn)
}

// OnError registers a function to be called when the transaction is rolled back.
func (t *txnWrapper) OnError(fn func()) {
	t.errorFns = append(t.errorFns, fn)
}

// OnDiscard registers a function to be called when the transaction is discarded.
func (t *txnWrapper) OnDiscard(fn func()) {
	t.discardFns = append(t.discardFns, fn)
}
