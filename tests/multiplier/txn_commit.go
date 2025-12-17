// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package multiplier

import (
	"slices"

	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/testo/multiplier"
)

func init() {
	multiplier.Register(&txnCommit{})
}

// The TxnCommit multiplier adapts tests to cover the commiting of a single transaction.
//
// It will ensure that any write actions are reflected within the transaction scope pre-commit,
// and will ensure that those writes are later persisted in the backing store after transaction
// commit.
//
// It does this by:
//   - Creating a new new transaction immediately after the last [action.NewNode] action.
//   - Scoping any following actions to the new transaction.
//   - Creating a [action.TxnCommit] action to commit the transaction.
//   - Duplicating the original read actions occuring after the last write action to re-execute them
//     outside of the transaction scope following commit.
//
// Whether or not an action is a 'read' or 'write' is currently handled by switches within this
// implementation, new action types will need to amend these switches should they wish to take
// advantage of this multiplier.  Long term we should find a better solution.
const TxnCommit Name = "txn-commit"

type txnCommit struct{}

var _ Multiplier = (*txnCommit)(nil)

func (n *txnCommit) Name() Name {
	return TxnCommit
}

func (n *txnCommit) Apply(source action.Actions) action.Actions {
	lastStartIndex := 0
	lastWriteIndex := 0
	firstCloseIndex := 0

	for i, a := range source {
		switch a.(type) {
		case *action.NewNode:
			lastStartIndex = i

		case *action.Add, *action.Sync:
			lastWriteIndex = i

		case *action.CloseNode:
			if firstCloseIndex == 0 {
				firstCloseIndex = i
			}

		case *action.TxnCreate:
			// If the action set already contains txns we should not adjust it
			return source
		}
	}

	result := make(action.Actions, 0, len(source)+2)

	for i, a := range source {
		switch a.(type) {
		case *action.NewNode:
			result = append(result, a)
			result = append(result, action.NewTxn())
			firstCloseIndex += 1

		case *action.CloseNode:
			result = append(result, a)
			continue

		default:
			if i > lastStartIndex && i < firstCloseIndex {
				result = append(result, action.WithTxn(a))
			} else {
				result = append(result, a)
			}
		}
	}

	var commitAction action.Action
	commitAction = &action.TxnCommit{}
	result = slices.Insert(result, firstCloseIndex, commitAction)
	firstCloseIndex += 1

	for i, a := range source {
		if i <= lastStartIndex {
			continue
		}

		if i <= lastWriteIndex {
			continue
		}

		switch a.(type) {
		// Append orginal, read-only actions that occured after the last write action,
		// after the transaction has been commited.  This allows the automatic coverage
		// of whether or not the transaction-state has been persisted.
		case *action.List:
			result = slices.Insert(result, firstCloseIndex, a)
			firstCloseIndex += 1
		}
	}

	return result
}
