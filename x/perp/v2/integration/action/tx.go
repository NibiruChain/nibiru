package action

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type msgServerMarketOrder struct {
	pair              asset.Pair
	traderAddress     sdk.AccAddress
	dir               types.Direction
	quoteAssetAmt     sdkmath.Int
	leverage          sdk.Dec
	baseAssetAmtLimit sdkmath.Int
}

func (m msgServerMarketOrder) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	// don't need to check response because it's already checked in clearing_house tests
	_, err := msgServer.MarketOrder(sdk.WrapSDKContext(ctx), &types.MsgMarketOrder{
		Pair:                 m.pair,
		Sender:               m.traderAddress.String(),
		Side:                 m.dir,
		QuoteAssetAmount:     m.quoteAssetAmt,
		Leverage:             m.leverage,
		BaseAssetAmountLimit: m.baseAssetAmtLimit,
	})

	return ctx, err, true
}

func MsgServerMarketOrder(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	dir types.Direction,
	quoteAssetAmt sdkmath.Int,
	leverage sdk.Dec,
	baseAssetAmtLimit sdkmath.Int,
) action.Action {
	return msgServerMarketOrder{
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
	_, err := msgServer.ClosePosition(sdk.WrapSDKContext(ctx), &types.MsgClosePosition{
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

type msgServerPartialClose struct {
	pair          asset.Pair
	traderAddress sdk.AccAddress
	size          sdk.Dec
}

func (m msgServerPartialClose) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	// don't need to check response because it's already checked in clearing_house tests
	_, err := msgServer.PartialClose(sdk.WrapSDKContext(ctx), &types.MsgPartialClose{
		Pair:   m.pair,
		Sender: m.traderAddress.String(),
		Size_:  m.size,
	})

	return ctx, err, true
}

func MsgServerPartialClosePosition(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	size sdk.Dec,
) action.Action {
	return msgServerPartialClose{
		pair:          pair,
		traderAddress: traderAddress,
		size:          size,
	}
}

type msgServerAddmargin struct {
	pair          asset.Pair
	traderAddress sdk.AccAddress
	amount        sdkmath.Int
}

func (m msgServerAddmargin) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	// don't need to check response because it's already checked in clearing_house tests
	_, err = msgServer.AddMargin(sdk.WrapSDKContext(ctx), &types.MsgAddMargin{
		Pair:   m.pair,
		Sender: m.traderAddress.String(),
		Margin: sdk.NewCoin(collateral, m.amount),
	})

	return ctx, err, true
}

func MsgServerAddMargin(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	amount sdkmath.Int,
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
	amount        sdkmath.Int
}

func (m msgServerRemoveMargin) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	// don't need to check response because it's already checked in clearing_house tests
	_, err = msgServer.RemoveMargin(sdk.WrapSDKContext(ctx), &types.MsgRemoveMargin{
		Pair:   m.pair,
		Sender: m.traderAddress.String(),
		Margin: sdk.NewCoin(collateral, m.amount),
	})

	return ctx, err, true
}

func MsgServerRemoveMargin(
	traderAddress sdk.AccAddress,
	pair asset.Pair,
	amount sdkmath.Int,
) action.Action {
	return msgServerRemoveMargin{
		pair:          pair,
		traderAddress: traderAddress,
		amount:        amount,
	}
}

type msgServerDonateToPerpEf struct {
	sender sdk.AccAddress
	amount sdkmath.Int
}

func (m msgServerDonateToPerpEf) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	_, err = msgServer.DonateToEcosystemFund(sdk.WrapSDKContext(ctx), &types.MsgDonateToEcosystemFund{
		Sender:   m.sender.String(),
		Donation: sdk.NewCoin(collateral, m.amount),
	})

	return ctx, err, true
}

func MsgServerDonateToPerpEf(
	traderAddress sdk.AccAddress,
	amount sdkmath.Int,
) action.Action {
	return msgServerDonateToPerpEf{
		sender: traderAddress,
		amount: amount,
	}
}

type msgServerMultiLiquidate struct {
	pairTraderTuples []PairTraderTuple
	liquidator       sdk.AccAddress
	shouldAllFail    bool
}

func (m msgServerMultiLiquidate) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

	liquidateMsgs := make([]*types.MsgMultiLiquidate_Liquidation, len(m.pairTraderTuples))
	for i, pairTraderTuple := range m.pairTraderTuples {
		liquidateMsgs[i] = &types.MsgMultiLiquidate_Liquidation{
			Pair:   pairTraderTuple.Pair,
			Trader: pairTraderTuple.Trader.String(),
		}
	}

	_, err := msgServer.MultiLiquidate(sdk.WrapSDKContext(ctx), &types.MsgMultiLiquidate{
		Sender:       m.liquidator.String(),
		Liquidations: liquidateMsgs,
	})

	return ctx, err, m.shouldAllFail
}

func MsgServerMultiLiquidate(liquidator sdk.AccAddress, shouldAllFail bool, pairTraderTuples ...PairTraderTuple) action.Action {
	return msgServerMultiLiquidate{
		pairTraderTuples: pairTraderTuples,
		liquidator:       liquidator,
		shouldAllFail:    shouldAllFail,
	}
}
