// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows && !js

package wasmer

import (
	"encoding/json"
	"errors"
	"io"
	"math"

	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/engine/pipes"

	"github.com/wasmerio/wasmer-go/wasmer"
)

const Name string = "wasmer"

type wRuntime struct {
	store *wasmer.Store
}

var _ module.Runtime = (*wRuntime)(nil)

func New() module.Runtime {
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)
	return &wRuntime{
		store: store,
	}
}

type wModule struct {
	runtime *wRuntime
	module  *wasmer.Module
}

var _ module.Module = (*wModule)(nil)

func (rt *wRuntime) Name() string {
	return Name
}

func (rt *wRuntime) NewModule(wasmBytes []byte) (module.Module, error) {
	module, err := wasmer.NewModule(rt.store, wasmBytes)
	if err != nil {
		return nil, err
	}

	return &wModule{
		runtime: rt,
		module:  module,
	}, nil
}

// instanceHandles holds all per-instance wasmer resources. Storing them behind a pointer
// lets Reset swap the contents in-place so that the Alloc/Transform/Memory closures — which
// all capture the same *instanceHandles — automatically see the fresh instance.
type instanceHandles struct {
	instance  *wasmer.Instance
	memory    *wasmer.Memory
	alloc     *wasmer.Function
	transform *wasmer.Function
}

// newInstanceHandles instantiates mod against the shared store, wires the nextFnPtr import,
// and applies set_param when params is non-empty.
func newInstanceHandles(
	store *wasmer.Store,
	mod *wasmer.Module,
	fnName string,
	params map[string]any,
	nextFnPtr *func() module.MemSize,
) (_ *instanceHandles, err error) {
	importObject := wasmer.NewImportObject()

	// Register the `lens.next` function required as an import for wasm lens modules
	importObject.Register(
		"lens",
		map[string]wasmer.IntoExtern{
			"next": wasmer.NewFunction(
				store,
				wasmer.NewFunctionType(
					wasmer.NewValueTypes(),
					// Warning: wasmer requires a concrete type here and as such this line is coupled to the module's runtime
					wasmer.NewValueTypes(wasmer.I32),
				),
				func(v []wasmer.Value) ([]wasmer.Value, error) {
					r := (*nextFnPtr)()
					return []wasmer.Value{wasmer.NewI32(r)}, nil
				},
			),
		},
	)

	instance, err := wasmer.NewInstance(mod, importObject)
	if err != nil {
		return nil, err
	}
	// Close the instance (freeing its C-allocated linear memory) if we fail before
	// returning successfully; the shared store is left intact (issue #158).
	defer func() {
		if err != nil {
			instance.Close()
		}
	}()

	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return nil, err
	}

	alloc, err := instance.Exports.GetRawFunction("alloc")
	if err != nil {
		return nil, err
	}

	transform, err := instance.Exports.GetRawFunction(fnName)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		setParam, err := instance.Exports.GetRawFunction("set_param")
		if err != nil {
			return nil, err
		}

		sourceBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		index, err := alloc.Call(module.TypeIdSize + module.MemSize(len(sourceBytes)) + module.LenSize)
		if err != nil {
			return nil, err
		}

		mem := module.NewBytesMemory(memory.Data())
		w := io.NewOffsetWriter(mem, int64(index.(module.MemSize)))

		err = pipes.WriteItem(w, module.JSONTypeID, sourceBytes)
		if err != nil {
			return nil, err
		}

		index, err = setParam.Call(index)
		if err != nil {
			return nil, err
		}

		r := io.NewSectionReader(mem, int64(index.(module.MemSize)), math.MaxInt64)

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
		instance:  instance,
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

	handles, err := newInstanceHandles(m.runtime.store, m.module, functionName, params, &nextFn)
	if err != nil {
		return module.Instance{}, err
	}

	var resetErr error

	return module.Instance{
		Alloc: func(u module.MemSize) (module.MemSize, error) {
			if resetErr != nil {
				return 0, resetErr
			}
			r, err := handles.alloc.Call(u)
			if err != nil {
				return 0, err
			}
			return r.(module.MemSize), err
		},
		Transform: func(next func() module.MemSize) (module.MemSize, error) {
			if resetErr != nil {
				return 0, resetErr
			}
			// By assigning the next function immediately prior to calling transform, we allow multiple
			// pipeline stages to share the same wasm instance - provided they are not called concurrently.
			// This also allows module state to be shared across pipeline stages.
			nextFn = next
			r, err := handles.transform.Call()
			if err != nil {
				return 0, err
			}
			return r.(module.MemSize), err
		},
		Memory: func() module.Memory {
			return module.NewBytesMemory(handles.memory.Data())
		},
		Reset: func() {
			nextFn = func() module.MemSize { return 0 }
			newHandles, err := newInstanceHandles(m.runtime.store, m.module, functionName, params, &nextFn)
			if err != nil {
				resetErr = err
				return
			}
			resetErr = nil
			oldInstance := handles.instance
			*handles = *newHandles
			// Free the old instance's C-allocated linear memory now; the GC finalizer would do
			// it eventually but wasm memory exerts no Go heap pressure (issue #159).
			oldInstance.Close()
		},
		OwnedBy: handles,
	}, nil
}
