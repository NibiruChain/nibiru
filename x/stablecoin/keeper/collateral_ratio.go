package keeper

import (
	"fmt"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
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
	params := types.NewParams(collRatio)
	k.ParamSubspace.SetParamSet(ctx, &params)
	return err
}

/*
GetNeededCollUSD is the collateral value in USD needed to reach a target
collateral ratio.
*/
func (k *Keeper) GetNeededCollUSD(ctx sdk.Context) (neededCollUSD sdk.Dec, err error) {
	stableSupply := k.GetSupplyUSDM(ctx)
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

func (k *Keeper) GetNeededCollAmount(
	ctx sdk.Context,
) (neededCollAmount sdk.Int, err error) {
	neededUSD, _ := k.GetNeededCollUSD(ctx)
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
*/
func (k *Keeper) GovAmtFromRecollateralize(
	ctx sdk.Context,
) (govOut sdk.Int, err error) {

	neededCollUSD, err0 := k.GetNeededCollUSD(ctx)

	bonusRate := sdk.MustNewDecFromStr("0.002") // TODO: Replace with attribute

	priceCollStable, err1 := k.PriceKeeper.GetCurrentPrice(ctx, common.CollStablePool)
	priceGovColl, err2 := k.PriceKeeper.GetCurrentPrice(ctx, common.GovCollPool)
	for _, err := range []error{err0, err1, err2} {
		if err != nil {
			return sdk.Int{}, err
		}
	}
	priceGovStable := priceGovColl.Price.Mul(priceCollStable.Price)
	govOut = neededCollUSD.Mul(sdk.OneDec().Add(bonusRate)).Quo(priceGovStable).TruncateInt()
	return govOut, err
}

/*
Recollateralize
*/
func (k *Keeper) Recollateralize(ctx sdk.Context, collRatio sdk.Dec) {
	// TODO https://github.com/MatrixDao/matrix/issues/118
}
