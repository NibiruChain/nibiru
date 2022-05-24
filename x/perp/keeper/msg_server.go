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

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) RemoveMargin(ctx context.Context, margin *types.MsgRemoveMargin) (*types.MsgRemoveMarginResponse, error) {
	return k.k.RemoveMargin(ctx, margin)
}

func (k msgServer) AddMargin(ctx context.Context, margin *types.MsgAddMargin) (*types.MsgAddMarginResponse, error) {
	return k.k.AddMargin(ctx, margin)
}

func (k msgServer) OpenPosition(goCtx context.Context, req *types.MsgOpenPosition) (*types.MsgOpenPositionResponse, error) {
	pair, err := common.NewTokenPairFromStr(req.TokenPair)
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
		return nil, err
	}

	return &types.MsgOpenPositionResponse{}, nil
}
