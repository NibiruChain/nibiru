package testutil

import (
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
