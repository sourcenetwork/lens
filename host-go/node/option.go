// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/store"
)

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

func WithTxnSource(txnSource store.TxnSource) Option {
	return func(opt *Options) {
		opt.TxnSource = immutable.Some(txnSource)
	}
}

// WithBlockstoreChunkSize will wrap the blockstore in a chunkstore with the given chunksize.
//
// This allows the store to hold values of indefinite size, even if the underlying
// corekv store does not support it (such as badger in-memory store).
//
// Setting this to true will reduce read-write speed, but will not affect the running
// of lenses, only their storage and P2P efficiency.
func WithBlockstoreChunkSize(blockstoreChunkSize int) Option {
	return func(opts *Options) {
		opts.BlockstoreChunkSize = immutable.Some(blockstoreChunkSize)
	}
}
