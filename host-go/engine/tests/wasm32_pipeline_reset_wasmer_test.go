// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows && !js

package tests

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/runtimes/wasmer"
)

// TestWasmerResetDoesNotLeakMemory asserts the wasmer runtime does not leak wasm
// linear memory across Reset cycles (issue #159).
func TestWasmerResetDoesNotLeakMemory(t *testing.T) {
	assertResetDoesNotLeak(t, wasmer.New())
}
