package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/perp/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// getLatestCumulativeFundingRate returns the last cumulative funding rate recorded for the
// specific pair.
func (k Keeper) getLatestCumulativeFundingRate(
	ctx sdk.Context, pair common.AssetPair,
) (sdk.Dec, error) {
	pairMetadata, err := k.PairsMetadata.Get(ctx, pair)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"pair",
			pair.String(),
		)
		return sdk.Dec{}, err
	}
	// this should never fail
	return pairMetadata.CumulativeFundingRates[len(pairMetadata.CumulativeFundingRates)-1], nil
}

// IncrementBadDebt increases the bad debt for the provided denom.
// And returns the newest bad-debt amount.
// If no prepaid bad debt for the given denom was recorded before
// then it is set using the provided amount and the provided amount is returned.
func (k Keeper) IncrementBadDebt(ctx sdk.Context, denom string, amount sdk.Int) sdk.Int {
	current := k.PrepaidBadDebt.GetOr(ctx, keys.String(denom), types.PrepaidBadDebt{
		Denom:  denom,
		Amount: sdk.ZeroInt(),
	})

	newBadDebt := current.Amount.Add(amount)
	k.PrepaidBadDebt.Insert(ctx, keys.String(denom), types.PrepaidBadDebt{
		Denom:  denom,
		Amount: newBadDebt,
	})

	return newBadDebt
}

// DecrementBadDebt decrements the amount of bad debt prepaid by denom.
// // The lowest it can be decremented to is zero. Trying to decrement a prepaid bad
// // debt balance to below zero will clip it at zero.
func (k Keeper) DecrementBadDebt(ctx sdk.Context, denom string, amount sdk.Int) sdk.Int {
	current := k.PrepaidBadDebt.GetOr(ctx, keys.String(denom), types.PrepaidBadDebt{
		Denom:  denom,
		Amount: sdk.ZeroInt(),
	})

	newBadDebt := sdk.MaxInt(current.Amount.Sub(amount), sdk.ZeroInt())

	k.PrepaidBadDebt.Insert(ctx, keys.String(denom), types.PrepaidBadDebt{
		Denom:  denom,
		Amount: newBadDebt,
	})
	return newBadDebt
}
