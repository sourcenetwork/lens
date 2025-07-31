package multiplier

import (
	"github.com/sourcenetwork/lens/host-go/config/model"
	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/testo/multiplier"
)

func init() {
	multiplier.Register(&txnDiscard{})
}

const TxnDiscard Name = "txn-discard"

// txnDiscard represents the transaction-discard complexity multiplier.
//
// Applying the multiplier will amend tests to assert that activities performed
// on a transaction are discarded on discard (or at least not applied without commit).
//
// It achieves this by:
//   - Inserting a new [TxnCreate] action immediately after the last [NewNode] action.
//   - Inserting a [TxnDiscard] immediately before the first [CloseNode] action.
//   - Modifying all original read and write actions to apply to the created transaction instead
//     of the underlying store.
//   - Inserting a [List] action immediately after the added [TxnDiscard] action asserting
//     that the underlying store is still empty.
type txnDiscard struct{}

var _ Multiplier = (*txnDiscard)(nil)

func (n *txnDiscard) Name() Name {
	return TxnDiscard
}

func (n *txnDiscard) Apply(source action.Actions) action.Actions {
	lastCreateStoreIndex := 0
	firstCloseIndex := 0
	indexOffset := 0

	result := make(action.Actions, len(source)+3)

	for i, a := range source {
		switch a.(type) {
		case *action.NewNode:
			lastCreateStoreIndex = i

		case *action.CloseNode:
			if firstCloseIndex == 0 {
				firstCloseIndex = i
			}

		case *action.TxnCreate:
			// If the action set already contains txns we should not adjust it
			return source
		}
	}

	for i, a := range source {
		newIndex := i + indexOffset

		switch a.(type) {
		case *action.NewNode:
			result[newIndex] = a

			if i == lastCreateStoreIndex {
				result[newIndex+1] = action.NewTxn()
				indexOffset += 1
			}

		default:
			if i == firstCloseIndex {
				firstCloseIndex = i

				result[newIndex] = action.DiscardTxn()
				result[newIndex+1] = &action.List{
					Expected: map[string]model.Lens{},
				}

				indexOffset += 2
				newIndex = i + indexOffset
			}

			if i > lastCreateStoreIndex && i < firstCloseIndex {
				result[newIndex] = action.WithTxn(a)
			} else {
				result[newIndex] = a
			}
		}
	}

	return result
}
