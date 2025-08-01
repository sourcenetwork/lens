// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/config/internal/json"
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/runtimes"
)

// LoadFromFile loads a lens file at the given path and applies it to the provided src.
//
// It does not enumerate the src.
func LoadFromFile[TSource any, TResult any](path string, src enumerable.Enumerable[TSource]) (enumerable.Enumerable[TResult], error) {
	// We only support json lens files at the moment, so we just trust that it is json.
	// In the future we'll need to determine which format the file is in.
	lensConfig, err := json.Load(path)
	if err != nil {
		return nil, err
	}

	return Load[TSource, TResult](lensConfig, src)
}

// Load constructs a lens from the given config and applies it to the provided src.
//
// It does not enumerate the src.
func Load[TSource any, TResult any](lensConfig model.Lens, src enumerable.Enumerable[TSource]) (enumerable.Enumerable[TResult], error) {
	runtime := runtimes.Default()
	modulesByPath := map[string]module.Module{}

	return LoadInto[TSource, TResult](runtime, modulesByPath, lensConfig, src)
}

// LoadIntoFromFile loads a lens file at the given path and applies it to the provided src
// extending the provided runtime and module cache.
//
// It does not enumerate the src. Any new modules will be added to the given module map.
func LoadIntoFromFile[TSource any, TResult any](
	runtime module.Runtime,
	modulesByPath map[string]module.Module,
	path string,
	src enumerable.Enumerable[TSource],
) (enumerable.Enumerable[TResult], error) {
	// We only support json lens files at the moment, so we just trust that it is json.
	// In the future we'll need to determine which format the file is in.
	lensConfig, err := json.Load(path)
	if err != nil {
		return nil, err
	}

	return LoadInto[TSource, TResult](runtime, modulesByPath, lensConfig, src)
}

// LoadInto constructs a lens from the given config and applies it to the provided src
// extending the provided runtime and module cache.
//
// It does not enumerate the src. Any new modules will be added to the given module map.
func LoadInto[TSource any, TResult any](
	runtime module.Runtime,
	modulesByPath map[string]module.Module,
	lensConfig model.Lens,
	src enumerable.Enumerable[TSource],
) (enumerable.Enumerable[TResult], error) {
	for _, moduleCfg := range lensConfig.Lenses {
		// Modules are fairly expensive objects, and they can be reused, so we de-duplicate
		// the WAT code paths here and make sure we only create unique module objects.
		if _, ok := modulesByPath[moduleCfg.Path]; ok {
			continue
		}

		lensModule, err := engine.NewModule(runtime, moduleCfg.Path)
		if err != nil {
			return nil, err
		}
		modulesByPath[moduleCfg.Path] = lensModule
	}

	instances := []module.Instance{}
	for _, moduleCfg := range lensConfig.Lenses {
		lensModule := modulesByPath[moduleCfg.Path]

		var instance module.Instance
		var err error
		if moduleCfg.Inverse {
			instance, err = engine.NewInverse(lensModule, moduleCfg.Arguments)
		} else {
			instance, err = engine.NewInstance(lensModule, moduleCfg.Arguments)
		}

		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}

	return engine.Append[TSource, TResult](src, instances...), nil
}
