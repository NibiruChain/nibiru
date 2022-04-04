// Module for Burning USDM
// See Example B of https://docs.frax.finance/minting-and-redeeming
package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/stablecoin/events"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
)

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
	priceGov, err := k.priceKeeper.GetCurrentPrice(ctx, common.GovPricePool)
	if err != nil {
		return nil, err
	}

	// priceColl: Price of the collateral token in USD
	priceColl, err := k.priceKeeper.GetCurrentPrice(ctx, common.CollPricePool)
	if err != nil {
		return nil, err
	}

	// The user receives a mixure of collateral (COLL) and governance (GOV) tokens
	// based on the collateral ratio.
	// TODO: Initialize 'collRatio' based on the collateral ratio of the protocol.
	collRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.NewDec(1).Sub(collRatio)

	redeemColl := msg.Stable.Amount.ToDec().Mul(collRatio).Quo(
		priceColl.Price).TruncateInt()
	redeemGov := msg.Stable.Amount.ToDec().Mul(govRatio).Quo(
		priceGov.Price).TruncateInt()

	// Send USDM from account to module
	stablesToBurn := sdk.NewCoins(msg.Stable)
	err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, msgCreator, types.ModuleName, stablesToBurn)
	if err != nil {
		return nil, err
	}
	events.EmitTransfer(ctx, msg.Stable, msgCreator.String(), types.ModuleName)

	// Mint GOV that will later be sent to the user.
	collToSend := sdk.NewCoin(common.CollDenom, redeemColl)
	govToSend := sdk.NewCoin(common.GovDenom, redeemGov)
	coinsNeededToSend := sdk.NewCoins(collToSend, govToSend)

	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(govToSend))
	if err != nil {
		panic(err)
	}
	events.EmitMintMtrx(ctx, govToSend)

	// Send tokens (GOV and COLL) to the account
	err = k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, msgCreator, coinsNeededToSend)
	if err != nil {
		panic(err)
	}
	for _, coin := range coinsNeededToSend {
		events.EmitTransfer(ctx, coin, types.ModuleName, msgCreator.String())
	}

	// Burn the USDM
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, stablesToBurn)
	if err != nil {
		panic(err)
	}
	events.EmitBurnStable(ctx, msg.Stable)

	return &types.MsgBurnStableResponse{Collateral: collToSend, Gov: govToSend}, nil
}
