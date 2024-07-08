package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

// TestKeeperSuite: Runs all the tests in the suite.
func TestKeeperSuite(t *testing.T) {
	s := new(Suite)
	suite.Run(t, s)
}
