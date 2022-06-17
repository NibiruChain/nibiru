package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// TODO test: https://github.com/NibiruChain/nibiru/issues/591
func HandleAddOracleProposal(ctx sdk.Context, k Keeper, proposal types.AddOracleProposal) error {
	if err := proposal.Validate(); err != nil {
		return err
	}
	oracle := sdk.MustAccAddressFromBech32(proposal.Oracle)

	k.WhitelistOraclesForPairs(
		ctx,
		/*oracles=*/ []sdk.AccAddress{oracle},
		/*assetPairs=*/ common.MustNewAssetPairsFromStr(proposal.Pairs),
	)

	return nil
}
