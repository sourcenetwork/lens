// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !js

package wazero

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/engine/pipes"

	"github.com/tetratelabs/wazero"
)

const Name string = "wazero"

type wRuntime struct {
	compilationCache wazero.CompilationCache
}

var _ module.Runtime = (*wRuntime)(nil)

// New creates a new wazero wasm runtime.
//
// WARNING: This runtime does not current allow for instance reuse within a single pipeline.
// Please see https://github.com/sourcenetwork/lens/issues/71 for more info.
func New() module.Runtime {
	return &wRuntime{
		compilationCache: wazero.NewCompilationCache(),
	}
}

type wModule struct {
	compilationCache wazero.CompilationCache
	moduleBytes      []byte
}

var _ module.Module = (*wModule)(nil)

func (rt *wRuntime) Name() string {
	return Name
}

func (rt *wRuntime) NewModule(wasmBytes []byte) (module.Module, error) {
	return &wModule{
		compilationCache: rt.compilationCache,
		moduleBytes:      wasmBytes,
	}, nil
}

func (m *wModule) NewInstance(functionName string, paramSets ...map[string]any) (module.Instance, error) {
	ctx := context.TODO()
	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(m.compilationCache)
	runtime := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	var nextFunction = func() module.MemSize { return 0 }
	_, err := runtime.NewHostModuleBuilder("lens").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context) module.MemSize {
			return nextFunction()
		}).
		Export("next").
		Instantiate(ctx)
	if err != nil {
		return module.Instance{}, err
	}

	instance, err := runtime.Instantiate(ctx, m.moduleBytes)
	if err != nil {
		return module.Instance{}, err
	}

	memory := instance.ExportedMemory("memory")
	if memory == nil {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "memory"))
	}

	alloc := instance.ExportedFunction("alloc")
	if alloc == nil {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "alloc"))
	}

	transform := instance.ExportedFunction(functionName)
	if transform == nil {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", functionName))
	}

	params := map[string]any{}
	// Merge the param sets into a single map in case more than
	// one map is provided.
	for _, paramSet := range paramSets {
		for key, value := range paramSet {
			params[key] = value
		}
	}

	if len(params) > 0 {
		setParam := instance.ExportedFunction("set_param")
		if err != nil {
			return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "set_param"))
		}

		sourceBytes, err := json.Marshal(params)
		if err != nil {
			return module.Instance{}, err
		}

		index, err := alloc.Call(ctx, uint64(module.TypeIdSize+module.MemSize(len(sourceBytes))+module.LenSize))
		if err != nil {
			return module.Instance{}, err
		}

		mem := newMemory(memory)
		w := io.NewOffsetWriter(mem, int64(index[0]))

		err = pipes.WriteItem(w, module.JSONTypeID, sourceBytes)
		if err != nil {
			return module.Instance{}, err
		}

		index, err = setParam.Call(ctx, index[0])
		if err != nil {
			return module.Instance{}, err
		}
		r := io.NewSectionReader(mem, int64(index[0]), math.MaxInt64)

		// The `set_param` wasm function may error, in which case the error needs to be retrieved
		// from memory using `pipes.GetItem`.
		id, data, err := pipes.ReadItem(r)
		if id.IsError() {
			return module.Instance{}, errors.New(string(data))
		}
		if err != nil {
			return module.Instance{}, err
		}
	}

	initialState, _ := memory.Read(0, memory.Size())

	return module.Instance{
		Alloc: func(u module.MemSize) (module.MemSize, error) {
			r, err := alloc.Call(ctx, uint64(u))
			if err != nil {
				return 0, err
			}
			return module.MemSize(r[0]), nil
		},
		Transform: func(next func() module.MemSize) (module.MemSize, error) {
			// By assigning the next function immediately prior to calling transform, we allow multiple
			// pipeline stages to share the same wasm instance - provided they are not called concurrently.
			// This also allows module state to be shared across pipeline stages.
			nextFunction = next
			r, err := transform.Call(ctx)
			if err != nil {
				return 0, err
			}
			return module.MemSize(r[0]), nil
		},
		Memory: func() module.Memory {
			return newMemory(memory)
		},
		Reset: func() {
			initialLen := uint32(len(initialState))
			currentLen := memory.Size()
			memory.Write(0, initialState)

			for i := initialLen; i < currentLen; i++ {
				memory.WriteByte(i, 0)
			}
		},
		OwnedBy: instance,
	}, nil
}
