// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build js

package tests

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/runtimes/js"
)

// TestJsNewInstanceErrorReturnsError guards that an errored NewInstance still surfaces
// its error after the next import is released on failure (issue #158).
func TestJsNewInstanceErrorReturnsError(t *testing.T) {
	assertNewInstanceErrorsOnUnknownParam(t, js.New())
}
