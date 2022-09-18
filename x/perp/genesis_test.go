package perp_test

import (
	"fmt"
	"testing"
	"time"

	simapp2 "github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
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
			require.NoError(t, app.PerpKeeper.PositionsState(ctx).Create(&types.Position{
				TraderAddress:                  sample.AccAddress().String(),
				Pair:                           common.Pair_NIBI_NUSD,
				Size_:                          sdk.NewDec(i + 1),
				Margin:                         sdk.NewDec(i * 2),
				OpenNotional:                   sdk.NewDec(i * 100),
				LatestCumulativeFundingPayment: sdk.NewDec(5 * 100),
				BlockNumber:                    i,
			}))
		}

		// create some prepaid bad debt
		for i := 0; i < 10; i++ {
			app.PerpKeeper.PrepaidBadDebtState(ctx).Set(fmt.Sprintf("%d", i), sdk.NewInt(int64(i)))
		}

		// export genesis
		genState := perp.ExportGenesis(ctx, app.PerpKeeper)

		// create new context and init genesis
		ctx, _ = ctxUncached.CacheContext()
		perp.InitGenesis(ctx, app.PerpKeeper, *genState)

		// export again to ensure they match
		genStateAfterInit := perp.ExportGenesis(ctx, app.PerpKeeper)
		require.Equal(t, genState, genStateAfterInit)
	})
}
