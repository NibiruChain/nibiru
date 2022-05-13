package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestFullLiquidate(t *testing.T) {
	tests := []struct {
		name                      string
		positionSize              sdk.Dec
		initialEF                 sdk.Dec
		indexPrice                sdk.Dec
		otherPositionSize         sdk.Dec
		expectedErr               error
		expectedLiquidatorBalance sdk.Dec
	}{
		{
			name:              "happy path",
			initialEF:         sdk.NewDec(100),
			indexPrice:        sdk.MustNewDecFromStr("1"),
			positionSize:      sdk.NewDec(1000),
			otherPositionSize: sdk.NewDec(1000),
			expectedErr:       types.MarginHighEnough,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)

			tokenPair, err := common.NewTokenPairFromStr("atom:unusd")
			require.NoError(t, err)

			t.Log("add margin funds (NUSD) to traders' account")
			traderAddr := sample.AccAddress()
			otherTraderAddr := sample.AccAddress()
			liquidatorAddr := sample.AccAddress()
			oracle := sample.AccAddress()

			// post oracle price
			mp := pftypes.Params{
				Pairs: []pftypes.Pair{
					{Token1: "atom", Token0: "unusd", Oracles: []sdk.AccAddress{oracle}, Active: true},
				},
			}
			app.PriceKeeper.SetParams(ctx, mp)
			_, err = app.PriceKeeper.SetPrice(
				ctx,
				oracle,
				"atom",
				"unusd",
				tc.indexPrice,
				time.Now().UTC().Add(1*time.Hour),
			)
			require.NoError(t, err, "Error posting price for pair")
			err = app.PriceKeeper.SetCurrentPrices(ctx, "atom", "unusd")
			require.NoError(t, err, "Error setting price for pair")

			// Create vPool to get the spot price
			app.VpoolKeeper.CreatePool(
				ctx,
				tokenPair.String(),
				sdk.MustNewDecFromStr("0.9"),  // 0.9 ratio
				sdk.NewDec(10_000_000),        // 10 tokens
				sdk.NewDec(5_000_000),         // 5 tokens
				sdk.MustNewDecFromStr("0.25"), // 0.25 ratio
				sdk.MustNewDecFromStr("0.25"), // 0.25 ratio
			)
			require.True(t, app.VpoolKeeper.ExistsPool(ctx, tokenPair))

			app.PerpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       tokenPair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			err = simapp.FundAccount(
				app.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(common.StableDenom, tc.positionSize.TruncateInt().Add(sdk.NewInt(2))), // Adding 2 for both fees (toll and spread)
				),
			)
			require.NoErrorf(t, err, "fund account call should work")
			err = simapp.FundAccount(
				app.BankKeeper,
				ctx,
				otherTraderAddr,
				sdk.NewCoins(
					sdk.NewCoin(common.StableDenom, tc.otherPositionSize.TruncateInt().Add(sdk.NewInt(2))),
				),
			)
			require.NoErrorf(t, err, "fund account call should work")

			t.Log("add liquidation funds to perp EF")
			err = simapp.FundModuleAccount(
				app.BankKeeper,
				ctx,
				common.TreasuryPoolModuleAccount,
				sdk.NewCoins(sdk.NewCoin(common.StableDenom, tc.initialEF.TruncateInt())),
			)
			require.NoErrorf(t, err, "fund module call should work")

			t.Log("establish initial position")
			err = app.PerpKeeper.OpenPosition(
				ctx, tokenPair, types.Side_BUY, traderAddr, tc.positionSize.TruncateInt(), sdk.OneDec(), sdk.NewInt(150),
			)
			require.NoError(t, err, "initial position should be opened")
			err = app.PerpKeeper.OpenPosition(
				ctx, tokenPair, types.Side_SELL, otherTraderAddr, tc.otherPositionSize.TruncateInt(), sdk.OneDec(), sdk.NewInt(500),
			)
			require.NoError(t, err, "second position should be opened")

			t.Log("liquidate position")
			err = app.PerpKeeper.Liquidate(ctx, tokenPair, traderAddr.String(), liquidatorAddr)
			require.ErrorIs(t, err, tc.expectedErr)
			panic(nil)
		})
	}
}
