// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"net/url"
	"os"
	"strings"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/stretchr/testify/require"
)

type Add struct {
	stateful

	Config model.Lens
}

var _ Action = (*Add)(nil)
var _ Stateful = (*Add)(nil)

func (a *Add) Execute() {
	id, err := a.s.Store.Add(a.s.Ctx, a.Config)
	require.NoError(a.s.T, err)

	a.s.LensIDs = append(a.s.LensIDs, id)

	for _, cfg := range a.Config.Lenses {
		if strings.HasPrefix(cfg.Path, "data://") {
			a.s.WasmBytes = append(a.s.WasmBytes, []byte(strings.TrimPrefix(cfg.Path, "data://")))
		} else if strings.HasPrefix(cfg.Path, "file://") {
			parsed, err := url.Parse(cfg.Path)
			require.NoError(a.s.T, err)

			wasm, err := os.ReadFile(parsed.Path)
			require.NoError(a.s.T, err)

			a.s.WasmBytes = append(a.s.WasmBytes, wasm)
		} else {
			require.Fail(a.s.T, "add action does not yet support wasm path type")
		}
	}
}
