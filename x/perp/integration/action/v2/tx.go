package action

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServerOpenPosition struct {
	pair              asset.Pair
	traderAddress     sdk.AccAddress
	dir               v2types.Direction
	quoteAssetAmt     sdk.Int
	leverage          sdk.Dec
	baseAssetAmtLimit sdk.Int
}

func (m msgServerOpenPosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	// don't need to check response because it's already checked in clearing_house tests
	_, err := msgServer.OpenPosition(sdk.WrapSDKContext(ctx), &v2types.MsgOpenPosition{
		Pair:                 m.pair,
		Sender:               m.traderAddress.String(),
		Side:                 m.dir,
		QuoteAssetAmount:     m.quoteAssetAmt,
		Leverage:             m.leverage,
		BaseAssetAmountLimit: m.baseAssetAmtLimit,
	})

	return ctx, err, true
}

func MsgServerOpenPosition(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	dir v2types.Direction,
	quoteAssetAmt sdk.Int,
	leverage sdk.Dec,
	baseAssetAmtLimit sdk.Int,
) action.Action {
	return msgServerOpenPosition{
		pair:              pair,
		traderAddress:     traderAddress,
		dir:               dir,
		quoteAssetAmt:     quoteAssetAmt,
		leverage:          leverage,
		baseAssetAmtLimit: baseAssetAmtLimit,
	}
}

type msgServerClosePosition struct {
	pair          asset.Pair
	traderAddress sdk.AccAddress
}

func (m msgServerClosePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	// don't need to check response because it's already checked in clearing_house tests
	_, err := msgServer.ClosePosition(sdk.WrapSDKContext(ctx), &v2types.MsgClosePosition{
		Pair:   m.pair,
		Sender: m.traderAddress.String(),
	})

	return ctx, err, true
}

func MsgServerClosePosition(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
) action.Action {
	return msgServerClosePosition{
		pair:          pair,
		traderAddress: traderAddress,
	}
}

type msgServerAddmargin struct {
	pair          asset.Pair
	traderAddress sdk.AccAddress
	amount        sdk.Int
}

func (m msgServerAddmargin) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	// don't need to check response because it's already checked in clearing_house tests
	_, err := msgServer.AddMargin(sdk.WrapSDKContext(ctx), &v2types.MsgAddMargin{
		Pair:   m.pair,
		Sender: m.traderAddress.String(),
		Margin: sdk.NewCoin(m.pair.QuoteDenom(), m.amount),
	})

	return ctx, err, true
}

func MsgServerAddMargin(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	amount sdk.Int,
) action.Action {
	return msgServerAddmargin{
		pair:          pair,
		traderAddress: traderAddress,
		amount:        amount,
	}
}

type msgServerRemoveMargin struct {
	pair          asset.Pair
	traderAddress sdk.AccAddress
	amount        sdk.Int
}

func (m msgServerRemoveMargin) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	// don't need to check response because it's already checked in clearing_house tests
	_, err := msgServer.RemoveMargin(sdk.WrapSDKContext(ctx), &v2types.MsgRemoveMargin{
		Pair:   m.pair,
		Sender: m.traderAddress.String(),
		Margin: sdk.NewCoin(m.pair.QuoteDenom(), m.amount),
	})

	return ctx, err, true
}

func MsgServerRemoveMargin(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	amount sdk.Int,
) action.Action {
	return msgServerRemoveMargin{
		pair:          pair,
		traderAddress: traderAddress,
		amount:        amount,
	}
}
