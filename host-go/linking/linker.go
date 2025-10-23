// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package linking

import (
	"bytes"
	"context"
	"slices"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/multiformats/go-multicodec"
)

type LinkSystem struct {
	maxSize int
	linking linking.LinkSystem
}

func New(linking linking.LinkSystem) *LinkSystem { //todo - might as well just take a blck store
	return &LinkSystem{
		linking: linking,
		maxSize: 3000000, //todo :)
	}
}

func (l *LinkSystem) Store(ctx context.Context, node ipld.Node) (datamodel.Link, error) {
	encoder, err := l.linking.EncoderChooser(getLinkPrototype())
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	encoder(node, &buf)
	nodeBytes := buf.Bytes()

	var multiBlock Multi
	if len(nodeBytes) > l.maxSize {
		chunkLinks := []datamodel.Link{}
		for byteChunk := range slices.Chunk(nodeBytes, l.maxSize) {
			chunkBlock := Chunk{
				Chunk: byteChunk,
			}

			chunkLink, err := l.linking.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), chunkBlock.generateNode())
			if err != nil {
				return nil, err
			}
			chunkLinks = append(chunkLinks, chunkLink)
		}
		multiBlock = Multi{
			Chunks: &Chunks{
				Chunks: chunkLinks,
			},
		}
	} else {
		multiBlock = Multi{
			Chunk: &Chunk{
				Chunk: nodeBytes,
			},
		}
	}

	return l.linking.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), multiBlock.generateNode())
}

func (l *LinkSystem) Load(
	ctx context.Context,
	link datamodel.Link,
	prototype datamodel.NodePrototype,
) (datamodel.Node, error) {
	node, err := l.linking.Load(linking.LinkContext{Ctx: ctx}, link, MultiBlockSchemaPrototype)
	if err != nil {
		return nil, err
	}
	multiBlock := bindnode.Unwrap(node).(*Multi)

	var blockBytes []byte
	if multiBlock.Chunks != nil {
		chunkBytes := [][]byte{}
		for _, chunk := range multiBlock.Chunks.Chunks {
			chunkNode, err := l.linking.Load(linking.LinkContext{Ctx: ctx}, chunk, ChunkBlockSchemaPrototype)
			if err != nil {
				return nil, err
			}

			chunkBlock := bindnode.Unwrap(chunkNode).(*Chunk)

			chunkBytes = append(chunkBytes, chunkBlock.Chunk)
		}

		blockBytes = bytes.Join(chunkBytes, []byte{})
	} else {
		blockBytes = multiBlock.Chunk.Chunk
	}

	var buf bytes.Buffer
	_, err = buf.Write(blockBytes)
	if err != nil {
		return nil, err
	}

	decoder, err := l.linking.DecoderChooser(link)
	if err != nil {
		return nil, err
	}

	builder := prototype.NewBuilder()
	err = decoder(builder, &buf)
	if err != nil {
		return nil, err
	}

	return builder.Build(), nil
}

func getLinkPrototype() cidlink.LinkPrototype {
	return cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  uint64(multicodec.Cidv1),
		Codec:    uint64(multicodec.DagCbor),
		MhType:   uint64(multicodec.Sha2_256),
		MhLength: 32,
	}}
}
