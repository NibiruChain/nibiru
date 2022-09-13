package keeper

import (
	"fmt"
	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"
	"os"
	"sort"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		CurrentPrices  collections.Map[common.AssetPair, types.CurrentPrice, *types.CurrentPrice]
		RawPrices      collections.Map[keys.Pair[common.AssetPair, keys.StringKey], types.PostedPrice, *types.PostedPrice]
		PriceSnapshots collections.Map[keys.Pair[common.AssetPair, keys.Uint64Key], types.PriceSnapshot, *types.PriceSnapshot]
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,

		CurrentPrices: collections.NewMap[common.AssetPair, types.CurrentPrice](cdc, storeKey, 0),
		RawPrices:     collections.NewMap[keys.Pair[common.AssetPair, keys.StringKey], types.PostedPrice](cdc, storeKey, 1),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// PostRawPrice updates the posted price for a specific oracle
func (k Keeper) PostRawPrice(
	ctx sdk.Context,
	oracle sdk.AccAddress,
	pairStr string,
	price sdk.Dec,
	expiry time.Time,
) (postedPrice types.PostedPrice, err error) {
	// If the posted price expires before the current block, it is invalid.
	if expiry.Before(ctx.BlockTime()) {
		return postedPrice, types.ErrExpired
	}

	if !price.IsPositive() {
		return postedPrice, fmt.Errorf("price must be positive, not: %s", price)
	}

	pair, err := common.NewAssetPair(pairStr)
	if err != nil {
		return postedPrice, err
	}

	// Set inverse price if the oracle gives the wrong string
	if k.IsActivePair(ctx, pair.Inverse().String()) {
		pair = pair.Inverse()
		price = sdk.OneDec().Quo(price)
	}

	if !k.IsWhitelistedOracle(ctx, pair.String(), oracle) {
		return types.PostedPrice{}, fmt.Errorf("oracle %s cannot post on pair %v", oracle, pair.String())
	}

	// Emit an event containing the oracle's new price
	if err = ctx.EventManager().EmitTypedEvent(&types.EventOracleUpdatePrice{
		PairId:    pair.String(),
		Oracle:    oracle.String(),
		PairPrice: price,
		Expiry:    expiry,
	}); err != nil {
		panic(err)
	}

	// Sets the raw price for a single oracle instead of an array of all oracle's raw prices
	newPostedPrice := types.NewPostedPrice(pair, oracle, price, expiry)
	k.RawPrices.Insert(ctx, keys.Join(pair, keys.String(oracle.String())), newPostedPrice)
	return newPostedPrice, nil
}

/*
GatherRawPrices updates the current price of an asset to the median of all valid posted oracle prices.

args:
  - ctx: cosmos-sdk context
  - token0: the base asset
  - token1: the quote asset

ret:
  - err: error if any
*/
func (k Keeper) GatherRawPrices(ctx sdk.Context, token0 string, token1 string) error {
	assetPair := common.AssetPair{Token0: token0, Token1: token1}
	pairID := assetPair.String()

	if !k.IsActivePair(ctx, pairID) {
		return sdkerrors.Wrap(types.ErrInvalidPair, pairID)
	}
	// store current price
	validPrevPrice := true
	prevPrice, err := k.GetCurrentPrice(ctx, token0, token1)
	if err != nil {
		validPrevPrice = false
	}

	var unexpiredPrices []types.CurrentPrice
	// filter out expired prices
	for _, rawPrice := range k.GetRawPrices(ctx, pairID) {
		if rawPrice.Expiry.After(ctx.BlockTime()) {
			unexpiredPrices = append(unexpiredPrices, types.NewCurrentPrice(token0, token1, rawPrice.Price))
		}
	}

	if len(unexpiredPrices) == 0 {
		// NOTE: The current price stored will continue storing the most recent (expired)
		// price if this is not set.
		// This zero's out the current price stored value for that market and ensures
		// that CDP methods that GetCurrentPrice will return error.
		k.CurrentPrices.Insert(ctx, assetPair, types.CurrentPrice{})
		return types.ErrNoValidPrice
	}

	medianPrice := k.CalculateMedianPrice(unexpiredPrices)

	// check case that market price was not set in genesis
	if validPrevPrice && !medianPrice.Equal(prevPrice.Price) {
		// only emit event if price has changed
		err = ctx.EventManager().EmitTypedEvent(&types.EventPairPriceUpdated{
			PairId:    pairID,
			PairPrice: medianPrice,
		})
		if err != nil {
			panic(err)
		}
	}

	currentPrice := types.NewCurrentPrice(token0, token1, medianPrice)
	k.CurrentPrices.Insert(ctx, assetPair, currentPrice)

	// Update the TWA prices
	k.saveOrUpdateSnapshot(ctx, pairID, currentPrice.Price)

	return nil
}

// CalculateMedianPrice calculates the median prices for the input prices.
func (k Keeper) CalculateMedianPrice(prices []types.CurrentPrice) sdk.Dec {
	l := len(prices)

	if l == 1 {
		// Return immediately if there's only one price
		return prices[0].Price
	}
	// sort the prices
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Price.LT(prices[j].Price)
	})
	// for even numbers of prices, the median is calculated as the mean of the two middle prices
	if l%2 == 0 {
		median := k.calculateMeanPrice(prices[l/2-1], prices[l/2])
		return median
	}
	// for odd numbers of prices, return the middle element
	return prices[l/2].Price
}

func (k Keeper) calculateMeanPrice(priceA, priceB types.CurrentPrice) sdk.Dec {
	sum := priceA.Price.Add(priceB.Price)
	mean := sum.Quo(sdk.NewDec(2))
	return mean
}

/*
GetCurrentPrice fetches the current median price of all oracles for a specific market.

args:
  - ctx: cosmos-sdk context
  - token0: the base asset
  - token1: the quote asset

ret:
  - currPrice: the current price
  - err: error if any
*/
func (k Keeper) GetCurrentPrice(ctx sdk.Context, token0 string, token1 string,
) (currPrice types.CurrentPrice, err error) {
	pair := common.AssetPair{Token0: token0, Token1: token1}
	givenIsActive := k.IsActivePair(ctx, pair.String())
	inverseIsActive := k.IsActivePair(ctx, pair.Inverse().String())
	if !givenIsActive && inverseIsActive {
		pair = pair.Inverse()
	}

	// Retrieve current price from the KV store
	price, err := k.CurrentPrices.Get(ctx, pair)
	if err != nil {
		return types.CurrentPrice{}, types.ErrNoValidPrice
	}
	if price.Price.Equal(sdk.ZeroDec()) {
		return types.CurrentPrice{}, types.ErrNoValidPrice
	}

	if inverseIsActive {
		// Return the inverse price if the tokens are not in params order.
		inversePrice := sdk.OneDec().Quo(price.Price)
		return types.NewCurrentPrice(
			/* token0 */ token1,
			/* token1 */ token0,
			/* price */ inversePrice), nil
	}

	return price, nil
}

/*
Gets the time-weighted average price from [ ctx.BlockTime() - interval, ctx.BlockTime() )
Note the open-ended right bracket.

Args:
- ctx: cosmos-sdk context
- pair: the token pair

Returns:
- twap: TWAP as sdk.Dec
- err: error
*/
func (k Keeper) GetCurrentTWAP(ctx sdk.Context, token0 string, token1 string,
) (twap sdk.Dec, err error) {
	// Ensure we still have valid prices
	_, err = k.GetCurrentPrice(ctx, token0, token1)
	if err != nil {
		return sdk.Dec{}, types.ErrNoValidPrice
	}

	assetPair := common.AssetPair{Token0: token0, Token1: token1}
	if err := assetPair.Validate(); err != nil {
		return sdk.Dec{}, err
	}

	// invert the asset pair if the given is not existent
	inverseIsActive := false
	if !k.IsActivePair(ctx, assetPair.String()) && k.IsActivePair(ctx, assetPair.Inverse().String()) {
		assetPair = assetPair.Inverse()
		inverseIsActive = true
	}

	// earliest timestamp we'll look back until
	lookbackWindow := k.GetParams(ctx).TwapLookbackWindow
	lowerLimitTimestampMs := ctx.BlockTime().Add(-lookbackWindow).UnixMilli()

	var cumulativePrice sdk.Dec = sdk.ZeroDec()
	var cumulativePeriodMs int64 = 0
	var prevTimestampMs int64 = ctx.BlockTime().UnixMilli()

	// traverse snapshots in reverse order
	startKey := types.PriceSnapshotKey(assetPair.String(), ctx.BlockHeight())
	var numSnapshots int64 = 0
	var snapshotPriceBuffer []sdk.Dec // contains snapshots at time 0
	k.IteratePriceSnapshotsFrom(
		/*ctx=*/ ctx,
		/*start=*/ startKey,
		/*end=*/ nil,
		/*reverse=*/ true,
		/*do=*/ func(ps *types.PriceSnapshot) (stop bool) {
			fmt.Fprintf(os.Stderr, "%s\n", ps)
			numSnapshots += 1
			var timeElapsedMs int64
			if ps.TimestampMs <= lowerLimitTimestampMs {
				// current snapshot is below the lower limit
				timeElapsedMs = prevTimestampMs - lowerLimitTimestampMs
			} else {
				timeElapsedMs = prevTimestampMs - ps.TimestampMs
			}
			cumulativePrice = cumulativePrice.Add(ps.Price.MulInt64(timeElapsedMs))
			cumulativePeriodMs += timeElapsedMs

			if cumulativePeriodMs <= 0 {
				snapshotPriceBuffer = append(snapshotPriceBuffer, ps.Price)
			}

			// end early if we're already beyond the lower limit timestamp
			if ps.TimestampMs <= lowerLimitTimestampMs {
				return true
			}
			prevTimestampMs = ps.TimestampMs
			return false
		})

	switch {
	case cumulativePeriodMs < 0:
		return sdk.Dec{}, fmt.Errorf("cumulativePeriodMs, %v, should never be negative", cumulativePeriodMs)
	case (cumulativePeriodMs == 0) && (numSnapshots > 0):
		sum := sdk.ZeroDec()
		for _, price := range snapshotPriceBuffer {
			sum = sum.Add(price)
		}
		return sum.QuoInt64(numSnapshots), nil
	case (cumulativePeriodMs == 0) && (numSnapshots == 0):
		return sdk.Dec{}, fmt.Errorf(`
			failed to calculate twap, no time passed and no snapshots have been taken since
			ctx.BlockTime: %v, 
			ctx.BlockHeight: %v,
			assetPair: %s, 
		`, prevTimestampMs, ctx.BlockHeight(), assetPair)
	}

	twap = cumulativePrice.QuoInt64(cumulativePeriodMs)

	if !twap.IsZero() && inverseIsActive {
		return sdk.OneDec().Quo(twap), nil
	}
	return twap, nil
}

// GetCurrentPrices returns all current price objects from the store
func (k Keeper) GetCurrentPrices(ctx sdk.Context) types.CurrentPrices {
	return k.CurrentPrices.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values()
}

// GetRawPrices fetches the set of all prices posted by oracles for an asset
func (k Keeper) GetRawPrices(ctx sdk.Context, pairStr string) types.PostedPrices {
	assetPair := common.MustNewAssetPair(pairStr)
	if k.IsActivePair(ctx, assetPair.Inverse().String()) {
		panic(fmt.Errorf(
			`cannot fetch posted prices using inverse pair, %v ;
			Use pair, %v, instead`, assetPair.Inverse().String(), pairStr))
	}

	prefix := keys.PairPrefix[common.AssetPair, keys.StringKey](assetPair)
	return k.RawPrices.
		Iterate(ctx, keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]().Prefix(prefix)).
		Values()
}
