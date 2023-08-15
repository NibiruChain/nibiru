package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// `Action` is a type of operation or task that can be performed in the
// Nibiru application.
type Action interface {
	// `Do` is a specific implementation of the `Action`. When `Do` is called,
	// the action is performed and some feedback is provided about the action's
	// success. `Do` can mutate the app.
	//
	// Returns:
	//   - outCtx: The new context after stateful changes
	//   - err: The error if one was raised.
	//   - isMandatory: Whether an error should have been raised.
	Do(app *app.NibiruApp, ctx sdk.Context) (
		outCtx sdk.Context, err error, isMandatory bool,
	)
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

func (t TestCase) Given(action ...Action) TestCase {
	t.given = append(t.given, action...)
	return t
}

func (t TestCase) When(action ...Action) TestCase {
	t.when = append(t.when, action...)
	return t
}

func (t TestCase) Then(action ...Action) TestCase {
	t.then = append(t.then, action...)
	return t
}

type TestSuite struct {
	t *testing.T

	testCases []TestCase
}

func NewTestSuite(t *testing.T) *TestSuite {
	return &TestSuite{t: t}
}

func (t *TestSuite) WithTestCases(testCase ...TestCase) *TestSuite {
	t.testCases = append(t.testCases, testCase...)
	return t
}

func (t *TestSuite) Run() {
	for _, testCase := range t.testCases {
		app, ctx := testapp.NewNibiruTestAppAndContext()
		var err error
		var isMandatory bool

		for _, action := range testCase.given {
			ctx, err, isMandatory = action.Do(app, ctx)
			if isMandatory {
				require.NoError(t.t, err, "failed to execute given action: %s", testCase.Name)
			} else {
				assert.NoError(t.t, err, "failed to execute given action: %s", testCase.Name)
			}
		}

		for _, action := range testCase.when {
			ctx, err, isMandatory = action.Do(app, ctx)
			if isMandatory {
				require.NoError(t.t, err, "failed to execute when action: %s", testCase.Name)
			} else {
				assert.NoError(t.t, err, "failed to execute when action: %s", testCase.Name)
			}
		}

		for _, action := range testCase.then {
			ctx, err, isMandatory = action.Do(app, ctx)
			if isMandatory {
				require.NoError(t.t, err, "failed to execute then action: %s", testCase.Name)
			} else {
				assert.NoError(t.t, err, "failed to execute then action: %s", testCase.Name)
			}
		}
	}
}
