package stablecoin

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

/*
NewHandler returns an sdk.Handler for "x/stablecoin" messages.
A handler defines the core state transition functions of an application.
First, the handler performs stateful checks to make sure each 'msg' is valid.
At this stage, the 'msg.ValidateBasic()' method has already been called, meaning
stateless checks on the message (like making sure parameters are correctly
formatted) have already been performed.
*/
func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		goCtx := sdk.WrapSDKContext(ctx)

		switch msg := msg.(type) {
		case *types.MsgMintStable:
			res, err := msgServer.MintStable(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgBurnStable:
			res, err := msgServer.BurnStable(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRecollateralize:
			res, err := msgServer.Recollateralize(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgBuyback:
			res, err := msgServer.Buyback(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			errMsg := fmt.Sprintf(
				"unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, sdkerrors.Wrap(errors.ErrUnknownRequest, errMsg)
		}
	}
}
