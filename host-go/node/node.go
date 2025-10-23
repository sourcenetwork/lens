// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"context"
	"errors"

	badgerds "github.com/dgraph-io/badger/v4"
	"github.com/sourcenetwork/corekv"

	"github.com/sourcenetwork/corekv/badger"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/repository"
	"github.com/sourcenetwork/lens/host-go/runtimes"
	"github.com/sourcenetwork/lens/host-go/store"
)

type closer = func() error

func New(ctx context.Context, opts ...Option) (*Node, error) {
	var o Options
	for _, option := range opts {
		option(&o)
	}

	onClose := []closer{}
	if !o.Rootstore.HasValue() {
		if !o.Path.HasValue() {
			// This is because the corekv memory store is not yet reliable due to
			// https://github.com/sourcenetwork/corekv/issues/58
			// and badger in memory store does not support values greater than 1MB
			// preventing it from storing lenses until we implement a work-around
			return nil, errors.New("either rootstore or path must be provided")
		}

		rootstore, err := badger.NewDatastore(o.Path.Value(), badgerds.DefaultOptions(o.Path.Value()))
		if err != nil {
			return nil, err
		}

		o.Rootstore = immutable.Some[corekv.TxnReaderWriter](rootstore)

		onClose = append(onClose, func() error {
			// We should only close the store on close if we own it
			return rootstore.Close()
		})
	}

	if !o.TxnSource.HasValue() {
		o.TxnSource = immutable.Some[store.TxnSource](&inMemoryTxnSource{store: o.Rootstore.Value()})
	}

	if !o.PoolSize.HasValue() {
		o.PoolSize = immutable.Some(DefaultPoolSize)
	}

	if !o.Runtime.HasValue() {
		o.Runtime = immutable.Some(runtimes.Default())
	}

	if !o.BlockstoreNamespace.HasValue() {
		o.BlockstoreNamespace = immutable.Some("b")
	}

	if !o.IndexstoreNamespace.HasValue() {
		o.IndexstoreNamespace = immutable.Some("i")
	}

	if !o.MaxBlockSize.HasValue() {
		o.MaxBlockSize = immutable.Some(1024 * 1024 * 3)
	}

	repo := repository.NewRepository(o.PoolSize.Value(), o.Runtime.Value(), &repositoryTxnSource{src: o.TxnSource.Value()})

	node, err := createNode(
		ctx,
		store.NewWithRepository(
			o.TxnSource.Value(),
			repo,
			o.BlockstoreNamespace.Value(),
			o.BlockstoreChunkSize,
			o.MaxBlockSize,
			o.IndexstoreNamespace.Value(),
		),
		repo,
		o,
		onClose,
	)
	if err != nil {
		return nil, err
	}

	// Reload on create, if the store is persisted, this will read any Lenses already in the store and
	// add them to the repository.
	err = node.Store.Reload(ctx)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (n *Node) Close() error {
	for _, closer := range n.onClose {
		err := closer()
		if err != nil {
			return err
		}
	}

	return nil
}

// repositoryTxnSource wraps a `TxnSource` so that it satisfies the `repository.TxnSource`
// interface and can be passed through.
//
// Without this, either the `store` package or the `repository` package would require an unnecessarily
// enlarged Txn interface, hindering consumption.
type repositoryTxnSource struct {
	src store.TxnSource
}

var _ repository.TxnSource = (*repositoryTxnSource)(nil)

func (s *repositoryTxnSource) NewTxn(readonly bool) (repository.Txn, error) {
	return s.src.NewTxn(readonly)
}
