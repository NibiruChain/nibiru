package oracle

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
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
	require.NoError(s.T(), err)

	s.writeClient, err = NewTxClient(grpcEndpoint, val.ValAddress, val.Address, &MemPrevoteCache{}, val.ClientCtx.Keyring)
	require.NoError(s.T(), err)

	conn, err := grpc.Dial(grpcEndpoint, grpc.WithInsecure())
	require.NoError(s.T(), err)
	s.oracle = oracletypes.NewQueryClient(conn)
}

func (s *IntegrationTestSuite) TestVoting() {
	targets := s.targetsUpdate()
	s.waitVotePeriod()                  // wait: VP 1
	pricesSent := s.sendPrices(targets) // VP 1: Only prevote
	s.waitVotePeriod()                  // wait VP 2
	s.sendPrices(targets)               // VP 2: Prevote VP 2 and Vote reveal VP 1
	s.waitVotePeriod()                  // VP 3: VP 1 Vote Consensus, VP 2 Vote Reveal, VP 3 Prevote
	gotPrices := s.getPrices()

	_ = pricesSent
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

func (s *IntegrationTestSuite) sendPrices(targets []string) []SymbolPrice {
	prices := make([]SymbolPrice, len(targets))
	for i, target := range targets {
		prices[i] = SymbolPrice{
			Symbol: target,
			Price:  1_000_000.1059459549,
		}
	}

	err := s.writeClient.SendPrices(prices)
	require.NoError(s.T(), err)

	return prices
}

func TestEventsClientSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
