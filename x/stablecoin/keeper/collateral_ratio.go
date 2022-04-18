package keeper

import (
	"context"
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin/events"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
The collateral ratio, or 'collRatio' (sdk.Dec), is a value beteween 0 and 1 that determines
what proportion of collateral and governance token is used during stablecoin mints
and burns.
*/

// GetCollRatio queries the 'collRatio'.
func (k *Keeper) GetCollRatio(ctx sdk.Context) (collRatio sdk.Dec) {
	return sdk.NewDec(k.GetParams(ctx).CollRatio).QuoInt64(1_000_000)
}

/*
SetCollRatio manually sets the 'collRatio'. This method is mainly used for
testing. When the chain is live, the collateral ratio cannot be manually set, only
adjusted by a fixed amount (e.g. 0.25%).
*/
func (k *Keeper) SetCollRatio(ctx sdk.Context, collRatio sdk.Dec) (err error) {
	collRatioTooHigh := collRatio.GT(sdk.OneDec())
	collRatioTooLow := collRatio.IsNegative()
	if collRatioTooHigh {
		return fmt.Errorf("input 'collRatio', %d, is higher than 1", collRatio)
	} else if collRatioTooLow {
		return fmt.Errorf("input 'collRatio', %d, is negative", collRatio)
	}

	params := k.GetParams(ctx)
	// TODO this should be rethought for production
	newParams := types.NewParams(
		collRatio,
		params.GetFeeRatioAsDec(),
		params.GetEfFeeRatioAsDec(),
		params.GetBonusRateRecollAsDec(),
	)
	k.ParamSubspace.SetParamSet(ctx, &newParams)

	return err
}

/*
GetCollUSDForTargetCollRatio is the collateral value in USD needed to reach a target
collateral ratio.
*/
func (k *Keeper) GetCollUSDForTargetCollRatio(ctx sdk.Context) (neededCollUSD sdk.Dec, err error) {
	stableSupply := k.GetSupplyNUSD(ctx)
	targetCollRatio := k.GetCollRatio(ctx)
	moduleAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	moduleCoins := k.BankKeeper.SpendableCoins(ctx, moduleAddr)
	collDenoms := []string{common.CollDenom}

	currentTotalCollUSD := sdk.ZeroDec()
	pricePools := map[string]string{
		common.CollDenom: common.CollStablePool,
	}
	for _, collDenom := range collDenoms {
		amtColl := moduleCoins.AmountOf(collDenom)
		priceColl, err := k.PriceKeeper.GetCurrentPrice(ctx, pricePools[collDenom])
		if err != nil {
			return sdk.ZeroDec(), err
		}
		collUSD := priceColl.Price.MulInt(amtColl)
		currentTotalCollUSD = currentTotalCollUSD.Add(collUSD)
	}

	targetCollUSD := targetCollRatio.MulInt(stableSupply.Amount)
	neededCollUSD = targetCollUSD.Sub(currentTotalCollUSD)
	return neededCollUSD, err
}

func (k *Keeper) GetCollAmtForTargetCollRatio(
	ctx sdk.Context,
) (neededCollAmount sdk.Int, err error) {
	neededUSD, _ := k.GetCollUSDForTargetCollRatio(ctx)
	priceCollStable, err := k.PriceKeeper.GetCurrentPrice(ctx, common.CollStablePool)
	if err != nil {
		return sdk.Int{}, err
	}

	neededCollAmountDec := neededUSD.Quo(priceCollStable.Price)
	return neededCollAmountDec.Ceil().TruncateInt(), err
}

/*
GovAmtFromRecollateralize computes the GOV token given as a reward for calling
recollateralize.
Args:
  ctx (sdk.Context): Carries information about the current state of the application.
  collDenom (string): 'Denom' of the collateral to be used for recollateralization.
Returns:
  govOut (sdk.Int): Amount of GOV token rewarded for 'Recollateralize'.
*/
func (k *Keeper) GovAmtFromRecollateralize(
	ctx sdk.Context, collUSD sdk.Dec,
) (govOut sdk.Int, err error) {

	params := k.GetParams(ctx)
	bonusRate := params.GetBonusRateRecollAsDec()

	priceGovStable, err := k.PriceKeeper.GetCurrentPrice(ctx, common.GovStablePool)
	if err != nil {
		return sdk.Int{}, err
	}
	govOut = collUSD.Mul(sdk.OneDec().Add(bonusRate)).
		Quo(priceGovStable.Price).TruncateInt()
	return govOut, err
}

func (k *Keeper) GovAmtFromFullRecollateralize(
	ctx sdk.Context,
) (govOut sdk.Int, err error) {

	neededCollUSD, err := k.GetCollUSDForTargetCollRatio(ctx)
	if err != nil {
		return sdk.Int{}, err
	}
	return k.GovAmtFromRecollateralize(ctx, neededCollUSD)
}

/*
Recollateralize
*/
func (k Keeper) Recollateralize(
	goCtx context.Context, msg *types.MsgRecollateralize,
) (response *types.MsgRecollateralizeResponse, err error) {

	ctx := sdk.UnwrapSDKContext(goCtx)
	caller, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return response, err
	}

	params := k.GetParams(ctx)
	targetCollRatio := params.GetCollRatioAsDec()

	neededCollAmt, err := k.GetCollAmtForTargetCollRatio(ctx)
	if err != nil {
		return response, err
	} else if neededCollAmt.LTE(sdk.ZeroInt()) {
		return response, fmt.Errorf(
			"protocol has sufficient COLL, so 'Recollateralize' is not needed")
	}

	// The caller doesn't need to be put in the full amount,
	// just a positive amount that is at most the 'neededCollAmount'.
	inColl := sdk.NewCoin(msg.Coll.Denom, sdk.ZeroInt())
	if msg.Coll.Amount.GT(neededCollAmt) {
		inColl.Amount = neededCollAmt
	} else if msg.Coll.Amount.LTE(sdk.ZeroInt()) {
		return response, fmt.Errorf(
			"collateral input, %v, must be positive", msg.Coll.String())
	} else {
		inColl.Amount = msg.Coll.Amount
	}

	// Send collateral from the caller to the module
	err = k.checkEnoughBalance(ctx, inColl, caller)
	if err != nil {
		return response, err
	}
	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx, caller, types.ModuleName, sdk.NewCoins(inColl),
	)
	if err != nil {
		return response, err
	}
	events.EmitTransfer(
		ctx, inColl,
		/* from */ k.AccountKeeper.GetModuleAddress(types.ModuleName).String(),
		/* to   */ caller.String(),
	)

	// Compute GOV rewarded to user
	priceCollStable, err := k.PriceKeeper.GetCurrentPrice(ctx, common.CollStablePool)
	if err != nil {
		return response, err
	}
	inCollUSD := priceCollStable.Price.MulInt(inColl.Amount)
	outGovAmount, err := k.GovAmtFromRecollateralize(ctx, inCollUSD)
	if err != nil {
		return response, err
	}
	outGov := sdk.NewCoin(common.GovDenom, outGovAmount)

	// Mint and send GOV reward from the module to the caller
	err = k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(outGov))
	if err != nil {
		return response, err
	}
	events.EmitMintNIBI(ctx, outGov)

	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, caller, sdk.NewCoins(outGov),
	)
	if err != nil {
		return response, err
	}
	events.EmitTransfer(
		ctx, outGov,
		/* from */ k.AccountKeeper.GetModuleAddress(types.ModuleName).String(),
		/* to   */ caller.String(),
	)

	events.EmitRecollateralize(
		ctx,
		/* inCoin    */ inColl,
		/* outCoin   */ outGov,
		/* caller    */ caller.String(),
		/* collRatio */ targetCollRatio,
	)
	return &types.MsgRecollateralizeResponse{
		Gov: outGov,
	}, err
}
