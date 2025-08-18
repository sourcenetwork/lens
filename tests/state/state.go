// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package state

import (
	"context"
	"testing"

	"github.com/sourcenetwork/lens/host-go/node"
)

type State struct {
	Ctx context.Context
	T   testing.TB

	Node *node.Node
}
