package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func (k msgServer) PostPrice(goCtx context.Context, msg *types.MsgPostPrice,
) (*types.MsgPostPriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return nil, err
	}

	pairID := common.PoolNameFromDenoms([]string{msg.Token0, msg.Token1})

	if !k.IsWhitelistedOracle(ctx, pairID, from) {
		return nil, fmt.Errorf("oracle %s\nis not whitelisted on pair %v", from, pairID)
	}

	_, err = k.SetPrice(ctx, from, msg.Token0, msg.Token1, msg.Price, msg.Expiry)

	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From),
		),
	)

	return &types.MsgPostPriceResponse{}, nil
}
