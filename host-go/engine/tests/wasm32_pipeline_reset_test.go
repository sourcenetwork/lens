// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Build-constrained to match the wasmtime runtime package, which is unavailable on
// windows/js builds. The wazero, wasmer, and js runtimes have their own reset tests.
//go:build !windows && !js

package tests

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/runtimes/wasmtime"
)

// TestWasm32ResetDoesNotLeakMemory asserts the wasmtime runtime does not leak wasm
// linear memory across Reset cycles (issue #155).
func TestWasm32ResetDoesNotLeakMemory(t *testing.T) {
	assertResetDoesNotLeak(t, wasmtime.New())
}
