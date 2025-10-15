// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build js

package node

import (
	"context"

	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/store"
)

type Options struct {
	Path                immutable.Option[string]
	Rootstore           immutable.Option[corekv.TxnReaderWriter]
	TxnSource           immutable.Option[store.TxnSource]
	PoolSize            immutable.Option[int]
	Runtime             immutable.Option[module.Runtime]
	BlockstoreNamespace immutable.Option[string]
	BlockstoreChunkSize immutable.Option[int]
	IndexstoreNamespace immutable.Option[string]
	DisableP2P          bool
}

type Node struct {
	onClose []closer
	Options Options
	Store   store.TxnStore
}

// WARNING - This option exists only for compatibility reasons, it can never have a
// `false` value.
func WithP2PDisabled(disableP2P bool) Option {
	if !disableP2P {
		panic("P2P cannot be enabled on JavaScript builds")
	}

	return func(opts *Options) {
		opts.DisableP2P = disableP2P
	}
}

func createNode(ctx context.Context, store store.TxnStore, o Options, onClose []closer) (*Node, error) {
	return &Node{
		onClose: onClose,
		Options: o,
		Store:   store,
	}, nil
}
