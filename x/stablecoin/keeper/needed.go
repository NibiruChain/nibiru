package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Computes the amount of MTRX needed to mint USDM given some COLL amount.
// Args:
//   collAmt sdk.Int: Amount of COLL given.
// Returns:
//   neededGovAmt sdk.Int: Amount of MTRX needed.
//   mintableStableAmt sdk.Int: Amount of USDM that can be minted.
func NeededGovAmtGivenColl(
	collAmt sdk.Int, priceGov sdk.Dec, priceColl sdk.Dec,
	collRatio sdk.Dec) (neededGovAmt sdk.Int, mintableStableAmt sdk.Int) {

	collUSD := collAmt.ToDec().Mul(priceColl)
	neededGovUSD := (collUSD.Quo(collRatio)).Sub(collUSD)

	neededGovAmt = neededGovUSD.Quo(priceGov).TruncateInt()
	mintableStableAmt = collUSD.Add(neededGovUSD).TruncateInt()
	return neededGovAmt, mintableStableAmt
}

// Computes the amount of COLL needed to mint USDM given some MTRX amount.
// Args:
//   govAmt sdk.Int: Amount of  MTRX given.
// Returns:
//   neededCollAmt sdk.Int: Amount of COLL needed.
//   mintableStableAmt sdk.Int: Amount of USDM that can be minted.
func NeededCollAmtGivenGov(
	govAmt sdk.Int, priceGov sdk.Dec, priceColl sdk.Dec,
	collRatio sdk.Dec) (neededCollAmt sdk.Int, mintableStableAmt sdk.Int) {

	govUSD := sdk.NewDecFromInt(govAmt).Mul(priceGov)
	govRatio := sdk.NewDec(1).Sub(collRatio)
	neededCollUSD := collRatio.Quo(govRatio).Mul(govUSD)

	neededCollAmt = neededCollUSD.Quo(priceColl).TruncateInt()
	mintableStableAmt = govUSD.Add(neededCollUSD).TruncateInt()
	return neededCollAmt, mintableStableAmt
}
