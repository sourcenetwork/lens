// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build js

package js

import (
	"syscall/js"
)

type memory struct {
	array js.Value
}

func newMemory(buffer js.Value) *memory {
	array := js.Global().Get("Uint8Array").New(buffer)
	return &memory{array}
}

func (m *memory) ReadAt(dst []byte, offset int64) (int, error) {
	src := m.array.Call("subarray", offset)
	n := js.CopyBytesToGo(dst, src)
	return n, nil
}

func (m *memory) WriteAt(src []byte, offset int64) (int, error) {
	dst := m.array.Call("subarray", offset)
	n := js.CopyBytesToJS(dst, src)
	return n, nil
}

func (m *memory) Size() uint32 {
	return uint32(m.array.Length())
}
