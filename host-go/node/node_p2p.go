// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !js

package node

import (
	"context"

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
	BlockstoreChunkSize immutable.Option[int]
	IndexstoreNamespace immutable.Option[string]
	P2P                 immutable.Option[p2p.Host]
	DisableP2P          bool

	P2POptions []sourceP2P.NodeOpt
}

type Node struct {
	onClose []closer
	Options Options
	Store   store.TxnStore
	P2P     immutable.Option[*p2p.P2P]
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

func createNode(ctx context.Context, store store.TxnStore, o Options, onClose []closer) (*Node, error) {
	var p2pSys immutable.Option[*p2p.P2P]
	if !o.DisableP2P {
		var host p2p.Host
		if o.P2P.HasValue() {
			host = o.P2P.Value()
		} else {
			p2pOptions := append(
				o.P2POptions,
				sourceP2P.WithBlockstoreNamespace(o.BlockstoreNamespace.Value()),
				sourceP2P.WithRootstore(o.Rootstore.Value()),
			)

			if o.BlockstoreChunkSize.HasValue() {
				p2pOptions = append(p2pOptions, sourceP2P.WithBlockstoreChunkSize(o.BlockstoreChunkSize.Value()))
			}

			var err error
			host, err = sourceP2P.NewPeer(
				ctx,
				p2pOptions...,
			)
			if err != nil {
				return nil, err
			}
		}

		p2pSys = immutable.Some(p2p.New(host, o.Rootstore.Value(), o.IndexstoreNamespace.Value()))
	}

	return &Node{
		onClose: onClose,
		Options: o,
		Store:   store,
		P2P:     p2pSys,
	}, nil
}
