// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func (k *Keeper) UpdateParams(
	goCtx context.Context, req *evm.MsgUpdateParams,
) (resp *evm.MsgUpdateParamsResponse, err error) {
	if err := req.ValidateBasic(); err != nil {
		return resp, err
	}

	sender := sdk.MustAccAddressFromBech32(req.Authority)
	ctx := sdk.UnwrapSDKContext(goCtx)

	sudoPermsErr := k.sudoKeeper.CheckPermissions(sender, ctx)
	havePerms := (sudoPermsErr == nil) || (k.authority.String() == req.Authority)
	if !havePerms {
		return resp, fmt.Errorf(
			"invalid signing authority, expected governance account %s or one of the sudoers defined by the x/sudo module. Sender was %s",
			k.authority, req.Authority,
		)
	}

	err = k.SetParams(ctx, req.Params)
	if err != nil {
		return nil, sdkioerrors.Wrapf(err, "failed to set params")
	}
	return &evm.MsgUpdateParamsResponse{}, nil
}
