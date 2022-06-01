package perp_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/perp"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestGenesis(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app := testutil.NewTestApp(false)
		ctxUncached := app.NewContext(false, tmproto.Header{})
		ctx, _ := ctxUncached.CacheContext()
		// fund module accounts
		require.NoError(t, simapp.FundModuleAccount(
			app.BankKeeper, ctx, types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin("test", 1000))))
		require.NoError(t, simapp.FundModuleAccount(
			app.BankKeeper, ctx, types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewInt64Coin("test", 1000))))
		require.NoError(t, simapp.FundModuleAccount(
			app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin("test", 1000))))

		// create some params
		app.PerpKeeper.SetParams(ctx, types.Params{
			Stopped:                 true,
			MaintenanceMarginRatio:  sdk.NewDec(100),
			TollRatio:               10,
			SpreadRatio:             5,
			LiquidationFee:          7,
			PartialLiquidationRatio: 10,
		})
		// create some positions
		for i := int64(0); i < 100; i++ {
			require.NoError(t, app.PerpKeeper.Positions().Create(ctx, &types.Position{
				TraderAddress:                       sample.AccAddress().String(),
				Pair:                                "NIBI:USDN",
				Size_:                               sdk.NewDec(i + 1),
				Margin:                              sdk.NewDec(i * 2),
				OpenNotional:                        sdk.NewDec(i * 100),
				LastUpdateCumulativePremiumFraction: sdk.NewDec(5 * 100),
				BlockNumber:                         i,
			}))
		}

		// create some prepaid bad debt
		for i := 0; i < 10; i++ {
			app.PerpKeeper.PrepaidBadDebtState().Set(ctx, fmt.Sprintf("%d", i), sdk.NewInt(int64(i)))
		}

		// whitelist some addrs
		for i := 0; i < 5; i++ {
			app.PerpKeeper.Whitelist().Whitelist(ctx, sample.AccAddress())
		}

		// export genesis
		genState := perp.ExportGenesis(ctx, app.PerpKeeper)

		// create new context and init genesis
		ctx, _ = ctxUncached.CacheContext()
		// simulate bank genesis
		require.NoError(t, simapp.FundModuleAccount(
			app.BankKeeper, ctx, types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin("test", 1000))))
		require.NoError(t, simapp.FundModuleAccount(
			app.BankKeeper, ctx, types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewInt64Coin("test", 1000))))
		require.NoError(t, simapp.FundModuleAccount(
			app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin("test", 1000))))

		perp.InitGenesis(ctx, app.PerpKeeper, *genState)

		// export again to ensure they match
		genStateAfterInit := perp.ExportGenesis(ctx, app.PerpKeeper)
		require.Equal(t, genState, genStateAfterInit)
	})
}
