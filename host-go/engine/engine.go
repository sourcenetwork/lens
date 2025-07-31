// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/engine/pipes"
)

// Append appends the given Module Instances to the given source Enumerable, returning the result.
//
// It will try and find the optimal way to communicate between the source and the new module instance, returning an enumerable of a type
// that best fits the situation. The source can be any type that implements the Enumerable interface, it does not need to be a
// lens module instance.
func Append[TSource any, TResult any](src enumerable.Enumerable[TSource], instances ...module.Instance) enumerable.Enumerable[TResult] {
	if len(instances) == 0 {
		return src.(enumerable.Enumerable[TResult])
	}

	if len(instances) == 1 {
		return append[TSource, TResult](src, instances[0])
	}

	intermediarySource := append[TSource, map[string]any](src, instances[0])
	for i := 1; i < len(instances)-1; i++ {
		intermediarySource = append[map[string]any, map[string]any](intermediarySource, instances[i])
	}

	return append[map[string]any, TResult](intermediarySource, instances[len(instances)-1])
}

func append[TSource any, TResult any](src enumerable.Enumerable[TSource], instance module.Instance) enumerable.Enumerable[TResult] {
	switch typedSrc := src.(type) {
	case pipes.Pipe[TSource]:
		return pipes.NewFromPipe[TSource, TResult](typedSrc, instance)
	default:
		return pipes.NewFromSource[TSource, TResult](src, instance)
	}
}

// NewModule instantiates a new module from the WAT code at the given path.
//
// The path must have one of the following prefixes:
// - "file:" local filesystem file
// - "http:" remote file served over http
// - "https:" remote file served over https
//
// This is a fairly expensive operation.
func NewModule(runtime module.Runtime, path string) (module.Module, error) {
	wasm, err := GetWasmBytes(path)
	if err != nil {
		return nil, err
	}
	return runtime.NewModule(wasm)
}

func GetWasmBytes(path string) ([]byte, error) {
	parsed, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		res, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		content, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return content, nil

	case "file":
		content, err := os.ReadFile(parsed.Path)
		if err != nil {
			return nil, err
		}
		return content, nil

	case "data":
		return []byte(parsed.Host), nil

	default:
		return nil, fmt.Errorf("invalid module path: %s", path)
	}
}

func NewInstance(module module.Module, paramSets ...map[string]any) (module.Instance, error) {
	return module.NewInstance("transform", paramSets...)
}

func NewInverse(module module.Module, paramSets ...map[string]any) (module.Instance, error) {
	return module.NewInstance("inverse", paramSets...)
}
