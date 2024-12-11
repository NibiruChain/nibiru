package keeper_test

import (
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

// TestSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

var _ suite.SetupTestSuite = (*Suite)(nil)

func (s *Suite) SetupTest() {
	log.Log().Msgf("SetupTest %v", s.T().Name())
}
