// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package repository

import (
	"context"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/config/model"
)

type implicitTxnRepository struct {
	repository *repository
	db         TxnSource
}

type explicitTxnRepository struct {
	repository *repository
	txn        Txn
}

var _ Repository = (*implicitTxnRepository)(nil)
var _ Repository = (*explicitTxnRepository)(nil)

func (r *implicitTxnRepository) WithTxn(txn Txn) Repository {
	return &explicitTxnRepository{
		repository: r.repository,
		txn:        txn,
	}
}

func (r *implicitTxnRepository) Add(ctx context.Context, collectionID string, cfg model.Lens) error {
	txn, err := r.db.NewTxn(false)
	if err != nil {
		return err
	}
	defer txn.Discard()
	txnCtx := r.repository.getCtx(txn, false)

	err = r.repository.add(txnCtx, collectionID, cfg)
	if err != nil {
		return err
	}

	return txn.Commit()
}

func (r *explicitTxnRepository) Add(ctx context.Context, collectionID string, cfg model.Lens) error {
	return r.repository.add(r.repository.getCtx(r.txn, false), collectionID, cfg)
}

func (r *implicitTxnRepository) Transform(
	ctx context.Context,
	src enumerable.Enumerable[Document],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	txn, err := r.db.NewTxn(true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard()
	txnCtx := newTxnCtx(txn)

	return r.repository.transform(txnCtx, src, collectionID)
}

func (r *explicitTxnRepository) Transform(
	ctx context.Context,
	src enumerable.Enumerable[Document],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	return r.repository.transform(r.repository.getCtx(r.txn, true), src, collectionID)
}

func (r *implicitTxnRepository) Inverse(
	ctx context.Context,
	src enumerable.Enumerable[Document],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	txn, err := r.db.NewTxn(true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard()
	txnCtx := newTxnCtx(txn)

	return r.repository.inverse(txnCtx, src, collectionID)
}

func (r *explicitTxnRepository) Inverse(
	ctx context.Context,
	src enumerable.Enumerable[Document],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	return r.repository.inverse(r.repository.getCtx(r.txn, true), src, collectionID)
}
