package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
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

func (k msgServer) RemoveMargin(ctx context.Context, margin *types.MsgRemoveMargin,
) (*types.MsgRemoveMarginResponse, error) {
	return k.k.RemoveMargin(ctx, margin)
}

func (k msgServer) AddMargin(ctx context.Context, margin *types.MsgAddMargin,
) (*types.MsgAddMarginResponse, error) {
	return k.k.AddMargin(ctx, margin)
}

func (k msgServer) OpenPosition(goCtx context.Context, req *types.MsgOpenPosition,
) (*types.MsgOpenPositionResponse, error) {
	pair, err := common.NewAssetPairFromStr(req.TokenPair)
	if err != nil {
		panic(err) // must not happen
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	err = k.k.OpenPosition(
		ctx,
		pair,
		req.Side,
		req.Sender,
		req.QuoteAssetAmount,
		req.Leverage,
		req.BaseAssetAmountLimit.ToDec(),
	)
	if err != nil {
		return nil, sdkerrors.Wrap(vpooltypes.ErrOpeningPosition, err.Error())
	}

	return &types.MsgOpenPositionResponse{}, nil
}

func (k msgServer) Liquidate(goCtx context.Context, msg *types.MsgLiquidate,
) (*types.MsgLiquidateResponse, error) {
	response, err := k.k.Liquidate(goCtx, msg)
	if err != nil {
		return nil, err
	}

	return response, nil
}
