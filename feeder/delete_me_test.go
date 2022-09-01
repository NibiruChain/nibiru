package feeder

import (
	"context"
	"encoding/hex"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/url"
	"testing"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network

	feeder *Feeder

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
	tmEndpoint = u.String()

	privKeyEncrypted, err := val.ClientCtx.Keyring.ExportPrivKeyArmorByAddress(val.Address, "hello")
	require.NoError(s.T(), err)

	privKeyDecrypted, _, err := crypto.UnarmorDecryptPrivKey(privKeyEncrypted, "hello")
	require.NoError(s.T(), err)

	rawConf := rawConfig{
		GRPCEndpoint:                grpcEndpoint,
		TendermintWebsocketEndpoint: tmEndpoint,
		Validator:                   val.ValAddress.String(),
		Feeder:                      val.Address.String(),
		Cache:                       MemCacheName,
		PrivateKeyHex:               hex.EncodeToString(privKeyDecrypted.Bytes()),
		ChainToExchangeSymbols: map[string]map[string]string{
			"binance": {
				"ubtc:unusd":  "BTCUSDT",
				"ueth:unusd":  "ETHUSDT",
				"uusdc:unusd": "USDCUSDT",
			},
		},
	}

	conf, err := rawConf.toConfig()
	require.NoError(s.T(), err)
	s.feeder, err = Dial(*conf)
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestVoting() {
	s.feeder.Run()
}

func (s *IntegrationTestSuite) getPrices() oracletypes.ExchangeRateTuples {
	prices, err := s.oracle.ExchangeRates(context.Background(), &oracletypes.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	return prices.ExchangeRates
}

func TestEventsClientSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
