package testutil

import (
	"strings"
	"testing"
)

type FunctionTestCase struct {
	Name string
	Test func()
}

type FunctionTestCases = []FunctionTestCase

func RunFunctionTests(t *testing.T, testCases []FunctionTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Test()
		})
	}
}

/*
BeforeIntegrationSuite: Skips a test if the `-short` flag is used:

All tests: `go test ./...`
Unit tests only: `go test ./... -short`
Integration tests only: `go test ./... -run Integration`

See: https://stackoverflow.com/a/41407042/13305627
*/
func BeforeIntegrationSuite(suiteT *testing.T) {
	if testing.Short() {
		suiteT.Skip("skipping integration test suite")
	}
	suiteT.Log("setting up integration test suite")
}

// RetrySuiteRunIfDbClosed runs a test suite with retries, recovering from a
// specific panic message, "pebbledb: closed" that often surfaces in CI when tests
// involve "Nibiru/x/common/testutil/testnetwork".
// For full context, see https://github.com/NibiruChain/nibiru/issues/1918.
func RetrySuiteRunIfDbClosed(t *testing.T, runTest func(), maxRetries int) {
	panicMessage := "pebbledb: closed"
	for attempt := 0; attempt < maxRetries; attempt++ {
		panicked := false

		func() {
			defer func() {
				if r := recover(); r != nil {
					if errMsg, ok := r.(string); ok && strings.Contains(errMsg, panicMessage) {
						t.Logf("Recovered from panic on attempt %d: %v", attempt, r)
						panicked = true
					} else {
						panic(r) // Re-panic if it's not the specific error
					}
				}
			}()

			// Run the test suite
			runTest()
			// suite.Run(t, suiteInstance)
		}()

		if !panicked {
			t.Logf("Test suite succeeded on attempt %d", attempt)
			return
		}

		t.Logf("Retrying test suite: attempt %d", attempt+1)
	}

	t.Fatalf("Test suite failed after %d attempts due to '%s'", maxRetries, panicMessage)
}
