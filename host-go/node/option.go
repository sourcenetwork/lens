// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"github.com/sourcenetwork/corekv"
	sourceP2P "github.com/sourcenetwork/go-p2p"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/p2p"
	"github.com/sourcenetwork/lens/host-go/store"
)

type Options struct {
	Path                immutable.Option[string]
	Rootstore           immutable.Option[corekv.TxnReaderWriter]
	TxnSource           immutable.Option[store.TxnSource]
	PoolSize            immutable.Option[int]
	Runtime             immutable.Option[module.Runtime]
	BlockstoreNamespace immutable.Option[string]
	BlockstoreChunksize immutable.Option[int]
	IndexstoreNamespace immutable.Option[string]
	P2P                 immutable.Option[p2p.Host]
	DisableP2P          bool

	P2POptions []sourceP2P.NodeOpt
}

// Option is a funtion that sets a config value on the db.
type Option func(*Options)

func WithRootstore(rootstore corekv.TxnReaderWriter) Option {
	return func(opts *Options) {
		opts.Rootstore = immutable.Some(rootstore)
	}
}

func WithPath(path string) Option {
	return func(opts *Options) {
		opts.Path = immutable.Some(path)
	}
}

const DefaultPoolSize int = 5

func WithPoolSize(size int) Option {
	return func(opts *Options) {
		opts.PoolSize = immutable.Some(size)
	}
}

func WithRuntime(runtime module.Runtime) Option {
	return func(opts *Options) {
		opts.Runtime = immutable.Some(runtime)
	}
}

func WithBlockstoreNamespace(blockstoreNamespace string) Option {
	return func(opts *Options) {
		opts.BlockstoreNamespace = immutable.Some(blockstoreNamespace)
	}
}

func WithIndextoreNamespace(indexstoreNamespace string) Option {
	return func(opts *Options) {
		opts.IndexstoreNamespace = immutable.Some(indexstoreNamespace)
	}
}

func WithP2P(host p2p.Host) Option {
	return func(opts *Options) {
		opts.P2P = immutable.Some(host)
	}
}

func WithP2PDisabled(disableP2P bool) Option {
	return func(opts *Options) {
		opts.DisableP2P = disableP2P
	}
}

func WithP2Poptions(opts ...sourceP2P.NodeOpt) Option {
	return func(opt *Options) {
		opt.P2POptions = opts
	}
}

func WithTxnSource(txnSource store.TxnSource) Option {
	return func(opt *Options) {
		opt.TxnSource = immutable.Some(txnSource)
	}
}

// WithBlockstoreChunksize will wrap the blockstore in a chunkstore with the given chunksize.
//
// This allows the store to hold values of indefinate size, even if the underlying
// corekv store does not support it (such as badger in-memory store).
//
// Setting this to true will reduce read-write speed, but will not affect the running
// of lenses, only their storage and P2P efficiency.
func WithBlockstoreChunksize(blockstoreChunksize int) Option {
	return func(opts *Options) {
		opts.BlockstoreChunksize = immutable.Some(blockstoreChunksize)
	}
}
