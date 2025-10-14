package multiplier

import (
	"github.com/sourcenetwork/testo/multiplier"

	"github.com/sourcenetwork/lens/tests/action"
)

func init() {
	multiplier.Register(&badgerFile{})
	multiplier.Register(&badgerMemory{})
}

const BadgerFile Name = "badger"

// badgerFile represents the badgerFile store complexity multiplier.
//
// Applying the multiplier will replace all [action.NewNode] actions
// with [action.NewBadgerFileNode] instances.
type badgerFile struct{}

var _ Multiplier = (*badgerFile)(nil)

func (n *badgerFile) Name() Name {
	return BadgerFile
}

func (n *badgerFile) Apply(source action.Actions) action.Actions {
	result := make([]action.Action, len(source))

	for i, sourceAction := range source {
		_, ok := sourceAction.(*action.NewNode)
		if ok {
			result[i] = &action.NewBadgerFileNode{}
		} else {
			result[i] = sourceAction
		}
	}

	return result
}

const BadgerMemory Name = "badger-memory"

// badgerMemory represents the memory store complexity multiplier.
//
// Applying the multiplier will replace all [action.NewNode] actions
// with [action.NewBadgerMemoryNode] instances.
type badgerMemory struct{}

var _ Multiplier = (*badgerMemory)(nil)

func (n *badgerMemory) Name() Name {
	return BadgerMemory
}

func (n *badgerMemory) Apply(source action.Actions) action.Actions {
	result := make([]action.Action, len(source))

	for i, sourceAction := range source {
		_, ok := sourceAction.(*action.NewNode)
		if ok {
			result[i] = &action.NewBadgerMemoryNode{}
		} else {
			result[i] = sourceAction
		}
	}

	return result
}
