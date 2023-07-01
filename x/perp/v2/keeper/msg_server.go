package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func (m msgServer) RemoveMargin(ctx context.Context, msg *types.MsgRemoveMargin,
) (*types.MsgRemoveMarginResponse, error) {
	// These fields should have already been validated by MsgRemoveMargin.ValidateBasic() prior to being sent to the msgServer.
	traderAddr := sdk.MustAccAddressFromBech32(msg.Sender)
	return m.k.RemoveMargin(sdk.UnwrapSDKContext(ctx), msg.Pair, traderAddr, msg.Margin)
}

func (m msgServer) AddMargin(ctx context.Context, msg *types.MsgAddMargin,
) (*types.MsgAddMarginResponse, error) {
	// These fields should have already been validated by MsgAddMargin.ValidateBasic() prior to being sent to the msgServer.
	traderAddr := sdk.MustAccAddressFromBech32(msg.Sender)
	return m.k.AddMargin(sdk.UnwrapSDKContext(ctx), msg.Pair, traderAddr, msg.Margin)
}

func (m msgServer) MarketOrder(goCtx context.Context, req *types.MsgMarketOrder,
) (response *types.MsgMarketOrderResponse, err error) {
	traderAddr := sdk.MustAccAddressFromBech32(req.Sender)

	positionResp, err := m.k.MarketOrder(
		sdk.UnwrapSDKContext(goCtx),
		req.Pair,
		req.Side,
		traderAddr,
		req.QuoteAssetAmount,
		req.Leverage,
		sdk.NewDecFromInt(req.BaseAssetAmountLimit),
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgMarketOrderResponse{
		Position:               &positionResp.Position,
		ExchangedNotionalValue: positionResp.ExchangedNotionalValue,
		ExchangedPositionSize:  positionResp.ExchangedPositionSize,
		FundingPayment:         positionResp.FundingPayment,
		RealizedPnl:            positionResp.RealizedPnl,
		UnrealizedPnlAfter:     positionResp.UnrealizedPnlAfter,
		MarginToVault:          positionResp.MarginToVault,
		PositionNotional:       positionResp.PositionNotional,
	}, nil
}

func (m msgServer) ClosePosition(goCtx context.Context, position *types.MsgClosePosition) (*types.MsgClosePositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	traderAddr := sdk.MustAccAddressFromBech32(position.Sender)

	resp, err := m.k.ClosePosition(ctx, position.Pair, traderAddr)
	if err != nil {
		return nil, err
	}

	return &types.MsgClosePositionResponse{
		ExchangedNotionalValue: resp.ExchangedNotionalValue,
		ExchangedPositionSize:  resp.ExchangedPositionSize,
		FundingPayment:         resp.FundingPayment,
		RealizedPnl:            resp.RealizedPnl,
		MarginToTrader:         resp.MarginToVault.Neg(),
	}, nil
}

func (m msgServer) MultiLiquidate(goCtx context.Context, req *types.MsgMultiLiquidate) (*types.MsgMultiLiquidateResponse, error) {
	resp, err := m.k.MultiLiquidate(sdk.UnwrapSDKContext(goCtx), sdk.MustAccAddressFromBech32(req.Sender), req.Liquidations)
	if err != nil {
		return nil, err
	}

	return &types.MsgMultiLiquidateResponse{Liquidations: resp}, nil
}

func (m msgServer) DonateToEcosystemFund(ctx context.Context, msg *types.MsgDonateToEcosystemFund) (*types.MsgDonateToEcosystemFundResponse, error) {
	if err := m.k.BankKeeper.SendCoinsFromAccountToModule(
		sdk.UnwrapSDKContext(ctx),
		sdk.MustAccAddressFromBech32(msg.Sender),
		types.PerpEFModuleAccount,
		sdk.NewCoins(msg.Donation),
	); err != nil {
		return nil, err
	}

	return &types.MsgDonateToEcosystemFundResponse{}, nil
}
