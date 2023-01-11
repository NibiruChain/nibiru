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
