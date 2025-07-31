// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"github.com/sourcenetwork/testo/action"

	"github.com/sourcenetwork/lens/tests/state"
)

type Action = action.Action
type Actions = action.Actions
type Stateful = action.Stateful[*state.State]

type stateful struct {
	s *state.State
}

var _ Stateful = (*stateful)(nil)

func (a *stateful) SetState(s *state.State) {
	if a == nil {
		a = &stateful{}
	}
	a.s = s
}
