// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// requireErr requires that if the expected string is empty, the given error is nil,
// and if the expected string is not empty, the given error's string contains the expected
// string.
//
// If it returns true, the expected error was found.
func requireErr(t testing.TB, expected string, actual error) bool {
	if len(expected) == 0 {
		require.NoError(t, actual)
		return false
	}

	if actual == nil {
		require.NotNil(t, actual, "expected: %s", expected)
	}

	require.Contains(t, actual.Error(), expected)
	return true
}
