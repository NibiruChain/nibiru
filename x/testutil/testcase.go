package testutil

import (
	"testing"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Action interface {
	Do(app *app.NibiruApp, ctx sdk.Context) error
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
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	for _, testCase := range t.testCases {
		for _, action := range testCase.given {
			err := action.Do(nibiruApp, ctx)
			require.NoError(t.t, err, "failed to execute given action: %s", testCase.Name)
		}

		for _, action := range testCase.when {
			err := action.Do(nibiruApp, ctx)
			require.NoError(t.t, err, "failed to execute when action: %s", testCase.Name)
		}

		for _, action := range testCase.then {
			err := action.Do(nibiruApp, ctx)
			require.NoError(t.t, err, "failed to execute then action: %s", testCase.Name)
		}
	}
}
