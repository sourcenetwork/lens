// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"

	cid "github.com/ipfs/go-cid"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type implicitTxnStore struct {
	repository *store
	db         repository.TxnSource
}

type explicitTxnStore struct {
	repository *store
	txn        repository.Txn
}

var _ Store = (*implicitTxnStore)(nil)
var _ Store = (*explicitTxnStore)(nil)

// todo - how to handle the txnsource/txnctx from here??

func (s *implicitTxnStore) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {

}

func (s *explicitTxnStore) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {

}

func (s *implicitTxnStore) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
}

func (s *explicitTxnStore) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
}
