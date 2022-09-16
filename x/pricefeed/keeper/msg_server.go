package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
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

// ---------------------------------------------------------------
// PostPrice
// ---------------------------------------------------------------

func (ms msgServer) PostPrice(goCtx context.Context, msg *types.MsgPostPrice,
) (*types.MsgPostPriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	from, err := sdk.AccAddressFromBech32(msg.Oracle)
	if err != nil {
		return nil, err
	}

	pair := common.AssetPair{Token0: msg.Token0, Token1: msg.Token1}

	isWhitelisted := ms.k.IsWhitelistedOracle(ctx, pair.String(), from)
	isWhitelistedForInverse := ms.k.IsWhitelistedOracle(
		ctx, pair.Inverse().String(), from)
	if !(isWhitelisted || isWhitelistedForInverse) {
		return nil, sdkerrors.Wrapf(types.ErrInvalidOracle,
			`oracle is not whitelisted for pair %v
			oracle: %s`, pair.String(), from)
	}

	var postedPrice sdk.Dec
	if isWhitelistedForInverse {
		postedPrice = sdk.OneDec().Quo(msg.Price)
		pair = pair.Inverse()
	} else {
		postedPrice = msg.Price
	}

	if err = ms.k.PostRawPrice(ctx, from, pair.String(), postedPrice, msg.Expiry); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Oracle),
		),
	)

	return &types.MsgPostPriceResponse{}, nil
}
