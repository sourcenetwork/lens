// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import "github.com/stretchr/testify/require"

type CloseNode struct {
	stateful
}

var _ Action = (*CloseNode)(nil)
var _ Stateful = (*CloseNode)(nil)

func Close() *CloseNode {
	return &CloseNode{}
}

func (a *CloseNode) Execute() {
	err := a.s.Node.Close()
	require.NoError(a.s.T, err)
}
