package keeper

import (
	"context"
	"fmt"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// UpdateFeeToken: gRPC tx msg for updating fee token
func (k Keeper) UpdateFeeToken(
	goCtx context.Context,
	msg *types.MsgUpdateFeeToken,
) (*types.MsgUpdateFeeTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != k.authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	switch msg.Action {
	case types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD:
		if err := k.AddFeeToken(ctx, *msg.FeeToken); err != nil {
			return nil, err
		}
	case types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE:
		if err := k.RemoveFeeToken(ctx, msg.FeeToken.Address); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid action %s; must be one of %s or %s",
			msg.Action,
			types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD,
			types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE)
	}

	return &types.MsgUpdateFeeTokenResponse{}, nil
}
