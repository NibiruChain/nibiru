package keeper_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

var _ suite.TearDownAllSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	cfg     testnetwork.Config
	network *testnetwork.Network
}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
}

func (s *TestSuite) SetupTest() {
	testapp.EnsureNibiruPrefix()
	homeDir := s.T().TempDir()

	genesisState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	s.cfg = testnetwork.BuildNetworkConfig(genesisState)
	s.cfg.NumValidators = 4
	s.cfg.GenesisState[types.ModuleName] = s.cfg.Codec.MustMarshalJSON(func() codec.ProtoMarshaler {
		gs := types.DefaultGenesisState()
		gs.Params.Whitelist = []asset.Pair{
			"nibi:usdc",
			"btc:usdc",
		}

		return gs
	}())

	network, err := testnetwork.New(
		s.T(),
		homeDir,
		s.cfg,
	)
	s.Require().NoError(err)
	s.network = network

	s.Require().NoError(s.network.WaitForNextBlock())
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
	prices := []map[asset.Pair]sdk.Dec{
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
		map[asset.Pair]sdk.Dec{
			"nibi:usdc": math.LegacyOneDec(),
			"btc:usdc":  math.LegacyMustNewDecFromStr("100200.9"),
		},
		gotPrices,
	)
}

func (s *TestSuite) sendPrevotes(prices []map[asset.Pair]sdk.Dec) []string {
	strVotes := make([]string, len(prices))
	for i, val := range s.network.Validators {
		raw := prices[i]
		votes := make(types.ExchangeRateTuples, 0, len(raw))
		for pair, price := range raw {
			votes = append(votes, types.NewExchangeRateTuple(pair, price))
		}

		pricesStr, err := votes.ToString()
		s.Require().NoError(err)
		_, err = s.network.BroadcastMsgs(val.Address, &types.MsgAggregateExchangeRatePrevote{
			Hash:      types.GetAggregateVoteHash("1", pricesStr, val.ValAddress).String(),
			Feeder:    val.Address.String(),
			Validator: val.ValAddress.String(),
		})
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())

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
		s.Require().NoError(err)
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

func (s *TestSuite) currentPrices() map[asset.Pair]sdk.Dec {
	rawRates, err := types.NewQueryClient(s.network.Validators[0].ClientCtx).ExchangeRates(context.Background(), &types.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	prices := make(map[asset.Pair]sdk.Dec)

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
