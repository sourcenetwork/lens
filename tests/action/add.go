// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
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
	_, err := a.s.Store.Add(a.s.Ctx, a.Config)
	require.NoError(a.s.T, err)
}
