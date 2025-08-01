// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package tests

import (
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/runtimes"
)

type type1 struct {
	Name string
	Age  int
}

type type2 struct {
	FullName string
	Age      int
}

func newRuntime() module.Runtime {
	return runtimes.Default()
}
