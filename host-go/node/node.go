// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"context"
	"errors"

	badgerds "github.com/dgraph-io/badger/v4"
	"github.com/sourcenetwork/corekv"
	sourceP2P "github.com/sourcenetwork/go-p2p"

	"github.com/sourcenetwork/corekv/badger"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/p2p"
	"github.com/sourcenetwork/lens/host-go/runtimes"
	"github.com/sourcenetwork/lens/host-go/store"
)

type closer = func() error

type Node struct {
	onClose []closer
	Options Options
	Store   store.TxnStore
	P2P     immutable.Option[*p2p.P2P]
}

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

	var p2pSys immutable.Option[*p2p.P2P]
	if !o.DisableP2P {
		var host p2p.Host
		if o.P2P.HasValue() {
			host = o.P2P.Value()
		} else {
			p2pOptions := append(
				o.P2POptions,
				sourceP2P.WithBlockstoreNamespace(o.BlockstoreNamespace.Value()),
				sourceP2P.WithRootstore(o.Rootstore.Value()),
			)

			if o.BlockstoreChunksize.HasValue() {
				p2pOptions = append(p2pOptions, sourceP2P.WithBlockstoreChunksize(o.BlockstoreChunksize.Value()))
			}

			var err error
			host, err = sourceP2P.NewPeer(
				ctx,
				p2pOptions...,
			)
			if err != nil {
				return nil, err
			}
		}

		p2pSys = immutable.Some(p2p.New(host, o.Rootstore.Value(), o.IndexstoreNamespace.Value()))
	}

	node := &Node{
		onClose: onClose,
		Options: o,
		Store: store.New(
			o.TxnSource.Value(),
			o.PoolSize.Value(),
			o.Runtime.Value(),
			o.BlockstoreNamespace.Value(),
			o.BlockstoreChunksize,
			o.IndexstoreNamespace.Value(),
		),
		P2P: p2pSys,
	}

	// Reload on create, if the store is persisted, this will read any Lenses already in the store and
	// add them to the repository.
	err := node.Store.Reload(ctx)
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
