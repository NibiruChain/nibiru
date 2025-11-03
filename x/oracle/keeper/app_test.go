package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/asset"
	"github.com/NibiruChain/nibiru/v2/x/nutil/denoms"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/oracle"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

type Suite struct {
	testutil.LogRoutingSuite
}

func TestOracleKeeper(t *testing.T) {
	suite.Run(t, new(Suite))
}

func setupKeeperTest(s *Suite) (
	nibiruApp *app.NibiruApp,
	ctx sdk.Context,
	msgServer types.MsgServer,
	validators []struct {
		valAddr sdk.ValAddress
		accAddr sdk.AccAddress
	},
) {
	// Initialize app and context
	nibiruApp, ctx = testapp.NewNibiruTestAppAndContext()

	// Create msg servers
	msgServer = oraclekeeper.NewMsgServerImpl(nibiruApp.OracleKeeper, nibiruApp.SudoKeeper)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(nibiruApp.StakingKeeper)

	// Generate validators with pubkeys using nutil helpers
	// Fund accounts and create validators in staking keeper
	testStakingAmt := sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)
	stakingCoins := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, testStakingAmt))

	for i := 0; i < 4; i++ {
		privKey, accAddr := testutil.PrivKey()
		pubKey := privKey.PubKey()
		valAddr := sdk.ValAddress(accAddr)

		// Fund account with staking tokens
		err := testapp.FundAccount(nibiruApp.BankKeeper, ctx, accAddr, stakingCoins)
		s.Require().NoError(err)

		// Create validator using the same helper as other tests
		createValMsg := oraclekeeper.NewTestMsgCreateValidator(valAddr, pubKey, testStakingAmt)

		_, err = stakingMsgServer.CreateValidator(ctx, createValMsg)
		s.Require().NoError(err)

		validators = append(validators, struct {
			valAddr sdk.ValAddress
			accAddr sdk.AccAddress
		}{valAddr: valAddr, accAddr: accAddr})
	}

	// Finalize validators by calling staking EndBlocker
	staking.EndBlocker(ctx, nibiruApp.StakingKeeper)

	// Set oracle params with whitelist pairs
	// Set VotePeriod to 1 like in Setup() for faster test execution
	params, err := nibiruApp.OracleKeeper.ModuleParams.Get(ctx)
	s.Require().NoError(err)
	whitelistPairs := []asset.Pair{"nibi:usdc", "btc:usdc"}
	params.Whitelist = whitelistPairs
	params.VotePeriod = 1 // Fast vote periods for testing
	nibiruApp.OracleKeeper.UpdateParams(ctx, params)

	// Manually sync whitelist pairs to the WhitelistedPairs collection
	// (this is needed because IsWhitelistedPair checks the collection, not params)
	for _, pair := range whitelistPairs {
		nibiruApp.OracleKeeper.WhitelistedPairs.Insert(ctx, pair)
	}

	return nibiruApp, ctx, msgServer, validators
}

func (s *Suite) TestOracleVoting() {
	// Setup
	nibiruApp, ctx, msgServer, validators := setupKeeperTest(s)

	// Price data - same as TestSuccessfulVoting
	prices := []map[asset.Pair]sdkmath.LegacyDec{
		{
			"nibi:usdc": sdkmath.LegacyOneDec(),
			"btc:usdc":  sdkmath.LegacyMustNewDecFromStr("100203.0"),
		},
		{
			"nibi:usdc": sdkmath.LegacyOneDec(),
			"btc:usdc":  sdkmath.LegacyMustNewDecFromStr("100150.5"),
		},
		{
			"nibi:usdc": sdkmath.LegacyOneDec(),
			"btc:usdc":  sdkmath.LegacyMustNewDecFromStr("100200.9"),
		},
		{
			"nibi:usdc": sdkmath.LegacyOneDec(),
			"btc:usdc":  sdkmath.LegacyMustNewDecFromStr("100300.9"),
		},
	}

	// Send prevotes - all in the same block/period for VotePeriod=1
	strVotes := make([]string, len(prices))
	for i, priceMap := range prices {
		// Reuse validators if we need more votes than validators
		val := validators[i%len(validators)]

		// Build exchange rate tuples
		votes := make(types.ExchangeRateTuples, 0, len(priceMap))
		for pair, price := range priceMap {
			votes = append(votes, types.NewExchangeRateTuple(pair, price))
		}

		pricesStr, err := votes.ToString()
		s.Require().NoError(err)

		// Create and send prevote message
		hash := types.GetAggregateVoteHash("1", pricesStr, val.valAddr).String()
		msg := &types.MsgAggregateExchangeRatePrevote{
			Hash:      hash,
			Feeder:    val.accAddr.String(),
			Validator: val.valAddr.String(),
		}

		_, err = msgServer.AggregateExchangeRatePrevote(ctx, msg)
		s.Require().NoError(err)

		strVotes[i] = pricesStr
	}

	// Advance to next vote period (where votes can be revealed)
	// With VotePeriod=1, we just need to advance one block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))

	// Send votes - must happen in the reveal period
	for i, rate := range strVotes {
		val := validators[i%len(validators)]
		msg := &types.MsgAggregateExchangeRateVote{
			Salt:          "1",
			ExchangeRates: rate,
			Feeder:        val.accAddr.String(),
			Validator:     val.valAddr.String(),
		}
		_, err := msgServer.AggregateExchangeRateVote(ctx, msg)
		s.Require().NoError(err)
	}

	// Advance to price update block (another vote period)
	// With VotePeriod=1, advance one more block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))

	// Call EndBlocker to trigger UpdateExchangeRates
	oracle.EndBlocker(ctx, nibiruApp.OracleKeeper)

	// Query exchange rates directly from keeper - only check the pairs we voted on
	// (NewNibiruTestAppAndContext sets default prices that we ignore)
	nibiPrice, err := nibiruApp.OracleKeeper.ExchangeRateMap.Get(ctx, "nibi:usdc")
	s.Require().NoError(err)
	btcPrice, err := nibiruApp.OracleKeeper.ExchangeRateMap.Get(ctx, "btc:usdc")
	s.Require().NoError(err)

	// Assert expected prices
	s.Equal(sdkmath.LegacyOneDec(), nibiPrice.ExchangeRate)
	s.Equal(sdkmath.LegacyMustNewDecFromStr("100200.9"), btcPrice.ExchangeRate)
}
