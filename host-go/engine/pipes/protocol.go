// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pipes

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/sourcenetwork/lens/host-go/engine/module"
)

// ReadTypeId returns the type id of the next item from the given reader.
func ReadTypeId(r io.Reader) (module.TypeIdType, error) {
	typeBuffer := make([]byte, module.TypeIdSize)
	typeReader := bytes.NewReader(typeBuffer)

	_, err := r.Read(typeBuffer)
	if err != nil {
		return 0, err
	}
	var typeId module.TypeIdType
	err = binary.Read(typeReader, module.TypeIdByteOrder, &typeId)
	if err != nil {
		return 0, err
	}
	return typeId, nil
}

// ReadItem returns the type id and bytes of the next item from the given reader.
func ReadItem(r io.Reader) (module.TypeIdType, []byte, error) {
	typeId, err := ReadTypeId(r)
	if err != nil {
		return typeId, nil, err
	}

	// type is nil so nothing else to read
	if typeId == module.NilTypeID {
		return typeId, nil, nil
	}

	lenBuffer := make([]byte, module.LenSize)
	lenReader := bytes.NewReader(lenBuffer)

	// read the item length
	_, err = r.Read(lenBuffer)
	if err != nil {
		return typeId, nil, err
	}
	var len module.LenType
	err = binary.Read(lenReader, module.LenByteOrder, &len)
	if err != nil {
		return typeId, nil, err
	}

	// read the item bytes
	data := make([]byte, len)
	_, err = r.Read(data)
	if err != nil {
		return typeId, nil, err
	}
	return typeId, data, nil
}

func WriteItem(w io.Writer, id module.TypeIdType, data []byte) error {
	// write the item type id
	err := binary.Write(w, module.TypeIdByteOrder, id)
	if err != nil {
		return err
	}

	// end of stream messages have no value component that needs writing
	if id.IsEOS() {
		return nil
	}

	// write the item length
	err = binary.Write(w, module.LenByteOrder, module.LenType(len(data)))
	if err != nil {
		return err
	}

	// write the item bytes
	_, err = w.Write(data)
	return err
}

// writeEOS writes the end-of-stream type id to the module memory and returns its location.
func writeEOS(instance module.Instance) (module.MemSize, error) {
	index, err := instance.Alloc(module.TypeIdSize)
	if err != nil {
		return 0, err
	}

	m := instance.Memory()
	w := io.NewOffsetWriter(m, int64(index))

	err = WriteItem(w, module.EOSTypeID, []byte{})
	if err != nil {
		return 0, err
	}

	return index, nil
}

// mustWriteErr writes the given error to the given module's memory, returning its location.
//
// Will panic if an error is generated during writing.
func mustWriteErr(instance module.Instance, err error) module.MemSize {
	errText := err.Error()

	index, err := instance.Alloc(module.TypeIdSize + module.LenSize + int32(len(errText)))
	if err != nil {
		panic(err)
	}

	m := instance.Memory()
	w := io.NewOffsetWriter(m, int64(index))

	err = WriteItem(w, module.ErrTypeID, []byte(errText))
	if err != nil {
		panic(err)
	}

	return index
}
