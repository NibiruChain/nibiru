package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) PostPrice(goCtx context.Context, msg *types.MsgPostPrice) (*types.MsgPostPriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return nil, err
	}

	_, err = k.GetOracle(ctx, msg.MarketID, from)
	if err != nil {
		return nil, err
	}

	_, err = k.SetPrice(ctx, from, msg.MarketID, msg.Price, msg.Expiry)

	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From),
		),
	)

	return &types.MsgPostPriceResponse{}, nil
}
