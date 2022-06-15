package keeper

import (
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO test: https://github.com/NibiruChain/nibiru/issues/591
func HandleAddOracleProposal(ctx sdk.Context, k Keeper, proposal types.AddOracleProposal) error {
	if err := proposal.Validate(); err != nil {
		return err
	}
	oracle := sdk.MustAccAddressFromBech32(proposal.Oracle)

	if err := k.WhitelistOraclesForPairs(
		ctx,
		/* oracles */ []sdk.AccAddress{oracle},
		/* pairs */ proposal.Pairs,
	); err != nil {
		return err
	}

	return nil
}
