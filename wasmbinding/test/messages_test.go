package wasmbinding

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/wasmbinding"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"

	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/stretchr/testify/assert"
)

func fundAccount(t *testing.T, ctx sdk.Context, app *app.NibiruApp, addr sdk.AccAddress, coins sdk.Coins) {
	err := simapp.FundAccount(
		app.BankKeeper,
		ctx,
		addr,
		coins,
	)
	require.NoError(t, err)
}

func TestOpenPosition(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	ctx = ctx.WithBlockTime(time.Now())
	tokenPair := asset.MustNewPair("BTC:NUSD")

	specs := map[string]struct {
		openPosition *bindings.OpenPosition
		expErr       bool
	}{
		"valid open-position": {
			openPosition: &bindings.OpenPosition{
				Pair:                 "BTC:NUSD",
				Side:                 int(perptypes.Side_BUY),
				QuoteAssetAmount:     sdk.NewInt(10),
				Leverage:             sdk.OneDec(),
				BaseAssetAmountLimit: sdk.ZeroInt(),
			},
		},
		"invalid open-position": {
			openPosition: &bindings.OpenPosition{
				Pair: "",
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Log("Create vpool")
			PreparePool(t, app, ctx, tokenPair)
			perpKeeper := &app.PerpKeeper

			t.Log("Fund trader account with sufficient quote")
			fundAccount(t, ctx, app, actor, sdk.NewCoins(sdk.NewInt64Coin("NUSD", 50_100)))

			t.Log("Increment block height and time for TWAP calculation")
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
				WithBlockTime(time.Now().Add(time.Minute))

			t.Log("Open position")
			gotErr := wasmbinding.PerformOpenPosition(perpKeeper, ctx, actor, spec.openPosition)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestClosePosition(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	ctx = ctx.WithBlockTime(time.Now())
	tokenPair := asset.MustNewPair("BTC:NUSD")

	specs := map[string]struct {
		closePosition *bindings.ClosePosition
		expErr        bool
	}{
		"valid close-position": {
			closePosition: &bindings.ClosePosition{
				Pair: "BTC:NUSD",
			},
		},
		"invalid close-position": {
			closePosition: &bindings.ClosePosition{
				Pair: "",
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Log("Create vpool")
			PreparePool(t, app, ctx, tokenPair)
			perpKeeper := &app.PerpKeeper

			t.Log("Fund trader account with sufficient quote")
			fundAccount(t, ctx, app, actor, sdk.NewCoins(sdk.NewInt64Coin("NUSD", 50_100)))

			t.Log("Increment block height and time for TWAP calculation")
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
				WithBlockTime(time.Now().Add(time.Minute))

			t.Log("Open position")
			assert.NoError(t, wasmbinding.PerformOpenPosition(perpKeeper, ctx, actor, &bindings.OpenPosition{
				Pair:                 "BTC:NUSD",
				Side:                 int(perptypes.Side_BUY),
				QuoteAssetAmount:     sdk.NewInt(10),
				Leverage:             sdk.OneDec(),
				BaseAssetAmountLimit: sdk.ZeroInt(),
			}))

			t.Log("Close position")
			gotErr := wasmbinding.PerformClosePosition(perpKeeper, ctx, actor, spec.closePosition)

			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}
