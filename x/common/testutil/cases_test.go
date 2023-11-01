package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
)

func TestRunFunctionTests(t *testing.T) {
	testutil.RunFunctionTests(t, testutil.FunctionTestCases{
		{
			Name: "test case A",
			Test: func() {
			},
		},
	})
}

func TestBeforeIntegrationSuite(t *testing.T) {
	testutil.BeforeIntegrationSuite(t)

	if testing.Short() {
		require.True(t, t.Skipped())
	} else {
		require.False(t, t.Skipped())
	}
}
