// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !js

package tests

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/runtimes/wazero"
)

// TestWazeroResetDoesNotLeakMemory asserts the wazero runtime does not leak wasm
// linear memory across Reset cycles (issue #159).
func TestWazeroResetDoesNotLeakMemory(t *testing.T) {
	assertResetDoesNotLeak(t, wazero.New())
}
