// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package module

import "io"

// Memory is an interface for reading and writing to a
// module's shared linear memory.
type Memory interface {
	io.ReaderAt
	io.WriterAt
	// Size returns the total size of linear memory in bytes.
	Size() uint32
}

var _ (Memory) = (*BytesMemory)(nil)

// BytesMemory converts a byte slice into a module Memory.
type BytesMemory struct {
	data []byte
}

// NewBytesMemory returns a ReadWriterAt that reads from the given bytes.
func NewBytesMemory(data []byte) *BytesMemory {
	return &BytesMemory{data}
}

// Read implements the io.ReaderAt interface.
func (s *BytesMemory) ReadAt(dst []byte, offset int64) (int, error) {
	n := copy(dst, s.data[offset:])
	return n, nil
}

// Write implements the io.WriterAt interface.
func (s *BytesMemory) WriteAt(src []byte, offset int64) (int, error) {
	n := copy(s.data[offset:], src)
	return n, nil
}

// Size returns the total size of linear memory in bytes.
func (s *BytesMemory) Size() uint32 {
	return uint32(len(s.data))
}
