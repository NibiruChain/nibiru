package testutil

import (
	"testing"

	"github.com/stretchr/testify/suite"
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

func RunFunctionTestSuite(s *suite.Suite, testCases []FunctionTestCase) {
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			tc.Test()
		})
	}
}
