// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"
	"errors"

	cid "github.com/ipfs/go-cid"
	// We must import the dagcbor in order to register it's encoders via it's private `init` function
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/storage"
	"github.com/multiformats/go-multicodec"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/immutable/enumerable"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type Store interface {
	// Add stores the given Lens and returns its content ID.
	//
	// Two identical Lenses will result in the same content ID, and only a single copy will be stored.
	Add(ctx context.Context, cfg model.Lens) (cid.Cid, error)

	// List fetches all the stored Lenses from the store and returns them mapped by their content ID.
	List(ctx context.Context) (map[cid.Cid]model.Lens, error)

	// Reload fetches all stored Lenses and uses them to overwrite any cached instances held by the in
	// memory wasm instance repository.
	//
	// This function is automatically called on `Node` startup.
	Reload(ctx context.Context) error

	// Transform returns an enumerable that feeds the given source through the Lens transform for the given
	// id, if there is no matching lens the given source will be returned.
	Transform(
		ctx context.Context,
		source enumerable.Enumerable[Document],
		id string,
	) (enumerable.Enumerable[Document], error)

	// Inverse returns an enumerable that feeds the given source through the Lens inverse for the given
	// id, if there is no matching lens the given source will be returned.
	Inverse(
		ctx context.Context,
		source enumerable.Enumerable[Document],
		id string,
	) (enumerable.Enumerable[Document], error)
}

type TxnStore interface {
	Store

	// NewTxn returns a new transaction.
	NewTxn(readonly bool) (Txn, error)

	// WithTxn returns a new `Store` instance scoped to the given transaction.
	//
	// Upon transaction commit any changes made via the returned value will be applied
	// to this `TxnStore`.
	WithTxn(txn Txn) Store
}

// New creates a new `TxnStore` using the given parameters.
//
// The blockstore namespace may safely be shared by other components if desired.
func New(
	txnSource TxnSource,
	poolSize int,
	runtime module.Runtime,
	blockstoreNamespace string,
	blockstoreChunksize immutable.Option[int],
	indexstoreNamespace string,
) TxnStore {
	return &implicitTxnStore{
		txnSource:           txnSource,
		repository:          repository.NewRepository(poolSize, runtime, &repositoryTxnSource{src: txnSource}),
		blockstoreNamespace: blockstoreNamespace,
		blockstoreChunksize: blockstoreChunksize,
		indexstoreNamespace: indexstoreNamespace,
	}
}

func add(ctx context.Context, cfg model.Lens, txn *txn) (cid.Cid, error) {
	configLink, err := writeConfigBlock(ctx, txn.linkSystem, cfg)
	if err != nil {
		return cid.Undef, err
	}

	err = txn.repository.Add(ctx, configLink.String(), cfg)
	if err != nil {
		return cid.Undef, err
	}

	// Index the ids of the config blocks so that we can rapidly access them without having to scan the entire blockstore
	// This is especially important if the blockstore is provided by users and may contain blocks not owned by LensVM.
	err = txn.indexstore.Set(ctx, []byte(configLink.Binary()), []byte{})
	if err != nil {
		return cid.Undef, err
	}

	_, configCID, err := cid.CidFromBytes([]byte(configLink.Binary()))
	if err != nil {
		return cid.Undef, err
	}

	return configCID, nil
}

func list(ctx context.Context, txn *txn) (map[cid.Cid]model.Lens, error) {
	iter, err := txn.indexstore.Iterator(
		ctx,
		corekv.IterOptions{
			KeysOnly: true,
		},
	)
	if err != nil {
		return nil, err
	}

	results := map[cid.Cid]model.Lens{}
	for {
		hasValue, err := iter.Next()
		if err != nil {
			return nil, errors.Join(err, iter.Close())
		}
		if !hasValue {
			break
		}

		configKey := iter.Key()
		_, configCID, err := cid.CidFromBytes(configKey)
		if err != nil {
			return nil, errors.Join(err, iter.Close())
		}

		config, err := loadLensModel(ctx, txn.linkSystem, configCID)
		if err != nil {
			return nil, err
		}
		results[configCID] = config
	}

	return results, iter.Close()
}

func transform(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
	txn *txn,
) (enumerable.Enumerable[Document], error) {
	return txn.repository.Transform(ctx, source, id)
}

func inverse(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
	txn *txn,
) (enumerable.Enumerable[Document], error) {
	return txn.repository.Inverse(ctx, source, id)
}

func reload(ctx context.Context, txn *txn) error {
	lenses, err := list(ctx, txn)
	if err != nil {
		return err
	}

	for cid, lens := range lenses {
		err = txn.repository.Add(ctx, cid.String(), lens)
		if err != nil {
			return err
		}
	}

	// todo - keys will not be removed from the repository until
	// https://github.com/sourcenetwork/lens/issues/118

	return nil
}

type blockstore struct {
	store corekv.ReaderWriter
}

var _ storage.ReadableStorage = (*blockstore)(nil)
var _ storage.WritableStorage = (*blockstore)(nil)

func newBlockstore(store corekv.ReaderWriter) *blockstore {
	return &blockstore{
		store: store,
	}
}

func (s *blockstore) Has(ctx context.Context, key string) (bool, error) {
	return s.store.Has(ctx, []byte(key))
}

func (s *blockstore) Get(ctx context.Context, key string) ([]byte, error) {
	return s.store.Get(ctx, []byte(key))
}

func (s *blockstore) Put(ctx context.Context, key string, content []byte) error {
	return s.store.Set(ctx, []byte(key), content)
}

func makeLinkSystem(store corekv.ReaderWriter) *linking.LinkSystem {
	blockstore := newBlockstore(store)

	linkSys := cidlink.DefaultLinkSystem()
	linkSys.SetWriteStorage(blockstore)
	linkSys.SetReadStorage(blockstore)
	linkSys.TrustedStorage = true

	return &linkSys
}

func getLinkPrototype() cidlink.LinkPrototype {
	return cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  uint64(multicodec.Cidv1),
		Codec:    uint64(multicodec.DagCbor),
		MhType:   uint64(multicodec.Sha2_256),
		MhLength: 32,
	}}
}
