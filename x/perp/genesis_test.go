package perp_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/testutil"

	simapp2 "github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestGenesis(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app := simapp2.NewTestNibiruApp(false)
		ctxUncached := app.NewContext(false, tmproto.Header{})
		ctx, _ := ctxUncached.CacheContext()

		// create some params
		app.PerpKeeper.SetParams(ctx, types.Params{
			Stopped:                 true,
			FeePoolFeeRatio:         sdk.MustNewDecFromStr("0.00001"),
			EcosystemFundFeeRatio:   sdk.MustNewDecFromStr("0.000005"),
			LiquidationFeeRatio:     sdk.MustNewDecFromStr("0.000007"),
			PartialLiquidationRatio: sdk.MustNewDecFromStr("0.00001"),
			TwapLookbackWindow:      15 * time.Minute,
		})

		// create some positions
		for i := int64(0); i < 100; i++ {
			addr := testutil.AccAddress()
			app.PerpKeeper.Positions.Insert(ctx, collections.Join(common.AssetRegistry.Pair(common.DenomNIBI, common.DenomNUSD), addr), types.Position{
				TraderAddress:                   addr.String(),
				Pair:                            common.AssetRegistry.Pair(common.DenomNIBI, common.DenomNUSD),
				Size_:                           sdk.NewDec(i + 1),
				Margin:                          sdk.NewDec(i * 2),
				OpenNotional:                    sdk.NewDec(i * 100),
				LatestCumulativePremiumFraction: sdk.NewDec(5 * 100),
				BlockNumber:                     i,
			})
		}

		// create some prepaid bad debt
		for i := 0; i < 10; i++ {
			app.PerpKeeper.PrepaidBadDebt.Insert(ctx, fmt.Sprintf("%d", i), types.PrepaidBadDebt{
				Denom:  fmt.Sprintf("%d", i),
				Amount: sdk.NewInt(int64(i)),
			})
		}

		// export genesis
		genState := perp.ExportGenesis(ctx, app.PerpKeeper)

		// create new context and init genesis
		ctx, _ = ctxUncached.CacheContext()
		perp.InitGenesis(ctx, app.PerpKeeper, *genState)

		// export again to ensure they match
		genStateAfterInit := perp.ExportGenesis(ctx, app.PerpKeeper)
		require.Equal(t, genState.Params, genStateAfterInit.Params)
		for i, pm := range genState.PairMetadata {
			require.Equal(t, pm, genStateAfterInit.PairMetadata[i])
		}
		require.Equalf(t, genState.PairMetadata, genStateAfterInit.PairMetadata, "%s <-> %s", genState.PairMetadata, genStateAfterInit.PairMetadata)
		require.Equal(t, genState.PrepaidBadDebts, genStateAfterInit.PrepaidBadDebts)
		require.Equal(t, len(genState.Positions), len(genStateAfterInit.Positions))
		for i, pos := range genState.Positions {
			require.Equalf(t, pos, genStateAfterInit.Positions[i], "%s <-> %s", pos, genStateAfterInit.Positions[i])
		}
	})
}
