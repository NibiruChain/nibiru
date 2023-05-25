package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Computes the amount of NIBI needed to mint NUSD given some COLL amount.
// Args:
//
//	collAmt sdk.Int: Amount of COLL given.
//
// Returns:
//
//	neededGovAmt sdk.Int: Amount of NIBI needed.
//	mintableStableAmt sdk.Int: Amount of NUSD that can be minted.
func NeededGovAmtGivenColl(
	collAmt sdkmath.Int, priceGov sdk.Dec, priceColl sdk.Dec,
	collRatio sdk.Dec) (neededGovAmt sdkmath.Int, mintableStableAmt sdkmath.Int) {
	collUSD := sdk.NewDecFromInt(collAmt).Mul(priceColl)
	neededGovUSD := (collUSD.Quo(collRatio)).Sub(collUSD)

	neededGovAmt = neededGovUSD.Quo(priceGov).TruncateInt()
	mintableStableAmt = collUSD.Add(neededGovUSD).TruncateInt()
	return neededGovAmt, mintableStableAmt
}

// Computes the amount of COLL needed to mint NUSD given some NIBI amount.
// Args:
//
//	govAmt sdk.Int: Amount of  NIBI given.
//
// Returns:
//
//	neededCollAmt sdk.Int: Amount of COLL needed.
//	mintableStableAmt sdk.Int: Amount of NUSD that can be minted.
func NeededCollAmtGivenGov(
	govAmt sdkmath.Int, priceGov sdk.Dec, priceColl sdk.Dec,
	collRatio sdk.Dec) (neededCollAmt sdk.Int, mintableStableAmt sdk.Int) {
	govUSD := sdk.NewDecFromInt(govAmt).Mul(priceGov)
	govRatio := sdk.NewDec(1).Sub(collRatio)
	neededCollUSD := collRatio.Quo(govRatio).Mul(govUSD)

	neededCollAmt = neededCollUSD.Quo(priceColl).TruncateInt()
	mintableStableAmt = govUSD.Add(neededCollUSD).TruncateInt()
	return neededCollAmt, mintableStableAmt
}
