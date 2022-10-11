package keeper

import (
	"fmt"
	"sort"
	"time"

	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"

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

		RawPrices     collections.Map[keys.Pair[common.AssetPair, keys.StringKey], types.PostedPrice, *types.PostedPrice]
		CurrentPrices collections.Map[common.AssetPair, types.CurrentPrice, *types.CurrentPrice]
		// PriceSnapshots maps types.PriceSnapshot to the common.AssetPair of the snapshot and the creation timestamp as keys.Uint64Key.
		// TODO(mercilex): maybe it's worth to create a keys.Timestamp key?
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

		CurrentPrices:  collections.NewMap[common.AssetPair, types.CurrentPrice](cdc, storeKey, 0),
		RawPrices:      collections.NewMap[keys.Pair[common.AssetPair, keys.StringKey], types.PostedPrice](cdc, storeKey, 1),
		PriceSnapshots: collections.NewMap[keys.Pair[common.AssetPair, keys.Uint64Key], types.PriceSnapshot](cdc, storeKey, 2),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

/*
PostRawPrice updates the posted price for a specific oracle

args:
  - ctx: the sdk context
  - oracle: the address of the oracle posting the raw price
  - pairStr: the string of the asset pair
  - price: the price
  - expiry: when the raw price should expire

ret:
  - err: error if any
*/
func (k Keeper) PostRawPrice(
	ctx sdk.Context,
	oracle sdk.AccAddress,
	pairStr string,
	price sdk.Dec,
	expiry time.Time,
) (err error) {
	// If the posted price expires before the current block, it is invalid.
	if expiry.Before(ctx.BlockTime()) {
		return types.ErrExpired
	}

	if !price.IsPositive() {
		return fmt.Errorf("price must be positive, not: %s", price)
	}

	pair, err := common.NewAssetPair(pairStr)
	if err != nil {
		return err
	}

	// Set inverse price if the oracle gives the wrong string
	if k.IsActivePair(ctx, pair.Inverse().String()) {
		pair = pair.Inverse()
		price = sdk.OneDec().Quo(price)
	}

	if !k.IsWhitelistedOracle(ctx, pair.String(), oracle) {
		return fmt.Errorf("oracle %s cannot post on pair %v", oracle, pair.String())
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
	return nil
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
		// price if this is not deleted.
		_ = k.CurrentPrices.Delete(ctx, assetPair)
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

	// Update the TWAP prices
	k.PriceSnapshots.Insert(ctx, keys.Join(assetPair, keys.Uint64(uint64(ctx.BlockTime().UnixMilli()))), types.PriceSnapshot{
		PairId:      pairID,
		Price:       currentPrice.Price,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	})

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

If there's only one snapshot, then this function returns the price from that single snapshot.

Returns -1 if there's no price.

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

	// traverse snapshots of the pair in reverse order from currentTime - TWAPLookBackWindow => now
	pair := keys.PairPrefix[common.AssetPair, keys.Uint64Key](assetPair)                  // only current AssetPair
	fromTime := ctx.BlockTime().Add(-1 * k.GetParams(ctx).TwapLookbackWindow).UnixMilli() // earliest timestamp aka current time - TWAPLookBackWindow
	toTime := ctx.BlockTime().UnixMilli()                                                 // current time
	start := keys.PairSuffix[common.AssetPair](keys.Uint64(uint64(fromTime)))             // this turns it into a suffix
	end := keys.PairSuffix[common.AssetPair](keys.Uint64(uint64(toTime)))
	rng := keys.NewRange[keys.Pair[common.AssetPair, keys.Uint64Key]]().
		Prefix(pair).
		Start(keys.Inclusive(start)).
		End(keys.Exclusive(end)) // current block shouldn't have a snapshot until EndBlock is run

	snapshots := k.PriceSnapshots.Iterate(ctx, rng).Values()
	if len(snapshots) == 0 {
		// if there are no snapshots, return -1 for the price
		return sdk.OneDec().Neg(), types.ErrNoValidTWAP
	}

	twap, err = calcTwap(ctx, snapshots)
	if err != nil {
		return sdk.Dec{}, err
	}
	if inverseIsActive {
		return sdk.OneDec().Quo(twap), nil
	}
	return twap, nil
}

/*
calcTwap walks through a slice of PriceSnapshots and tallies up the prices weighted by the amount of time they were active for.

Callers of this function should already check if the snapshot slice is empty. Passing an empty snapshot slice will result in a panic.
*/
func calcTwap(ctx sdk.Context, snapshots []types.PriceSnapshot) (sdk.Dec, error) {
	cumulativeTime := ctx.BlockTime().UnixMilli() - snapshots[0].TimestampMs
	cumulativePrice := sdk.ZeroDec()

	for i, s := range snapshots {
		var nextTimestampMs int64
		if i == len(snapshots)-1 {
			// if we're at the last snapshot, then consider that price as ongoing until the current blocktime
			nextTimestampMs = ctx.BlockTime().UnixMilli()
		} else {
			nextTimestampMs = snapshots[i+1].TimestampMs
		}
		price := s.Price.MulInt64(nextTimestampMs - s.TimestampMs)
		cumulativePrice = cumulativePrice.Add(price)
	}
	return cumulativePrice.QuoInt64(cumulativeTime), nil
}

// GetCurrentPrices returns all current price objects from the store
func (k Keeper) GetCurrentPrices(ctx sdk.Context) types.CurrentPrices {
	return k.CurrentPrices.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values()
}

// GetRawPrices fetches the set of all prices posted by oracles for an asset
func (k Keeper) GetRawPrices(ctx sdk.Context, pairStr string) types.PostedPrices {
	pair := common.MustNewAssetPair(pairStr)
	if k.IsActivePair(ctx, pair.Inverse().String()) {
		panic(fmt.Errorf(
			`cannot fetch posted prices using inverse pair, %v ;
			Use pair, %v, instead`, pair.Inverse().String(), pairStr))
	}

	prefix := keys.PairPrefix[common.AssetPair, keys.StringKey](pair)
	return k.
		RawPrices.
		Iterate(ctx,
			keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]().Prefix(prefix),
		).Values()
}
