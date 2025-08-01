// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package tests

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/engine"
	"github.com/sourcenetwork/lens/host-go/engine/module"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppendLensWithoutWasm asserts that AppendLens can function independently of anything wasm related.
// This may be very important at a later date, as it means this function can (in-theory, minus any internal
// dependencies) can be executed in a wasm/wasi runtime. It should also help flag any changes to AppendLens'
// externally visible behaviour that may otherwise be hidden (e.g. if otherwise testing using wasm modules).
//
// TestAppendLensWithoutWasm also helps document how AppendLens works.
//
// It is not recomended to actually use AppendLens like this - if you are not transforming data via a wasm module,
// you should probably be using enumerable.Select or similar instead.
func TestAppendLensWithoutWasm(t *testing.T) {
	sourceSlice := []type1{
		{
			Name: "John",
			Age:  32,
		},
		{
			Name: "Fred",
			Age:  55,
		},
	}
	source := enumerable.New(sourceSlice)

	memory := make([]byte, math.MaxUint16)
	results := engine.Append[type1, type2](
		source,
		module.Instance{
			Alloc: func(size module.MemSize) (module.MemSize, error) {
				var arbitraryIndex module.MemSize = 5
				return arbitraryIndex, nil
			},
			Transform: func(next func() module.MemSize) (module.MemSize, error) {
				startIndex := next()
				typeBuffer := make([]byte, module.TypeIdSize)
				copy(typeBuffer, memory[startIndex:startIndex+module.TypeIdSize])
				var inputTypeId module.TypeIdType
				buf := bytes.NewReader(typeBuffer)
				err := binary.Read(buf, module.TypeIdByteOrder, &inputTypeId)
				if err != nil {
					return 0, err
				}

				if inputTypeId.IsEOS() {
					arbitraryReturnIndex := math.MaxUint16 / 4
					typeWriter := bytes.NewBuffer([]byte{})
					err = binary.Write(typeWriter, module.TypeIdByteOrder, int8(module.EOSTypeID))
					if err != nil {
						return 0, err
					}
					dst := memory[arbitraryReturnIndex:]
					copy(dst, typeWriter.Bytes())
					return module.MemSize(arbitraryReturnIndex), nil
				}

				lenBuffer := make([]byte, module.LenSize)
				copy(lenBuffer, memory[startIndex+module.TypeIdSize:startIndex+module.TypeIdSize+module.LenSize])
				var inputLen module.LenType
				buf = bytes.NewReader(lenBuffer)
				err = binary.Read(buf, module.LenByteOrder, &inputLen)
				if err != nil {
					return 0, err
				}

				sourceJson := memory[startIndex+module.TypeIdSize+module.LenSize : startIndex+module.TypeIdSize+module.MemSize(inputLen)+module.LenSize]
				sourceItem := &type1{}

				err = json.Unmarshal([]byte(sourceJson), sourceItem)
				if err != nil {
					return 0, err
				}

				resultItem := type2{
					FullName: sourceItem.Name,
					Age:      sourceItem.Age,
				}

				returnBytes, err := json.Marshal(resultItem)
				if err != nil {
					return 0, err
				}

				typeWriter := bytes.NewBuffer([]byte{})
				err = binary.Write(typeWriter, module.TypeIdByteOrder, int8(1))
				if err != nil {
					return 0, err
				}

				returnLen := module.LenType(len(returnBytes))
				lenWriter := bytes.NewBuffer([]byte{})
				err = binary.Write(lenWriter, module.LenByteOrder, returnLen)
				if err != nil {
					return 0, err
				}

				arbitraryReturnIndex := math.MaxUint16 / 2
				dst := memory[arbitraryReturnIndex:]
				copy(dst, typeWriter.Bytes())
				copy(dst[module.TypeIdSize:], lenWriter.Bytes())
				copy(dst[module.TypeIdSize+module.LenSize:], returnBytes)

				return module.MemSize(arbitraryReturnIndex), nil
			},
			Memory: func() module.Memory {
				return module.NewBytesMemory(memory)
			},
		},
	)

	hasNext, err := results.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err := results.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "John",
		Age:      32,
	}, val)

	hasNext, err = results.Next()
	if err != nil {
		t.Error(err)
	}
	assert.True(t, hasNext)

	val, err = results.Value()
	require.Nil(t, err)
	assert.Equal(t, type2{
		FullName: "Fred",
		Age:      55,
	}, val)

	hasNext, err = results.Next()
	if err != nil {
		t.Error(err)
	}
	assert.False(t, hasNext)
}
