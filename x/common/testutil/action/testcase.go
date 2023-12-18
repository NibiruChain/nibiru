package action

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Action is a type of operation or task that can be performed in the
// Nibiru application.
type Action interface {
	// Do is a specific implementation of the `Action`. When `Do` is called,
	// the action is performed and some feedback is provided about the action's
	// success. `Do` can mutate the app.
	//
	// Returns:
	//   - outCtx: The new context after stateful changes
	//   - err: The error if one was raised.
	//   - isMandatory: Whether an error should have been raised.
	Do(app *app.NibiruApp, ctx sdk.Context) (
		outCtx sdk.Context, err error,
	)
}

// IsNotMandatory is a marker interface for actions that are not mandatory, and it does not stop the test when there is an error.
type IsNotMandatory interface {
	IsNotMandatory()
}

func ActionResp(ctx sdk.Context, respErr error) (outCtx sdk.Context, err error) {
	return ctx, respErr
}

type TestCases []TestCase

type TestCase struct {
	Name string

	given []Action
	when  []Action
	then  []Action
}

// TC creates a new test case
func TC(name string) TestCase {
	return TestCase{Name: name}
}

func (tc TestCase) Given(action ...Action) TestCase {
	tc.given = append(tc.given, action...)
	return tc
}

func (tc TestCase) When(action ...Action) TestCase {
	tc.when = append(tc.when, action...)
	return tc
}

func (tc TestCase) Then(action ...Action) TestCase {
	tc.then = append(tc.then, action...)
	return tc
}

func (tc TestCase) Run(t *testing.T) {
	t.Run(tc.Name, func(t *testing.T) {
		app, ctx := testapp.NewNibiruTestAppAndContextAtTime(time.UnixMilli(0))
		var err error
		var isNotMandatory bool

		for _, action := range tc.given {
			_, isNotMandatory = action.(IsNotMandatory)

			ctx, err = action.Do(app, ctx)
			if isNotMandatory {
				assert.NoError(t, err, "failed to execute given action: %s", tc.Name)
			} else {
				require.NoError(t, err, "failed to execute given action: %s", tc.Name)
			}
		}

		for _, action := range tc.when {
			_, isNotMandatory = action.(IsNotMandatory)

			ctx, err = action.Do(app, ctx)
			if isNotMandatory {
				assert.NoError(t, err, "failed to execute when action: %s", tc.Name)
			} else {
				require.NoError(t, err, "failed to execute when action: %s", tc.Name)
			}
		}

		for _, action := range tc.then {
			_, isNotMandatory = action.(IsNotMandatory)

			ctx, err = action.Do(app, ctx)
			if isNotMandatory {
				assert.NoError(t, err, "failed to execute then action: %s", tc.Name)
			} else {
				require.NoError(t, err, "failed to execute then action: %s", tc.Name)
			}
		}
	})
}

type TestSuite struct {
	t *testing.T

	testCases []TestCase
}

func NewTestSuite(t *testing.T) *TestSuite {
	return &TestSuite{t: t}
}

func (ts *TestSuite) WithTestCases(testCase ...TestCase) *TestSuite {
	ts.testCases = append(ts.testCases, testCase...)
	return ts
}

func (ts *TestSuite) Run() {
	for _, testCase := range ts.testCases {
		testCase.Run(ts.t)
	}
}
