// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pipes

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"

	"github.com/sourcenetwork/lens/host-go/engine/module"
)

type fromPipe[TSource any, TResult any] struct {
	source   Pipe[TSource]
	instance module.Instance

	currentIndex module.MemSize
}

func NewFromPipe[TSource any, TResult any](
	source Pipe[TSource],
	instance module.Instance,
) Pipe[TResult] {
	return &fromPipe[TSource, TResult]{
		source:   source,
		instance: instance,
	}
}

var _ Pipe[int] = (*fromPipe[bool, int])(nil)

func (p *fromPipe[TSource, TResult]) Next() (bool, error) {
	index, err := p.instance.Transform(p.mustGetNext)
	if err != nil {
		return false, err
	}

	m := p.instance.Memory()
	r := io.NewSectionReader(m, int64(index), math.MaxInt64)

	typeId, err := ReadTypeId(r)
	if err != nil {
		return false, err
	}
	if typeId.IsEOS() {
		return false, nil
	}

	p.currentIndex = index
	return true, nil
}

func (p *fromPipe[TSource, TResult]) Value() (TResult, error) {
	var result TResult

	mem := p.instance.Memory()
	r := io.NewSectionReader(mem, int64(p.currentIndex), math.MaxInt64)

	id, data, err := ReadItem(r)
	if err != nil {
		return result, err
	}
	if id.IsError() {
		return result, errors.New(string(data))
	}
	if id != module.JSONTypeID {
		return result, nil
	}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (p *fromPipe[TSource, TResult]) Bytes() ([]byte, error) {
	m := p.instance.Memory()
	r := io.NewSectionReader(m, int64(p.currentIndex), math.MaxInt64)

	id, data, err := ReadItem(r)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := WriteItem(&out, id, data); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (p *fromPipe[TSource, TResult]) Reset() {
	p.instance.Reset()
	p.source.Reset()
}

// mustGetNext tries to get the next value from source and copy it into the memory buffer.
//
// If there are no more values in source it will write an EOS message. If an error is
// generated, it will attempt to write the error to the memory buffer - if the writing of the
// error to the buffer fails it will panic.
func (p *fromPipe[TSource, TResult]) mustGetNext() module.MemSize {
	index, err := p.getNext()
	if err != nil {
		return mustWriteErr(p.instance, err)
	}

	return index
}

func (p *fromPipe[TSource, TResult]) getNext() (module.MemSize, error) {
	hasNext, err := p.source.Next()
	if err != nil {
		return 0, err
	}
	if !hasNext {
		return writeEOS(p.instance)
	}

	value, err := p.source.Bytes()
	if err != nil {
		return 0, err
	}

	// allocate space for the next item
	index, err := p.instance.Alloc(module.MemSize(len(value)))
	if err != nil {
		return 0, err
	}

	m := p.instance.Memory()
	w := io.NewOffsetWriter(m, int64(index))

	// write the item to memory
	_, err = w.Write(value)
	if err != nil {
		return 0, err
	}
	return index, nil
}
