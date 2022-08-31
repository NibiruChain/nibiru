package oracle

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/url"
	"testing"
	"time"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network

	eventsClient EventsClient
}

func (s *IntegrationTestSuite) SetupSuite() {
	app.SetPrefixes(app.AccountAddressPrefix)
	s.cfg = testutilcli.BuildNetworkConfig(simapp.NewTestGenesisStateFromDefault())
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	require.NoError(s.T(), err)

	val := s.network.Validators[0]
	grpcEndpoint, tmEndpoint := val.AppConfig.GRPC.Address, val.RPCAddress
	u, err := url.Parse(tmEndpoint)
	require.NoError(s.T(), err)
	u.Scheme = "ws"
	u.Path = "/websocket"

	s.eventsClient, err = NewEventsClient(u.String(), grpcEndpoint)

	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestVotingPeriod() {
	select {
	case <-time.Tick(1 * time.Minute):
		s.T().Fatal("no voting period detected")
	case <-s.eventsClient.NewVotingPeriod():
	}
}
func TestEventsClientSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
