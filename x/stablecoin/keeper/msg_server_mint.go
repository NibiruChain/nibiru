// Module for minting USDM  Minting USDM
// See Example B of https://docs.frax.finance/minting-and-redeeming
package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	stableDenom string = "usdm"
	govDenom    string = "umtrx"
	collDenom   string = "uust"
)

// govDeposited: Units of GOV burned
// govDeposited = (1 - collRatio) * (collDeposited * 1) / (collRatio * priceGOV)
func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	// priceGov: Price of the governance token in USD
	// TODO: Read the governance token price (MTRX per USD) from an oracle.
	priceGov, _ := sdk.NewDecFromStr("20")

	// priceColl: Price of the collateral token in USD
	// TODO: Read the collateral token price (per USD) from an oracle.
	priceColl, _ := sdk.NewDecFromStr("1")

	// The user deposits a mixure of collateral and GOV tokens based on the collateral ratio.
	// TODO: Initialize these two vars based on the collateral ratio of the protocol.
	collRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.NewDec(1).Sub(collRatio)

	neededCollUSD := sdk.NewDecFromInt(msg.Stable.Amount).Mul(collRatio)
	neededCollAmt := sdk.NewIntFromBigInt(neededCollUSD.Quo(priceColl).BigInt())
	neededColl := sdk.NewCoin(collDenom, neededCollAmt)

	neededGovUSD := sdk.NewDecFromInt(msg.Stable.Amount).Mul(govRatio)
	neededGovAmt := sdk.NewIntFromBigInt(neededGovUSD.Quo(priceGov).BigInt())
	neededGov := sdk.NewCoin(govDenom, neededGovAmt)

	coinsNeededToMint := sdk.NewCoins(neededColl, neededGov)

	for _, coin := range coinsNeededToMint {
		hasEnoughBalance, err := k.CheckEnoughBalance(ctx, coin, fromAddr)
		if err != nil {
			return nil, err
		}

		if !hasEnoughBalance {
			return nil, types.NotEnoughBalance.Wrap(coin.String())
		}
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

// Computes the maximum amount of USDM mintable for a MsgMint.Creator's
// input of collateral and governance tokens
// TODO
func MaxMintableStable(
	msg *types.MsgMint,
	collRatio sdk.Dec,
	priceGov sdk.Dec,
	priceColl sdk.Dec) sdk.Coin {

	// msgCollUSD := sdk.NewDecFromInt(msg.Coll.Amount).Mul(priceColl)
	// msgGovUSD := sdk.NewDecFromInt(msg.Gov.Amount).Mul(priceGov)

	// govUSDNeeded := msgCollUSD.Quo(collRatio).Sub(msgCollUSD)
	// ^ should be LE sum(msgCollUSD, msgGovUSD)
	maxStable := sdk.NewCoin(stableDenom, sdk.ZeroInt())
	return maxStable
}
