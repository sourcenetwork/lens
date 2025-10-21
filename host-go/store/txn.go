// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"github.com/ipld/go-ipld-prime/linking"
	"github.com/sourcenetwork/corekv"

	"github.com/sourcenetwork/lens/host-go/repository"
)

type TxnSource interface {
	// NewTxn returns a new transaction.
	NewTxn(bool) (Txn, error)
}

type Txn interface {
	repository.Txn
	corekv.Txn
}

type txn struct {
	repository.Txn
	linkSystem *linking.LinkSystem
	indexstore corekv.ReaderWriter
	repository repository.Repository
}

// repositoryTxnSource wraps a `TxnSource` so that it satisfies the `repository.TxnSource`
// interface and can be passed through.
//
// Without this, either the `store` package or the `repository` package would require an unnecessarily
// enlarged Txn interface, hindering consumption.
type repositoryTxnSource struct {
	src TxnSource
}

var _ repository.TxnSource = (*repositoryTxnSource)(nil)

func (s *repositoryTxnSource) NewTxn(readonly bool) (repository.Txn, error) {
	return s.src.NewTxn(readonly)
}
