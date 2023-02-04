package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
)

var LiquidityRatioBands = sdk.MustNewDecFromStr("0.001")

func (k Keeper) GetSupplyNUSD(
	ctx sdk.Context,
) sdk.Coin {
	return k.BankKeeper.GetSupply(ctx, denoms.NUSD)
}

func (k Keeper) GetSupplyNIBI(
	ctx sdk.Context,
) sdk.Coin {
	return k.BankKeeper.GetSupply(ctx, denoms.NIBI)
}

func (k Keeper) GetStableMarketCap(ctx sdk.Context) sdk.Int {
	return k.GetSupplyNUSD(ctx).Amount
}

func (k Keeper) GetGovMarketCap(ctx sdk.Context) (sdk.Int, error) {
	pool, err := k.SpotKeeper.FetchPoolFromPair(ctx, denoms.NIBI, denoms.NUSD)
	if err != nil {
		return sdk.Int{}, err
	}

	price, err := pool.CalcSpotPrice(denoms.NIBI, denoms.NUSD)
	if err != nil {
		return sdk.Int{}, err
	}

	nibiSupply := k.GetSupplyNIBI(ctx)

	return nibiSupply.Amount.ToDec().Mul(price).RoundInt(), nil
}

// GetLiquidityRatio returns the liquidity ratio defined as govMarketCap / stableMarketCap
func (k Keeper) GetLiquidityRatio(ctx sdk.Context) (sdk.Dec, error) {
	govMarketCap, err := k.GetGovMarketCap(ctx)
	if err != nil {
		return sdk.Dec{}, err
	}

	stableMarketCap := k.GetStableMarketCap(ctx)
	if stableMarketCap.Equal(sdk.ZeroInt()) {
		return sdk.Dec{}, fmt.Errorf("stable maket cap is equal to zero")
	}

	return govMarketCap.ToDec().Quo(stableMarketCap.ToDec()), nil
}

func (k Keeper) GetLiquidityRatioBands(ctx sdk.Context) (
	lowBand sdk.Dec, upBand sdk.Dec, err error) {
	liquidityRatio, err := k.GetLiquidityRatio(ctx)
	if err != nil {
		return sdk.Dec{}, sdk.Dec{}, err
	}

	lowBand = liquidityRatio.Mul(sdk.OneDec().Sub(LiquidityRatioBands))
	upBand = liquidityRatio.Mul(sdk.OneDec().Add(LiquidityRatioBands))

	return lowBand, upBand, err
}
