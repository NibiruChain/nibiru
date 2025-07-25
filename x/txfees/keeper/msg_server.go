package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// UpdateFeeToken: gRPC tx msg for updating fee token
func (k Keeper) UpdateFeeToken(
	goCtx context.Context, msg *types.MsgUpdateFeeToken,
) (resp *types.MsgUpdateFeeTokenResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Stateless field validation was already performed in msg.ValidateBasic()
	// before the current scope is reached.

	if k.authority != msg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	err = k.SetFeeToken(ctx, types.FeeToken{
		Address: msg.ContractAddress,
	})
	if err != nil {
		return nil, err
	}

	resp = &types.MsgUpdateFeeTokenResponse{}
	return resp, err
}
