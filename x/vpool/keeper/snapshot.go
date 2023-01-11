package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
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
	pair           common.AssetPair
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
		return snapshot.QuoteAssetReserve.Quo(snapshot.BaseAssetReserve), nil

	case types.TwapCalcOption_QUOTE_ASSET_SWAP:
		pool := &types.Vpool{
			Pair:              snapshotPriceOpts.pair,
			QuoteAssetReserve: snapshot.QuoteAssetReserve,
			BaseAssetReserve:  snapshot.BaseAssetReserve,
			Config: types.VpoolConfig{
				FluctuationLimitRatio:  sdk.ZeroDec(), // unused
				MaintenanceMarginRatio: sdk.ZeroDec(), // unused
				MaxLeverage:            sdk.ZeroDec(), // unused
				MaxOracleSpreadRatio:   sdk.ZeroDec(), // unused
				TradeLimitRatio:        sdk.ZeroDec(), // unused
			},
		}
		return pool.GetBaseAmountByQuoteAmount(snapshotPriceOpts.assetAmount.MulInt64(snapshotPriceOpts.direction.ToMultiplier()))

	case types.TwapCalcOption_BASE_ASSET_SWAP:
		pool := &types.Vpool{
			Pair:              snapshotPriceOpts.pair,
			QuoteAssetReserve: snapshot.QuoteAssetReserve,
			BaseAssetReserve:  snapshot.BaseAssetReserve,
			Config: types.VpoolConfig{
				FluctuationLimitRatio:  sdk.ZeroDec(), // unused
				MaintenanceMarginRatio: sdk.ZeroDec(), // unused
				MaxLeverage:            sdk.ZeroDec(), // unused
				MaxOracleSpreadRatio:   sdk.ZeroDec(), // unused
				TradeLimitRatio:        sdk.ZeroDec(), // unused
			},
		}
		return pool.GetQuoteAmountByBaseAmount(
			snapshotPriceOpts.assetAmount.MulInt64(snapshotPriceOpts.direction.ToMultiplier()),
		)
	}

	return sdk.ZeroDec(), nil
}
