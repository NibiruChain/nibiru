package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
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
	pair := common.MustNewAssetPair(msg.TokenPair)

	marginOut, fundingPayment, position, err := m.k.RemoveMargin(sdk.UnwrapSDKContext(ctx), pair, traderAddr, msg.Margin)
	if err != nil {
		return nil, err
	}

	return &types.MsgRemoveMarginResponse{
		MarginOut:      marginOut,
		FundingPayment: fundingPayment,
		Position:       position,
	}, nil
}

func (m msgServer) AddMargin(ctx context.Context, msg *types.MsgAddMargin,
) (*types.MsgAddMarginResponse, error) {
	// These fields should have already been validated by MsgAddMargin.ValidateBasic() prior to being sent to the msgServer.
	traderAddr := sdk.MustAccAddressFromBech32(msg.Sender)
	pair := common.MustNewAssetPair(msg.TokenPair)
	return m.k.AddMargin(sdk.UnwrapSDKContext(ctx), pair, traderAddr, msg.Margin)
}

func (m msgServer) OpenPosition(goCtx context.Context, req *types.MsgOpenPosition,
) (response *types.MsgOpenPositionResponse, err error) {
	pair := common.MustNewAssetPair(req.TokenPair)
	traderAddr := sdk.MustAccAddressFromBech32(req.Sender)

	positionResp, err := m.k.OpenPosition(
		sdk.UnwrapSDKContext(goCtx),
		pair,
		req.Side,
		traderAddr,
		req.QuoteAssetAmount,
		req.Leverage,
		req.BaseAssetAmountLimit.ToDec(),
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgOpenPositionResponse{
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

func (m msgServer) ClosePosition(goCtx context.Context, position *types.MsgClosePosition) (*types.MsgClosePositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	traderAddr := sdk.MustAccAddressFromBech32(position.Sender)
	tokenPair := common.MustNewAssetPair(position.TokenPair)

	resp, err := m.k.ClosePosition(ctx, tokenPair, traderAddr)
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

func (m msgServer) Liquidate(goCtx context.Context, msg *types.MsgLiquidate,
) (*types.MsgLiquidateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	liquidatorAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	traderAddr, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return nil, err
	}

	pair, err := common.NewAssetPair(msg.TokenPair)
	if err != nil {
		return nil, err
	}

	feeToLiquidator, feeToFund, err := m.k.Liquidate(ctx, liquidatorAddr, pair, traderAddr)
	if err != nil {
		return nil, err
	}

	return &types.MsgLiquidateResponse{
		FeeToLiquidator:        feeToLiquidator,
		FeeToPerpEcosystemFund: feeToFund,
	}, nil
}

func (m msgServer) MultiLiquidate(goCtx context.Context, req *types.MsgMultiLiquidate) (*types.MsgMultiLiquidateResponse, error) {
	liquidatorAddr, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		panic(err)
	}

	liquidate := func(ctx sdk.Context, liquidator sdk.AccAddress, tokenPair string, trader string) (*types.MsgLiquidateResponse, error) {
		traderAddr, err := sdk.AccAddressFromBech32(trader)
		if err != nil {
			return nil, err
		}

		pair, err := common.NewAssetPair(tokenPair)
		if err != nil {
			return nil, err
		}

		feeToLiquidator, feeToFund, err := m.k.Liquidate(ctx, liquidatorAddr, pair, traderAddr)
		if err != nil {
			return nil, err
		}

		return &types.MsgLiquidateResponse{
			FeeToLiquidator:        feeToLiquidator,
			FeeToPerpEcosystemFund: feeToFund,
		}, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	resp := make([]*types.MsgMultiLiquidateResponse_MultiLiquidateResponse, len(req.Liquidations))
	for i, liquidation := range req.Liquidations {
		cachedCtx, commit := ctx.CacheContext()
		liq, err := liquidate(cachedCtx, liquidatorAddr, liquidation.TokenPair, liquidation.Trader)
		// in case of error, add error to liquidation responses, and skip commit
		if err != nil {
			resp[i] = &types.MsgMultiLiquidateResponse_MultiLiquidateResponse{
				Response: &types.MsgMultiLiquidateResponse_MultiLiquidateResponse_Error{Error: err.Error()},
			}
			// since there was an error we skip
			continue
		}

		// this is not skipped in case there are no errors
		resp[i] = &types.MsgMultiLiquidateResponse_MultiLiquidateResponse{
			Response: &types.MsgMultiLiquidateResponse_MultiLiquidateResponse_Liquidation{Liquidation: liq},
		}
		// tragically the sdk's CacheContext.Write function does not commit events,
		// so we have to manually do it. yikes.
		ctx.EventManager().EmitEvents(cachedCtx.EventManager().Events())
		commit()
	}

	return &types.MsgMultiLiquidateResponse{LiquidationResponses: resp}, nil
}
