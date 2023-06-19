package oracle

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestOracleTallyTiming(t *testing.T) {
	input, h := keeper.Setup(t)

	// all the Addrs vote for the block ... not last period block yet, so tally fails
	for i := range keeper.Addrs[:4] {
		keeper.MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.OneDec()},
		}, i)
	}

	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)

	params.VotePeriod = 10 // set vote period to 10 for now, for convenience
	params.ExpirationBlocks = 100
	input.OracleKeeper.Params.Set(input.Ctx, params)
	require.Equal(t, 1, int(input.Ctx.BlockHeight()))

	EndBlocker(input.Ctx, input.OracleKeeper)
	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.Error(t, err)

	input.Ctx = input.Ctx.WithBlockHeight(int64(params.VotePeriod - 1))

	EndBlocker(input.Ctx, input.OracleKeeper)
	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.NoError(t, err)
}

// Set prices for 2 pairs, one that is updated and the other which is updated only once.
// Ensure that the updated pair is not deleted and the other pair is deleted after a certain time.
func TestOraclePriceExpiration(t *testing.T) {
	input, h := keeper.Setup(t)
	pair1 := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	pair2 := asset.Registry.Pair(denoms.ETH, denoms.NUSD)

	// Set prices for both pairs
	for i := range keeper.Addrs[:4] {
		keeper.MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: pair1, ExchangeRate: sdk.OneDec()},
			{Pair: pair2, ExchangeRate: sdk.OneDec()},
		}, i)
	}

	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)

	params.VotePeriod = 10
	params.ExpirationBlocks = 10
	input.OracleKeeper.Params.Set(input.Ctx, params)

	// Wait for prices to set
	input.Ctx = input.Ctx.WithBlockHeight(int64(params.VotePeriod - 1))
	EndBlocker(input.Ctx, input.OracleKeeper)

	// Check if both prices are set
	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, pair1)
	require.NoError(t, err)
	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, pair2)
	require.NoError(t, err)

	// Set prices for pair 1
	for i := range keeper.Addrs[:4] {
		keeper.MakeAggregatePrevoteAndVote(t, input, h, 19, types.ExchangeRateTuples{
			{Pair: pair1, ExchangeRate: sdk.NewDec(2)},
		}, i)
	}

	// Set price
	input.Ctx = input.Ctx.WithBlockHeight(19)
	EndBlocker(input.Ctx, input.OracleKeeper)

	// Set the block height to 1 block after the expiration
	// End blocker should delete the price of pair2
	input.Ctx = input.Ctx.WithBlockHeight(int64(params.ExpirationBlocks+params.VotePeriod) + 1)
	EndBlocker(input.Ctx, input.OracleKeeper)

	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, pair1)
	require.NoError(t, err)
	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, pair2)
	require.Error(t, err)
}
