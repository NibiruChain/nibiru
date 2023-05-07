package integration_test_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func (s *IntegrationTestSuite) SetupTest() {
	app.SetPrefixes(app.AccountAddressPrefix)
	s.cfg = testutilcli.BuildNetworkConfig(genesis.NewTestGenesisState())
	s.cfg.NumValidators = 4
	s.cfg.GenesisState[types.ModuleName] = s.cfg.Codec.MustMarshalJSON(func() codec.ProtoMarshaler {
		gs := types.DefaultGenesisState()
		gs.Params.Whitelist = []asset.Pair{
			"nibi:usdc",
			"btc:usdc",
		}

		return gs
	}())

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	_, err := s.network.WaitForHeight(2)
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestSuccessfulVoting() {
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
			"nibi:usdc": sdk.MustNewDecFromStr("1"),
			"btc:usdc":  sdk.MustNewDecFromStr("100203.0"),
		},
		{
			"nibi:usdc": sdk.MustNewDecFromStr("1"),
			"btc:usdc":  sdk.MustNewDecFromStr("100150.5"),
		},
		{
			"nibi:usdc": sdk.MustNewDecFromStr("1"),
			"btc:usdc":  sdk.MustNewDecFromStr("100200.9"),
		},
		{
			"nibi:usdc": sdk.MustNewDecFromStr("1"),
			"btc:usdc":  sdk.MustNewDecFromStr("100300.9"),
		},
	}
	votes := s.sendPrevotes(prices)

	s.waitVoteRevealBlock()

	s.sendVotes(votes)

	s.waitPriceUpdateBlock()

	gotPrices := s.currentPrices()
	require.Equal(s.T(),
		map[asset.Pair]sdk.Dec{
			"nibi:usdc": sdk.MustNewDecFromStr("1"),
			"btc:usdc":  sdk.MustNewDecFromStr("100200.9"),
		},
		gotPrices,
	)
}

func (s *IntegrationTestSuite) sendPrevotes(prices []map[asset.Pair]sdk.Dec) []string {
	strVotes := make([]string, len(prices))
	for i, val := range s.network.Validators {
		raw := prices[i]
		votes := make(types.ExchangeRateTuples, 0, len(raw))
		for pair, price := range raw {
			votes = append(votes, types.NewExchangeRateTuple(pair, price))
		}

		pricesStr, err := votes.ToString()
		require.NoError(s.T(), err)
		_, err = s.network.SendTx(val.Address, &types.MsgAggregateExchangeRatePrevote{
			Hash:      types.GetAggregateVoteHash("1", pricesStr, val.ValAddress).String(),
			Feeder:    val.Address.String(),
			Validator: val.ValAddress.String(),
		})
		require.NoError(s.T(), err)

		strVotes[i] = pricesStr
	}

	return strVotes
}

func (s *IntegrationTestSuite) sendVotes(rates []string) {
	for i, val := range s.network.Validators {
		_, err := s.network.SendTx(val.Address, &types.MsgAggregateExchangeRateVote{
			Salt:          "1",
			ExchangeRates: rates[i],
			Feeder:        val.Address.String(),
			Validator:     val.ValAddress.String(),
		})
		require.NoError(s.T(), err)
	}
}

func (s *IntegrationTestSuite) waitVoteRevealBlock() {
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
func (s *IntegrationTestSuite) waitPriceUpdateBlock() {
	s.waitVoteRevealBlock()
}

func (s *IntegrationTestSuite) currentPrices() map[asset.Pair]sdk.Dec {
	rawRates, err := types.NewQueryClient(s.network.Validators[0].ClientCtx).ExchangeRates(context.Background(), &types.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	prices := make(map[asset.Pair]sdk.Dec)

	for _, p := range rawRates.ExchangeRates {
		prices[p.Pair] = p.ExchangeRate
	}

	return prices
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
