// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package p2p

import (
	"context"

	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/namespace"

	"github.com/sourcenetwork/lens/host-go/repository"
	"github.com/sourcenetwork/lens/host-go/store"
)

type implicitTxnP2P struct {
	host           Host
	txnSource      store.TxnSource
	indexNamespace string
	repository     repository.TxnRepository
}

type explicitTxnP2P struct {
	host       Host
	repository repository.Repository
	indexstore corekv.ReaderWriter
}

var _ TxnP2P = (*implicitTxnP2P)(nil)
var _ P2P = (*explicitTxnP2P)(nil)

// WithTxn returns a new `P2P` instance based off of this instance, but wrapped in a transaction.
//
// The transaction will not cover the syncing of the backing IPFS blocks.
func (r *implicitTxnP2P) WithTxn(txn Txn) P2P {
	return &explicitTxnP2P{
		host:       r.host,
		repository: r.repository.WithTxn(txn),
		indexstore: namespace.Wrap(txn, []byte(r.indexNamespace)),
	}
}

func (r *implicitTxnP2P) Host() Host {
	return r.host
}

func (r *explicitTxnP2P) Host() Host {
	return r.host
}

func (r *implicitTxnP2P) SyncLens(ctx context.Context, id string) error {
	txn, err := r.txnSource.NewTxn(false)
	if err != nil {
		return err
	}
	defer txn.Discard()

	repository := r.repository.WithTxn(txn)
	indexstore := namespace.Wrap(txn, []byte(r.indexNamespace))

	err = syncLens(ctx, id, r.host, repository, indexstore)
	if err != nil {
		return err
	}

	return txn.Commit()
}

func (r *explicitTxnP2P) SyncLens(ctx context.Context, id string) error {
	return syncLens(ctx, id, r.host, r.repository, r.indexstore)
}
