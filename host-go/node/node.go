// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/storage/bsrvadapter"
	"github.com/multiformats/go-multicodec"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/memory"
	"github.com/sourcenetwork/corekv/namespace"
	"github.com/sourcenetwork/immutable"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/repository"
	"github.com/sourcenetwork/lens/host-go/runtimes"
	"github.com/sourcenetwork/lens/host-go/store"
)

type StreamHandler func(stream io.Reader, peerID string)
type PubsubMessageHandler func(from string, topic string, msg []byte) ([]byte, error)
type BlockAccessFunc func(ctx context.Context, peerID string, c cid.Cid) bool

type PeerInfo struct {
	ID        string
	Addresses []string
}

func (p PeerInfo) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

type PubsubResponse struct {
	// ID is the cid.Cid of the received message.
	ID string
	// From is the ID of the sender.
	From string
	// Data is the message data.
	Data []byte
	// Err is an error from the sender.
	Err error
}

// todo - this (and PubsubResponse etc) is copy-pasted from Defra, consider breaking out to repo
type Host interface {
	// ID returns the peer ID of the host.
	ID() string
	// Addrs returns the host's list of addresses.
	Addrs() []string
	// PeerInfo returns the host's [PeerInfo].
	PeerInfo() PeerInfo
	// Pubkey return the byte slice representation of the host's public key.
	Pubkey() ([]byte, error)
	// Connect tries to connect to the peer with the given [PeerInfo].
	Connect(ctx context.Context, info PeerInfo) error
	// Disconnect will try to disconnect from the peer with the given ID.
	Disconnect(ctx context.Context, peerID string) error
	// Send will try to send the given data to a peer.
	Send(ctx context.Context, data []byte, peerID string, protocolID string) error
	// Sign will return a hash of the provided data signed with the private key of the host.
	Sign(data []byte) ([]byte, error)
	// SetStreamHandler tells the host to listen for messages of the provided protocol ID and
	// handle them with the given handler.
	SetStreamHandler(protocolID string, handler StreamHandler)
	// AddPubSubTopic adds a pubsub topic to the host.
	AddPubSubTopic(topicName string, subscribe bool, handler PubsubMessageHandler) error
	// RemovePubSubTopic removes the given topic from the host.
	RemovePubSubTopic(topic string) error
	// PublishToTopicAsync sends a new message on the given topic without waiting for a response.
	PublishToTopicAsync(ctx context.Context, topic string, data []byte) error
	// PublishToTopic sends a new message on the given topic, returning a response channel.
	// It provides the option to allow responses from multiple peers.
	PublishToTopic(
		ctx context.Context,
		topic string,
		data []byte,
		withMultiResponse bool,
	) (<-chan PubsubResponse, error)
	// BlockService returns the host's block service.
	BlockService() blockservice.BlockService
	// SetBlockAccessFunc set the function to use to determine if a peer has access to
	// the requested blocks on the block service.
	SetBlockAccessFunc(accessFunc BlockAccessFunc)
}

type options struct {
	p2p         immutable.Option[Host]
	rootstore   immutable.Option[corekv.ReaderWriter] // todo - should this be a mandatory param?
	blockstore  immutable.Option[corekv.ReaderWriter]
	txnProvider immutable.Option[repository.TxnSource]
}

// Option is a funtion that sets a config value on the db.
type Option func(*options) // todo - private param is wierd/broken

func WithP2P(host Host) Option {
	return func(opts *options) {
		opts.p2p = immutable.Some(host)
	}
}

func WithRootstore(rootstore corekv.ReaderWriter) Option {
	return func(opts *options) {
		opts.rootstore = immutable.Some(rootstore)
	}
}

type Node struct {
	p2p   immutable.Option[Host]
	store store.Store
}

func New(ctx context.Context, opts ...Option) (*Node, error) {
	var o options
	for _, option := range opts {
		option(&o)
	}

	if !o.rootstore.HasValue() {
		//store, err := badger.NewDatastore("", badgerds.DefaultOptions("").WithInMemory(true)) // todo - we must close if we own the store
		store := memory.NewDatastore(ctx)
		/*if err != nil {
			return nil, err
		}*/

		if !o.txnProvider.HasValue() {
			o.txnProvider = immutable.Some[repository.TxnSource](&storeWrapper{store: store})
		}

		o.rootstore = immutable.Some[corekv.ReaderWriter](store)
	}

	if !o.blockstore.HasValue() {
		o.blockstore = immutable.Some[corekv.ReaderWriter](namespace.Wrap(o.rootstore.Value(), []byte("b/"))) // todo - const
	}

	node := &Node{
		p2p: o.p2p,
		store: *store.New(
			o.blockstore.Value(),
			namespace.Wrap(o.rootstore.Value(), []byte("i/")), // todo - const
			5,
			runtimes.Default(),
		), // todo - opts
	}

	node.store.Init(ctx, o.txnProvider.Value()) // todo - handle panic if `o.rootstore` and `o.txnProvider` is nil

	return node, nil
}

func (n *Node) Add(ctx context.Context, cfg model.Lens) (cid.Cid, error) {
	return n.store.Add(ctx, cfg)
}

func (n *Node) List(ctx context.Context) (map[cid.Cid]model.Lens, error) {
	return n.store.List(ctx)
}

// todo - would we want this?
// a seperate publish command allows broadcasting when users want, and, allows publishing of lenses
// using the collection[version]id - which can only be generated by defra after calling Add.
// Although, defra will probably be reliant on the link system instead of this.
/*func (n *Node) Publish(ctx context.Context, id cid.Cid, topics ...string) error {
	panic("todo")
}rm - unwanted */

// todo - equivelent of Defra's `SyncDocuments` func (sync on demand)
func (n *Node) Sync(ctx context.Context, CIDs []cid.Cid) error {
	strCIDs := make([]string, len(CIDs))
	for i, cid := range CIDs {
		strCIDs[i] = cid.String()
	}

	return n.sync(ctx, strCIDs)
}

func (n *Node) sync(ctx context.Context, CIDs []string) error {
	pubsubReq := &syncRequest{CIDs: CIDs}

	data, err := cbor.Marshal(pubsubReq)
	if err != nil {
		return err
	}

	pubSubRespChan, err := n.p2p.Value().PublishToTopic(ctx, syncTopic, data, true)
	if err != nil {
		return err
	}

	return n.waitAndHandleDocSyncResponses(ctx, CIDs, pubSubRespChan)
}

// waitAndHandleDocSyncResponses handles multiple responses from different peers.
func (n *Node) waitAndHandleDocSyncResponses(
	ctx context.Context,
	CIDs []string,
	pubSubRespChan <-chan PubsubResponse,
) error { // todo - this should probs return a channel instead?  Lots of async stuff and users should control the wait
	results := map[cid.Cid]struct{}{}

loop:
	for {
		select {
		case resp := <-pubSubRespChan:
			if resp.Err != nil {
				// log.ErrorE("Received error response from peer", resp.Err) todo?
				continue
			}

			var reply syncReply
			if err := cbor.Unmarshal(resp.Data, &reply); err != nil {
				// log.ErrorE("Failed to unmarshal doc sync reply", err) todo?
				continue
			}

			for _, item := range reply.Results {
				n.handleDocSyncItem(ctx, item, reply.Sender, results)
			}

			if len(results) >= len(CIDs) {
				break loop
			}

		case <-ctx.Done():
			if len(results) == 0 {
				return ErrTimeoutDocSync
			}
			break loop
		}
	}

	return nil
}

// handleDocSyncItem handles a single document sync item from a peer response.
// It mutates the results map with the document IDs and their corresponding CIDs.
func (n *Node) handleDocSyncItem(
	ctx context.Context,
	item syncItem,
	senderID string,
	results map[cid.Cid]struct{},
) error {
	_, CID, err := cid.CidFromBytes(item.CID)
	if err != nil {
		return err
	}

	if _, exists := results[CID]; exists {
		// we've seen this head already, just skip
		return nil
	} else {
		results[CID] = struct{}{}
	}

	return n.syncDocumentAndMerge(ctx, senderID, CID)
}

// syncDocumentAndMerge synchronizes a document from a remote peer and publishes a merge event.
func (n *Node) syncDocumentAndMerge(
	ctx context.Context,
	senderID string,
	CID cid.Cid,
) error {
	err := n.syncDocumentDAG(ctx, CID)
	if err != nil {
		return err
	}
	return nil //?

	/*
		evt := event.Merge{
			DocID:        docID,
			ByPeer:       senderID,
			FromPeer:     p.host.ID(),
			Cid:          head,
			CollectionID: collectionID,
		}

		return p.db.Merge(ctx, evt)
	*/
}

// syncDocumentDAG synchronizes the DAG for a specific document CID.
func (n *Node) syncDocumentDAG(ctx context.Context, docCid cid.Cid) error {
	linkSys := makeLinkSystem(n.p2p.Value().BlockService())

	nd, err := linkSys.Load(linking.LinkContext{Ctx: ctx}, cidlink.Link{Cid: docCid}, store.ConfigBlockSchemaPrototype)
	if err != nil {
		return err
	}

	switch block := bindnode.Unwrap(nd).(type) {
	case store.ConfigBlock:
		_ = block
		n.syncConfigBlock()

	case store.ModuleBlock:

	case store.LensBlock:

	}

	// todo - add cfg block to repository once all blocks stored

	return nil
	//return syncDAG(ctx, n.p2p.Value().BlockService(), block)
}

func (n *Node) syncConfigBlock() error {
	panic("todo")
}

const syncTopic = "sync"

// docSyncRequest represents a request to synchronize specific documents.
type syncRequest struct {
	CIDs []string `json:"cids"`
}

var ErrTimeoutDocSync = errors.New("timeout while syncing doc")

// docSyncReply represents the response to a document sync request.
type syncReply struct {
	Results []syncItem `json:"results"`
	Sender  string     `json:"sender"`
}

// docSyncItem represents the sync result for a single document.
type syncItem struct {
	CID []byte `json:"cid"`
}

// syncBlockLinkTimeout is the maximum amount of time
// to wait for a block link to be fetched.
var syncBlockLinkTimeout = 5 * time.Second

func makeLinkSystem(blockService blockservice.BlockService) linking.LinkSystem {
	blockStore := &bsrvadapter.Adapter{Wrapped: blockService}

	linkSys := cidlink.DefaultLinkSystem()
	linkSys.SetWriteStorage(blockStore)
	linkSys.SetReadStorage(blockStore)
	linkSys.TrustedStorage = true

	return linkSys
}

// syncDAG synchronizes the DAG starting with the given block
// using the blockservice to fetch remote blocks.
//
// This process walks the entire DAG until the issue below is resolved.
// https://github.com/sourcenetwork/defradb/issues/2722
func syncDAG(ctx context.Context, blockService blockservice.BlockService, block any) error {
	// use a session to make remote fetches more efficient
	ctx = blockservice.ContextWithSession(ctx, blockService)

	linkSystem := makeLinkSystem(blockService)

	// Store the block in the DAG store
	/* todo - don't rely on linkSystem.Store for this - synced lenses need to be added to repo and this wont do that
	_, err := linkSystem.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), bindnode.Wrap(block, store.BlockSchema).Representation())
	if err != nil {
		return err
	}
	*/

	err := loadBlockLinks(ctx, &linkSystem, block)
	if err != nil {
		return err
	}
	return nil
}

func loadBlockLinks(ctx context.Context, linkSys *linking.LinkSystem, block any) error {
	panic("todo")
	/*
		ctxWithCancel, cancel := context.WithCancel(ctx)
		defer cancel()
		var wg sync.WaitGroup
		var asyncErr error
		var asyncErrOnce sync.Once

		setAsyncErr := func(err error) {
			asyncErr = err
			cancel()
		}

		for _, lnk := range block.AllLinks() {
			wg.Add(1)
			go func(lnk cidlink.Link) {
				defer wg.Done()
				if ctxWithCancel.Err() != nil {
					return
				}
				ctxWithTimeout, cancel := context.WithTimeout(ctx, syncBlockLinkTimeout)
				defer cancel()
				nd, err := linkSys.Load(linking.LinkContext{Ctx: ctxWithTimeout}, lnk, coreblock.BlockSchemaPrototype)
				if err != nil {
					asyncErrOnce.Do(func() { setAsyncErr(err) })
					return
				}
				linkBlock, err := coreblock.GetFromNode(nd)
				if err != nil {
					asyncErrOnce.Do(func() { setAsyncErr(err) })
					return
				}

				err = loadBlockLinks(ctx, linkSys, linkBlock)
				if err != nil {
					asyncErrOnce.Do(func() { setAsyncErr(err) })
					return
				}
			}(lnk)
		}

		wg.Wait()

		return asyncErr
	*/
}

// GetLinkPrototype returns the link prototype for the block.
func getLinkPrototype() cidlink.LinkPrototype {
	return cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  uint64(multicodec.Cidv1),
		Codec:    uint64(multicodec.DagCbor),
		MhType:   uint64(multicodec.Sha2_256),
		MhLength: 32,
	}}
}

type storeWrapper struct {
	store corekv.TxnStore
}

var _ repository.TxnSource = (*storeWrapper)(nil)

func (s *storeWrapper) NewTxn(ctx context.Context, readonly bool) (repository.Txn, error) {
	return &txnWrapper{txn: s.store.NewTxn(readonly)}, nil
}

type txnWrapper struct {
	txn corekv.Txn
}

var _ repository.Txn = (*txnWrapper)(nil)

func (s *txnWrapper) ID() uint64 {
	return 0 // todo!!!
}

// Commit finalizes a transaction, attempting to commit it to the Datastore.
// May return an error if the transaction has gone stale. The presence of an
// error is an indication that the data was not committed to the Datastore.
func (s *txnWrapper) Commit(ctx context.Context) error {
	return s.txn.Commit()
}

// Discard throws away changes recorded in a transaction without committing
// them to the underlying Datastore. Any calls made to Discard after Commit
// has been successfully called will have no effect on the transaction and
// state of the Datastore, making it safe to defer.
func (s *txnWrapper) Discard(ctx context.Context) {
	s.txn.Discard()
}

// OnSuccess registers a function to be called when the transaction is committed.
func (s *txnWrapper) OnSuccess(fn func()) {
	// todo!!!
}

// OnError registers a function to be called when the transaction is rolled back.
func (s *txnWrapper) OnError(fn func()) {
	// todo!!!
}

// OnDiscard registers a function to be called when the transaction is discarded.
func (s *txnWrapper) OnDiscard(fn func()) {
	// todo!!!
}
