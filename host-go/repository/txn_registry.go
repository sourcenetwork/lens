// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package repository

import (
	"context"

	"github.com/lens-vm/lens/host-go/config/model"
	"github.com/sourcenetwork/immutable/enumerable"
)

type implicitTxnLensRegistry struct {
	registry *lensRegistry
	db       TxnSource
}

type explicitTxnLensRegistry struct {
	registry *lensRegistry
	txn      Txn
}

var _ LensRegistry = (*implicitTxnLensRegistry)(nil)
var _ LensRegistry = (*explicitTxnLensRegistry)(nil)

func (r *implicitTxnLensRegistry) Init(txnSource TxnSource) {
	r.db = txnSource
}

func (r *explicitTxnLensRegistry) Init(txnSource TxnSource) {}

func (r *explicitTxnLensRegistry) WithTxn(txn Txn) LensRegistry {
	return &explicitTxnLensRegistry{
		registry: r.registry,
		txn:      txn,
	}
}

func (r *implicitTxnLensRegistry) SetMigration(ctx context.Context, collectionID string, cfg model.Lens) error {
	txn, err := r.db.NewTxn(ctx, false)
	if err != nil {
		return err
	}
	defer txn.Discard(ctx)
	txnCtx := r.registry.getCtx(txn, false)

	err = r.registry.setMigration(ctx, txnCtx, collectionID, cfg)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

func (r *explicitTxnLensRegistry) SetMigration(ctx context.Context, collectionID string, cfg model.Lens) error {
	return r.registry.setMigration(ctx, r.registry.getCtx(r.txn, false), collectionID, cfg)
}

func (r *implicitTxnLensRegistry) MigrateUp(
	ctx context.Context,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	txn, err := r.db.NewTxn(ctx, true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard(ctx)
	txnCtx := newTxnCtx(txn)

	return r.registry.migrateUp(txnCtx, src, collectionID)
}

func (r *explicitTxnLensRegistry) MigrateUp(
	ctx context.Context,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	return r.registry.migrateUp(r.registry.getCtx(r.txn, true), src, collectionID)
}

func (r *implicitTxnLensRegistry) MigrateDown(
	ctx context.Context,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	txn, err := r.db.NewTxn(ctx, true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard(ctx)
	txnCtx := newTxnCtx(txn)

	return r.registry.migrateDown(txnCtx, src, collectionID)
}

func (r *explicitTxnLensRegistry) MigrateDown(
	ctx context.Context,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[map[string]any], error) {
	return r.registry.migrateDown(r.registry.getCtx(r.txn, true), src, collectionID)
}
