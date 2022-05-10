package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetSpotPrice(ctx sdk.Context, pair common.TokenPair) (sdk.Dec, error) {
	//TODO implement me
	panic("implement me")
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
