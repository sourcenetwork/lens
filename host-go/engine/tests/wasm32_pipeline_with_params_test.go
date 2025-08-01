// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package tests

import (
	"testing"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/tests/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWasm32PipelineWithAddtionalParams(t *testing.T) {
	runtime := newRuntime()

	module, err := engine.NewModule(runtime, modules.WasmPath4)
	if err != nil {
		t.Error(err)
	}

	instance, err := engine.NewInstance(
		module,
		map[string]any{
			"src": "Name",
			"dst": "FullName",
		},
	)
	if err != nil {
		t.Error(err)
	}

	input := type1{
		Name: "John",
		Age:  32,
	}
	source := enumerable.New([]type1{input})

	pipe := engine.Append[type1, type2](source, instance)

	hasNext, err := pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := pipe.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "John",
		Age:      32,
	}, val)

	hasNext, err = pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}

func TestWasm32PipelineMultipleModulesAndWithAddtionalParams(t *testing.T) {
	runtime := newRuntime()

	module, err := engine.NewModule(runtime, modules.WasmPath4)
	if err != nil {
		t.Error(err)
	}

	instance1, err := engine.NewInstance(
		module,
		map[string]any{
			"src": "Name",
			"dst": "FirstName",
		},
	)
	if err != nil {
		t.Error(err)
	}

	instance2, err := engine.NewInstance(
		module,
		map[string]any{
			"src": "FirstName",
			"dst": "FullName",
		},
	)
	if err != nil {
		t.Error(err)
	}

	input := type1{
		Name: "John",
		Age:  32,
	}
	source := enumerable.New([]type1{input})

	pipe := engine.Append[type1, type2](
		source,
		instance1,
		instance2,
	)

	hasNext, err := pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := pipe.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "John",
		Age:      32,
	}, val)

	hasNext, err = pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}

func TestWasm32PipelineWithAddtionalParamsErrors(t *testing.T) {
	runtime := newRuntime()

	module, err := engine.NewModule(runtime, modules.WasmPath4)
	if err != nil {
		t.Error(err)
	}

	instance, err := engine.NewInstance(
		module,
		map[string]any{
			"src": "NotAField",
			"dst": "FullName",
		},
	)
	if err != nil {
		t.Error(err)
	}

	input := type1{
		Name: "John",
		Age:  32,
	}
	source := enumerable.New([]type1{input})

	pipe := engine.Append[type1, type2](source, instance)

	hasNext, err := pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	_, err = pipe.Value()
	assert.ErrorContains(t, err, "The requested property was not found. Requested: NotAField")
}

func TestWasm32PipelineWithAddtionalParamsErrorsAndNilItem(t *testing.T) {
	runtime := newRuntime()

	module, err := engine.NewModule(runtime, modules.WasmPath4)
	if err != nil {
		t.Error(err)
	}

	instance, err := engine.NewInstance(
		module,
		map[string]any{
			"src": "FirstName",
			"dst": "FullName",
		},
	)
	if err != nil {
		t.Error(err)
	}

	source := enumerable.New([]*type1{nil})

	pipe := engine.Append[*type1, *type2](source, instance)

	hasNext, err := pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	value, err := pipe.Value()
	require.Nil(t, err)
	assert.Nil(t, value)
}
