// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Excluded on windows/js builds: those platforms default to the wazero/js runtimes, which
// still reset by overwriting linear memory and therefore still leak (see issue #155).
//go:build !windows && !js

package tests

import (
	"testing"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/tests/modules"

	"github.com/stretchr/testify/require"
)

// TestWasm32ResetDoesNotLeakMemory drives a single instance through many pipe.Reset() cycles
// and asserts that wasm linear memory does not grow unboundedly.
//
// Without the store-per-instance fix, dlmalloc-rs bookkeeping is overwritten on each Reset,
// causing memory.grow to be called every cycle and leak ~64 KiB per call (issue #155).
func TestWasm32ResetDoesNotLeakMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory-growth test under -short")
	}

	const (
		warmupCycles = 50
		totalCycles  = 2_000
		// Allow at most two extra 64 KiB pages after warm-up to absorb any
		// legitimate one-time allocator adjustment; more than that indicates a leak.
		maxGrowthBytes = uint32(2 * 64 * 1024)
	)

	rt := newRuntime()
	mod, err := engine.NewModule(rt, modules.WasmPath1)
	require.NoError(t, err)

	instance, err := engine.NewInstance(mod)
	require.NoError(t, err)

	var baseline uint32
	for cycle := 1; cycle <= totalCycles; cycle++ {
		source := enumerable.New([]type1{{Name: "John", Age: cycle}})
		pipe := engine.Append[type1, type2](source, instance)

		hasNext, err := pipe.Next()
		require.NoError(t, err)
		require.True(t, hasNext, "expected one record on cycle %d", cycle)

		_, err = pipe.Value()
		require.NoError(t, err)

		hasNext, err = pipe.Next()
		require.NoError(t, err)
		require.False(t, hasNext, "expected EOS on cycle %d", cycle)

		pipe.Reset()

		if cycle == warmupCycles {
			baseline = instance.Memory().Size()
		}
	}

	final := instance.Memory().Size()
	growth := final - baseline
	require.LessOrEqualf(t, growth, maxGrowthBytes,
		"wasm linear memory grew by %d bytes over %d Reset cycles "+
			"(baseline after %d-cycle warm-up = %d bytes, final = %d bytes); "+
			"expected growth <= %d bytes — Reset is likely leaking memory (issue #155)",
		growth, totalCycles-warmupCycles, warmupCycles, baseline, final, maxGrowthBytes,
	)
}
