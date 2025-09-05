// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package store

import (
	"github.com/ipld/go-ipld-prime/linking"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/corekv/namespace"

	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/repository"
)

type TxnSource interface {
	NewTxn(bool) (Txn, error)
}

type PTxnSource interface {
	NewTxn(bool) (PTxn, error)
}

type PTxn interface {
	repository.Txn
	corekv.Txn
}

type Txn interface { // todo - rn, make private and/or split
	repository.Txn

	LinkSystem() *linking.LinkSystem
	Indexstore() corekv.ReaderWriter
	Repository() repository.Repository
}

type Root interface {
	TxnSource
	/*
	   LinkSystem() *linking.LinkSystem
	   Indexstore() corekv.ReaderWriter
	   Repository() repository.TxnRepository
	*/
}

func NewRoot(
	txnSource PTxnSource,
	rootstore corekv.ReaderWriter,
	poolSize int, // should repo be passed in instead?
	runtime module.Runtime,
) Root {
	return &root{
		txnSource:  txnSource,
		rootstore:  rootstore,
		linkSystem: makeLinkSystem(namespace.Wrap(rootstore, []byte("b/"))),
		indexstore: namespace.Wrap(rootstore, []byte("i/")),
		repository: repository.NewRepository(poolSize, runtime, &repositoryTxnSource{src: txnSource}),
	}
}

type root struct {
	txnSource  PTxnSource
	rootstore  corekv.ReaderWriter
	linkSystem *linking.LinkSystem
	indexstore corekv.ReaderWriter
	repository repository.TxnRepository
}

var _ Root = (*root)(nil)

func (r *root) NewTxn(readonly bool) (Txn, error) {
	t, err := r.txnSource.NewTxn(readonly)
	if err != nil {
		return nil, err
	}

	return &txn{
		Txn:        t,
		linkSystem: makeLinkSystem(namespace.Wrap(t, []byte("b/"))),
		indexstore: namespace.Wrap(t, []byte("i/")),
		repository: r.Repository().WithTxn(t),
	}, nil
}

func (r *root) LinkSystem() *linking.LinkSystem {
	return r.linkSystem
}

func (r *root) Indexstore() corekv.ReaderWriter {
	return r.indexstore
}

func (r *root) Repository() repository.TxnRepository {
	return r.repository
}

type txn struct {
	repository.Txn
	linkSystem *linking.LinkSystem
	indexstore corekv.ReaderWriter
	repository repository.Repository
}

var _ Txn = (*txn)(nil)

func (r *txn) LinkSystem() *linking.LinkSystem {
	return r.linkSystem
}

func (r *txn) Indexstore() corekv.ReaderWriter {
	return r.indexstore
}

func (r *txn) Repository() repository.Repository {
	return r.repository
}

type repositoryTxnSource struct {
	src PTxnSource
}

var _ repository.TxnSource = (*repositoryTxnSource)(nil)

func (s *repositoryTxnSource) NewTxn(readonly bool) (repository.Txn, error) {
	txn, err := s.NewTxn(readonly)
	if err != nil {
		return nil, err
	}

	return txn, nil
}
