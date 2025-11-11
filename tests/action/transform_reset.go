// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

type TransformReset struct {
	Nodeful

	TransformIndex int
}

var _ Action = (*Add)(nil)
var _ Stateful = (*Add)(nil)

func (a *TransformReset) Execute() {
	for _, n := range a.Nodes() {
		n.Transforms[a.TransformIndex].Reset()
	}
}
