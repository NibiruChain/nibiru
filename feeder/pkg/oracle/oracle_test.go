package oracle

import (
	"context"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"net/url"
	"testing"
	"time"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network

	eventsClient EventsClient
	writeClient  *TxClient

	oracle oracletypes.QueryClient
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
	s.writeClient, err = NewTxClient(grpcEndpoint, val.ValAddress, val.Address, &MemPrevoteCache{}, val.ClientCtx.Keyring)
	require.NoError(s.T(), err)

	conn, err := grpc.Dial(grpcEndpoint, grpc.WithInsecure())
	require.NoError(s.T(), err)
	s.oracle = oracletypes.NewQueryClient(conn)
}

func (s *IntegrationTestSuite) TestVoting() {
	targets := s.targetsUpdate()
	s.waitVotePeriod()

	prices := make([]SymbolPrice, len(targets))
	for i, target := range targets {
		prices[i] = SymbolPrice{
			Symbol: target,
			Price:  1_000_000.1059459549,
		}
	}

	err := s.writeClient.SendPrices(prices)
	require.NoError(s.T(), err)

	s.waitVotePeriod()

	gotPrices := s.getPrices()

	s.T().Logf("%#v", gotPrices)
}

func (s *IntegrationTestSuite) waitVotePeriod() {
	select {
	case <-time.Tick(1 * time.Minute):
		s.T().Fatal("no voting period detected")
	case <-s.eventsClient.NewVotingPeriod():
	}
}

func (s *IntegrationTestSuite) targetsUpdate() []string {
	select {
	case <-time.Tick(1 * time.Minute):
		s.T().Fatal("no vote targets")
	case targets := <-s.eventsClient.SymbolsUpdate():
		return targets
	}
	// unreachable
	return nil
}

func (s *IntegrationTestSuite) getPrices() oracletypes.ExchangeRateTuples {
	prices, err := s.oracle.ExchangeRates(context.Background(), &oracletypes.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	return prices.ExchangeRates
}

func TestEventsClientSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
