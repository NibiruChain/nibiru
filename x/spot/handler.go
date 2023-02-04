package spot

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/spot/keeper"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

// NewHandler ...
func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		goCtx := sdk.WrapSDKContext(ctx)

		switch msg := msg.(type) {
		case *types.MsgCreatePool:
			res, err := msgServer.CreatePool(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgJoinPool:
			res, err := msgServer.JoinPool(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgExitPool:
			res, err := msgServer.ExitPool(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgSwapAssets:
			res, err := msgServer.SwapAssets(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}
