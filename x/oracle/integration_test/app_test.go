package integration_test_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func (s *IntegrationTestSuite) SetupTest() {
	app.SetPrefixes(app.AccountAddressPrefix)
	s.cfg = testutilcli.BuildNetworkConfig(simapp.NewTestGenesisStateFromDefault())
	s.cfg.NumValidators = 4
	s.cfg.GenesisState[oracletypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(func() codec.ProtoMarshaler {
		gs := oracletypes.DefaultGenesisState()
		gs.Params.Whitelist = []common.AssetPair{
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
	prices := []map[common.AssetPair]sdk.Dec{
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
		map[common.AssetPair]sdk.Dec{
			"nibi:usdc": sdk.MustNewDecFromStr("1"),
			"btc:usdc":  sdk.MustNewDecFromStr("100200.9"),
		},
		gotPrices,
	)
}

func (s *IntegrationTestSuite) sendPrevotes(prices []map[common.AssetPair]sdk.Dec) []string {
	strVotes := make([]string, len(prices))
	for i, val := range s.network.Validators {
		raw := prices[i]
		votes := make(oracletypes.ExchangeRateTuples, 0, len(raw))
		for pair, price := range raw {
			votes = append(votes, oracletypes.NewExchangeRateTuple(pair, price))
		}

		pricesStr, err := votes.ToString()
		require.NoError(s.T(), err)
		_, err = s.network.SendTx(val.Address, &oracletypes.MsgAggregateExchangeRatePrevote{
			Hash:      oracletypes.GetAggregateVoteHash("1", pricesStr, val.ValAddress).String(),
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
		_, err := s.network.SendTx(val.Address, &oracletypes.MsgAggregateExchangeRateVote{
			Salt:          "1",
			ExchangeRates: rates[i],
			Feeder:        val.Address.String(),
			Validator:     val.ValAddress.String(),
		})
		require.NoError(s.T(), err)
	}
}

func (s *IntegrationTestSuite) waitVoteRevealBlock() {
	params, err := oracletypes.NewQueryClient(s.network.Validators[0].ClientCtx).Params(context.Background(), &oracletypes.QueryParamsRequest{})
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

func (s *IntegrationTestSuite) currentPrices() map[common.AssetPair]sdk.Dec {
	rawRates, err := oracletypes.NewQueryClient(s.network.Validators[0].ClientCtx).ExchangeRates(context.Background(), &oracletypes.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	prices := make(map[common.AssetPair]sdk.Dec)

	for _, p := range rawRates.ExchangeRates {
		prices[p.Pair] = p.ExchangeRate
	}

	return prices
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
