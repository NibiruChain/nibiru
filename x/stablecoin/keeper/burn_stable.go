// Module for Burning USDM
// See Example B of https://docs.frax.finance/minting-and-redeeming
package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) BurnStable(goCtx context.Context, msg *types.MsgBurnStable) (*types.MsgBurnStableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	toAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}

	// Check if the user has the fund necessary
	err = k.CheckEnoughBalances(ctx, sdk.NewCoins(msg.Stable), toAddr)
	if err != nil {
		panic(err)
	}

	// priceGov: Price of the governance token in USD
	priceGov, err := k.priceKeeper.GetCurrentPrice(ctx, govDenom)
	if err != nil {
		panic(err)
	}

	// priceColl: Price of the collateral token in USD
	priceColl, err := k.priceKeeper.GetCurrentPrice(ctx, collDenom)
	if err != nil {
		panic(err)
	}

	// The user deposits a mixure of collateral and GOV tokens based on the collateral ratio.
	// TODO: Initialize these two vars based on the collateral ratio of the protocol.
	collRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.NewDec(1).Sub(collRatio)

	redeemColl := AsInt(sdk.NewDecFromInt(msg.Stable.Amount).Mul(collRatio).Quo(priceColl.Price))
	redeemGov := AsInt(sdk.NewDecFromInt(msg.Stable.Amount).Mul(govRatio).Quo(priceGov.Price))

	// Send USDM from account to module
	stableToBurn := msg.Stable
	stablesToBurn := sdk.NewCoins(stableToBurn)
	err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, toAddr, types.ModuleName, stablesToBurn)
	if err != nil {
		panic(err)
	}

	// Mint the GOV that the user gave to the protocol.
	collToSend := sdk.NewCoin(collDenom, redeemColl)
	govToSend := sdk.NewCoin(govDenom, redeemGov)
	coinsNeededToSend := sdk.NewCoins(collToSend, govToSend)

	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(govToSend))
	if err != nil {
		panic(err)
	}

	// Send tokens to the account
	err = k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, toAddr, coinsNeededToSend)
	if err != nil {
		panic(err)
	}

	// Burn the USDM
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, stablesToBurn)
	if err != nil {
		panic(err)
	}

	return &types.MsgBurnStableResponse{Collateral: collToSend, Gov: govToSend}, nil
}
