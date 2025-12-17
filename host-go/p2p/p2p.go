// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package p2p

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/blockstore"
	"github.com/sourcenetwork/lens/host-go/repository"
	"github.com/sourcenetwork/lens/host-go/store"
)

type P2P interface {
	Host() Host

	// SyncLens synchronously tries that the given lens ID exists locally in this node.
	//
	// If the lens does not exist locally it will try and fetch it from any connected nodes.
	//
	// It will keep trying to fetch the lens until it either succeeds, or the context times out.
	SyncLens(ctx context.Context, id string) error
}

type TxnP2P interface {
	P2P

	WithTxn(txn Txn) P2P
}
type Txn interface {
	repository.Txn
	corekv.ReaderWriter
}

func New(
	host Host,
	repository repository.TxnRepository,
	txnSource store.TxnSource,
	indexNamespace string,
) TxnP2P {
	return &implicitTxnP2P{
		host:           host,
		repository:     repository,
		txnSource:      txnSource,
		indexNamespace: indexNamespace,
	}
}

func syncLens(
	ctx context.Context,
	id string,
	host Host,
	repository repository.Repository,
	indexstore corekv.ReaderWriter,
) error {
	cid, err := cid.Parse(id)
	if err != nil {
		return err
	}

	linkSys := makeLinkSystem(host.IPLDStore())

	model, err := store.LoadLensModel(ctx, &linkSys, cid)
	if err != nil {
		return err
	}

	err = repository.Add(ctx, id, model)
	if err != nil {
		return err
	}

	// Store the top level index so that it may be fetched efficiently from the Store
	//
	// We can avoid bothering with a txn here so long as this is the only write required outside of the blockstore.
	// If another call is added, they likely should be written as part of the same transaction.
	err = indexstore.Set(ctx, cid.Bytes(), []byte{})
	if err != nil {
		return err
	}

	return nil
}

func makeLinkSystem(blockStore blockstore.IPLDStore) linking.LinkSystem {
	linkSys := cidlink.DefaultLinkSystem()
	linkSys.SetWriteStorage(blockStore)
	linkSys.SetReadStorage(blockStore)
	linkSys.TrustedStorage = true

	return linkSys
}
