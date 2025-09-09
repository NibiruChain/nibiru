package keeper

import (
	"context"
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// UpdateFeeToken: gRPC tx msg for updating fee token
func (k Keeper) UpdateFeeToken(
	goCtx context.Context,
	msg *types.MsgUpdateFeeToken,
) (*types.MsgUpdateFeeTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sudoAddr, err := k.sudoKeeper.GetRootAddr(ctx)
	if err != nil {
		return nil, sdkioerrors.Wrap(sdkerrors.ErrUnauthorized, "failed to get root address")
	}
	if msg.Sender != sudoAddr.String() {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", sudoAddr, msg.Sender)
	}

	switch msg.Action {
	case types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD:
		if msg.FeeToken == nil {
			return nil, sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "fee_token must be provided")
		}
		if err := k.AddFeeToken(ctx, *msg.FeeToken); err != nil {
			return nil, err
		}
	case types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE:
		if msg.FeeToken == nil || msg.FeeToken.Erc20Address == "" {
			return nil, sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "fee_token.Erc20Address must be provided")
		}
		if err := k.RemoveFeeToken(ctx, msg.FeeToken.Erc20Address); err != nil {
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

func (k Keeper) UpdateParams(
	goCtx context.Context,
	msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sudoAddr, err := k.sudoKeeper.GetRootAddr(ctx)
	if err != nil {
		return nil, sdkioerrors.Wrap(sdkerrors.ErrUnauthorized, "failed to get root address")
	}
	if msg.Sender != sudoAddr.String() {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", sudoAddr, msg.Sender)
	}

	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
