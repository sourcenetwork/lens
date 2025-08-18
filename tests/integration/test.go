package integration

import (
	"context"
	"testing"
	"time"

	"github.com/sourcenetwork/testo"

	"github.com/sourcenetwork/lens/tests/action"
	"github.com/sourcenetwork/lens/tests/state"
)

// Test is a single, self-contained, test.
type Test struct {
	// Actions contains the set of actions that should be
	// executed as part of this test.
	Actions action.Actions
}

func (test *Test) Execute(t testing.TB) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Prepend a New action if there is not already one present, this saves each test from
	// having to redeclare the same initial action.
	actions := prependNew(test.Actions)

	testo.Log(t, actions)

	state := &state.State{
		T:   t,
		Ctx: ctx,
	}
	testo.ExecuteS(actions, state)
}

func prependNew(actions action.Actions) action.Actions {
	if hasType[*action.NewAction](actions) {
		return actions
	}

	result := make(action.Actions, 1, len(actions)+1)
	result[0] = action.New()
	result = append(result, actions...)

	return result
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
