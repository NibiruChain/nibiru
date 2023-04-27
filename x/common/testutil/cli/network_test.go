package cli_test

// Alteration of [network/network_test.go](https://github.com/cosmos/cosmos-sdk/blob/v0.45.15/testutil/network/network_test.go)
//
// ```go
// //go:build norace
// // +build norace
// ````

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationTestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

type IntegrationTestSuite struct {
	suite.Suite

	network *cli.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	/* 	Make test skip if -short is not used:
	All tests: `go test ./...`
	Unit tests only: `go test ./... -short`
	Integration tests only: `go test ./... -run Integration`
	https://stackoverflow.com/a/41407042/13305627 */
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	s.network = cli.NewNetwork(
		s.T(),
		cli.BuildNetworkConfig(genesis.NewTestGenesisState()),
	)
	s.Require().NotNil(s.network)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestNetwork_Liveness() {
	height, err := s.network.WaitForHeightWithTimeout(4, time.Minute)
	s.Require().NoError(err, "expected to reach 4 blocks; got %d", height)
}

func (s *IntegrationTestSuite) TestNetwork_LatestHeight() {
	height, err := s.network.LatestHeight()
	s.NoError(err)
	s.Positive(height)

	sadNetwork := new(cli.Network)
	_, err = sadNetwork.LatestHeight()
	s.Error(err)
}
