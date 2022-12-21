/*
Package keeper that mints Nibiru stablecoins, maintains their price stability,
and ensures that the protocol remains collateralized enough for stablecoins to
be redeemed.
*/
package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
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
	if !params.IsCollateralRatioValid {
		return nil, types.NoValidCollateralRatio
	}

	feeRatio := params.GetFeeRatioAsDec()
	collRatio := params.GetCollRatioAsDec()
	efFeeRatio := params.GetEfFeeRatioAsDec()
	govRatio := sdk.OneDec().Sub(collRatio)

	// The user deposits a mixture of collateral and GOV tokens based on the collateral ratio.
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

	err = k.sendCoinsToModuleAccount(ctx, msgCreator, coinsNeededToMint)
	if err != nil {
		panic(err)
	}

	err = k.burnGovTokens(ctx, neededGov)
	if err != nil {
		panic(err)
	}

	err = k.splitAndSendFeesToEfAndTreasury(ctx, msgCreator, efFeeRatio, sdk.NewCoins(collFees, govFees))
	if err != nil {
		panic(err)
	}

	err = k.mintStable(ctx, msg.Stable)
	if err != nil {
		panic(err)
	}

	err = k.sendCoinsFromModuleAccountToUser(ctx, msgCreator, sdk.NewCoins(msg.Stable))
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
func (k Keeper) calcNeededGovAndFees(
	ctx sdk.Context, stable sdk.Coin, govRatio sdk.Dec, feeRatio sdk.Dec,
) (sdk.Coin, sdk.Coin, error) {
	priceGov, err := k.OracleKeeper.GetCurrentPrice(
		ctx, common.DenomNIBI, common.DenomNUSD)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	neededGovUSD := stable.Amount.ToDec().Mul(govRatio)
	neededGovAmt := neededGovUSD.Quo(priceGov.Price).TruncateInt()
	neededGov := sdk.NewCoin(common.DenomNIBI, neededGovAmt)
	govFeeAmt := neededGovAmt.ToDec().Mul(feeRatio).RoundInt()
	govFee := sdk.NewCoin(common.DenomNIBI, govFeeAmt)

	return neededGov, govFee, nil
}

// calcNeededCollateralAndFees returns the needed collateral and the collateral fees
func (k Keeper) calcNeededCollateralAndFees(
	ctx sdk.Context,
	stable sdk.Coin,
	collRatio sdk.Dec,
	feeRatio sdk.Dec,
) (sdk.Coin, sdk.Coin, error) {
	priceColl, err := k.OracleKeeper.GetCurrentPrice(
		ctx, common.DenomUSDC, common.DenomNUSD)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	neededCollUSD := stable.Amount.ToDec().Mul(collRatio)
	neededCollAmt := neededCollUSD.Quo(priceColl.Price).TruncateInt()
	neededColl := sdk.NewCoin(common.DenomUSDC, neededCollAmt)
	collFeeAmt := neededCollAmt.ToDec().Mul(feeRatio).RoundInt()
	collFee := sdk.NewCoin(common.DenomUSDC, collFeeAmt)

	return neededColl, collFee, nil
}

// sendCoinsToModuleAccount sends coins from account to the module account
func (k Keeper) sendCoinsToModuleAccount(
	ctx sdk.Context, from sdk.AccAddress, coins sdk.Coins,
) (err error) {
	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, from, types.ModuleName, coins)
	if err != nil {
		return err
	}

	for _, coin := range coins {
		moduleAddress := k.AccountKeeper.GetModuleAddress(types.ModuleName)
		err = ctx.EventManager().EmitTypedEvent(&types.EventTransfer{
			Coin: coin, From: from.String(), To: moduleAddress.String()})
		if err != nil {
			return err
		}
	}
	return nil
}

// sendCoinsFromModuleAccountToUser sends coins minted in Module Account to address to
func (k Keeper) sendCoinsFromModuleAccountToUser(
	ctx sdk.Context, to sdk.AccAddress, coins sdk.Coins,
) error {
	err := k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, to, coins)
	if err != nil {
		return err
	}

	for _, coin := range coins {
		moduleAddress := k.AccountKeeper.GetModuleAddress(types.ModuleName)
		err = ctx.EventManager().EmitTypedEvent(&types.EventTransfer{
			Coin: coin, From: moduleAddress.String(), To: to.String()})
		if err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) burnCoins(ctx sdk.Context, coins sdk.Coins) error {
	err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return err
	}

	return nil
}

// burnGovTokens burns governance coins
func (k Keeper) burnGovTokens(ctx sdk.Context, govTokens sdk.Coin) error {
	err := k.burnCoins(ctx, sdk.NewCoins(govTokens))
	if err != nil {
		return err
	}

	err = ctx.EventManager().EmitTypedEvent(
		&types.EventBurnNIBI{Amount: govTokens.Amount})
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) burnStableTokens(ctx sdk.Context, stable sdk.Coin) error {
	err := k.burnCoins(ctx, sdk.NewCoins(stable))
	if err != nil {
		return err
	}

	err = ctx.EventManager().EmitTypedEvent(
		&types.EventBurnStable{Amount: stable.Amount})
	if err != nil {
		return err
	}

	return nil
}

// mintStable mints NIBI tokens into module account
func (k Keeper) mintStable(ctx sdk.Context, stable sdk.Coin) error {
	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(stable))
	if err != nil {
		return err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.EventMintStable{Amount: stable.Amount})
	if err != nil {
		return err
	}

	return nil
}

// mintGov mints governance tokens into module account
func (k Keeper) mintGov(ctx sdk.Context, gov sdk.Coin) error {
	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(gov))
	if err != nil {
		return err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.EventMintNIBI{Amount: gov.Amount})
	if err != nil {
		return err
	}

	return nil
}

// splitAndSendFeesToEfAndTreasury sends fees to the Stable Ecosystem Fund and treasury pool
func (k Keeper) splitAndSendFeesToEfAndTreasury(
	ctx sdk.Context, account sdk.AccAddress, efFeeRatio sdk.Dec, coins sdk.Coins,
) error {
	efCoins := sdk.Coins{}
	treasuryCoins := sdk.Coins{}
	for _, c := range coins {
		amountEf := c.Amount.ToDec().Mul(efFeeRatio).TruncateInt()
		amountTreasury := c.Amount.Sub(amountEf)

		if c.Denom == common.DenomNIBI {
			stableCoins := sdk.NewCoins(sdk.NewCoin(c.Denom, amountEf))
			err := k.BankKeeper.SendCoinsFromAccountToModule(
				ctx, account, types.StableEFModuleAccount, stableCoins)
			if err != nil {
				return err
			}

			err = k.BankKeeper.BurnCoins(ctx, types.StableEFModuleAccount, stableCoins)
			if err != nil {
				return err
			}
		} else {
			efCoins = efCoins.Add(sdk.NewCoin(c.Denom, amountEf))
		}
		treasuryCoins = treasuryCoins.Add(sdk.NewCoin(c.Denom, amountTreasury))
	}

	err := k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, account, types.StableEFModuleAccount, efCoins)
	if err != nil {
		return err
	}

	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, account, common.TreasuryPoolModuleAccount, treasuryCoins,
	)
	if err != nil {
		return err
	}

	return nil
}

// BurnStable burns stable coin (plus fees) and returns the equivalent of collateral and gov token.
// Fees are distributed between ecosystem fund and treasury based on feeRatio.
func (k Keeper) BurnStable(goCtx context.Context, msg *types.MsgBurnStable,
) (*types.MsgBurnStableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	msgCreator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	if msg.Stable.Amount == sdk.ZeroInt() {
		return nil, sdkerrors.Wrap(types.NoCoinFound, msg.Stable.Denom)
	}

	params := k.GetParams(ctx)
	if !params.IsCollateralRatioValid {
		return nil, types.NoValidCollateralRatio
	}

	feeRatio := params.GetFeeRatioAsDec()
	collRatio := params.GetCollRatioAsDec()
	govRatio := sdk.OneDec().Sub(collRatio)

	redeemGovCoin, govFees, err := k.calcNeededGovAndFees(ctx, msg.Stable, govRatio, feeRatio)
	if err != nil {
		return nil, err
	}
	redeemCollCoin, collFees, err := k.calcNeededCollateralAndFees(ctx, msg.Stable, collRatio, feeRatio)
	if err != nil {
		return nil, err
	}

	if err = k.mintGov(ctx, redeemGovCoin); err != nil {
		return nil, err
	}
	// The user receives a mixure of collateral (COLL) and governance (GOV) tokens
	// based on the collateral ratio.

	// Send NUSD from account to module
	stablesToBurn := sdk.NewCoins(msg.Stable)
	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, msgCreator, types.ModuleName, stablesToBurn)
	if err != nil {
		return nil, err
	}

	redeemedCoins := sdk.NewCoins(redeemCollCoin, redeemGovCoin)
	err = k.sendCoinsFromModuleAccountToUser(ctx, msgCreator, redeemedCoins)
	if err != nil {
		return nil, err
	}

	feesToSendEF := sdk.NewCoins(govFees, collFees)
	err = k.splitAndSendFeesToEfAndTreasury(
		ctx,
		msgCreator,
		params.GetEfFeeRatioAsDec(),
		feesToSendEF,
	)
	if err != nil {
		return nil, err
	}

	err = k.burnStableTokens(ctx, msg.Stable)
	if err != nil {
		panic(err)
	}

	return &types.MsgBurnStableResponse{
		Collateral: redeemCollCoin.Sub(collFees),
		Gov:        redeemGovCoin.Sub(govFees),
		FeesPayed:  sdk.NewCoins(govFees, collFees),
	}, nil
}
