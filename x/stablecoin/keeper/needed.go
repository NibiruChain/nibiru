// Module for minting USDM  Minting USDM
// See Example B of https://docs.frax.finance/minting-and-redeeming
package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Truncates a decimal (sdk.Dec) to convert it into an integer (sdk.Int).
func AsInt(dec sdk.Dec) sdk.Int {
	sdkInt18 := sdk.NewIntFromBigInt(dec.BigInt())
	var ten18 sdk.Int = sdk.NewIntFromBigInt(sdk.MustNewDecFromStr("1").BigInt())
	sdkInt := sdkInt18.Quo(sdk.OneInt().Mul(ten18))
	return sdkInt
}

// Computes the amount of MTRX needed to mint USDM given some COLL amount.
// Args:
//   collAmt sdk.Int: Amount of COLL given.
// Returns:
//   neededGovAmt sdk.Int: Amount of MTRX needed.
//   mintableStableAmt sdk.Int: Amount of USDM that can be minted.
func NeededGovAmtGivenColl(
	collAmt sdk.Int, priceGov sdk.Dec, priceColl sdk.Dec,
	collRatio sdk.Dec) (neededGovAmt sdk.Int, mintableStableAmt sdk.Int) {

	collUSD := sdk.NewDecFromInt(collAmt).Mul(priceColl)
	neededGovUSD := (collUSD.Quo(collRatio)).Sub(collUSD)

	neededGovAmt = AsInt(neededGovUSD.Quo(priceGov))
	mintableStableAmt = AsInt(collUSD.Add(neededGovUSD))
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

	neededCollAmt = AsInt(neededCollUSD.Quo(priceColl))
	mintableStableAmt = AsInt(govUSD.Add(neededCollUSD))
	return neededCollAmt, mintableStableAmt
}
