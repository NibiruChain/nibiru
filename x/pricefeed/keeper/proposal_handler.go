package keeper

import (
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func NewPriceFeedProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *pftypes.WhitelistPriceOracleProposal:
			return HandleWhitelistPriceOracleProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized pricefeed proposal content type: %T", c)
		}
	}
}

// HandleWhitelistPriceOracleProposal is a handler for executing a passed whitelist price oracle proposal
func HandleWhitelistPriceOracleProposal(ctx sdk.Context, k Keeper, p *pftypes.WhitelistPriceOracleProposal) error {
	return nil
}
