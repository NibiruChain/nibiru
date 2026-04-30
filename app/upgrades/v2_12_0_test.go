package upgrades_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/asset"
	"github.com/NibiruChain/nibiru/v2/x/nutil/denoms"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestV2_12_0_ClearsOracleState(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	oracleKeeper := deps.App.OracleKeeper

	valAddr := deps.App.StakingKeeper.GetValidators(ctx, 1)[0].GetOperator()

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second)).WithBlockHeight(ctx.BlockHeight() + 1)
	deps.SetCtx(ctx)
	oracleKeeper.SetPrice(ctx, asset.NewPair(denoms.ETH, denoms.NUSD), sdkmath.LegacyNewDec(1701))

	oracleKeeper.Prevotes.Insert(
		ctx,
		valAddr,
		oracletypes.NewAggregateExchangeRatePrevote(oracletypes.AggregateVoteHash{}, valAddr, 1),
	)

	oracleKeeper.Votes.Insert(
		ctx,
		valAddr,
		oracletypes.NewAggregateExchangeRateVote(
			oracletypes.ExchangeRateTuples{
				{Pair: asset.NewPair(denoms.ETH, denoms.NUSD), ExchangeRate: sdkmath.LegacyNewDec(1701)},
			},
			valAddr,
		),
	)

	oracleKeeper.MissCounters.Insert(ctx, valAddr, 7)

	params, err := oracleKeeper.ModuleParams.Get(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, params.Whitelist)
	require.NotEmpty(t, oracleKeeper.WhitelistedPairs.Iterate(ctx, collections.Range[asset.Pair]{}).Keys())
	require.NotEmpty(t, oracleKeeper.ExchangeRateMap.Iterate(ctx, collections.Range[asset.Pair]{}).Keys())
	require.NotEmpty(
		t,
		oracleKeeper.PriceSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}).Keys(),
	)
	require.NotEmpty(t, oracleKeeper.Prevotes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys())
	require.NotEmpty(t, oracleKeeper.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys())
	require.NotEmpty(t, oracleKeeper.MissCounters.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys())

	err = deps.RunUpgrade(upgrades.Upgrade2_12_0)
	require.NoError(t, err)

	params, err = oracleKeeper.ModuleParams.Get(deps.Ctx())
	require.NoError(t, err)
	require.Empty(t, params.Whitelist)
	require.Empty(t, oracleKeeper.WhitelistedPairs.Iterate(deps.Ctx(), collections.Range[asset.Pair]{}).Keys())
	require.Empty(t, oracleKeeper.ExchangeRateMap.Iterate(deps.Ctx(), collections.Range[asset.Pair]{}).Keys())
	require.Empty(
		t,
		oracleKeeper.PriceSnapshots.Iterate(deps.Ctx(), collections.PairRange[asset.Pair, time.Time]{}).Keys(),
	)
	require.Empty(t, oracleKeeper.Prevotes.Iterate(deps.Ctx(), collections.Range[sdk.ValAddress]{}).Keys())
	require.Empty(t, oracleKeeper.Votes.Iterate(deps.Ctx(), collections.Range[sdk.ValAddress]{}).Keys())
	require.Empty(t, oracleKeeper.MissCounters.Iterate(deps.Ctx(), collections.Range[sdk.ValAddress]{}).Keys())
}
