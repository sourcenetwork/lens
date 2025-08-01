// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package repository

import (
	"context"
	"sync"

	"github.com/sourcenetwork/immutable/enumerable"

	"github.com/sourcenetwork/lens/host-go/config"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/engine/module"
)

// todo: This file, particularly the `pool` stuff, contains fairly sensitive code that is both
// cumbersome to fully test with integration/benchmark tests, and can have a significant affect on
// the users if broken (deadlocks, large performance degradation).  It should have dedicated tests.
// https://github.com/sourcenetwork/defradb/issues/1596
//
// todo: This package currently relies on Defra tests for coverage - this needs to be resolved in:
// https://github.com/sourcenetwork/lens/issues/108

// Repository is a thread-safe store of pooled, reuseable, lens-wasm instances, allowing consumers
// to initialize a fixed amount up front and then use them as many times as required later.
//
// Should all instances within the pool be busy when accessing, temporary instances will be spun up
// on demand.  This is relatively expensive compared to the typical execution cost of a Lens function.
type Repository interface {
	// Init initializes the repository with the provided transaction source.
	//
	// Transactions are created from the source in order to ensure that partial changes are not applied on
	// execution of other functions on the [Repository] interface.
	Init(TxnSource)

	// Add caches the given lenses for the given ID, a reuseable set of wasm instances - the number of instances
	// created is determined by the `poolSize` parameter provided on repository creation.
	Add(ctx context.Context, id string, cfg model.Lens) error

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

// repository is responsible for managing all migration related state within a local
// database instance.
type repository struct {
	poolSize int

	// The runtime used to execute lens wasm modules.
	runtime module.Runtime

	// The modules by file path used to instantiate lens wasm module instances.
	modulesByPath map[string]module.Module
	moduleLock    sync.Mutex

	lensPoolsByID     map[string]*pool
	reversedPoolsByID map[string]*pool
	poolLock          sync.RWMutex

	// Writable transaction contexts by transaction ID.
	//
	// Read-only transaction contexts are not tracked.
	txnCtxs map[uint64]*txnContext
	txnLock sync.RWMutex
}

// txnContext contains uncommitted transaction state tracked by the repository,
// stuff within here should be accessible from within this transaction but not
// from outside.
type txnContext struct {
	txn               Txn
	lensPoolsByID     map[string]*pool
	reversedPoolsByID map[string]*pool
}

func newTxnCtx(txn Txn) *txnContext {
	return &txnContext{
		txn:               txn,
		lensPoolsByID:     map[string]*pool{},
		reversedPoolsByID: map[string]*pool{},
	}
}

// NewRepository instantiates a new repository.
func NewRepository(
	poolSize int,
	runtime module.Runtime,
) Repository {
	return &implicitTxnRepository{
		repository: &repository{
			poolSize:          poolSize,
			runtime:           runtime,
			modulesByPath:     map[string]module.Module{},
			lensPoolsByID:     map[string]*pool{},
			reversedPoolsByID: map[string]*pool{},
			txnCtxs:           map[uint64]*txnContext{},
		},
	}
}

func (r *repository) getCtx(txn Txn, readonly bool) *txnContext {
	r.txnLock.RLock()
	if txnCtx, ok := r.txnCtxs[txn.ID()]; ok {
		r.txnLock.RUnlock()
		return txnCtx
	}
	r.txnLock.RUnlock()

	txnCtx := newTxnCtx(txn)
	if readonly {
		return txnCtx
	}

	r.txnLock.Lock()
	r.txnCtxs[txn.ID()] = txnCtx
	r.txnLock.Unlock()

	txnCtx.txn.OnSuccess(func() {
		r.poolLock.Lock()
		for id, locker := range txnCtx.lensPoolsByID {
			r.lensPoolsByID[id] = locker
		}
		for id, locker := range txnCtx.reversedPoolsByID {
			r.reversedPoolsByID[id] = locker
		}
		r.poolLock.Unlock()

		r.txnLock.Lock()
		delete(r.txnCtxs, txn.ID())
		r.txnLock.Unlock()
	})

	txn.OnError(func() {
		r.txnLock.Lock()
		delete(r.txnCtxs, txn.ID())
		r.txnLock.Unlock()
	})

	txn.OnDiscard(func() {
		// Delete it to help reduce the build up of memory, the txnCtx will be re-contructed if the
		// txn is reused after discard.
		r.txnLock.Lock()
		delete(r.txnCtxs, txn.ID())
		r.txnLock.Unlock()
	})

	return txnCtx
}

func (r *repository) add(
	txnCtx *txnContext,
	id string,
	cfg model.Lens,
) error {
	inversedModuleCfgs := make([]model.LensModule, len(cfg.Lenses))
	for i, moduleCfg := range cfg.Lenses {
		// Reverse the order of the lenses for the inverse migration.
		inversedModuleCfgs[len(cfg.Lenses)-i-1] = model.LensModule{
			Path: moduleCfg.Path,
			// Reverse the direction of the lens.
			// This needs to be done on a clone of the original cfg or we may end up mutating
			// the original.
			Inverse:   !moduleCfg.Inverse,
			Arguments: moduleCfg.Arguments,
		}
	}

	reversedCfg := model.Lens{
		Lenses: inversedModuleCfgs,
	}

	err := r.cachePool(txnCtx.lensPoolsByID, cfg, id)
	if err != nil {
		return err
	}
	err = r.cachePool(txnCtx.reversedPoolsByID, reversedCfg, id)
	// For now, checking this error is the best way of determining if a migration has an inverse.
	// Inverses are optional.
	if err != nil && err.Error() != "Export `inverse` does not exist" {
		return err
	}

	return nil
}

func (r *repository) cachePool(
	target map[string]*pool,
	cfg model.Lens,
	id string,
) error {
	pool := r.newPool(r.poolSize, cfg)

	for i := 0; i < r.poolSize; i++ {
		lensPipe, err := r.newLensPipe(cfg)
		if err != nil {
			return err
		}
		pool.returnLens(lensPipe)
	}

	target[id] = pool

	return nil
}

func (r *repository) transform(
	txnCtx *txnContext,
	src enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	return r.appendLens(r.lensPoolsByID, txnCtx.lensPoolsByID, src, id)
}

func (r *repository) inverse(
	txnCtx *txnContext,
	src enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	return r.appendLens(r.reversedPoolsByID, txnCtx.reversedPoolsByID, src, id)
}

func (r *repository) appendLens(
	pools map[string]*pool,
	txnPools map[string]*pool,
	src enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	lensPool, ok := r.getPool(pools, txnPools, id)
	if !ok {
		// If there are no migrations for this schema version, just return the given source.
		return src, nil
	}

	lens, err := lensPool.borrow()
	if err != nil {
		return nil, err
	}

	lens.SetSource(src)

	return lens, nil
}

func (r *repository) getPool(
	pools map[string]*pool,
	txnPools map[string]*pool,
	id string,
) (*pool, bool) {
	if pool, ok := txnPools[id]; ok {
		return pool, true
	}

	r.poolLock.RLock()
	pool, ok := pools[id]
	r.poolLock.RUnlock()
	return pool, ok
}

// pool provides a pool-like mechanic for caching a limited number of wasm lens modules in
// a thread safe fashion.
//
// Instanstiating a lens module is pretty expensive as it has to spin up the wasm runtime environment
// so we need to limit how frequently we do this.
type pool struct {
	// The config used to create the lenses within this locker.
	cfg model.Lens

	repository *repository

	// Using a buffered channel provides an easy way to manage a finite
	// number of lenses.
	//
	// We wish to limit this as creating lenses is expensive, and we do not want
	// to be dynamically resizing this collection and spinning up new lens instances
	// in user time, or holding on to large numbers of them.
	pipes chan *pipe
}

func (r *repository) newPool(lensPoolSize int, cfg model.Lens) *pool {
	return &pool{
		cfg:        cfg,
		repository: r,
		pipes:      make(chan *pipe, lensPoolSize),
	}
}

// borrow attempts to borrow a module from the locker, if one is not available
// it will return a new, temporary instance that will not be returned to the locker
// after use.
func (l *pool) borrow() (enumerable.Socket[Document], error) {
	select {
	case lens := <-l.pipes:
		return &borrowedEnumerable{
			source: lens,
			pool:   l,
		}, nil
	default:
		// If there are no free cached migrations within the locker, create a new temporary one
		// instead of blocking.
		return l.repository.newLensPipe(l.cfg)
	}
}

// returnLens returns a borrowed module to the locker, allowing it to be reused by other contexts.
func (l *pool) returnLens(lens *pipe) {
	l.pipes <- lens
}

// borrowedEnumerable is an enumerable tied to a locker.
//
// it exposes the source enumerable and amends the Reset function so that when called, the source
// pipe is returned to the locker.
type borrowedEnumerable struct {
	source *pipe
	pool   *pool
}

var _ enumerable.Socket[Document] = (*borrowedEnumerable)(nil)

func (s *borrowedEnumerable) SetSource(newSource enumerable.Enumerable[Document]) {
	s.source.SetSource(newSource)
}

func (s *borrowedEnumerable) Next() (bool, error) {
	return s.source.Next()
}

func (s *borrowedEnumerable) Value() (Document, error) {
	return s.source.Value()
}

func (s *borrowedEnumerable) Reset() {
	s.pool.returnLens(s.source)
	s.source.Reset()
}

// pipe provides a mechanic where the underlying wasm module can be hidden from consumers
// and allow input sources to be swapped in and out as different actors borrow it from the locker.
type pipe struct {
	input      enumerable.Socket[Document]
	enumerable enumerable.Enumerable[Document]
}

var _ enumerable.Socket[Document] = (*pipe)(nil)

func (r *repository) newLensPipe(cfg model.Lens) (*pipe, error) {
	socket := enumerable.NewSocket[Document]()

	r.moduleLock.Lock()
	enumerable, err := config.LoadInto[Document, Document](r.runtime, r.modulesByPath, cfg, socket)
	r.moduleLock.Unlock()

	if err != nil {
		return nil, err
	}

	return &pipe{
		input:      socket,
		enumerable: enumerable,
	}, nil
}

func (p *pipe) SetSource(newSource enumerable.Enumerable[Document]) {
	p.input.SetSource(newSource)
}

func (p *pipe) Next() (bool, error) {
	return p.enumerable.Next()
}

func (p *pipe) Value() (Document, error) {
	return p.enumerable.Value()
}

func (p *pipe) Reset() {
	p.input.Reset()
	// WARNING: Currently the wasm module state is not reset by calling reset on the enumerable
	// this means that state from one context may leak to the next useage.  There is a ticket here
	// to fix this: https://github.com/sourcenetwork/lens/issues/46
	p.enumerable.Reset()
}
