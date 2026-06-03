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
	"github.com/tetratelabs/wazero/api"
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

// instanceHandles holds all per-instance wazero resources. Storing them behind a pointer
// lets Reset swap the contents in-place so that the Alloc/Transform/Memory closures — which
// all capture the same *instanceHandles — automatically see the fresh runtime.
type instanceHandles struct {
	runtime   wazero.Runtime
	instance  api.Module
	memory    api.Memory
	alloc     api.Function
	transform api.Function
}

// newInstanceHandles creates a fresh wazero runtime (reusing the shared compilation cache),
// wires the nextFnPtr import, instantiates the module, and applies set_param when params is
// non-empty.
func newInstanceHandles(
	compilationCache wazero.CompilationCache,
	moduleBytes []byte,
	fnName string,
	params map[string]any,
	nextFnPtr *func() module.MemSize,
) (_ *instanceHandles, err error) {
	ctx := context.TODO()
	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(compilationCache)
	rt := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	// Close the runtime (and the instance/memory it owns) if we fail before returning
	// successfully, so an errored NewInstance/Reset does not leak it (issue #158). The
	// close error is discarded so it cannot mask the construction error we return.
	defer func() {
		if err != nil {
			_ = rt.Close(ctx)
		}
	}()

	_, err = rt.NewHostModuleBuilder("lens").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context) module.MemSize {
			return (*nextFnPtr)()
		}).
		Export("next").
		Instantiate(ctx)
	if err != nil {
		return nil, err
	}

	inst, err := rt.Instantiate(ctx, moduleBytes)
	if err != nil {
		return nil, err
	}

	memory := inst.ExportedMemory("memory")
	if memory == nil {
		return nil, fmt.Errorf("Export `%s` does not exist", "memory")
	}

	alloc := inst.ExportedFunction("alloc")
	if alloc == nil {
		return nil, fmt.Errorf("Export `%s` does not exist", "alloc")
	}

	transform := inst.ExportedFunction(fnName)
	if transform == nil {
		return nil, fmt.Errorf("Export `%s` does not exist", fnName)
	}

	if len(params) > 0 {
		setParam := inst.ExportedFunction("set_param")
		if setParam == nil {
			return nil, fmt.Errorf("Export `%s` does not exist", "set_param")
		}

		sourceBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		index, err := alloc.Call(ctx, uint64(module.TypeIdSize+module.MemSize(len(sourceBytes))+module.LenSize))
		if err != nil {
			return nil, err
		}

		mem := newMemory(memory)
		w := io.NewOffsetWriter(mem, int64(index[0]))

		err = pipes.WriteItem(w, module.JSONTypeID, sourceBytes)
		if err != nil {
			return nil, err
		}

		index, err = setParam.Call(ctx, index[0])
		if err != nil {
			return nil, err
		}
		r := io.NewSectionReader(mem, int64(index[0]), math.MaxInt64)

		// The `set_param` wasm function may error, in which case the error needs to be retrieved
		// from memory using `pipes.GetItem`.
		id, data, err := pipes.ReadItem(r)
		if id.IsError() {
			return nil, errors.New(string(data))
		}
		if err != nil {
			return nil, err
		}
	}

	return &instanceHandles{
		runtime:   rt,
		instance:  inst,
		memory:    memory,
		alloc:     alloc,
		transform: transform,
	}, nil
}

func (m *wModule) NewInstance(functionName string, paramSets ...map[string]any) (module.Instance, error) {
	params := map[string]any{}
	// Merge the param sets into a single map in case more than
	// one map is provided.
	for _, paramSet := range paramSets {
		for key, value := range paramSet {
			params[key] = value
		}
	}

	var nextFn func() module.MemSize = func() module.MemSize { return 0 }

	handles, err := newInstanceHandles(m.compilationCache, m.moduleBytes, functionName, params, &nextFn)
	if err != nil {
		return module.Instance{}, err
	}

	ctx := context.TODO()
	var resetErr error

	return module.Instance{
		Alloc: func(u module.MemSize) (module.MemSize, error) {
			if resetErr != nil {
				return 0, resetErr
			}
			r, err := handles.alloc.Call(ctx, uint64(u))
			if err != nil {
				return 0, err
			}
			return module.MemSize(r[0]), nil
		},
		Transform: func(next func() module.MemSize) (module.MemSize, error) {
			if resetErr != nil {
				return 0, resetErr
			}
			// By assigning the next function immediately prior to calling transform, we allow multiple
			// pipeline stages to share the same wasm instance - provided they are not called concurrently.
			// This also allows module state to be shared across pipeline stages.
			nextFn = next
			r, err := handles.transform.Call(ctx)
			if err != nil {
				return 0, err
			}
			return module.MemSize(r[0]), nil
		},
		Memory: func() module.Memory {
			return newMemory(handles.memory)
		},
		Reset: func() {
			nextFn = func() module.MemSize { return 0 }
			newHandles, err := newInstanceHandles(m.compilationCache, m.moduleBytes, functionName, params, &nextFn)
			if err != nil {
				resetErr = err
				return
			}
			resetErr = nil
			oldRuntime := handles.runtime
			*handles = *newHandles
			// Free the old runtime (and its instance and linear memory) now rather than waiting
			// for the GC, mirroring the wasmtime fix (issue #159). The close error is intentionally
			// discarded, not routed to resetErr: it concerns cleanup of the old, already-replaced
			// runtime and must not fail Alloc/Transform on the freshly-built instance.
			_ = oldRuntime.Close(ctx)
		},
		OwnedBy: handles,
	}, nil
}
