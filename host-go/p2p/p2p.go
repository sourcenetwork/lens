// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package p2p

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/blockstore"
	"github.com/sourcenetwork/corekv/namespace"
	"github.com/sourcenetwork/lens/host-go/store"
)

type P2P struct {
	Host       Host
	indexstore corekv.ReaderWriter
}

func New(host Host, rootstore corekv.ReaderWriter, indexNamespace string) *P2P {
	return &P2P{
		Host:       host,
		indexstore: namespace.Wrap(rootstore, []byte(indexNamespace)),
	}
}

// SyncLens synchronously tries that the given lens ID exists locally in this node.
//
// If the lens does not exist locally it will try and fetch it from any connected nodes.
//
// It will keep trying to fetch the lens until it either succeeds, or the context times out.
func (p *P2P) SyncLens(ctx context.Context, id string) error {
	cid, err := cid.Parse(id)
	if err != nil {
		return err
	}

	linkSys := makeLinkSystem(p.Host.IPLDStore())

	configNode, err := linkSys.Load(linking.LinkContext{Ctx: ctx}, cidlink.Link{Cid: cid}, store.ConfigBlockSchemaPrototype)
	if err != nil {
		return err
	}

	configBlock := bindnode.Unwrap(configNode).(*store.ConfigBlock)

	for _, moduleLink := range configBlock.Modules {
		moduleNode, err := linkSys.Load(linking.LinkContext{Ctx: ctx}, moduleLink, store.ModuleBlockSchemaPrototype)
		if err != nil {
			return err
		}
		moduleBlock := bindnode.Unwrap(moduleNode).(*store.ModuleBlock)

		_, err = linkSys.Load(linking.LinkContext{Ctx: ctx}, moduleBlock.Lens, store.LensBlockSchemaPrototype)
		if err != nil {
			return err
		}
	}

	// Store the top level index so that it may be fetched efficiently from the Store
	//
	// We can avoid bothering with a txn here so long as this is the only write required outside of the blockstore.
	// If another call is added, they likely should be written as part of the same transaction.
	err = p.indexstore.Set(ctx, cid.Bytes(), []byte{})
	if err != nil {
		return err
	}

	return nil
}

func makeLinkSystem(blockStore blockstore.IPLDStore) linking.LinkSystem {
	linkSys := cidlink.DefaultLinkSystem()
	linkSys.SetWriteStorage(blockStore)
	linkSys.SetReadStorage(blockStore)
	linkSys.TrustedStorage = true

	return linkSys
}
