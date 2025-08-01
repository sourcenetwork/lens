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

func TestWasm32PipelineFromSourceAsFull(t *testing.T) {
	runtime := newRuntime()

	module, err := engine.NewModule(runtime, modules.WasmPath1)
	if err != nil {
		t.Error(err)
	}

	instance, err := engine.NewInstance(module)
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

func TestWasm32PipelineFromSourceAsFullToModuleAsFull(t *testing.T) {
	runtime := newRuntime()

	module1, err := engine.NewModule(runtime, modules.WasmPath1)
	if err != nil {
		t.Error(err)
	}
	module2, err := engine.NewModule(runtime, modules.WasmPath2)
	if err != nil {
		t.Error(err)
	}

	instance1, err := engine.NewInstance(module1)
	if err != nil {
		t.Error(err)
	}

	instance2, err := engine.NewInstance(module2)
	if err != nil {
		t.Error(err)
	}

	input := type1{
		Name: "John",
		Age:  32,
	}
	source := enumerable.New([]type1{input})

	pipe1 := engine.Append[type1, type2](source, instance1)
	pipe2 := engine.Append[type2, type2](pipe1, instance2)

	hasNext, err := pipe2.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := pipe2.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "John",
		Age:      33,
	}, val)

	hasNext, err = pipe2.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}

func TestWasm32PipelineFromSourceAsFullToModuleAsFullToModuleAsFull(t *testing.T) {
	runtime := newRuntime()

	module1, err := engine.NewModule(runtime, modules.WasmPath1)
	if err != nil {
		t.Error(err)
	}
	module2, err := engine.NewModule(runtime, modules.WasmPath2)
	if err != nil {
		t.Error(err)
	}

	instance1, err := engine.NewInstance(module1)
	if err != nil {
		t.Error(err)
	}

	instance2, err := engine.NewInstance(module2)
	if err != nil {
		t.Error(err)
	}

	input := type1{
		Name: "John",
		Age:  32,
	}
	source := enumerable.New([]type1{input})

	pipe1 := engine.Append[type1, type2](source, instance1)
	pipe2 := engine.Append[type2, type2](pipe1, instance2)
	pipe3 := engine.Append[type2, type2](pipe2, instance2)

	hasNext, err := pipe3.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := pipe3.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "John",
		Age:      34,
	}, val)

	hasNext, err = pipe3.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}

func TestWasm32PipelineFromSourceAsFullToModuleAsFullToASModuleAsFull(t *testing.T) {
	runtime := newRuntime()

	module1, err := engine.NewModule(runtime, modules.WasmPath1)
	if err != nil {
		t.Error(err)
	}
	module2, err := engine.NewModule(runtime, modules.WasmPath2)
	if err != nil {
		t.Error(err)
	}
	module3, err := engine.NewModule(runtime, modules.WasmPath3)
	if err != nil {
		t.Error(err)
	}

	instance1, err := engine.NewInstance(module1)
	if err != nil {
		t.Error(err)
	}
	instance2, err := engine.NewInstance(module2)
	if err != nil {
		t.Error(err)
	}
	instance3, err := engine.NewInstance(module3)
	if err != nil {
		t.Error(err)
	}

	input := type1{
		Name: "John",
		Age:  32,
	}
	source := enumerable.New([]type1{input})

	pipe1 := engine.Append[type1, type2](source, instance1)
	pipe2 := engine.Append[type2, type2](pipe1, instance2)
	pipe3 := engine.Append[type2, type2](pipe2, instance3)

	hasNext, err := pipe3.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := pipe3.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "John",
		Age:      43,
	}, val)

	hasNext, err = pipe3.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}

func TestWasm32PipelineFromSourceAsFullToModuleAsFullToModuleAsFullWithSingleAppend(t *testing.T) {
	runtime := newRuntime()

	module1, err := engine.NewModule(runtime, modules.WasmPath1)
	if err != nil {
		t.Error(err)
	}
	module2, err := engine.NewModule(runtime, modules.WasmPath2)
	if err != nil {
		t.Error(err)
	}

	instance1, err := engine.NewInstance(module1)
	if err != nil {
		t.Error(err)
	}

	instance2, err := engine.NewInstance(module2)
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
		Age:      34,
	}, val)

	hasNext, err = pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}
