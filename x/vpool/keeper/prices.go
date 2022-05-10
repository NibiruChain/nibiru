package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
GetSpotPrice retrieves the price of the base asset denominated in quote asset.

The convention is the amount of quote assets required to buy one base asset.

e.g. If the tokenPair is BTC:NUSD, the method would return sdk.Dec(40,000.00)
because the instantaneous tangent slope on the vpool curve is 40,000.00,
so it would cost ~40,000.00 to buy one BTC:NUSD perp.

args:
  - ctx: cosmos-sdk context
  - pair: the token pair to get price for

ret:
  - price: the price of the token pair as sdk.Dec
  - err: error
*/
func (k Keeper) GetSpotPrice(ctx sdk.Context, pair common.TokenPair) (
	price sdk.Dec, err error,
) {
	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	return pool.QuoteAssetReserve.ToDec().
		Quo(pool.BaseAssetReserve.ToDec()), nil
}

/*
Retrieves the base asset's price from PricefeedKeeper (oracle).
The price is denominated in quote asset, so # of quote asset to buy one base asset.

args:
  - ctx: cosmos-sdk context
  - pair: token pair

ret:
  - price: price as sdk.Dec
  -
*/
func (k Keeper) GetUnderlyingPrice(ctx sdk.Context, pair common.TokenPair) (
	price sdk.Dec, err error,
) {
	currentPrice, err := k.pricefeedKeeper.GetCurrentPrice(
		ctx,
		pair.GetBaseTokenDenom(),
		pair.GetQuoteTokenDenom(),
	)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	return currentPrice.Price, nil
}

func (k Keeper) GetOutputPrice(ctx sdk.Context, pair common.TokenPair, dir types.Direction, abs sdk.Int) (sdk.Dec, error) {
	//TODO implement me
	panic("implement me")
}

func (k Keeper) GetOutputTWAP(ctx sdk.Context, pair common.TokenPair, dir types.Direction, abs sdk.Int) (sdk.Dec, error) {
	//TODO implement me
	panic("implement me")
}
