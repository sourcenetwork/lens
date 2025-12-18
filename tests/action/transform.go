// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"fmt"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/store"
	"github.com/stretchr/testify/require"
)

// Transform action executes the transform of the given ID with the given input
// and asserts that the output matches the given expected output.
type Transform struct {
	Nodeful

	LensID        string
	Input         enumerable.Enumerable[store.Document]
	Expected      enumerable.Enumerable[store.Document]
	Inverse       bool
	ExpectedError string
}

var _ Action = (*Add)(nil)
var _ Stateful = (*Add)(nil)

func (a *Transform) Execute() {
	lensID := replace(a.s, a.LensID)
	for nodeIndex, n := range a.Nodes() {
		var output enumerable.Enumerable[store.Document]
		var err error
		if a.Inverse {
			output, err = n.Store.Inverse(a.s.Ctx, a.Input, lensID)
		} else {
			output, err = n.Store.Transform(a.s.Ctx, a.Input, lensID)
		}
		if requireErr(a.s.T, a.ExpectedError, err) {
			continue
		}

		n.Transforms = append(n.Transforms, output)

		if output != nil {
			for i := 0; true; i++ {
				println(fmt.Sprintf("TransformEval: Node: {%v} item: {%v}", nodeIndex, i))

				hasNext, err := output.Next()
				require.NoError(a.s.T, err)

				expectedHasNext, err := a.Expected.Next()
				require.NoError(a.s.T, err)

				require.Equal(a.s.T, expectedHasNext, hasNext)

				if !hasNext {
					break
				}

				value, err := output.Value()
				require.NoError(a.s.T, err)

				expectedValue, err := a.Expected.Value()
				require.NoError(a.s.T, err)

				require.Equal(a.s.T, expectedValue, value)
			}
		}

		if a.Expected != nil {
			expectedHasNext, err := a.Expected.Next()
			require.NoError(a.s.T, err)
			require.False(a.s.T, expectedHasNext)

			a.Expected.Reset()
		}
	}
}
