package oracle_test

import (
	"context"
	"github.com/NibiruChain/nibiru/app"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"log"
	"testing"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func (s *IntegrationTestSuite) SetupTest() {
	app.SetPrefixes(app.AccountAddressPrefix)
	s.cfg = testutilcli.BuildNetworkConfig(testapp.NewTestGenesisStateFromDefault())
	s.cfg.NumValidators = 1
	s.cfg.GenesisState[oracletypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(func() codec.ProtoMarshaler {
		gs := oracletypes.DefaultGenesisState()
		gs.Params.Whitelist = oracletypes.PairList{
			oracletypes.Pair{Name: "unibi:usdc"},
		}

		return gs
	}())

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	_, err := s.network.WaitForHeight(2)
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestSuccessfulVoting() {
	prices := []map[string]sdk.Dec{
		{
			"unibi:usdc": sdk.MustNewDecFromStr("1"),
		},
		{
			"unibi:usdc": sdk.MustNewDecFromStr("1"),
		},
		{
			"unibi:usdc": sdk.MustNewDecFromStr("1"),
		},
	}
	votes := s.sendPrevotes(prices)

	s.waitRevealVotePeriod()

	s.sendVotes(votes)

	gotPrices := s.prices()

	require.Equal(s.T(), prices, gotPrices)
}

func (s *IntegrationTestSuite) sendPrevotes(prices []map[string]sdk.Dec) []string {
	strVotes := make([]string, len(prices))
	for i, val := range s.network.Validators {
		log.Printf("%s", val.Address.String())
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

func (s *IntegrationTestSuite) waitRevealVotePeriod() {

	params, err := oracletypes.NewQueryClient(s.network.Validators[0].ClientCtx).Params(context.Background(), &oracletypes.QueryParamsRequest{})
	require.NoError(s.T(), err)

	votePeriod := params.Params.VotePeriod

	height, err := s.network.LatestHeight()
	require.NoError(s.T(), err)

	waitBlock := (uint64(height)/votePeriod)*votePeriod + votePeriod

	_, err = s.network.WaitForHeight(int64(waitBlock + 1))
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) prices() map[string]sdk.Dec {
	rawRates, err := oracletypes.NewQueryClient(s.network.Validators[0].ClientCtx).ExchangeRates(context.Background(), &oracletypes.QueryExchangeRatesRequest{})
	require.NoError(s.T(), err)

	prices := make(map[string]sdk.Dec)

	for _, p := range rawRates.ExchangeRates {
		prices[p.Pair] = p.ExchangeRate
	}

	return prices
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
