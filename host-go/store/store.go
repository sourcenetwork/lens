// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"

	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/immutable/enumerable"

	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type Store struct {
	store      corekv.ReaderWriter
	repository repository.Repository
}

func New(store corekv.ReaderWriter) *Store {
	panic("todo")
}

func Add(ctx context.Context, cfg model.Lens) (string, error) {
	// do we need to bother with ipld.NodePrototype schemas and all that, or can we do something simpler?

	panic("todo")
}

func Remove(ctx context.Context, id string) error {
	panic("todo")
}

func List(ctx context.Context) (map[string]model.Lens, error) {
	panic("todo")
}

func Transform(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	panic("todo")
}

func Inverse(
	ctx context.Context,
	source enumerable.Enumerable[Document],
	id string,
) (enumerable.Enumerable[Document], error) {
	panic("todo")
}
