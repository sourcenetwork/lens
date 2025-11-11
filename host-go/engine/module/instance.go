// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package module

// Instance is the representation of loaded lens module. This will often be sourced from a WASM binary
// but it does not have to be.
type Instance struct {
	// Alloc allocates the given number of bytes in memory and returns the start index to the allocated block.
	Alloc func(size MemSize) (MemSize, error)

	// Transform transforms the data stored at the given start index, returning the start index of the result.
	//
	// The next function provided should return a wasm memory pointer to the next source item to be transformed.
	Transform func(next func() MemSize) (MemSize, error)

	// Memory returns an interface that can be used to read or write to the
	// linear memory that this module uses.
	//
	// Values written to memory will be made available to this module, however changes made by the
	// module after this function has been called are not guaranteed to be visible to the returned io.Reader.
	Memory func() Memory

	// Reset resets the wasm memory to the state it was in immediately after configuration, before any items were
	// pulled through it.
	Reset func()

	// OwnedBy hosts a reference to any object(s) that may be required to live in memory for the lifetime of this Module.
	//
	// This is very important when working with some libraries (such as wasmer-go), as without this, dependencies of other members
	// of this Module may be garbage collected prematurely.
	OwnedBy any
}
