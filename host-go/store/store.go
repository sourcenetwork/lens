// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"context"

	"github.com/sourcenetwork/corekv"

	"github.com/lens-vm/lens/host-go/config/model"
	"github.com/lens-vm/lens/host-go/repository"
)

type Store struct {
	store      corekv.ReaderWriter
	repository repository.Repository
}

func New(store corekv.ReaderWriter) *Store {
	panic("todo")
}

func Add(ctx context.Context, cfg model.Lens) error {
	panic("todo")
}
