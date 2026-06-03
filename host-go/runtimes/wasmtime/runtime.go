// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows && !js

package wasmtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/engine/pipes"

	"github.com/bytecodealliance/wasmtime-go/v35"
)

const Name string = "wasmtime"

type wRuntime struct {
	engine *wasmtime.Engine
}

var _ module.Runtime = (*wRuntime)(nil)

func New() module.Runtime {
	return &wRuntime{
		engine: wasmtime.NewEngine(),
	}
}

type wModule struct {
	rt     *wRuntime
	module *wasmtime.Module
}

var _ module.Module = (*wModule)(nil)

func (rt *wRuntime) Name() string {
	return Name
}

func (rt *wRuntime) NewModule(wasmBytes []byte) (module.Module, error) {
	mod, err := wasmtime.NewModule(rt.engine, wasmBytes)
	if err != nil {
		return nil, err
	}

	return &wModule{
		rt:     rt,
		module: mod,
	}, nil
}

// instanceHandles holds all per-instance wasmtime resources. Storing them behind a pointer
// lets Reset swap the contents in-place so that the Alloc/Transform/Memory closures — which
// all capture the same *instanceHandles — automatically see the fresh store.
type instanceHandles struct {
	store     *wasmtime.Store
	memory    *wasmtime.Memory
	alloc     *wasmtime.Func
	transform *wasmtime.Func
}

// newInstanceHandles creates a fresh wasmtime store, instantiates mod, wires the nextFnPtr
// import, and applies set_param when params is non-empty.
func newInstanceHandles(
	eng *wasmtime.Engine,
	mod *wasmtime.Module,
	fnName string,
	params map[string]any,
	nextFnPtr *func() module.MemSize,
) (_ *instanceHandles, err error) {
	store := wasmtime.NewStore(eng)
	// Close the store (freeing its instance and C-allocated linear memory) if we fail
	// before returning successfully, so an errored NewInstance/Reset does not leak it (issue #158).
	defer func() {
		if err != nil {
			store.Close()
		}
	}()

	nextImport := wasmtime.WrapFunc(store, func() module.MemSize {
		return (*nextFnPtr)()
	})

	inst, err := wasmtime.NewInstance(store, mod, []wasmtime.AsExtern{nextImport})
	if err != nil {
		return nil, err
	}

	memExport := inst.GetExport(store, "memory")
	if memExport == nil {
		return nil, errors.New(fmt.Sprintf("Export `%s` does not exist", "memory"))
	}
	memory := memExport.Memory()
	if memory == nil {
		return nil, errors.New(fmt.Sprintf("Export `%s` does not exist", "memory"))
	}

	alloc := inst.GetFunc(store, "alloc")
	if alloc == nil {
		return nil, errors.New(fmt.Sprintf("Export `%s` does not exist", "alloc"))
	}

	transform := inst.GetFunc(store, fnName)
	if transform == nil {
		return nil, errors.New(fmt.Sprintf("Export `%s` does not exist", fnName))
	}

	if len(params) > 0 {
		setParam := inst.GetFunc(store, "set_param")
		if setParam == nil {
			return nil, errors.New(fmt.Sprintf("Export `%s` does not exist", "set_param"))
		}

		sourceBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		index, err := alloc.Call(store, module.TypeIdSize+module.MemSize(len(sourceBytes))+module.LenSize)
		if err != nil {
			return nil, err
		}

		mem := module.NewBytesMemory(memory.UnsafeData(store))
		w := io.NewOffsetWriter(mem, int64(index.(module.MemSize)))

		err = pipes.WriteItem(w, module.JSONTypeID, sourceBytes)
		if err != nil {
			return nil, err
		}

		index, err = setParam.Call(store, index)
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
		store:     store,
		memory:    memory,
		alloc:     alloc,
		transform: transform,
	}, nil
}

func (m *wModule) NewInstance(functionName string, paramSets ...map[string]any) (module.Instance, error) {
	params := map[string]any{}
	for _, paramSet := range paramSets {
		for key, value := range paramSet {
			params[key] = value
		}
	}

	var nextFn func() module.MemSize = func() module.MemSize { return 0 }

	handles, err := newInstanceHandles(m.rt.engine, m.module, functionName, params, &nextFn)
	if err != nil {
		return module.Instance{}, err
	}

	var resetErr error

	return module.Instance{
		Alloc: func(u module.MemSize) (module.MemSize, error) {
			if resetErr != nil {
				return 0, resetErr
			}
			r, err := handles.alloc.Call(handles.store, u)
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
			r, err := handles.transform.Call(handles.store)
			if err != nil {
				return 0, err
			}
			return r.(module.MemSize), err
		},
		Memory: func() module.Memory {
			return module.NewBytesMemory(handles.memory.UnsafeData(handles.store))
		},
		Reset: func() {
			nextFn = func() module.MemSize { return 0 }
			newHandles, err := newInstanceHandles(m.rt.engine, m.module, functionName, params, &nextFn)
			if err != nil {
				resetErr = err
				return
			}
			resetErr = nil
			oldStore := handles.store
			*handles = *newHandles
			// Free the old store's C-allocated resources (instance state and linear memory) now.
			// The GC finalizer would do this eventually, but wasm memory lives outside the Go heap
			// and exerts no pressure on the Go GC, so dropped stores pile up faster than they are
			// collected (issue #155).
			oldStore.Close()
		},
		OwnedBy: handles,
	}, nil
}
