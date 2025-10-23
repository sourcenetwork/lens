// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package linking

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

type MultiBlock struct {
	Children []datamodel.Link
}

var _ schemaDefinition = (*MultiBlock)(nil)

func (b *MultiBlock) IPLDSchemaBytes() []byte {
	return []byte(`
		type multiBlock struct {
			children [Link]
		}
	`)
}

func (b *MultiBlock) generateNode() ipld.Node {
	return bindnode.Wrap(b, MultiBlockSchema).Representation()
}

type ChunkBlock struct {
	Chunk []byte
}

var _ schemaDefinition = (*MultiBlock)(nil)

func (b *ChunkBlock) IPLDSchemaBytes() []byte {
	return []byte(`
		type chunkBlock struct {
			chunk Bytes
		}
	`)
}

func (b *ChunkBlock) generateNode() ipld.Node {
	return bindnode.Wrap(b, ChunkBlockSchema).Representation()
}

var (
	MultiBlockSchema          schema.Type
	MultiBlockSchemaPrototype ipld.NodePrototype
	ChunkBlockSchema          schema.Type
	ChunkBlockSchemaPrototype ipld.NodePrototype
)

func init() {
	MultiBlockSchema, MultiBlockSchemaPrototype = mustSetSchema(
		"multiBlock",
		&MultiBlock{},
	)
	ChunkBlockSchema, ChunkBlockSchemaPrototype = mustSetSchema(
		"chunkBlock",
		&ChunkBlock{},
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
