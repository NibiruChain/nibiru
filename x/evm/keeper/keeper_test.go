package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type KeeperSuite struct {
	suite.Suite
}

// TestKeeperSuite: Runs all the tests in the suite.
func TestKeeperSuite(t *testing.T) {
	s := new(KeeperSuite)
	suite.Run(t, s)
}
