package tests

import (
	"testing"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/tests/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test asserts how state can be shared between pipe stages if the same module-instance is
// appended multiple times.
func TestWasm32PipelineWithSharedState(t *testing.T) {
	type Value struct {
		Id   int
		Name string
	}
	runtime := newRuntime()
	if runtime.Name() == "wazero" {
		// This is due to https://github.com/sourcenetwork/lens/issues/71
		t.Skipf("runtime does not support instance reuse")
	}

	module, err := engine.NewModule(runtime, modules.WasmPath5)
	if err != nil {
		t.Error(err)
	}

	instance, err := engine.NewInstance(module)
	if err != nil {
		t.Error(err)
	}

	source := enumerable.New([]Value{
		{
			Name: "John",
		},
		{
			Name: "Shahzad",
		},
		{
			Name: "Addo",
		},
	})

	pipe := engine.Append[Value, Value](source, instance)
	pipe = engine.Append[Value, Value](pipe, instance)
	pipe = engine.Append[Value, Value](pipe, instance)

	hasNext, err := pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := pipe.Value()
	require.NoError(t, err)
	assert.Equal(t, Value{
		// As the same module instance is shared 3 times, the counter will have been incremented
		// 3 times, from a start point of 0
		Id:   3,
		Name: "John",
	}, val)

	hasNext, err = pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err = pipe.Value()
	require.NoError(t, err)
	assert.Equal(t, Value{
		// As the same module instance is shared 3 times, the counter will have been incremented
		// 3 times, from a start point of 3
		Id:   6,
		Name: "Shahzad",
	}, val)

	hasNext, err = pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err = pipe.Value()
	require.NoError(t, err)
	assert.Equal(t, Value{
		// As the same module instance is shared 3 times, the counter will have been incremented
		// 3 times, from a start point of 6
		Id:   9,
		Name: "Addo",
	}, val)

	hasNext, err = pipe.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}
