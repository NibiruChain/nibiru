package pricefeed

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// NewHandler ...
func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *types.MsgPostPrice:
			res, err := msgServer.PostPrice(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

/* NewPricefeedPropsalHandler defines a function that handles a proposal after it has
passed the governance process */
func NewPricefeedProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch contentType := content.(type) {
		case *types.AddOracleProposal:
			return handleAddOracleProposal(ctx, k, contentType)
		default:
			return sdkerrors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"unrecognized %s proposal content type: %T", types.ModuleName, contentType)
		}
	}
}

func handleAddOracleProposal(
	ctx sdk.Context, k keeper.Keeper, proposal *types.AddOracleProposal) error {
	if err := proposal.Validate(); err != nil {
		return err
	}
	oracles := common.StringsToAddrs(proposal.Oracles...)

	k.WhitelistOraclesForPairs(
		ctx,
		/*oracles=*/ oracles,
		/*assetPairs=*/ common.NewAssetPairs(proposal.Pairs...),
	)

	// TODO Emit typed event for when oracles get added
	return nil
}
