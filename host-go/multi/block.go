// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package multi

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

type schemaDefinition interface {
	// IPLDSchemaBytes returns the IPLD schema representation for the type.
	IPLDSchemaBytes() []byte
}

type Multi struct {
	Chunks *Chunks
	Chunk  *Chunk
}

var _ schemaDefinition = (*Multi)(nil)

func (b *Multi) IPLDSchemaBytes() []byte {
	return []byte(`
		type multi union {
			| chunks "chunks"
			| chunk "chunk"
		} representation keyed

		type chunks struct {
			chunks [Link]
		}

		type chunk struct {
			chunk Bytes
		}
	`)
}

func (b *Multi) generateNode() ipld.Node {
	return bindnode.Wrap(b, MultiBlockSchema).Representation()
}

type Chunk struct {
	Chunk []byte
}

var _ schemaDefinition = (*Multi)(nil)

func (b *Chunk) IPLDSchemaBytes() []byte {
	return []byte(`
		type chunk struct {
			chunk Bytes
		}
	`)
}

func (b *Chunk) generateNode() ipld.Node {
	return bindnode.Wrap(b, ChunkBlockSchema).Representation()
}

type Chunks struct {
	Chunks []datamodel.Link
}

var _ schemaDefinition = (*Multi)(nil)

func (b *Chunks) IPLDSchemaBytes() []byte {
	return []byte(`
		type chunks struct {
			chunks [Link]
		}
	`)
}

func (b *Chunks) generateNode() ipld.Node {
	return bindnode.Wrap(b, ChunksBlockSchema).Representation()
}

var (
	MultiBlockSchema           schema.Type
	MultiBlockSchemaPrototype  ipld.NodePrototype
	ChunkBlockSchema           schema.Type
	ChunkBlockSchemaPrototype  ipld.NodePrototype
	ChunksBlockSchema          schema.Type
	ChunksBlockSchemaPrototype ipld.NodePrototype
)

func init() {
	MultiBlockSchema, MultiBlockSchemaPrototype = mustSetSchema(
		"multi",
		&Multi{},
	)
	ChunkBlockSchema, ChunkBlockSchemaPrototype = mustSetSchema(
		"chunk",
		&Chunk{},
	)
	ChunksBlockSchema, ChunksBlockSchemaPrototype = mustSetSchema(
		"chunks",
		&Chunks{},
	)
}

func mustSetSchema(schemaName string, schema schemaDefinition) (schema.Type, ipld.NodePrototype) {
	ts, err := ipld.LoadSchemaBytes(schema.IPLDSchemaBytes())
	if err != nil {
		panic(err)
	}

	schemaType := ts.TypeByName(schemaName)

	// Calling bindnode.Prototype here ensure that [Block] and all the types it contains
	// are compatible with the IPLD schema defined by [schemaDefinition].
	// If [Block] and [schemaType] do not match, this will panic.
	proto := bindnode.Prototype(schema, schemaType)

	return schemaType, proto.Representation()
}
