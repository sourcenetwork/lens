// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package node

import (
	"sync/atomic"

	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/lens/host-go/store"
)

type inMemoryTxnSource struct {
	previousTxnID uint64
	store         corekv.TxnReaderWriter
}

var _ store.TxnSource = (*inMemoryTxnSource)(nil)

func (s *inMemoryTxnSource) NewTxn(readonly bool) (store.Txn, error) {
	txnID := atomic.AddUint64(&s.previousTxnID, 1)

	return &txnWrapper{
		id:  txnID,
		Txn: s.store.NewTxn(readonly),
	}, nil
}
