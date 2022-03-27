// Module for minting USDM  Minting USDM
// See Example B of https://docs.frax.finance/minting-and-redeeming
package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func AsInt(dec sdk.Dec) sdk.Int {
	return sdk.NewIntFromBigInt(dec.BigInt())
}

// Computes the amount of MTRX needed to mint USDM given some COLL amount.
// Args:
//   collAmt sdk.Int: Amount of COLL given.
// Returns:
//   neededGovAmt sdk.Int: Amount of MTRX needed.
//   mintableStableAmt sdk.Int: Amount of USDM that can be minted.
func NeededGovAmtGivenColl(
	collAmt sdk.Int, priceGov sdk.Dec, priceColl sdk.Dec,
	collRatio sdk.Dec) (sdk.Int, sdk.Int) {

	collUSD := sdk.NewDecFromInt(collAmt).Mul(priceColl)
	neededGovUSD := (collUSD.Quo(collRatio)).Sub(collUSD)

	neededGovAmt := AsInt(neededGovUSD.Quo(priceGov))
	mintableStableAmt := AsInt(collUSD.Add(neededGovUSD))
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
	collRatio sdk.Dec) (sdk.Int, sdk.Int) {

	govUSD := sdk.NewDecFromInt(govAmt).Mul(priceGov)
	govRatio := sdk.NewDec(1).Sub(collRatio)
	neededCollUSD := collRatio.Quo(govRatio).Mul(govUSD)

	neededCollAmt := AsInt(neededCollUSD.Quo(priceColl))
	mintableStableAmt := AsInt(govUSD.Add(neededCollUSD))
	return neededCollAmt, mintableStableAmt
}
