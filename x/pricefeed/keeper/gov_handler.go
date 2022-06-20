package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func HandleAddOracleProposal(ctx sdk.Context, k Keeper, proposal types.AddOracleProposal) error {
	if err := proposal.Validate(); err != nil {
		return err
	}
	oracle := sdk.MustAccAddressFromBech32(proposal.Oracle)

	k.WhitelistOraclesForPairs(
		ctx,
		/*oracles=*/ []sdk.AccAddress{oracle},
		/*assetPairs=*/ common.NewAssetPairs(proposal.Pairs),
	)

	return nil
}
