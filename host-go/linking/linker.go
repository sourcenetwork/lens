// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package linking

import (
	"bytes"
	"context"
	"slices"
	"strings"

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
		maxSize: 100000, //todo :)
	}
}

func (l *LinkSystem) Store(ctx context.Context, node ipld.Node) (datamodel.Link, error) {
	enc, err := l.linking.EncoderChooser(getLinkPrototype())
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	enc(node, &b)
	bts := b.Bytes()

	chunkLinks := []datamodel.Link{}
	if len(bts) > l.maxSize {
		for byteChunk := range slices.Chunk(bts, l.maxSize) { // todo - needs to be smaller as the block has overhead
			chunkBlock := ChunkBlock{
				Chunk: byteChunk,
			}

			chunkLink, err := l.linking.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), chunkBlock.generateNode())
			if err != nil {
				return nil, err
			}
			chunkLinks = append(chunkLinks, chunkLink)
		}
	}

	if len(chunkLinks) == 1 {
		return chunkLinks[0], nil
	}

	multiBlock := MultiBlock{
		Children: chunkLinks,
	}

	return l.linking.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), multiBlock.generateNode())
}

func (l *LinkSystem) Load(
	ctx context.Context,
	link datamodel.Link,
	prototype datamodel.NodePrototype,
) (datamodel.Node, error) {
	node, err := l.linking.Load(linking.LinkContext{Ctx: ctx}, link, prototype)
	if err != nil {
		if strings.Contains(err.Error(), "invalid key: \"children\" is not a field in type") {
			multiNode, err := l.linking.Load(linking.LinkContext{Ctx: ctx}, link, MultiBlockSchemaPrototype)
			if err != nil {
				return nil, err
			}
			multiBlock := bindnode.Unwrap(multiNode).(*MultiBlock)

			childBytes := [][]byte{}
			for _, child := range multiBlock.Children {
				childNode, err := l.linking.Load(linking.LinkContext{Ctx: ctx}, child, ChunkBlockSchemaPrototype)
				if err != nil {
					return nil, err
				}

				childBlock := bindnode.Unwrap(childNode).(*ChunkBlock)

				childBytes = append(childBytes, childBlock.Chunk)
			}

			b := bytes.Join(childBytes, []byte{})
			var buf bytes.Buffer
			_, err = buf.Write(b)
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
		return nil, err
	}
	return node, err
}

func getLinkPrototype() cidlink.LinkPrototype {
	return cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  uint64(multicodec.Cidv1),
		Codec:    uint64(multicodec.DagCbor),
		MhType:   uint64(multicodec.Sha2_256),
		MhLength: 32,
	}}
}
