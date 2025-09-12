// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package module

// Runtime represents the runtime hosting lens instances.
type Runtime interface {
	// Name returns the type-name of this runtime.
	//
	// It is useful if a direct reference to certain runtime packages is undesirable.
	Name() string

	// NewModule instantiates a new module from the given WAT code.
	//
	// This is a fairly expensive operation.
	NewModule([]byte) (Module, error)
}
