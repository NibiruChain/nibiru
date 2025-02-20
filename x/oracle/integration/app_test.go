package integration_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

var _ suite.TearDownAllSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
}

func (s *TestSuite) SetupTest() {
	testapp.EnsureNibiruPrefix()
	homeDir := s.T().TempDir()

	genesisState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	s.cfg = testutilcli.BuildNetworkConfig(genesisState)
	s.cfg.NumValidators = 4
	s.cfg.GenesisState[types.ModuleName] = s.cfg.Codec.MustMarshalJSON(func() codec.ProtoMarshaler {
		gs := types.DefaultGenesisState()
		gs.Params.Whitelist = []asset.Pair{
			"nibi:usdc",
			"btc:usdc",
		}

		return gs
	}())

	network, err := testutilcli.New(
		s.T(),
		homeDir,
		s.cfg,
	)
	s.Require().NoError(err)
	s.network = network

	_, err = s.network.WaitForHeight(2)
	require.NoError(s.T(), err)
}

func (s *TestSuite) TestSuccessfulVoting() {
	// assuming validators have equal power
	// we use the weighted median.
	// what happens is that prices are ordered
	// based on exchange rate, from lowest to highest.
	// then the median is picked, based on consensus power
	// so obviously, in this case, since validators have the same power
	// once weight (based on power) >= total power (sum of weights)
	// then the number picked is the one in the middle always.
	prices := []map[asset.Pair]math.LegacyDec{
		{
			"nibi:usdc": math.LegacyOneDec(),
			"btc:usdc":  math.LegacyMustNewDecFromStr("100203.0"),
		},
		{
			"nibi:usdc": math.LegacyOneDec(),
			"btc:usdc":  math.LegacyMustNewDecFromStr("100150.5"),
		},
		{
			"nibi:usdc": math.LegacyOneDec(),
			"btc:usdc":  math.LegacyMustNewDecFromStr("100200.9"),
		},
		{
			"nibi:usdc": math.LegacyOneDec(),
			"btc:usdc":  math.LegacyMustNewDecFromStr("100300.9"),
		},
	}
	votes := s.sendPrevotes(prices)

	s.waitVoteRevealBlock()

	s.sendVotes(votes)

	s.waitPriceUpdateBlock()

	gotPrices := s.currentPrices()
	require.Equal(s.T(),
		map[asset.Pair]math.LegacyDec{
			"nibi:usdc": math.LegacyOneDec(),
			"btc:usdc":  math.LegacyMustNewDecFromStr("100200.9"),
		},
		gotPrices,
	)
}

func (s *IntegrationTestSuite) sendPrevotes(prices []map[asset.Pair]math.LegacyDec) []string {
	strVotes := make([]string, len(prices))
	for i, val := range s.network.Validators {
		raw := prices[i]
		votes := make(types.ExchangeRateTuples, 0, len(raw))
		for pair, price := range raw {
			votes = append(votes, types.NewExchangeRateTuple(pair, price))
		}

		pricesStr, err := votes.ToString()
		require.NoError(s.T(), err)
		_, err = s.network.BroadcastMsgs(val.Address, &types.MsgAggregateExchangeRatePrevote{
			Hash:      types.GetAggregateVoteHash("1", pricesStr, val.ValAddress).String(),
			Feeder:    val.Address.String(),
			Validator: val.ValAddress.String(),
		})
		require.NoError(s.T(), err)

		strVotes[i] = pricesStr
	}

	return strVotes
}

func (s *TestSuite) sendVotes(rates []string) {
	for i, val := range s.network.Validators {
		_, err := s.network.BroadcastMsgs(val.Address, &types.MsgAggregateExchangeRateVote{
			Salt:          "1",
			ExchangeRates: rates[i],
			Feeder:        val.Address.String(),
			Validator:     val.ValAddress.String(),
		})
		require.NoError(s.T(), err)
	}
}

func (s *TestSuite) waitVoteRevealBlock() {
	params, err := types.NewQueryClient(s.network.Validators[0].ClientCtx).Params(context.Background(), &types.QueryParamsRequest{})
	require.NoError(s.T(), err)

	votePeriod := params.Params.VotePeriod

	height, err := s.network.LatestHeight()
	require.NoError(s.T(), err)

	waitBlock := (uint64(height)/votePeriod)*votePeriod + votePeriod

	_, err = s.network.WaitForHeight(int64(waitBlock + 1))
	require.NoError(s.T(), err)
}

// it's an alias, but it exists to give better understanding of what we're doing in test cases scenarios
func (s *TestSuite) waitPriceUpdateBlock() {
	s.waitVoteRevealBlock()
}

func (s *IntegrationTestSuite) currentPrices() map[asset.Pair]math.LegacyDec {
	rawRates, err := types.NewQueryClient(s.network.Validators[0].ClientCtx).ExchangeRates(context.Background(), &types.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	prices := make(map[asset.Pair]math.LegacyDec)

	for _, p := range rawRates.ExchangeRates {
		prices[p.Pair] = p.ExchangeRate
	}

	return prices
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
