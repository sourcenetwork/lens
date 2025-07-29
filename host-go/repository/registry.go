// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package repository

import (
	"context"
	"sync"

	"github.com/lens-vm/lens/host-go/config"
	"github.com/lens-vm/lens/host-go/config/model"
	"github.com/lens-vm/lens/host-go/engine/module"
	"github.com/sourcenetwork/immutable/enumerable"
)

// todo: This file, particularly the `lensPool` stuff, contains fairly sensitive code that is both
// cumbersome to fully test with integration/benchmark tests, and can have a significant affect on
// the users if broken (deadlocks, large performance degradation).  It should have proper unit tests.
// https://github.com/sourcenetwork/defradb/issues/1596

// LensDoc represents a document that will be sent to/from a Lens.
type LensDoc = map[string]any

// TxnSource represents an object capable of constructing the transactions that
// implicit-transaction registries need internally.
type TxnSource interface {
	NewTxn(context.Context, bool) (Txn, error)
}

// Txn is a common interface to the BasicTxn struct.
type Txn interface {
	// ID returns the unique immutable identifier for this transaction.
	ID() uint64

	// Commit finalizes a transaction, attempting to commit it to the Datastore.
	// May return an error if the transaction has gone stale. The presence of an
	// error is an indication that the data was not committed to the Datastore.
	Commit(ctx context.Context) error
	// Discard throws away changes recorded in a transaction without committing
	// them to the underlying Datastore. Any calls made to Discard after Commit
	// has been successfully called will have no effect on the transaction and
	// state of the Datastore, making it safe to defer.
	Discard(ctx context.Context)

	// OnSuccess registers a function to be called when the transaction is committed.
	OnSuccess(fn func())

	// OnError registers a function to be called when the transaction is rolled back.
	OnError(fn func())

	// OnDiscard registers a function to be called when the transaction is discarded.
	OnDiscard(fn func())
}

// LensRegistry exposes several useful thread-safe migration related functions which may
// be used to manage migrations.
type LensRegistry interface {
	// Init initializes the registry with the provided transaction source.
	Init(TxnSource)

	// SetMigration caches the migration for the given collection ID. It does not persist the migration in long
	// term storage, for that one should call [Store.SetMigration(ctx, cfg)].
	//
	// There may only be one migration per collection.  If another migration was registered it will be
	// overwritten by this migration.
	//
	// Migrations will only run if there is a complete path from the document schema version to the latest local
	// schema version.
	SetMigration(context.Context, string, model.Lens) error

	// MigrateUp returns an enumerable that feeds the given source through the Lens migration for the given
	// collection id if one is found, if there is no matching migration the given source will be returned.
	MigrateUp(
		context.Context,
		enumerable.Enumerable[map[string]any],
		string,
	) (enumerable.Enumerable[map[string]any], error)

	// MigrateDown returns an enumerable that feeds the given source through the Lens migration for the given
	// collection id in reverse if one is found, if there is no matching migration the given source will be returned.
	//
	// This downgrades any documents in the source enumerable if/when enumerated.
	MigrateDown(
		context.Context,
		enumerable.Enumerable[map[string]any],
		string,
	) (enumerable.Enumerable[map[string]any], error)
}

// lensRegistry is responsible for managing all migration related state within a local
// database instance.
type lensRegistry struct {
	poolSize int

	// The runtime used to execute lens wasm modules.
	runtime module.Runtime

	// The modules by file path used to instantiate lens wasm module instances.
	modulesByPath map[string]module.Module
	moduleLock    sync.Mutex

	lensPoolsByCollectionID     map[string]*lensPool
	reversedPoolsByCollectionID map[string]*lensPool
	poolLock                    sync.RWMutex

	// Writable transaction contexts by transaction ID.
	//
	// Read-only transaction contexts are not tracked.
	txnCtxs map[uint64]*txnContext
	txnLock sync.RWMutex
}

// txnContext contains uncommitted transaction state tracked by the registry,
// stuff within here should be accessible from within this transaction but not
// from outside.
type txnContext struct {
	txn                         Txn
	lensPoolsByCollectionID     map[string]*lensPool
	reversedPoolsByCollectionID map[string]*lensPool
}

func newTxnCtx(txn Txn) *txnContext {
	return &txnContext{
		txn:                         txn,
		lensPoolsByCollectionID:     map[string]*lensPool{},
		reversedPoolsByCollectionID: map[string]*lensPool{},
	}
}

// NewRegistry instantiates a new registery.
//
// It will be of size 5 (per schema version) if a size is not provided.
func NewRegistry(
	poolSize int,
	runtime module.Runtime,
) LensRegistry {
	registry := &lensRegistry{
		poolSize:                    poolSize,
		runtime:                     runtime,
		modulesByPath:               map[string]module.Module{},
		lensPoolsByCollectionID:     map[string]*lensPool{},
		reversedPoolsByCollectionID: map[string]*lensPool{},
		txnCtxs:                     map[uint64]*txnContext{},
	}

	return &implicitTxnLensRegistry{
		registry: registry,
	}
}

func (r *lensRegistry) getCtx(txn Txn, readonly bool) *txnContext {
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
		for collectionID, locker := range txnCtx.lensPoolsByCollectionID {
			r.lensPoolsByCollectionID[collectionID] = locker
		}
		for collectionID, locker := range txnCtx.reversedPoolsByCollectionID {
			r.reversedPoolsByCollectionID[collectionID] = locker
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

func (r *lensRegistry) setMigration(
	_ context.Context,
	txnCtx *txnContext,
	collectionID string,
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

	err := r.cachePool(txnCtx.lensPoolsByCollectionID, cfg, collectionID)
	if err != nil {
		return err
	}
	err = r.cachePool(txnCtx.reversedPoolsByCollectionID, reversedCfg, collectionID)
	// For now, checking this error is the best way of determining if a migration has an inverse.
	// Inverses are optional.
	if err != nil && err.Error() != "Export `inverse` does not exist" {
		return err
	}

	return nil
}

func (r *lensRegistry) cachePool(
	target map[string]*lensPool,
	cfg model.Lens,
	collectionID string,
) error {
	pool := r.newPool(r.poolSize, cfg)

	for i := 0; i < r.poolSize; i++ {
		lensPipe, err := r.newLensPipe(cfg)
		if err != nil {
			return err
		}
		pool.returnLens(lensPipe)
	}

	target[collectionID] = pool

	return nil
}

func (r *lensRegistry) migrateUp(
	txnCtx *txnContext,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[LensDoc], error) {
	return r.migrate(r.lensPoolsByCollectionID, txnCtx.lensPoolsByCollectionID, src, collectionID)
}

func (r *lensRegistry) migrateDown(
	txnCtx *txnContext,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[LensDoc], error) {
	return r.migrate(r.reversedPoolsByCollectionID, txnCtx.reversedPoolsByCollectionID, src, collectionID)
}

func (r *lensRegistry) migrate(
	pools map[string]*lensPool,
	txnPools map[string]*lensPool,
	src enumerable.Enumerable[LensDoc],
	collectionID string,
) (enumerable.Enumerable[LensDoc], error) {
	lensPool, ok := r.getPool(pools, txnPools, collectionID)
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

func (r *lensRegistry) getPool(
	pools map[string]*lensPool,
	txnPools map[string]*lensPool,
	collectionID string,
) (*lensPool, bool) {
	if pool, ok := txnPools[collectionID]; ok {
		return pool, true
	}

	r.poolLock.RLock()
	pool, ok := pools[collectionID]
	r.poolLock.RUnlock()
	return pool, ok
}

// lensPool provides a pool-like mechanic for caching a limited number of wasm lens modules in
// a thread safe fashion.
//
// Instanstiating a lens module is pretty expensive as it has to spin up the wasm runtime environment
// so we need to limit how frequently we do this.
type lensPool struct {
	// The config used to create the lenses within this locker.
	cfg model.Lens

	registry *lensRegistry

	// Using a buffered channel provides an easy way to manage a finite
	// number of lenses.
	//
	// We wish to limit this as creating lenses is expensive, and we do not want
	// to be dynamically resizing this collection and spinning up new lens instances
	// in user time, or holding on to large numbers of them.
	pipes chan *lensPipe
}

func (r *lensRegistry) newPool(lensPoolSize int, cfg model.Lens) *lensPool {
	return &lensPool{
		cfg:      cfg,
		registry: r,
		pipes:    make(chan *lensPipe, lensPoolSize),
	}
}

// borrow attempts to borrow a module from the locker, if one is not available
// it will return a new, temporary instance that will not be returned to the locker
// after use.
func (l *lensPool) borrow() (enumerable.Socket[LensDoc], error) {
	select {
	case lens := <-l.pipes:
		return &borrowedEnumerable{
			source: lens,
			pool:   l,
		}, nil
	default:
		// If there are no free cached migrations within the locker, create a new temporary one
		// instead of blocking.
		return l.registry.newLensPipe(l.cfg)
	}
}

// returnLens returns a borrowed module to the locker, allowing it to be reused by other contexts.
func (l *lensPool) returnLens(lens *lensPipe) {
	l.pipes <- lens
}

// borrowedEnumerable is an enumerable tied to a locker.
//
// it exposes the source enumerable and amends the Reset function so that when called, the source
// pipe is returned to the locker.
type borrowedEnumerable struct {
	source *lensPipe
	pool   *lensPool
}

var _ enumerable.Socket[LensDoc] = (*borrowedEnumerable)(nil)

func (s *borrowedEnumerable) SetSource(newSource enumerable.Enumerable[LensDoc]) {
	s.source.SetSource(newSource)
}

func (s *borrowedEnumerable) Next() (bool, error) {
	return s.source.Next()
}

func (s *borrowedEnumerable) Value() (LensDoc, error) {
	return s.source.Value()
}

func (s *borrowedEnumerable) Reset() {
	s.pool.returnLens(s.source)
	s.source.Reset()
}

// lensPipe provides a mechanic where the underlying wasm module can be hidden from consumers
// and allow input sources to be swapped in and out as different actors borrow it from the locker.
type lensPipe struct {
	input      enumerable.Socket[LensDoc]
	enumerable enumerable.Enumerable[LensDoc]
}

var _ enumerable.Socket[LensDoc] = (*lensPipe)(nil)

func (r *lensRegistry) newLensPipe(cfg model.Lens) (*lensPipe, error) {
	socket := enumerable.NewSocket[LensDoc]()

	r.moduleLock.Lock()
	enumerable, err := config.LoadInto[LensDoc, LensDoc](r.runtime, r.modulesByPath, cfg, socket)
	r.moduleLock.Unlock()

	if err != nil {
		return nil, err
	}

	return &lensPipe{
		input:      socket,
		enumerable: enumerable,
	}, nil
}

func (p *lensPipe) SetSource(newSource enumerable.Enumerable[LensDoc]) {
	p.input.SetSource(newSource)
}

func (p *lensPipe) Next() (bool, error) {
	return p.enumerable.Next()
}

func (p *lensPipe) Value() (LensDoc, error) {
	return p.enumerable.Value()
}

func (p *lensPipe) Reset() {
	p.input.Reset()
	// WARNING: Currently the wasm module state is not reset by calling reset on the enumerable
	// this means that state from one context may leak to the next useage.  There is a ticket here
	// to fix this: https://github.com/lens-vm/lens/issues/46
	p.enumerable.Reset()
}
