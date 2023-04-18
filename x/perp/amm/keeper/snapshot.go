package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

/*
An object parameter for getPriceWithSnapshot().

Specifies how to read the price from a single snapshot. There are three ways:
SPOT: spot price
QUOTE_ASSET_SWAP: price when swapping y amount of quote assets
BASE_ASSET_SWAP: price when swapping x amount of base assets
*/
type snapshotPriceOptions struct {
	// required
	pair           asset.Pair
	twapCalcOption types.TwapCalcOption

	// required only if twapCalcOption == QUOTE_ASSET_SWAP or BASE_ASSET_SWAP
	direction   types.Direction
	assetAmount sdk.Dec
}

/*
Pure function that returns a price from a snapshot.

Can choose from three types of calc options: SPOT, QUOTE_ASSET_SWAP, and BASE_ASSET_SWAP.
QUOTE_ASSET_SWAP and BASE_ASSET_SWAP require the `direction“ and `assetAmount“ args.
SPOT does not require `direction` and `assetAmount`.

args:
  - pair: the token pair
  - snapshot: a reserve snapshot
  - twapCalcOption: SPOT, QUOTE_ASSET_SWAP, or BASE_ASSET_SWAP
  - direction: add or remove; only required for QUOTE_ASSET_SWAP or BASE_ASSET_SWAP
  - assetAmount: the amount of base or quote asset; only required for QUOTE_ASSET_SWAP or BASE_ASSET_SWAP

ret:
  - price: the price as sdk.Dec
  - err: error
*/
func getPriceWithSnapshot(
	snapshot types.ReserveSnapshot,
	snapshotPriceOpts snapshotPriceOptions,
) (price sdk.Dec, err error) {
	switch snapshotPriceOpts.twapCalcOption {
	case types.TwapCalcOption_SPOT:
		return snapshot.QuoteReserve.Quo(snapshot.BaseReserve).Mul(snapshot.PegMultiplier), nil

	case types.TwapCalcOption_QUOTE_ASSET_SWAP:
		pool := &types.Market{
			Pair:          snapshotPriceOpts.pair,
			QuoteReserve:  snapshot.QuoteReserve,
			BaseReserve:   snapshot.BaseReserve,
			SqrtDepth:     common.MustSqrtDec(snapshot.QuoteReserve.Mul(snapshot.BaseReserve)),
			PegMultiplier: snapshot.PegMultiplier,
			Bias:          snapshot.Bias,
			Config: types.MarketConfig{
				FluctuationLimitRatio:  sdk.ZeroDec(), // unused
				MaintenanceMarginRatio: sdk.ZeroDec(), // unused
				MaxLeverage:            sdk.ZeroDec(), // unused
				MaxOracleSpreadRatio:   sdk.ZeroDec(), // unused
				TradeLimitRatio:        sdk.ZeroDec(), // unused
			},
		}
		price, err = pool.GetBaseAmountByQuoteAmount(snapshotPriceOpts.assetAmount.Quo(pool.PegMultiplier).MulInt64(snapshotPriceOpts.direction.ToMultiplier()))
		if err != nil {
			return
		}
		return

	case types.TwapCalcOption_BASE_ASSET_SWAP:
		pool := &types.Market{
			Pair:          snapshotPriceOpts.pair,
			QuoteReserve:  snapshot.QuoteReserve,
			BaseReserve:   snapshot.BaseReserve,
			SqrtDepth:     common.MustSqrtDec(snapshot.QuoteReserve.Mul(snapshot.BaseReserve)),
			PegMultiplier: snapshot.PegMultiplier,
			Bias:          snapshot.Bias,

			Config: types.MarketConfig{
				FluctuationLimitRatio:  sdk.ZeroDec(), // unused
				MaintenanceMarginRatio: sdk.ZeroDec(), // unused
				MaxLeverage:            sdk.ZeroDec(), // unused
				MaxOracleSpreadRatio:   sdk.ZeroDec(), // unused
				TradeLimitRatio:        sdk.ZeroDec(), // unused
			},
		}
		price, err = pool.GetQuoteAmountByBaseAmount(
			snapshotPriceOpts.assetAmount.MulInt64(snapshotPriceOpts.direction.ToMultiplier()),
		)
		if err != nil {
			return
		}
		price = price.Mul(pool.PegMultiplier)
		return
	}

	return sdk.ZeroDec(), nil
}
