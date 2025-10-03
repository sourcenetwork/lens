// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package p2p

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/sourcenetwork/corekv/blockstore"
)

type BlockAccessFunc = func(ctx context.Context, peerID string, c cid.Cid) bool

// Host represents an object capable of managing peer to peer connections and the
// syncing of blocks to/from this process.
//
// This interface is a trimmed down version of the `defra/client.Host` interface.
// An implementation can be found in https://github.com/sourcenetwork/go-p2p.
type Host interface {
	// ID returns the peer ID of the host.
	ID() string
	// Addrs returns the host's list of addresses.
	Addresses() ([]string, error)
	// Pubkey return the byte slice representation of the host's public key.
	Pubkey() ([]byte, error)
	// Connect tries to connect to the peer with the given peer info.
	Connect(ctx context.Context, addresses []string) error
	// Disconnect will try to disconnect from the peer with the given ID.
	Disconnect(ctx context.Context, peerID string) error
	// BlockService returns the host's block service.
	IPLDStore() blockstore.IPLDStore
	// SetBlockAccessFunc set the function to use to determine if a peer has access to
	// the requested blocks on the block service.
	SetBlockAccessFunc(accessFunc BlockAccessFunc)
}
