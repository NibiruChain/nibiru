package vpool

import (
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// NewHandler ...
func NewHandler(k keeper.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		// ctx = ctx.WithEventManager(sdk.NewEventManager())

		errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
	}
}

func NewGovProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch proposal := content.(type) {
		case *types.CreatePoolProposal:
			return handleProposalCreatePool(ctx, k, proposal)
		case *types.EditPoolConfigProposal:
			return handleProposalEditPoolConfig(ctx, k, proposal)
		default:
			return sdkerrors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"unrecognized %s proposal content type: %T", types.ModuleName, proposal)
		}
	}
}

func handleProposalCreatePool(
	ctx sdk.Context, k keeper.Keeper, proposal *types.CreatePoolProposal,
) error {
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

	pair, err := common.NewAssetPair(proposal.Pair)
	if err != nil {
		return err
	}

	k.CreatePool(
		ctx,
		pair,
		proposal.QuoteAssetReserve,
		proposal.BaseAssetReserve,
		proposal.Config,
	)
	return nil
}

func handleProposalEditPoolConfig(
	ctx sdk.Context, k keeper.Keeper, proposal *types.EditPoolConfigProposal,
) error {
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

	pair, err := common.NewAssetPair(proposal.Pair)
	if err != nil {
		return err
	}

	// Grab current pool from state
	vpool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return err
	}

	newVpool := types.Vpool{
		Pair:              vpool.Pair,
		BaseAssetReserve:  vpool.BaseAssetReserve,
		QuoteAssetReserve: vpool.QuoteAssetReserve,
		Config:            proposal.Config, // main change is here
	}
	if err := newVpool.Validate(); err != nil {
		return err
	}

	err = k.UpdatePool(
		ctx,
		newVpool,
		/*skipFluctuationLimitCheck*/ true)
	if err != nil {
		return err
	}

	return nil
}
