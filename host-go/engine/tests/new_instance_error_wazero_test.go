// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !js

package tests

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/runtimes/wazero"
)

// TestWazeroNewInstanceErrorReturnsError guards that an errored NewInstance still
// surfaces its error after the factory closes the runtime on failure (issue #158).
func TestWazeroNewInstanceErrorReturnsError(t *testing.T) {
	assertNewInstanceErrorsOnUnknownParam(t, wazero.New())
}
