package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"

	sdkioerrors "cosmossdk.io/errors"
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

	whitelistedAddresses := server.keeper.GetParams(ctx).WhitelistedFeeTokenSetters

	isWhitelisted := Contains(whitelistedAddresses, msg.Sender)
	if !isWhitelisted {
		return nil, sdkioerrors.Wrapf(types.ErrNotWhitelistedFeeTokenSetter, "%s", msg.Sender)
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
