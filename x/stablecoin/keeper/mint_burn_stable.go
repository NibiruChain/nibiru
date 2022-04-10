/*
Package keeper that mints Matrix stablecoins, maintains their price stability,
and ensures that the protocol remains collateralized enough for stablecoins to
be redeemed.
*/
package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/stablecoin/events"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MintStable mints stable coins given collateral (COLL) and governance (GOV)
func (k Keeper) MintStable(
	goCtx context.Context, msg *types.MsgMintStable,
) (*types.MsgMintStableResponse, error) {

	ctx := sdk.UnwrapSDKContext(goCtx)

	msgCreator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	// priceGov: Price of the governance token in USD
	priceGov, err := k.PriceKeeper.GetCurrentPrice(ctx, common.GovCollPool)
	if err != nil {
		return nil, err
	}

	// priceColl: Price of the collateral token in USD
	priceColl, err := k.PriceKeeper.GetCurrentPrice(ctx, common.CollStablePool)
	if err != nil {
		return nil, err
	}

	// The user deposits a mixture of collateral and GOV tokens based on the collateral ratio.
	// TODO: Initialize these two vars based on the collateral ratio of the protocol.
	collRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.OneDec().Sub(collRatio)

	neededCollUSD := collRatio.MulInt(msg.Stable.Amount)
	neededCollAmt := neededCollUSD.Quo(priceColl.Price).TruncateInt()
	neededColl := sdk.NewCoin(common.CollDenom, neededCollAmt)

	neededGovUSD := govRatio.MulInt(msg.Stable.Amount)
	neededGovAmt := neededGovUSD.Quo(priceGov.Price).TruncateInt()
	neededGov := sdk.NewCoin(common.GovDenom, neededGovAmt)

	coinsNeededToMint := sdk.NewCoins(neededColl, neededGov)

	err = k.CheckEnoughBalances(ctx, coinsNeededToMint, msgCreator)
	if err != nil {
		return nil, err
	}

	err = k.sendInputCoinsToModule(ctx, msgCreator, coinsNeededToMint)
	if err != nil {
		panic(err)
	}

	err = k.burnGovTokens(ctx, neededGov)
	if err != nil {
		panic(err)
	}

	err = k.mintStable(ctx, msg.Stable)
	if err != nil {
		panic(err)
	}

	err = k.sendMintedTokensToUser(ctx, msgCreator, msg.Stable)
	if err != nil {
		panic(err)
	}

	return &types.MsgMintStableResponse{Stable: msg.Stable}, nil
}

// sendInputCoinsToModule sends coins from the 'msg.Creator' to the module account
func (k Keeper) sendInputCoinsToModule(
	ctx sdk.Context, msgCreator sdk.AccAddress, coins sdk.Coins,
) (err error) {
	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, msgCreator, types.ModuleName, coins)
	if err != nil {
		return err
	}

	for _, coin := range coins {
		events.EmitTransfer(ctx, coin, msgCreator.String(), types.ModuleName)
	}

	return nil
}

// sendMintedTokensToUser sends coins minted in Module Account to address to
func (k Keeper) sendMintedTokensToUser(
	ctx sdk.Context, to sdk.AccAddress, stable sdk.Coin,
) error {
	err := k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, to, sdk.NewCoins(stable))
	if err != nil {
		return err
	}

	events.EmitTransfer(ctx, stable, types.ModuleName, to.String())

	return nil
}

// burnGovTokens burns governance coins
func (k Keeper) burnGovTokens(ctx sdk.Context, neededGov sdk.Coin) error {
	err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(neededGov))
	if err != nil {
		return err
	}

	events.EmitBurnMtrx(ctx, neededGov)

	return nil
}

func (k Keeper) mintStable(ctx sdk.Context, stable sdk.Coin) error {
	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(stable))
	if err != nil {
		return err
	}

	events.EmitMintStable(ctx, stable)

	return nil
}

func (k Keeper) BurnStable(
	goCtx context.Context, msg *types.MsgBurnStable,
) (*types.MsgBurnStableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	msgCreator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	if msg.Stable.Amount == sdk.ZeroInt() {
		return nil, sdkerrors.Wrap(types.NoCoinFound, msg.Stable.Denom)
	}

	// priceGov: Price of the governance token in USD
	priceGov, err := k.PriceKeeper.GetCurrentPrice(ctx, common.GovCollPool)
	if err != nil {
		return nil, err
	}

	// priceColl: Price of the collateral token in USD
	priceColl, err := k.PriceKeeper.GetCurrentPrice(ctx, common.CollStablePool)
	if err != nil {
		return nil, err
	}

	// The user receives a mixure of collateral (COLL) and governance (GOV) tokens
	// based on the collateral ratio.
	// TODO: Initialize 'collRatio' based on the collateral ratio of the protocol.
	collRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.NewDec(1).Sub(collRatio)

	redeemColl := collRatio.MulInt(msg.Stable.Amount).Quo(
		priceColl.Price).TruncateInt()
	redeemGov := govRatio.MulInt(msg.Stable.Amount).Quo(
		priceGov.Price).TruncateInt()

	// Send USDM from account to module
	stablesToBurn := sdk.NewCoins(msg.Stable)
	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, msgCreator, types.ModuleName, stablesToBurn)
	if err != nil {
		return nil, err
	}
	events.EmitTransfer(ctx, msg.Stable, msgCreator.String(), types.ModuleName)

	// Mint GOV that will later be sent to the user.
	collToSend := sdk.NewCoin(common.CollDenom, redeemColl)
	govToSend := sdk.NewCoin(common.GovDenom, redeemGov)
	coinsNeededToSend := sdk.NewCoins(collToSend, govToSend)

	err = k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(govToSend))
	if err != nil {
		panic(err)
	}
	events.EmitMintMtrx(ctx, govToSend)

	// Send tokens (GOV and COLL) to the account
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, msgCreator, coinsNeededToSend)
	if err != nil {
		return nil, err
	}
	for _, coin := range coinsNeededToSend {
		events.EmitTransfer(ctx, coin, types.ModuleName, msgCreator.String())
	}

	// Burn the USDM
	err = k.BankKeeper.BurnCoins(ctx, types.ModuleName, stablesToBurn)
	if err != nil {
		panic(err)
	}
	events.EmitBurnStable(ctx, msg.Stable)

	return &types.MsgBurnStableResponse{Collateral: collToSend, Gov: govToSend}, nil
}
