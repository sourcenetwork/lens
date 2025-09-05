// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"

	cid "github.com/ipfs/go-cid"
	"github.com/sourcenetwork/lens/host-go/config/model"
)

type implicitTxnStore struct {
	store *store
	db    Root
}

type explicitTxnStore struct {
	store *store
	txn   Txn
}

var _ Store = (*implicitTxnStore)(nil)
var _ Store = (*explicitTxnStore)(nil)

// todo - how to handle the txnsource/txnctx from here??

func (s *implicitTxnStore) WithTxn(txn Txn) *explicitTxnStore {
	return &explicitTxnStore{
		store: s.store,
		txn:   txn,
	}
}

func (s *implicitTxnStore) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {
	txn, err := s.db.NewTxn(false)
	if err != nil {
		return cid.Undef, err
	}
	defer txn.Discard()

	id, err := s.store.Add(ctx, cfg, txn)
	if err != nil {
		return cid.Undef, err
	}

	err = txn.Commit()
	if err != nil {
		return cid.Undef, err
	}

	return id, nil
}

func (s *explicitTxnStore) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {
	return s.store.Add(ctx, cfg, s.txn)
}

func (s *implicitTxnStore) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
	txn, err := s.db.NewTxn(true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	return s.store.List(ctx, txn)
}

func (s *explicitTxnStore) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
	return s.store.List(ctx, s.txn)
}
