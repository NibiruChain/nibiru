package wasmbinding

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/NibiruChain/nibiru/wasmbinding"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/stretchr/testify/assert"
)

func TestOpenPositionMsg(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	tokenPair := "BTC:NUSD"
	pair := asset.MustNewPair(tokenPair)

	PreparePool(t, app, ctx, pair)
	lucky := RandomAccountAddress()
	perp := instantiatePerpContract(t, ctx, app, lucky)
	require.NotEmpty(t, perp)

	t.Log("Fund trader account with sufficient quote")
	fundAccount(t, ctx, app, perp, sdk.NewCoins(sdk.NewInt64Coin("NUSD", 50_100)))

	msg := bindings.NibiruMsg{OpenPosition: &bindings.OpenPosition{
		Pair:                 tokenPair,
		Side:                 int(perptypes.Side_BUY),
		QuoteAssetAmount:     sdk.NewInt(10),
		Leverage:             sdk.OneDec(),
		BaseAssetAmountLimit: sdk.ZeroInt(),
	}}
	err := executeCustom(t, ctx, app, perp, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// query the denom and see if it matches
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

func TestClosePositionMsg(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)
	tokenPair := "BTC:NUSD"
	pair := asset.MustNewPair(tokenPair)

	PreparePool(t, app, ctx, pair)
	perpKeeper := &app.PerpKeeper
	lucky := RandomAccountAddress()
	perp := instantiatePerpContract(t, ctx, app, lucky)
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

	msg := bindings.NibiruMsg{ClosePosition: &bindings.ClosePosition{
		Pair: tokenPair,
	}}
	err := executeCustom(t, ctx, app, perp, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// query the denom and see if it matches
	query := bindings.NibiruQuery{
		Position: &bindings.Position{
			Trader: perp.String(),
			Pair:   tokenPair,
		},
	}
	resp := perptypes.QueryPositionResponse{}
	queryCustom(t, ctx, app, perp, query, &resp, true)
}

func executeCustom(t *testing.T, ctx sdk.Context, app *app.NibiruApp, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.NibiruMsg, funds sdk.Coin) error {
	perpBz, err := json.Marshal(msg)
	require.NoError(t, err)

	// no funds sent if amount is 0
	var coins sdk.Coins
	if !funds.Amount.IsNil() {
		coins = sdk.Coins{funds}
	}

	contractKeeper := keeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	_, err = contractKeeper.Execute(ctx, contract, sender, perpBz, coins)
	return err
}
