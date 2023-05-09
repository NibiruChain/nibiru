package keeper

import (
	"context"

	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	k Keeper
}

var _ v2types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) v2types.MsgServer {
	return &msgServer{k: keeper}
}

func (m msgServer) RemoveMargin(ctx context.Context, msg *v2types.MsgRemoveMargin,
) (*v2types.MsgRemoveMarginResponse, error) {
	// These fields should have already been validated by MsgRemoveMargin.ValidateBasic() prior to being sent to the msgServer.
	traderAddr := sdk.MustAccAddressFromBech32(msg.Sender)
	return m.k.RemoveMargin(sdk.UnwrapSDKContext(ctx), msg.Pair, traderAddr, msg.Margin)
}

func (m msgServer) AddMargin(ctx context.Context, msg *v2types.MsgAddMargin,
) (*v2types.MsgAddMarginResponse, error) {
	// These fields should have already been validated by MsgAddMargin.ValidateBasic() prior to being sent to the msgServer.
	traderAddr := sdk.MustAccAddressFromBech32(msg.Sender)
	return m.k.AddMargin(sdk.UnwrapSDKContext(ctx), msg.Pair, traderAddr, msg.Margin)
}

func (m msgServer) OpenPosition(goCtx context.Context, req *v2types.MsgOpenPosition,
) (response *v2types.MsgOpenPositionResponse, err error) {
	traderAddr := sdk.MustAccAddressFromBech32(req.Sender)

	positionResp, err := m.k.OpenPosition(
		sdk.UnwrapSDKContext(goCtx),
		req.Pair,
		req.Side,
		traderAddr,
		req.QuoteAssetAmount,
		req.Leverage,
		req.BaseAssetAmountLimit.ToDec(),
	)
	if err != nil {
		return nil, err
	}

	return &v2types.MsgOpenPositionResponse{
		Position:               positionResp.Position,
		ExchangedNotionalValue: positionResp.ExchangedNotionalValue,
		ExchangedPositionSize:  positionResp.ExchangedPositionSize,
		FundingPayment:         positionResp.FundingPayment,
		RealizedPnl:            positionResp.RealizedPnl,
		UnrealizedPnlAfter:     positionResp.UnrealizedPnlAfter,
		MarginToVault:          positionResp.MarginToVault,
		PositionNotional:       positionResp.PositionNotional,
	}, nil
}

func (m msgServer) ClosePosition(goCtx context.Context, position *v2types.MsgClosePosition) (*v2types.MsgClosePositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	traderAddr := sdk.MustAccAddressFromBech32(position.Sender)

	resp, err := m.k.ClosePosition(ctx, position.Pair, traderAddr)
	if err != nil {
		return nil, err
	}

	return &v2types.MsgClosePositionResponse{
		ExchangedNotionalValue: resp.ExchangedNotionalValue,
		ExchangedPositionSize:  resp.ExchangedPositionSize,
		FundingPayment:         resp.FundingPayment,
		RealizedPnl:            resp.RealizedPnl,
		MarginToTrader:         resp.MarginToVault.Neg(),
	}, nil
}

func (m msgServer) MultiLiquidate(goCtx context.Context, req *v2types.MsgMultiLiquidate) (*v2types.MsgMultiLiquidateResponse, error) {
	resp, err := m.k.MultiLiquidate(sdk.UnwrapSDKContext(goCtx), sdk.MustAccAddressFromBech32(req.Sender), req.Liquidations)
	if err != nil {
		return nil, err
	}

	return &v2types.MsgMultiLiquidateResponse{Liquidations: resp}, nil
}

func (m msgServer) DonateToEcosystemFund(ctx context.Context, msg *v2types.MsgDonateToEcosystemFund) (*v2types.MsgDonateToEcosystemFundResponse, error) {
	if err := m.k.BankKeeper.SendCoinsFromAccountToModule(
		sdk.UnwrapSDKContext(ctx),
		sdk.MustAccAddressFromBech32(msg.Sender),
		v2types.PerpEFModuleAccount,
		sdk.NewCoins(msg.Donation),
	); err != nil {
		return nil, err
	}

	return &v2types.MsgDonateToEcosystemFundResponse{}, nil
}
