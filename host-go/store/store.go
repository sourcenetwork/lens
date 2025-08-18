// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	cid "github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/storage"
	"github.com/multiformats/go-multicodec"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/immutable/enumerable"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type Store struct {
	store      linking.LinkSystem
	indexstore corekv.ReaderWriter
	repository repository.Repository
}

func New(
	store corekv.ReaderWriter,
	indexstore corekv.ReaderWriter,
	poolSize int, // should repo be passed in instead?
	runtime module.Runtime,
) *Store {
	return &Store{
		store:      makeLinkSystem(store),
		indexstore: indexstore,
		repository: repository.NewRepository(poolSize, runtime),
	}
}

func (s *Store) Init(ctx context.Context, txnSrc repository.TxnSource) error {
	s.repository.Init(txnSrc)
	return s.Reload(ctx)
}

func (s *Store) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {
	moduleLinks := make([]datamodel.Link, 0, len(cfg.Lenses))

	for _, module := range cfg.Lenses {
		wasmBytes, err := engine.GetWasmBytes(module.Path)
		if err != nil {
			return cid.Undef, err
		}
		lensBlock := LensBlock{
			WasmBytes: wasmBytes,
		}

		lensLink, err := s.store.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), lensBlock.GenerateNode())
		if err != nil {
			return cid.Undef, err
		}

		orderedArgs := make([]keyValue, len(module.Arguments))
		for argKey, value := range module.Arguments {
			// The linking system does not tolerate the `any` type, so we store them as json
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return cid.Undef, err
			}

			orderedArgs = append(orderedArgs, keyValue{
				Key:   argKey,
				Value: string(jsonBytes),
			})
		}
		// The blocks must be deterministic, so sort the arguments by key before storing
		sort.Slice(orderedArgs, func(i, j int) bool { return orderedArgs[i].Key < orderedArgs[j].Key })

		moduleBlock := ModuleBlock{
			Inverse:   module.Inverse,
			Arguments: orderedArgs,
			Lens:      lensLink,
		}

		moduleLink, err := s.store.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), moduleBlock.GenerateNode())
		if err != nil {
			return cid.Undef, err
		}

		moduleLinks = append(moduleLinks, moduleLink)
	}

	configBlock := ConfigBlock{
		Modules: moduleLinks,
	}
	configLink, err := s.store.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), configBlock.GenerateNode())

	err = s.repository.Add(ctx, configLink.String(), cfg)
	if err != nil {
		return cid.Undef, err
	}

	// Index the ids of the config blocks so that we can rapidly access them without having to scan the entire blockstore
	// This is especially important if the blockstore is provided by users and may contain blocks not owned by LensVM.
	err = s.indexstore.Set(ctx, []byte(configLink.Binary()), []byte{})
	if err != nil {
		return cid.Undef, err
	}

	_, configCID, err := cid.CidFromBytes([]byte(configLink.Binary()))
	if err != nil {
		return cid.Undef, err
	}

	return configCID, nil
}

func (s *Store) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
	iter, err := s.indexstore.Iterator(
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

		configNode, err := s.store.Load(linking.LinkContext{Ctx: ctx}, cidlink.Link{Cid: configCID}, ConfigBlockSchemaPrototype)
		if err != nil {
			return nil, err
		}

		configBlock := bindnode.Unwrap(configNode).(*ConfigBlock)
		config, err := configBlock.ToLensModel(ctx, s.store)
		if err != nil {
			return nil, err
		}
		results[configCID] = config
	}

	return results, iter.Close()
}

func (s *Store) Transform(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	return s.repository.Transform(ctx, source, id)
}

func (s *Store) Inverse(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	return s.repository.Inverse(ctx, source, id)
}

func (s *Store) Reload(ctx context.Context) error {
	lenses, err := s.List(ctx)
	if err != nil {
		return err
	}

	for cid, lens := range lenses {
		err = s.repository.Add(ctx, cid.String(), lens)
		if err != nil {
			return err
		}
	}

	// todo - old stuff will be overwritten by `Add`, but keys that no longer exist will continue to do so in the repo

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

func makeLinkSystem(store corekv.ReaderWriter) linking.LinkSystem {
	blockstore := newBlockstore(store)

	linkSys := cidlink.DefaultLinkSystem()
	linkSys.SetWriteStorage(blockstore)
	linkSys.SetReadStorage(blockstore)
	linkSys.TrustedStorage = true // todo - is needed?

	return linkSys
}

func getLinkPrototype() cidlink.LinkPrototype {
	return cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  uint64(multicodec.Cidv1),
		Codec:    uint64(multicodec.DagCbor),
		MhType:   uint64(multicodec.Sha2_256),
		MhLength: 32,
	}}
}
