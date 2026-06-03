// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package tests

import (
	"testing"

	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/tests/modules"

	"github.com/stretchr/testify/require"
)

// assertNewInstanceErrorsOnUnknownParam drives NewInstance into a factory error path:
// WasmPath1 has no `set_param` export, so providing params makes the factory fail after
// the store/runtime/instance has been created. It asserts the error is still surfaced —
// guarding against the error-path cleanup (issue #158) swallowing it.
//
// This is a regression guard, not a leak assertion: the resource freed by the #158
// cleanup is C-allocated/host memory invisible to Go's GC, so a leak cannot be observed
// in-tree (cf. #157). Actual reclamation is verified out-of-tree via host RSS.
func assertNewInstanceErrorsOnUnknownParam(t *testing.T, rt module.Runtime) {
	mod, err := engine.NewModule(rt, modules.WasmPath1)
	require.NoError(t, err)

	_, err = engine.NewInstance(mod, map[string]any{"unknown": "param"})
	require.Error(t, err)
}
