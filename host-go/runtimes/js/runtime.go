// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build js

package js

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"syscall/js"

	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/engine/pipes"
)

const Name string = "js"

type wRuntime struct {
	webAssembly js.Value
}

var _ module.Runtime = (*wRuntime)(nil)

func New() module.Runtime {
	// Get the global WebAssembly object.
	//
	// https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface
	webAssembly := js.Global().Get("WebAssembly")
	return &wRuntime{webAssembly}
}

func (rt *wRuntime) Name() string {
	return Name
}

func (rt *wRuntime) NewModule(wasmBytes []byte) (module.Module, error) {
	// Copy bytes from Go to a JavaScript Uint8Array
	//
	// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Uint8Array/Uint8Array
	wasmBytesJS := js.Global().Get("Uint8Array").New(len(wasmBytes))
	js.CopyBytesToJS(wasmBytesJS, wasmBytes)

	// Compile the WASM bytes into a WebAssembly.Module
	//
	// https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/compile_static
	promise := rt.webAssembly.Call("compile", wasmBytesJS)
	results, err := await(promise)
	if err != nil {
		return nil, err
	}
	return &wModule{
		module:  results[0],
		runtime: rt,
	}, nil
}

type wModule struct {
	module  js.Value
	runtime *wRuntime
}

var _ module.Module = (*wModule)(nil)

func (m *wModule) NewInstance(functionName string, paramSets ...map[string]any) (module.Instance, error) {
	var nextFunction = func() module.MemSize { return 0 }
	// Register the `lens.next` function required as an import for wasm lens modules
	importObject := map[string]any{
		"lens": map[string]any{
			"next": js.FuncOf(func(this js.Value, args []js.Value) any {
				return nextFunction()
			}),
		},
	}

	// Instantiates a WebAssembly.Instance from a WebAssembly.Module with imports.
	//
	// https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/instantiate_static
	promise := m.runtime.webAssembly.Call("instantiate", m.module, importObject)
	results, err := await(promise)
	if err != nil {
		return module.Instance{}, err
	}
	instance := results[0]

	// Get the WebAssembly.Instance.exports object.
	//
	// https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/Instance/exports
	exports := instance.Get("exports")

	// Get the WebAssembly.Memory from the exports.
	//
	// https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/Memory
	memory := exports.Get("memory")
	if memory.Type() != js.TypeObject {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "memory"))
	}

	alloc := exports.Get("alloc")
	if alloc.Type() != js.TypeFunction {
		return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "alloc"))
	}

	transform := exports.Get(functionName)
	if transform.Type() != js.TypeFunction {
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
		setParam := exports.Get("set_param")
		if setParam.Type() != js.TypeFunction {
			return module.Instance{}, errors.New(fmt.Sprintf("Export `%s` does not exist", "set_param"))
		}

		sourceBytes, err := json.Marshal(params)
		if err != nil {
			return module.Instance{}, err
		}

		// allocate memory to write to
		index := alloc.Invoke(module.TypeIdSize + module.MemSize(len(sourceBytes)) + module.LenSize)
		mem := newMemory(memory.Get("buffer"))
		w := io.NewOffsetWriter(mem, int64(index.Int()))

		err = pipes.WriteItem(w, module.JSONTypeID, sourceBytes)
		if err != nil {
			return module.Instance{}, err
		}

		// set param from JavaScript memory
		index = setParam.Invoke(index)
		r := io.NewSectionReader(mem, int64(index.Int()), math.MaxInt64)

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
			result := alloc.Invoke(int32(u))
			return module.MemSize(result.Int()), nil
		},
		Transform: func(next func() module.MemSize) (module.MemSize, error) {
			// By assigning the next function immediately prior to calling transform, we allow multiple
			// pipeline stages to share the same wasm instance - provided they are not called concurrently.
			// This also allows module state to be shared across pipeline stages.
			nextFunction = next
			result := transform.Invoke()
			return module.MemSize(result.Int()), nil
		},
		Memory: func() module.Memory {
			buffer := memory.Get("buffer")
			return newMemory(buffer)
		},
		OwnedBy: instance,
	}, nil
}

// await is a helper that waits for and returns results from the given promise.
func await(promise js.Value) ([]js.Value, error) {
	var (
		res []js.Value
		err error
		wg  sync.WaitGroup
	)

	success := js.FuncOf(func(this js.Value, args []js.Value) any {
		defer wg.Done()
		res = args
		return nil
	})
	defer success.Release()

	failure := js.FuncOf(func(this js.Value, args []js.Value) any {
		defer wg.Done()
		err = errors.New(args[0].Call("toString").String())
		return nil
	})
	defer failure.Release()

	wg.Add(1)
	promise.Call("then", success, failure)
	wg.Wait()

	return res, err
}
