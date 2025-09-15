// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"
	"encoding/json"
	"sort"

	cid "github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/engine"
)

type schemaDefinition interface {
	// IPLDSchemaBytes returns the IPLD schema representation for the type.
	IPLDSchemaBytes() []byte
}

var (
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

// KeyValue represents a simple key-value pairing.
type KeyValue struct {
	// Key is the identifier of this key-value pairing.
	Key string

	// Value is the JSON serialized string representation of the value of this pairing.
	Value string
}

// ConfigBlock represents a Lens configuration.
//
// It may contain zero to many Lens ModuleBlocks.
type ConfigBlock struct {
	// Modules is a set of `ModuleBlock`s that form this configuration.
	Modules []datamodel.Link
}

var _ schemaDefinition = (*ConfigBlock)(nil)

// ModuleBlock represents the configuration of a single lens.
//
// Many of these form a Lens ConfigBlock.
type ModuleBlock struct {
	// Inverse, if true, indicates that the `Inverse` function of the wasm binary should be called
	// to transform data instead of the defaul `Transform` function.
	Inverse bool

	// Arguments is the ordered, json serialized set of parameters to provide to the lens binary
	// before execution.
	Arguments []KeyValue

	// Lens is a link to the LensBlock containing the wasm binary that will be executed using
	// the above parameters.
	Lens datamodel.Link
}

var _ schemaDefinition = (*ModuleBlock)(nil)

// LensBlock represents the wasm binary of a compiled lens.
//
// A single LensBlock maybe referenced by an unlimited number of lens modules/configurations.
type LensBlock struct {
	// WasmBytes is the executable wasm binary of the lens.
	WasmBytes []byte
}

var _ schemaDefinition = (*LensBlock)(nil)

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

func (b *LensBlock) generateNode() ipld.Node {
	return bindnode.Wrap(b, LensBlockSchema).Representation()
}

func (b *ModuleBlock) generateNode() ipld.Node {
	return bindnode.Wrap(b, ModuleBlockSchema).Representation()
}

func (b *ConfigBlock) generateNode() ipld.Node {
	return bindnode.Wrap(b, ConfigBlockSchema).Representation()
}

func loadLensModel(ctx context.Context, linkSys *linking.LinkSystem, cid cid.Cid) (model.Lens, error) {
	configNode, err := linkSys.Load(linking.LinkContext{Ctx: ctx}, cidlink.Link{Cid: cid}, ConfigBlockSchemaPrototype)
	if err != nil {
		return model.Lens{}, err
	}

	configBlock := bindnode.Unwrap(configNode).(*ConfigBlock)

	result := model.Lens{}
	for _, moduleLink := range configBlock.Modules {
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
			path = "data://" + string(lensBlock.WasmBytes)
		}

		result.Lenses = append(result.Lenses, model.LensModule{
			Path:      path,
			Inverse:   moduleBlock.Inverse,
			Arguments: arguments,
		})
	}

	return result, nil
}

func writeConfigBlock(
	ctx context.Context,
	linkSys *linking.LinkSystem,
	cfg model.Lens,
) (datamodel.Link, error) {
	moduleLinks := make([]datamodel.Link, 0, len(cfg.Lenses))

	for _, module := range cfg.Lenses {
		wasmBytes, err := engine.GetWasmBytes(module.Path)
		if err != nil {
			return nil, err
		}
		lensBlock := LensBlock{
			WasmBytes: wasmBytes,
		}

		lensLink, err := linkSys.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), lensBlock.generateNode())
		if err != nil {
			return nil, err
		}

		orderedArgs := make([]KeyValue, len(module.Arguments))
		for argKey, value := range module.Arguments {
			// The linking system does not tolerate the `any` type, so we store them as json.  The arguments
			// are passed to the lenses as json anyway so we can be confident in the safety of this.
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}

			orderedArgs = append(orderedArgs, KeyValue{
				Key:   argKey,
				Value: string(jsonBytes),
			})
		}
		// The blocks must be deterministic, so sort the arguments by key before storing
		sort.Slice(orderedArgs, func(i, j int) bool { return orderedArgs[i].Key < orderedArgs[j].Key })

		moduleBlock := ModuleBlock{
			Inverse:   module.Inverse,
			Arguments: orderedArgs,
			Lens:      lensLink,
		}

		moduleLink, err := linkSys.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), moduleBlock.generateNode())
		if err != nil {
			return nil, err
		}

		moduleLinks = append(moduleLinks, moduleLink)
	}

	configBlock := ConfigBlock{
		Modules: moduleLinks,
	}
	return linkSys.Store(linking.LinkContext{Ctx: ctx}, getLinkPrototype(), configBlock.generateNode())
}
