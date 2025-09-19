package integration

import (
	"context"
	"testing"
	"time"

	"github.com/sourcenetwork/testo"
	"github.com/sourcenetwork/testo/multiplier"

	"github.com/sourcenetwork/lens/tests/action"
	_ "github.com/sourcenetwork/lens/tests/multiplier"
	"github.com/sourcenetwork/lens/tests/state"
)

func init() {
	multiplier.Init("LENS_MULTIPLIERS")
}

// Test is a single, self-contained, test.
type Test struct {
	// The test will be skipped if the current active set of multipliers
	// does not contain all of the given multiplier names.
	Includes []multiplier.Name

	// The test will be skipped if the current active set of multipliers
	// contains any of the given multiplier names.
	Excludes []multiplier.Name

	// Actions contains the set of actions that should be
	// executed as part of this test.
	Actions action.Actions
}

func (test *Test) Execute(t testing.TB) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	multiplier.Skip(t, test.Includes, test.Excludes)

	// Prepend a New action if there is not already one present, this saves each test from
	// having to redeclare the same initial action.
	actions := prependNew(test.Actions)
	actions = appendClose(actions)

	actions = multiplier.Apply(actions)

	testo.Log(t, actions)

	state := &state.State{
		T:   t,
		Ctx: ctx,
	}
	testo.ExecuteS(actions, state)
}

func prependNew(actions action.Actions) action.Actions {
	if hasType[*action.NewNode](actions) {
		return actions
	}

	result := make(action.Actions, 1, len(actions)+1)
	result[0] = action.New()
	result = append(result, actions...)

	return result
}

// appendClose appends an [*action.CloseNode] action to the end of the given
// action set if the set does not already contain a close action.
//
// This is done so that we don't have to bother adding (and reading) the same action
// in 99% of our tests.
func appendClose(actions action.Actions) action.Actions {
	if hasType[*action.CloseNode](actions) {
		return actions
	}

	actions = append(actions, &action.CloseNode{})
	return actions
}

// hasType returns true if any of the items in the given set are of the given type.
func hasType[TAction any](actions action.Actions) bool {
	for _, action := range actions {
		_, ok := action.(TAction)
		if ok {
			return true
		}
	}

	return false
}
