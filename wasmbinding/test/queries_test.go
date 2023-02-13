package wasmbinding

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/x/common/asset"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/wasmbinding"
	"github.com/NibiruChain/nibiru/x/common"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestPosition(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	ctx = ctx.WithBlockTime(time.Now())
	tokenPair := asset.MustNewPair("BTC:NUSD")
	querier := wasmbinding.NewQueryPlugin(&app.PerpKeeper)

	specs := map[string]struct {
		trader string
		pair   string
		expErr bool
	}{
		"valid trader and pair": {
			trader: actor.String(),
			pair:   "BTC:NUSD",
		},
		"empty trader": {
			trader: "",
			pair:   "BTC:NUSD",
			expErr: true,
		},
		"empty pair": {
			trader: actor.String(),
			pair:   "",
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Log("Create vpool")
			vpoolKeeper := &app.VpoolKeeper
			perpKeeper := &app.PerpKeeper
			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				sdk.NewDec(10*common.Precision),
				sdk.NewDec(5*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))
			app.OracleKeeper.SetPrice(ctx, tokenPair, sdk.NewDec(2))

			pairMetadata := perptypes.PairMetadata{
				Pair:                            tokenPair,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			}
			perpKeeper.PairsMetadata.Insert(ctx, pairMetadata.Pair, pairMetadata)

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

			res, gotErr := querier.GetPosition(ctx, spec.trader, spec.pair)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}

			require.NoError(t, gotErr)
			require.Equal(t, res.Position.Pair.String(), spec.pair)
			require.Equal(t, res.Position.TraderAddress, spec.trader)
		})
	}
}

func TestPositions(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	ctx = ctx.WithBlockTime(time.Now())
	tokenPair := asset.MustNewPair("BTC:NUSD")
	querier := wasmbinding.NewQueryPlugin(&app.PerpKeeper)

	specs := map[string]struct {
		trader string
		pair   string
		expErr bool
	}{
		"valid trader": {
			trader: actor.String(),
		},
		"empty trader": {
			trader: "",
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Log("Create vpool")
			vpoolKeeper := &app.VpoolKeeper
			perpKeeper := &app.PerpKeeper
			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				sdk.NewDec(10*common.Precision),
				sdk.NewDec(5*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))
			app.OracleKeeper.SetPrice(ctx, tokenPair, sdk.NewDec(2))

			pairMetadata := perptypes.PairMetadata{
				Pair:                            tokenPair,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			}
			perpKeeper.PairsMetadata.Insert(ctx, pairMetadata.Pair, pairMetadata)

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

			res, gotErr := querier.GetPositions(ctx, spec.trader)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}

			require.NoError(t, gotErr)
			require.Equal(t, res.Positions[0].Position.TraderAddress, spec.trader)
		})
	}
}
