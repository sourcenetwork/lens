// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/sourcenetwork/lens/host-go/config/model"
)

type schemaDefinition interface {
	// IPLDSchemaBytes returns the IPLD schema representation for the type.
	IPLDSchemaBytes() []byte
}

var (
	// BlockSchema is the IPLD schema type that represents a `block`.
	ConfigBlockSchema          schema.Type
	ConfigBlockSchemaPrototype ipld.NodePrototype
	ModuleBlockSchema          schema.Type
	ModuleBlockSchemaPrototype ipld.NodePrototype
	LensBlockSchema            schema.Type
	LensBlockSchemaPrototype   ipld.NodePrototype
)

func init() {
	ConfigBlockSchema, ConfigBlockSchemaPrototype = mustSetSchema(
		"configBlock",
		&ConfigBlock{},
	)
	ModuleBlockSchema, ModuleBlockSchemaPrototype = mustSetSchema(
		"moduleBlock",
		&ModuleBlock{},
	)
	LensBlockSchema, LensBlockSchemaPrototype = mustSetSchema(
		"lensBlock",
		&LensBlock{},
	)
}
func mustSetSchema(schemaName string, schemas ...schemaDefinition) (schema.Type, ipld.NodePrototype) { // todo - rm schema name param
	schemaBytes := make([][]byte, 0, len(schemas))
	for _, s := range schemas {
		schemaBytes = append(schemaBytes, s.IPLDSchemaBytes())
	}

	ts, err := ipld.LoadSchemaBytes(bytes.Join(schemaBytes, nil))
	if err != nil {
		panic(err)
	}

	schemaType := ts.TypeByName(schemaName)

	// Calling bindnode.Prototype here ensure that [Block] and all the types it contains
	// are compatible with the IPLD schema defined by [schemaDefinition].
	// If [Block] and [schemaType] do not match, this will panic.
	proto := bindnode.Prototype(schemas[0], schemaType)

	return schemaType, proto.Representation()
}

type block interface {
	Bytes() ([]byte, error)
	GenerateNode() ipld.Node
}

type keyValue struct {
	Key   string
	Value string
}

type ConfigBlock struct {
	Modules []datamodel.Link
}

var _ schemaDefinition = (*ConfigBlock)(nil)
var _ block = (*ConfigBlock)(nil)

type ModuleBlock struct {
	Inverse   bool
	Arguments []keyValue
	Lens      datamodel.Link
}

var _ schemaDefinition = (*ModuleBlock)(nil)
var _ block = (*ModuleBlock)(nil)

type LensBlock struct {
	WasmBytes []byte
}

var _ schemaDefinition = (*LensBlock)(nil)
var _ block = (*LensBlock)(nil)

func (b *ConfigBlock) IPLDSchemaBytes() []byte {
	return []byte(`
		type configBlock struct {
			modules [Link]
		}
	`)
}

func (b *ModuleBlock) IPLDSchemaBytes() []byte {
	return []byte(`
		type moduleBlock struct {
			inverse   Bool
			arguments [KeyValue]
			lens Link
		}
		type KeyValue struct {
			key String
			value String
		} 
	`)
}

func (b *LensBlock) IPLDSchemaBytes() []byte {
	return []byte(`
		type lensBlock struct {
			wasmBytes Bytes
		}
	`)
}

func (b *LensBlock) Bytes() ([]byte, error) {
	return marshalNode(b, LensBlockSchema)
}

func (b *ModuleBlock) Bytes() ([]byte, error) {
	return marshalNode(b, ModuleBlockSchema)
}

func (b *ConfigBlock) Bytes() ([]byte, error) {
	return marshalNode(b, ConfigBlockSchema)
}

func (b *LensBlock) GenerateNode() ipld.Node {
	return bindnode.Wrap(b, LensBlockSchema).Representation()
}

func (b *ModuleBlock) GenerateNode() ipld.Node {
	return bindnode.Wrap(b, ModuleBlockSchema).Representation()
}

func (b *ConfigBlock) GenerateNode() ipld.Node {
	return bindnode.Wrap(b, ConfigBlockSchema).Representation()
}

func (b *ConfigBlock) ToLensModel(ctx context.Context, linkSys linking.LinkSystem) (model.Lens, error) {
	result := model.Lens{}
	for _, moduleLink := range b.Modules {
		moduleNode, err := linkSys.Load(ipld.LinkContext{Ctx: ctx}, moduleLink, ModuleBlockSchemaPrototype)
		if err != nil {
			return model.Lens{}, err
		}
		moduleBlock := bindnode.Unwrap(moduleNode).(*ModuleBlock)

		var arguments map[string]any
		if moduleBlock.Arguments != nil {
			arguments = make(map[string]any, len(moduleBlock.Arguments))
			for _, kv := range moduleBlock.Arguments {
				var value any
				err := json.Unmarshal([]byte(kv.Value), &value)
				if err != nil {
					return model.Lens{}, err
				}

				arguments[kv.Key] = value
			}
		}

		lensNode, err := linkSys.Load(ipld.LinkContext{Ctx: ctx}, moduleBlock.Lens, LensBlockSchemaPrototype)
		if err != nil {
			return model.Lens{}, err
		}
		lensBlock := bindnode.Unwrap(lensNode).(*LensBlock)

		var path string
		if len(lensBlock.WasmBytes) != 0 {
			path = "data://" + string(lensBlock.WasmBytes) // todo - nicer
		}

		result.Lenses = append(result.Lenses, model.LensModule{
			Path:      path,
			Inverse:   moduleBlock.Inverse,
			Arguments: arguments,
		})
	}

	return result, nil
}

func NewConfigBlock(model model.Lens) *ConfigBlock {
	panic("todo")
}

// marshalNode encodes a ipld node using CBOR encoding.
func marshalNode(node any, schema schema.Type) ([]byte, error) {
	b, err := ipld.Marshal(dagcbor.Encode, node, schema)
	if err != nil {
		return nil, err
	}
	return b, nil
}
