package perp_test

import (
	"testing"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perp "github.com/NibiruChain/nibiru/x/perp/module/v2"
	types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestGenesis(t *testing.T) {
	app, ctxUncached := testapp.NewNibiruTestAppAndContext(true)
	ctx, _ := ctxUncached.CacheContext()

	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	// create some params
	app.PerpKeeperV2.Markets.Insert(ctx, pair, *mock.TestMarket())
	app.PerpKeeperV2.AMMs.Insert(ctx, pair, *mock.TestAMMDefault())

	// create some positions
	for i := int64(0); i < 100; i++ {
		trader := testutil.AccAddress()
		app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(asset.Registry.Pair(denoms.NIBI, denoms.NUSD), trader), types.Position{
			TraderAddress:                   trader.String(),
			Pair:                            asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			Size_:                           sdk.NewDec(i + 1),
			Margin:                          sdk.NewDec(i * 2),
			OpenNotional:                    sdk.NewDec(i * 100),
			LatestCumulativePremiumFraction: sdk.NewDec(5 * 100),
			LastUpdatedBlockNumber:          i,
		})
	}

	// export genesis
	genState := perp.ExportGenesis(ctx, app.PerpKeeperV2)

	// create new context and init genesis
	ctx, _ = ctxUncached.CacheContext()
	perp.InitGenesis(ctx, app.PerpKeeperV2, *genState)

	// export again to ensure they match
	genStateAfterInit := perp.ExportGenesis(ctx, app.PerpKeeperV2)

	require.Equal(t, genState.Params, genStateAfterInit.Params)

	for i, pm := range genState.Markets {
		require.Equal(t, pm, genStateAfterInit.Markets[i])
	}
	for i, amm := range genState.Amms {
		require.Equal(t, amm, genStateAfterInit.Amms[i])
	}

	require.Equal(t, len(genState.Positions), len(genStateAfterInit.Positions))
	for i, pos := range genState.Positions {
		require.Equalf(t, pos, genStateAfterInit.Positions[i], "%s <-> %s", pos, genStateAfterInit.Positions[i])
	}
}
