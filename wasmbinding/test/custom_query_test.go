package wasmbinding

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
)

func TestQueryPosition(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	tokenPair := "BTC:NUSD"
	pair := asset.MustNewPair(tokenPair)

	PreparePool(t, app, ctx, pair)
	perpKeeper := &app.PerpKeeper
	perp := instantiatePerpContract(t, ctx, app, actor)
	require.NotEmpty(t, perp)

	t.Log("Fund trader account with sufficient quote")
	fundAccount(t, ctx, app, perp, sdk.NewCoins(sdk.NewInt64Coin("NUSD", 50_100)))

	t.Log("Increment block height and time for TWAP calculation")
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
		WithBlockTime(time.Now().Add(time.Minute))

	t.Log("Open position")
	assert.NoError(t, wasmbinding.PerformOpenPosition(perpKeeper, ctx, perp, &bindings.OpenPosition{
		Pair:                 tokenPair,
		Side:                 int(perptypes.Side_BUY),
		QuoteAssetAmount:     sdk.NewInt(10),
		Leverage:             sdk.OneDec(),
		BaseAssetAmountLimit: sdk.ZeroInt(),
	}))

	query := bindings.NibiruQuery{
		Position: &bindings.Position{
			Trader: perp.String(),
			Pair:   tokenPair,
		},
	}
	resp := perptypes.QueryPositionResponse{}
	queryCustom(t, ctx, app, perp, query, &resp, false)

	require.Equal(t, resp.Position.Pair, pair)
}

func TestQueryPositions(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	tokenPair := "BTC:NUSD"
	pair := asset.MustNewPair(tokenPair)

	PreparePool(t, app, ctx, pair)
	perpKeeper := &app.PerpKeeper
	perp := instantiatePerpContract(t, ctx, app, actor)
	require.NotEmpty(t, perp)

	t.Log("Fund trader account with sufficient quote")
	fundAccount(t, ctx, app, perp, sdk.NewCoins(sdk.NewInt64Coin("NUSD", 50_100)))

	t.Log("Increment block height and time for TWAP calculation")
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
		WithBlockTime(time.Now().Add(time.Minute))

	t.Log("Open position")
	assert.NoError(t, wasmbinding.PerformOpenPosition(perpKeeper, ctx, perp, &bindings.OpenPosition{
		Pair:                 tokenPair,
		Side:                 int(perptypes.Side_BUY),
		QuoteAssetAmount:     sdk.NewInt(10),
		Leverage:             sdk.OneDec(),
		BaseAssetAmountLimit: sdk.ZeroInt(),
	}))

	query := bindings.NibiruQuery{
		Positions: &bindings.Positions{
			Trader: perp.String(),
		},
	}
	resp := perptypes.QueryPositionsResponse{}
	queryCustom(t, ctx, app, perp, query, &resp, false)

	require.Equal(t, resp.Positions[0].Position.Pair, pair)
}

func queryCustom(t *testing.T, ctx sdk.Context, app *app.NibiruApp, contract sdk.AccAddress, request bindings.NibiruQuery, response interface{}, shouldFail bool) {
	queryBz, err := json.Marshal(request)
	require.NoError(t, err)

	resBz, err := app.WasmKeeper.QuerySmart(ctx, contract, queryBz)
	if shouldFail {
		require.Error(t, err)
		return
	}

	require.NoError(t, err)
	err = json.Unmarshal(resBz, response)
	require.NoError(t, err)
}
