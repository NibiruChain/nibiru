package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

type msgServer struct {
	keeper *Keeper
}

func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{
		keeper: keeper,
	}
}

// SetFeeTokens sets the provided fee tokens for the chain. The sender must be whitelisted to set fee tokens.
func (server msgServer) SetFeeTokens(goCtx context.Context, msg *types.MsgSetFeeTokens) (*types.MsgSetFeeTokensResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if len(msg.FeeTokens) != 1 {
		return nil, fmt.Errorf("exactly one fee token must be provided, got %d", len(msg.FeeTokens))
	}

	err := server.keeper.setFeeToken(ctx, msg.FeeTokens[0])
	if err != nil {
		return nil, fmt.Errorf("failed to set fee token: %w", err)
	}

	return &types.MsgSetFeeTokensResponse{}, nil
}

// Contains returns true if the slice contains the item, false otherwise.
func Contains[T comparable](slice []T, item T) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
