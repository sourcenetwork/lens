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
	"github.com/sourcenetwork/corekv/chunk"
	"github.com/sourcenetwork/corekv/namespace"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/immutable/enumerable"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/repository"
)

const defaultBlockMaxSize int = 1024 * 1024 * 3

// The key used to calculate keyLen is descriptive only, they are all the same length, and there
// is nothing special about this one.
var keyLen int = len([]byte("bafybeiet6foxcipesjurdqi4zpsgsiok5znqgw4oa5poef6qtiby5hlpzy"))

type Store interface {
	// Add stores the given Lens and returns its content ID.
	//
	// Two identical Lenses will result in the same content ID, and only a single copy will be stored.
	Add(ctx context.Context, cfg model.Lens) (string, error)

	// List fetches all the stored Lenses from the store and returns them mapped by their content ID.
	List(ctx context.Context) (map[string]model.Lens, error)

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

type store struct {
	repository   repository.Repository
	linkSystem   *linking.LinkSystem
	indexstore   corekv.ReaderWriter
	maxBlockSize int
}

// New creates a new `TxnStore` using the given parameters.
//
// The blockstore namespace may safely be shared by other components if desired.
func New(
	rootstore corekv.ReaderWriter,
	poolSize int,
	runtime module.Runtime,
	blockstoreNamespace string,
	blockstoreChunksize immutable.Option[int],
	maxBlockSize immutable.Option[int],
	indexstoreNamespace string,
) Store {
	var maxSize int
	if maxBlockSize.HasValue() {
		maxSize = maxBlockSize.Value()
	} else {
		maxSize = defaultBlockMaxSize
	}
	var bStore corekv.ReaderWriter = namespace.Wrap(rootstore, []byte(blockstoreNamespace))
	if blockstoreChunksize.HasValue() {
		bStore = chunk.NewSized(bStore, blockstoreChunksize.Value(), keyLen)
	}
	return &store{
		repository:   repository.NewRepository(poolSize, runtime),
		linkSystem:   makeLinkSystem(bStore),
		indexstore:   namespace.Wrap(rootstore, []byte(indexstoreNamespace)),
		maxBlockSize: maxSize,
	}
}

// NewWithRepository creates a new `TxnStore` using the given parameters.
//
// The blockstore namespace may safely be shared by other components if desired.
func NewWithRepository(
	repository repository.Repository,
	rootstore corekv.ReaderWriter,
	blockstoreNamespace string,
	blockstoreChunksize immutable.Option[int],
	maxBlockSize immutable.Option[int],
	indexstoreNamespace string,
) Store {
	var maxSize int
	if maxBlockSize.HasValue() {
		maxSize = maxBlockSize.Value()
	} else {
		maxSize = defaultBlockMaxSize
	}
	var bStore corekv.ReaderWriter = namespace.Wrap(rootstore, []byte(blockstoreNamespace))
	if blockstoreChunksize.HasValue() {
		bStore = chunk.NewSized(bStore, blockstoreChunksize.Value(), keyLen)
	}
	return &store{
		repository:   repository,
		linkSystem:   makeLinkSystem(bStore),
		indexstore:   namespace.Wrap(rootstore, []byte(indexstoreNamespace)),
		maxBlockSize: maxSize,
	}
}

func (s *store) Add(ctx context.Context, cfg model.Lens) (string, error) {
	configLink, err := writeConfigBlock(ctx, s.linkSystem, s.maxBlockSize, cfg)
	if err != nil {
		return "", err
	}

	err = s.repository.Add(ctx, configLink.String(), cfg)
	if err != nil {
		return "", err
	}

	// Index the ids of the config blocks so that we can rapidly access them without having to scan the entire blockstore
	// This is especially important if the blockstore is provided by users and may contain blocks not owned by LensVM.
	err = s.indexstore.Set(ctx, []byte(configLink.Binary()), []byte{})
	if err != nil {
		return "", err
	}

	_, configCID, err := cid.CidFromBytes([]byte(configLink.Binary()))
	if err != nil {
		return "", err
	}

	return configCID.String(), nil
}

func (s *store) List(ctx context.Context) (map[string]model.Lens, error) {
	iter, err := s.indexstore.Iterator(
		ctx,
		corekv.IterOptions{
			KeysOnly: true,
		},
	)
	if err != nil {
		return nil, err
	}

	results := map[string]model.Lens{}
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

		config, err := LoadLensModel(ctx, s.linkSystem, configCID)
		if err != nil {
			return nil, errors.Join(err, iter.Close())
		}
		results[configCID.String()] = config
	}

	return results, iter.Close()
}

func (s *store) Transform(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	if err := assertIsCid(id); err != nil {
		return nil, err
	}

	return s.repository.Transform(ctx, source, id)
}

func (s *store) Inverse(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	if err := assertIsCid(id); err != nil {
		return nil, err
	}

	return s.repository.Inverse(ctx, source, id)
}

func (s *store) Reload(ctx context.Context) error {
	lenses, err := s.List(ctx)
	if err != nil {
		return err
	}

	for cid, lens := range lenses {
		err = s.repository.Add(ctx, cid, lens)
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

func assertIsCid(input string) error {
	_, err := cid.Parse(input)
	return err
}
