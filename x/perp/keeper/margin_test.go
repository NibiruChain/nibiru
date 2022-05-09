package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestAddMargin(t *testing.T) {
	tests := []struct {
		name           string
		initialMargin  sdk.Int
		addedMargin    sdk.Int
		expectedMargin sdk.Int
	}{
		{
			name:           "add margin",
			initialMargin:  sdk.NewIntFromUint64(100),
			addedMargin:    sdk.NewIntFromUint64(100),
			expectedMargin: sdk.NewIntFromUint64(200),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)

			tokenPair, err := common.NewTokenPairFromStr("atom:nusd")
			require.NoError(t, err)

			t.Log("add margin funds (NUSD) to trader's account")
			traderAddr := sample.AccAddress()
			err = simapp.FundAccount(
				app.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(common.StableDenom, tc.addedMargin),
				),
			)
			require.NoErrorf(t, err, "fund account call should work")

			t.Log("establish initial position")
			app.PerpKeeper.SetPosition(
				ctx,
				tokenPair,
				traderAddr.String(),
				&types.Position{
					Address: traderAddr.String(),
					Pair:    tokenPair.String(),
					Size_:   sdk.NewIntFromUint64(9999),
					Margin:  tc.initialMargin,
				},
			)

			require.NoError(t,
				app.PerpKeeper.AddMargin(ctx, tokenPair, traderAddr, tc.addedMargin))

			position, err := app.PerpKeeper.GetPosition(
				ctx, tokenPair, traderAddr.String())
			require.NoError(t, err)
			require.Equal(t, tc.expectedMargin, position.Margin)
		})
	}
}
