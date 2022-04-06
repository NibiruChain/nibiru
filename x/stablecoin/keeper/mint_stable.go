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

	params := k.GetParams(ctx)
	feeRatio := params.GetFeeRatioAsDec()
	collRatio := params.GetCollRatioAsDec()
	govRatio := sdk.OneDec().Sub(collRatio)

	neededColl, collFees, err := k.
		calcNeededCollateralAndFees(ctx, msg.Stable, collRatio, feeRatio)
	if err != nil {
		return nil, err
	}

	neededGov, govFees, err := k.
		calcNeededGovAndFees(ctx, msg.Stable, govRatio, feeRatio)
	if err != nil {
		return nil, err
	}

	coinsNeededToMint := sdk.NewCoins(neededColl, neededGov)
	coinsNeededToMintPlusFees := coinsNeededToMint.Add(govFees, collFees)

	err = k.CheckEnoughBalances(ctx, coinsNeededToMintPlusFees, msgCreator)
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

	err = k.sendFeesToPool(ctx, msgCreator, sdk.NewCoins(collFees, govFees))
	if err != nil {
		panic(err)
	}

	err = k.sendMintedTokensToUser(ctx, msgCreator, msg.Stable)
	if err != nil {
		panic(err)
	}

	return &types.MsgMintStableResponse{
		Stable:    msg.Stable,
		UsedCoins: sdk.NewCoins(neededGov, neededColl),
		FeesPayed: sdk.NewCoins(govFees, collFees),
	}, nil
}

// calcNeededGovAndFees returns the needed governance tokens and fees
func (k Keeper) calcNeededGovAndFees(ctx sdk.Context, stable sdk.Coin, govRatio sdk.Dec, feeRatio sdk.Dec) (sdk.Coin, sdk.Coin, error) {
	priceGov, err := k.priceKeeper.GetCurrentPrice(ctx, common.GovPricePool)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	neededGovUSD := stable.Amount.ToDec().Mul(govRatio)
	neededGovAmt := neededGovUSD.Quo(priceGov.Price).TruncateInt()
	neededGov := sdk.NewCoin(common.GovDenom, neededGovAmt)
	govFeeAmt := neededGovAmt.ToDec().Mul(feeRatio).RoundInt()
	govFee := sdk.NewCoin(common.GovDenom, govFeeAmt)

	return neededGov, govFee, nil
}

// calcNeededCollateralAndFees returns the needed collateral and the collateral fees
func (k Keeper) calcNeededCollateralAndFees(
	ctx sdk.Context,
	stable sdk.Coin,
	collRatio sdk.Dec,
	feeRatio sdk.Dec,
) (sdk.Coin, sdk.Coin, error) {
	priceColl, err := k.priceKeeper.GetCurrentPrice(ctx, common.CollPricePool)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	neededCollUSD := stable.Amount.ToDec().Mul(collRatio)
	neededCollAmt := neededCollUSD.Quo(priceColl.Price).TruncateInt()
	neededColl := sdk.NewCoin(common.CollDenom, neededCollAmt)
	collFeeAmt := neededCollAmt.ToDec().Mul(feeRatio).RoundInt()
	collFee := sdk.NewCoin(common.CollDenom, collFeeAmt)

	return neededColl, collFee, nil
}

// sendInputCoinsToModule sends coins from the 'msg.Creator' to the module account
func (k Keeper) sendInputCoinsToModule(
	ctx sdk.Context, msgCreator sdk.AccAddress, coins sdk.Coins,
) (err error) {
	err = k.bankKeeper.SendCoinsFromAccountToModule(
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
	err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, to, sdk.NewCoins(stable))
	if err != nil {
		return err
	}

	events.EmitTransfer(ctx, stable, types.ModuleName, to.String())

	return nil
}

// burnGovTokens burns governance coins
func (k Keeper) burnGovTokens(ctx sdk.Context, govTokens sdk.Coin) error {
	err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(govTokens))
	if err != nil {
		return err
	}

	events.EmitBurnMtrx(ctx, govTokens)

	return nil
}

func (k Keeper) mintStable(ctx sdk.Context, stable sdk.Coin) error {
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(stable))
	if err != nil {
		return err
	}

	events.EmitMintStable(ctx, stable)

	return nil
}

// sendFeesToPool sends the coins to the Stable Ecosystem Fund.
func (k Keeper) sendFeesToPool(ctx sdk.Context, account sdk.AccAddress, coins sdk.Coins) error {
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, account, types.StableEFModuleAccount, coins)
	if err != nil {
		return err
	}

	return nil
}
