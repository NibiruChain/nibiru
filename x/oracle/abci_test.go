package oracle

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestOracleTallyTiming(t *testing.T) {
	input, h := keeper.Setup(t)

	// all the Addrs vote for the block ... not last period block yet, so tally fails
	for i := range keeper.Addrs[:4] {
		keeper.MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.OneDec()},
		}, i)
	}

	params := input.OracleKeeper.GetParams(input.Ctx)
	params.VotePeriod = 10 // set vote period to 10 for now, for convenience
	input.OracleKeeper.SetParams(input.Ctx, params)
	require.Equal(t, 0, int(input.Ctx.BlockHeight()))

	EndBlocker(input.Ctx, input.OracleKeeper)
	_, err := input.OracleKeeper.ExchangeRates.Get(input.Ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.Error(t, err)

	input.Ctx = input.Ctx.WithBlockHeight(int64(params.VotePeriod - 1))

	EndBlocker(input.Ctx, input.OracleKeeper)
	_, err = input.OracleKeeper.ExchangeRates.Get(input.Ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.NoError(t, err)
}
