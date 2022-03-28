// Module for minting USDM  Minting USDM
// See Example B of https://docs.frax.finance/minting-and-redeeming
package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// stableDenom string = "usdm"
	govDenom  string = "umtrx"
	collDenom string = "uust"
)

func AsInt(dec sdk.Dec) sdk.Int {
	return sdk.NewIntFromBigInt(dec.BigInt())
}

// govDeposited: Units of GOV burned
// govDeposited = (1 - collRatio) * (collDeposited * 1) / (collRatio * priceGOV)
func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	// priceGov: Price of the governance token in USD
	priceGov, err := k.priceKeeper.GetCurrentPrice(ctx, govDenom)
	if err != nil {
		return nil, err
	}

	// priceColl: Price of the collateral token in USD
	priceColl, err := k.priceKeeper.GetCurrentPrice(ctx, collDenom)
	if err != nil {
		return nil, err
	}

	// The user deposits a mixure of collateral and GOV tokens based on the collateral ratio.
	// TODO: Initialize these two vars based on the collateral ratio of the protocol.
	collRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.NewDec(1).Sub(collRatio)

	neededCollUSD := sdk.NewDecFromInt(msg.Stable.Amount).Mul(collRatio)
	neededCollAmt := sdk.NewIntFromBigInt(neededCollUSD.Quo(priceColl.Price).BigInt())
	neededColl := sdk.NewCoin(collDenom, neededCollAmt)

	neededGovUSD := sdk.NewDecFromInt(msg.Stable.Amount).Mul(govRatio)
	neededGovAmt := sdk.NewIntFromBigInt(neededGovUSD.Quo(priceGov.Price).BigInt())
	neededGov := sdk.NewCoin(govDenom, neededGovAmt)

	coinsNeededToMint := sdk.NewCoins(neededColl, neededGov)

	err = k.CheckEnoughBalances(ctx, coinsNeededToMint, fromAddr)
	if err != nil {
		panic(err)
	}

	// Take assets out of the user account.
	err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, fromAddr, types.ModuleName, coinsNeededToMint)
	if err != nil {
		panic(err)
		// return nil, err
		// Q: Ask about panic vs. return nil and reverting an entire method.
	}

	// Mint the USDM
	stableToMint := msg.Stable
	stablesToMint := sdk.NewCoins(stableToMint)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, stablesToMint)
	if err != nil {
		panic(err)
	}
	// TODO: Burn the GOV that the user gave to the protocol.

	// Send the minted tokens to the user.
	err = k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, fromAddr, stablesToMint)
	if err != nil {
		panic(err)
	}

	return &types.MsgMintResponse{Stable: stableToMint}, nil
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
