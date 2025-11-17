// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"

	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/chunk"
	"github.com/sourcenetwork/corekv/namespace"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type implicitTxnStore struct {
	txnSource  TxnSource
	repository repository.TxnRepository

	blockstoreNamespace string
	blockstoreChunksize immutable.Option[int]
	indexstoreNamespace string
	maxBlockSize        int
}

type explicitTxnStore struct {
	txn *txn
}

var _ TxnStore = (*implicitTxnStore)(nil)
var _ Store = (*explicitTxnStore)(nil)

func (s *implicitTxnStore) WithTxn(txn Txn) Store {
	return &explicitTxnStore{
		txn: s.wrapTxn(txn),
	}
}

func (s *implicitTxnStore) NewTxn(readonly bool) (Txn, error) {
	return s.txnSource.NewTxn(readonly)
}

func (s *implicitTxnStore) newTxn(readonly bool) (*txn, error) {
	t, err := s.txnSource.NewTxn(readonly)
	if err != nil {
		return nil, err
	}

	return s.wrapTxn(t), nil
}

// The key used to calculate keyLen is descriptive only, they are all the same length, and there
// is nothing special about this one.
var keyLen int = len([]byte("bafybeiet6foxcipesjurdqi4zpsgsiok5znqgw4oa5poef6qtiby5hlpzy"))

func (s *implicitTxnStore) wrapTxn(t Txn) *txn {
	var bStore corekv.ReaderWriter = namespace.Wrap(t, []byte(s.blockstoreNamespace))
	if s.blockstoreChunksize.HasValue() {
		bStore = chunk.NewSized(bStore, s.blockstoreChunksize.Value(), keyLen)
	}

	return &txn{
		Txn:          t,
		maxBlockSize: s.maxBlockSize,
		linkSystem:   makeLinkSystem(bStore),
		indexstore:   namespace.Wrap(t, []byte(s.indexstoreNamespace)),
		repository:   s.repository.WithTxn(t),
	}
}

func (s *implicitTxnStore) Add(ctx context.Context, cfg model.Lens) (string, error) {
	txn, err := s.newTxn(false)
	if err != nil {
		return "", err
	}
	defer txn.Discard()

	id, err := add(ctx, cfg, txn)
	if err != nil {
		return "", err
	}

	err = txn.Commit()
	if err != nil {
		return "", err
	}

	return id, nil
}

func (s *explicitTxnStore) Add(ctx context.Context, cfg model.Lens) (string, error) {
	return add(ctx, cfg, s.txn)
}

func (s *implicitTxnStore) List(ctx context.Context) (map[string]model.Lens, error) {
	txn, err := s.newTxn(true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	return list(ctx, txn)
}

func (s *explicitTxnStore) List(ctx context.Context) (map[string]model.Lens, error) {
	return list(ctx, s.txn)
}

func (s *implicitTxnStore) Transform(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	txn, err := s.newTxn(true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	return transform(ctx, source, id, txn)
}

func (s *explicitTxnStore) Transform(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	return transform(ctx, source, id, s.txn)
}

func (s *implicitTxnStore) Inverse(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	txn, err := s.newTxn(true)
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	return inverse(ctx, source, id, txn)
}

func (s *explicitTxnStore) Inverse(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	return inverse(ctx, source, id, s.txn)
}

func (s *implicitTxnStore) Reload(
	ctx context.Context,
) error {
	txn, err := s.newTxn(false)
	if err != nil {
		return err
	}
	defer txn.Discard()

	err = reload(ctx, txn)
	if err != nil {
		return err
	}

	return txn.Commit()
}

func (s *explicitTxnStore) Reload(
	ctx context.Context,
) error {
	return reload(ctx, s.txn)
}
