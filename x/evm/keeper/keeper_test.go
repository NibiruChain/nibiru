package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

// TestSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
