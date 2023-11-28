package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
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

func (m msgServer) ClosePosition(goCtx context.Context, req *types.MsgClosePosition) (*types.MsgClosePositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	traderAddr := sdk.MustAccAddressFromBech32(req.Sender)

	resp, err := m.k.ClosePosition(ctx, req.Pair, traderAddr)
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

func (m msgServer) PartialClose(goCtx context.Context, req *types.MsgPartialClose) (*types.MsgPartialCloseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	traderAddr := sdk.MustAccAddressFromBech32(req.Sender)

	resp, err := m.k.PartialClose(ctx, req.Pair, traderAddr, req.Size_)
	if err != nil {
		return nil, err
	}

	return &types.MsgPartialCloseResponse{
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

func (m msgServer) SettlePosition(ctx context.Context, msg *types.MsgSettlePosition) (*types.MsgClosePositionResponse, error) {
	// These fields should have already been validated by MsgSettlePosition.ValidateBasic() prior to being sent to the msgServer.
	traderAddr := sdk.MustAccAddressFromBech32(msg.Sender)
	resp, err := m.k.SettlePosition(sdk.UnwrapSDKContext(ctx), msg.Pair, msg.Version, traderAddr)
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

// ChangeCollateralDenom: Updates the collateral denom. A denom is valid if it is
// possible to make an sdk.Coin using it. [Admin] Only callable by sudoers.
func (m msgServer) ChangeCollateralDenom(
	goCtx context.Context, txMsg *types.MsgChangeCollateralDenom,
) (resp *types.MsgChangeCollateralDenomResponse, err error) {
	if txMsg == nil {
		return resp, common.ErrNilMsg()
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	err = m.k.Admin.ChangeCollateralDenom(ctx, txMsg.NewDenom, txMsg.GetSigners()[0])

	return &types.MsgChangeCollateralDenomResponse{}, err
}

func (m msgServer) AllocateEpochRebates(ctx context.Context, msg *types.MsgAllocateEpochRebates) (*types.MsgAllocateEpochRebatesResponse, error) {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	total, err := m.k.AllocateEpochRebates(sdk.UnwrapSDKContext(ctx), sender, msg.Rebates)
	if err != nil {
		return nil, err
	}

	return &types.MsgAllocateEpochRebatesResponse{TotalEpochRebates: total}, nil
}

func (m msgServer) WithdrawEpochRebates(ctx context.Context, msg *types.MsgWithdrawEpochRebates) (*types.MsgWithdrawEpochRebatesResponse, error) {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalWithdrawn := sdk.NewCoins()
	for _, epoch := range msg.Epochs {
		withdrawn, err := m.k.WithdrawEpochRebates(sdkCtx, epoch, sender)
		if err != nil {
			return nil, err
		}
		totalWithdrawn = totalWithdrawn.Add(withdrawn...)
	}
	return &types.MsgWithdrawEpochRebatesResponse{
		WithdrawnRebates: totalWithdrawn,
	}, nil
}

// ShiftPegMultiplier: gRPC tx msg for changing a market's peg multiplier.
// [Admin] Only callable by sudoers.
func (m msgServer) ShiftPegMultiplier(
	goCtx context.Context, msg *types.MsgShiftPegMultiplier,
) (*types.MsgShiftPegMultiplierResponse, error) {
	// The `msg` here is checked with ValidateBasic before this fn is called.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	ctx := sdk.UnwrapSDKContext(goCtx)
	err := m.k.Admin.ShiftPegMultiplier(ctx, msg.Pair, msg.NewPegMult, sender)
	return &types.MsgShiftPegMultiplierResponse{}, err
}

// ShiftSwapInvariant: gRPC tx msg for changing a market's swap invariant.
// [Admin] Only callable by sudoers.
func (m msgServer) ShiftSwapInvariant(
	goCtx context.Context, msg *types.MsgShiftSwapInvariant,
) (*types.MsgShiftSwapInvariantResponse, error) {
	// The `msg` here is checked with ValidateBasic before this fn is called.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	ctx := sdk.UnwrapSDKContext(goCtx)
	err := m.k.Admin.ShiftSwapInvariant(ctx, msg.Pair, msg.NewSwapInvariant, sender)
	return &types.MsgShiftSwapInvariantResponse{}, err
}
