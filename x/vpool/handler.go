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

func NewVpoolProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch proposal := content.(type) {
		case *types.CreatePoolProposal:
			return handleProposalCreatePool(ctx, k, proposal)
		case *types.EditPoolConfigProposal:
			return handleProposalEditPoolConfig(ctx, k, proposal)
		case *types.EditSwapInvariantsProposal:
			return handleProposalEditSwapInvariants(ctx, k, proposal)
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

	return k.CreatePool(
		ctx,
		pair,
		proposal.QuoteAssetReserve,
		proposal.BaseAssetReserve,
		proposal.Config,
	)
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

	return k.EditPoolConfig(ctx, pair, proposal.Config)
}

func handleProposalEditSwapInvariants(
	ctx sdk.Context, k keeper.Keeper, proposal *types.EditSwapInvariantsProposal,
) error {
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}
	for _, swapInvariantMap := range proposal.SwapInvariantMaps {
		err := k.EditSwapInvariant(ctx, swapInvariantMap)
		if err != nil {
			return err
		}
	}
	return nil
}
