package testutil_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
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
