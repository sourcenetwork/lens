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
	store *wasmtime.Store
}

var _ module.Runtime = (*wRuntime)(nil)

func New() module.Runtime {
	engine := wasmtime.NewEngine()
	store := wasmtime.NewStore(engine)

	return &wRuntime{
		store: store,
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
	module, err := wasmtime.NewModule(rt.store.Engine, wasmBytes)
	if err != nil {
		return nil, err
	}

	return &wModule{
		rt:     rt,
		module: module,
	}, nil
}

func (m *wModule) NewInstance(functionName string, paramSets ...map[string]any) (module.Instance, error) {
	// We require a non-nil placeholder else Go will panic upon reassignment (nil pointer de-reference)
	nextFunction := func() module.MemSize { return 0 }
	nextImport := wasmtime.WrapFunc(
		m.rt.store,
		func() module.MemSize {
			return nextFunction()
		},
	)

	instance, err := wasmtime.NewInstance(m.rt.store, m.module, []wasmtime.AsExtern{nextImport})
	if err != nil {
		return module.Instance{}, err
	}

	mem := instance.GetExport(m.rt.store, "memory")
	if mem == nil {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "memory"))
	}

	memory := mem.Memory()
	if memory == nil {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "memory"))
	}

	alloc := instance.GetFunc(m.rt.store, "alloc")
	if alloc == nil {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "alloc"))
	}

	transform := instance.GetFunc(m.rt.store, functionName)
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
		setParam := instance.GetFunc(m.rt.store, "set_param")
		if setParam == nil {
			return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "set_param"))
		}

		sourceBytes, err := json.Marshal(params)
		if err != nil {
			return module.Instance{}, err
		}

		index, err := alloc.Call(m.rt.store, module.TypeIdSize+module.MemSize(len(sourceBytes))+module.LenSize)
		if err != nil {
			return module.Instance{}, err
		}

		mem := module.NewBytesMemory(memory.UnsafeData(m.rt.store))
		w := io.NewOffsetWriter(mem, int64(index.(module.MemSize)))

		err = pipes.WriteItem(w, module.JSONTypeID, sourceBytes)
		if err != nil {
			return module.Instance{}, err
		}

		index, err = setParam.Call(m.rt.store, index)
		if err != nil {
			return module.Instance{}, err
		}
		r := io.NewSectionReader(mem, int64(index.(module.MemSize)), math.MaxInt64)

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

	return module.Instance{
		Alloc: func(u module.MemSize) (module.MemSize, error) {
			r, err := alloc.Call(m.rt.store, u)
			if err != nil {
				return 0, err
			}
			return r.(module.MemSize), err
		},
		Transform: func(next func() module.MemSize) (module.MemSize, error) {
			// By assigning the next function immediately prior to calling transform, we allow multiple
			// pipeline stages to share the same wasm instance - provided they are not called concurrently.
			// This also allows module state to be shared across pipeline stages.
			nextFunction = next
			r, err := transform.Call(m.rt.store)
			if err != nil {
				return 0, err
			}
			return r.(module.MemSize), err
		},
		Memory: func() module.Memory {
			return module.NewBytesMemory(memory.UnsafeData(m.rt.store))
		},
		OwnedBy: instance,
	}, nil
}
