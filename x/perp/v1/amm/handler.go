package amm

import (
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/perp/v1/amm/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

// NewHandler ...
func NewHandler(k keeper.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		// ctx = ctx.WithEventManager(sdk.NewEventManager())

		errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
	}
}

func NewMarketProposalHandler(k keeper.Keeper) govtypes.Handler {
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

	err := proposal.Pair.Validate()
	if err != nil {
		return err
	}

	return k.CreatePool(
		ctx,
		proposal.Pair,
		proposal.QuoteReserve,
		proposal.BaseReserve,
		proposal.Config,
		sdk.OneDec(), // TODO: peg multiplier is 1 by default
	)
}

func handleProposalEditPoolConfig(
	ctx sdk.Context, k keeper.Keeper, proposal *types.EditPoolConfigProposal,
) error {
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

	err := proposal.Pair.Validate()
	if err != nil {
		return err
	}

	return k.EditPoolConfig(ctx, proposal.Pair, proposal.Config)
}

func handleProposalEditSwapInvariants(
	ctx sdk.Context, k keeper.Keeper, proposal *types.EditSwapInvariantsProposal,
) error {
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}
	for _, swapInvariantMap := range proposal.SwapInvariantMaps {
		if err := swapInvariantMap.Validate(); err != nil {
			return err
		}
		_, err := k.EditSwapInvariant(ctx, swapInvariantMap.Pair, swapInvariantMap.Multiplier)
		if err != nil {
			return err
		}
	}
	return nil
}
