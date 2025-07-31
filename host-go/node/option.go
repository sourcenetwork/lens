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

type Options struct {
	Path                immutable.Option[string]
	Rootstore           immutable.Option[corekv.TxnReaderWriter]
	TxnProvider         immutable.Option[store.TxnSource]
	PoolSize            immutable.Option[int]
	Runtime             immutable.Option[module.Runtime]
	BlockstoreNamespace immutable.Option[string]
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
