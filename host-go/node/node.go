// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"context"
	"sync/atomic"

	//badgerds "github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/memory"

	//"github.com/sourcenetwork/corekv/badger"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/repository"
	"github.com/sourcenetwork/lens/host-go/runtimes"
	"github.com/sourcenetwork/lens/host-go/store"
)

type NodeI interface { // todo - rename
	Add(ctx context.Context, cfg model.Lens) (cid.Cid, error)
	List(ctx context.Context) (map[cid.Cid]model.Lens, error)
}

type options struct {
	rootstore   immutable.Option[corekv.ReaderWriter] // todo - should this be a mandatory param?
	txnProvider immutable.Option[store.PTxnSource]
}

// Option is a funtion that sets a config value on the db.
type Option func(*options) // todo - private param is wierd/broken

func WithRootstore(rootstore corekv.ReaderWriter) Option {
	return func(opts *options) {
		opts.rootstore = immutable.Some(rootstore)
	}
}

type Node struct { // todo - rename
	store store.Store
}

func New(ctx context.Context, opts ...Option) (*Node, error) {
	var o options
	for _, option := range opts {
		option(&o)
	}

	if !o.rootstore.HasValue() {
		/*
			rootstore, err := badger.NewDatastore("", badgerds.DefaultOptions("").WithInMemory(true)) // todo - we must close if we own the store
			if err != nil {
				return nil, err
			}
		*/
		rootstore := memory.NewDatastore(ctx)

		if !o.txnProvider.HasValue() {
			o.txnProvider = immutable.Some[store.PTxnSource](&inMemoryTxnSource{store: rootstore})
		}

		o.rootstore = immutable.Some[corekv.ReaderWriter](rootstore)
	}

	node := &Node{
		store: store.New(
			o.txnProvider.Value(),
			o.rootstore.Value(),
			5,
			runtimes.Default(),
		), // todo - opts
	}

	return node, nil
}

func (n *Node) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {
	return n.store.Add(ctx, cfg)
}

func (n *Node) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
	return n.store.List(ctx)
}

type inMemoryTxnSource struct {
	previousTxnID uint64
	store         corekv.TxnStore
}

var _ store.PTxnSource = (*inMemoryTxnSource)(nil)

func (s *inMemoryTxnSource) NewTxn(readonly bool) (store.PTxn, error) {
	txnID := atomic.AddUint64(&s.previousTxnID, 1)

	return &txnWrapper{
		id:  txnID,
		Txn: s.store.NewTxn(readonly),
	}, nil
}

/*
func (s *inMemoryTxnSource) NewTxn(readonly bool) (repository.Txn, error) {
	txnID := atomic.AddUint64(&s.previousTxnID, 1)

	return &txnWrapper{
		id:  txnID,
		Txn: s.store.NewTxn(readonly),
	}, nil
}
*/

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
