package keeper

import (
	"fmt"
	"sort"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
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
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetPrice updates the posted price for a specific oracle
func (k Keeper) SetPrice(
	ctx sdk.Context,
	oracle sdk.AccAddress,
	token0 string,
	token1 string,
	price sdk.Dec,
	expiry time.Time) (types.PostedPrice, error) {
	// If the expiry is less than or equal to the current blockheight, we consider the price valid
	if !expiry.After(ctx.BlockTime()) {
		return types.PostedPrice{}, types.ErrExpired
	}

	// TODO: test this behavior when setting the inverse pair
	pairName := common.RawPoolNameFromDenoms([]string{token0, token1})
	pairID := common.PoolNameFromDenoms([]string{token0, token1})
	if (pairName != pairID) && (!price.Equal(sdk.ZeroDec())) {
		price = sdk.OneDec().Quo(price)
	}

	_, err := k.GetOracle(ctx, pairID, oracle)
	if err != nil {
		return types.PostedPrice{}, err
	}

	store := ctx.KVStore(k.storeKey)

	newRawPrice := types.NewPostedPrice(pairID, oracle, price, expiry)

	// Emit an event containing the oracle's new price
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleUpdatedPrice,
			sdk.NewAttribute(types.AttributePairID, pairID),
			sdk.NewAttribute(types.AttributeOracle, oracle.String()),
			sdk.NewAttribute(types.AttributePairPrice, price.String()),
			sdk.NewAttribute(types.AttributeExpiry, expiry.UTC().String()),
		),
	)

	// Sets the raw price for a single oracle instead of an array of all oracle's raw prices
	store.Set(types.RawPriceKey(pairID, oracle), k.cdc.MustMarshal(&newRawPrice))
	return newRawPrice, nil
}

// SimSetPrice simulate SetPrice without needing an oracle and for one hour
func (k Keeper) SimSetPrice(
	ctx sdk.Context,
	token0 string,
	token1 string,
	price sdk.Dec) (types.PostedPrice, error) {
	store := ctx.KVStore(k.storeKey)
	expiry := ctx.BlockTime().UTC().Add(time.Hour * 1)

	pairName := common.RawPoolNameFromDenoms([]string{token0, token1})
	pairID := common.PoolNameFromDenoms([]string{token0, token1})
	if (pairName != pairID) && (!price.Equal(sdk.ZeroDec())) {
		price = sdk.OneDec().Quo(price)
	}

	oracle := sample.AccAddress()
	newRawPrice := types.NewPostedPrice(pairID, oracle, price, expiry)

	// Emit an event containing the oracle's new price
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleUpdatedPrice,
			sdk.NewAttribute(types.AttributePairID, pairID),
			sdk.NewAttribute(types.AttributeOracle, oracle.String()),
			sdk.NewAttribute(types.AttributePairPrice, price.String()),
			sdk.NewAttribute(types.AttributeExpiry, expiry.UTC().String()),
		),
	)

	// Sets the raw price for a single oracle instead of an array of all oracle's raw prices
	store.Set(types.RawPriceKey(pairID, oracle), k.cdc.MustMarshal(&newRawPrice))
	return newRawPrice, nil
}

// SetCurrentPrices updates the price of an asset to the median of all valid oracle inputs
func (k Keeper) SetCurrentPrices(ctx sdk.Context, token0 string, token1 string) error {
	assetPair := common.AssetPair{Token0: token0, Token1: token1}
	pairID := assetPair.Name()
	tokens := common.DenomsFromPoolName(pairID)
	token0, token1 = tokens[0], tokens[1]

	_, ok := k.GetPair(ctx, pairID)
	if !ok {
		return sdkerrors.Wrap(types.ErrInvalidPair, pairID)
	}
	// store current price
	validPrevPrice := true
	prevPrice, err := k.GetCurrentPrice(ctx, token0, token1)
	if err != nil {
		validPrevPrice = false
	}

	postedPrices := k.GetRawPrices(ctx, pairID)

	var notExpiredPrices []types.CurrentPrice
	// filter out expired prices
	for _, post := range postedPrices {
		if post.Expiry.After(ctx.BlockTime()) {
			notExpiredPrices = append(
				notExpiredPrices,
				types.NewCurrentPrice(token0, token1, post.Price))
		}
	}

	if len(notExpiredPrices) == 0 {
		// NOTE: The current price stored will continue storing the most recent (expired)
		// price if this is not set.
		// This zero's out the current price stored value for that market and ensures
		// that CDP methods that GetCurrentPrice will return error.
		k.setCurrentPrice(ctx, pairID, types.CurrentPrice{})
		return types.ErrNoValidPrice
	}

	medianPrice := k.CalculateMedianPrice(notExpiredPrices)

	// check case that market price was not set in genesis
	if validPrevPrice && !medianPrice.Equal(prevPrice.Price) {
		// only emit event if price has changed
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePairPriceUpdated,
				sdk.NewAttribute(types.AttributePairID, pairID),
				sdk.NewAttribute(types.AttributePairPrice, medianPrice.String()),
			),
		)
	}

	currentPrice := types.NewCurrentPrice(token0, token1, medianPrice)
	k.setCurrentPrice(ctx, pairID, currentPrice)

	return nil
}

func (k Keeper) setCurrentPrice(ctx sdk.Context, pairID string, currentPrice types.CurrentPrice) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CurrentPriceKey(pairID), k.cdc.MustMarshal(&currentPrice))
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

// GetCurrentPrice fetches the current median price of all oracles for a specific market
func (k Keeper) GetCurrentPrice(ctx sdk.Context, token0 string, token1 string,
) (currPrice types.CurrentPrice, err error) {
	assetPair := common.AssetPair{Token0: token0, Token1: token1}
	pairID := assetPair.Name()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CurrentPriceKey(pairID))

	if bz == nil {
		return types.CurrentPrice{}, types.ErrNoValidPrice
	}

	var price types.CurrentPrice
	err = k.cdc.Unmarshal(bz, &price)
	if err != nil {
		return types.CurrentPrice{}, err
	}
	if price.Price.Equal(sdk.ZeroDec()) {
		return types.CurrentPrice{}, types.ErrNoValidPrice
	}

	if !assetPair.IsProperOrder() {
		// Return the inverse price if the tokens are not in "proper" order.
		inversePrice := sdk.OneDec().Quo(price.Price)
		return types.NewCurrentPrice(
			/* token0 */ token1,
			/* token1 */ token0,
			/* price */ inversePrice), nil
	}

	return price, nil
}

// GetCurrentTWAPPrice fetches the current median price of all oracles for a specific market
// TODO: Replace this with actual TWAP logic
func (k Keeper) GetCurrentTWAPPrice(ctx sdk.Context, token0 string, token1 string) (types.CurrentPrice, error) {
	return k.GetCurrentPrice(ctx, token0, token1)
}

// IterateCurrentPrices iterates over all current price objects in the store and performs a callback function
func (k Keeper) IterateCurrentPrices(ctx sdk.Context, cb func(cp types.CurrentPrice) (stop bool)) {
	iterator := sdk.KVStorePrefixIterator(ctx.KVStore(k.storeKey), types.CurrentPricePrefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var cp types.CurrentPrice
		k.cdc.MustUnmarshal(iterator.Value(), &cp)
		if cb(cp) {
			break
		}
	}
}

// GetCurrentPrices returns all current price objects from the store
func (k Keeper) GetCurrentPrices(ctx sdk.Context) types.CurrentPrices {
	var cps types.CurrentPrices
	k.IterateCurrentPrices(ctx, func(cp types.CurrentPrice) (stop bool) {
		cps = append(cps, cp)
		return false
	})
	return cps
}

// GetRawPrices fetches the set of all prices posted by oracles for an asset
func (k Keeper) GetRawPrices(ctx sdk.Context, marketId string) types.PostedPrices {
	var pps types.PostedPrices
	k.IterateRawPricesByPair(ctx, marketId, func(pp types.PostedPrice) (stop bool) {
		pps = append(pps, pp)
		return false
	})
	return pps
}

// IterateRawPrices iterates over all raw prices in the store and performs a callback function
func (k Keeper) IterateRawPricesByPair(ctx sdk.Context, marketId string, cb func(record types.PostedPrice) (stop bool)) {
	iterator := sdk.KVStorePrefixIterator(ctx.KVStore(k.storeKey), types.RawPriceIteratorKey((marketId)))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var record types.PostedPrice
		k.cdc.MustUnmarshal(iterator.Value(), &record)
		if cb(record) {
			break
		}
	}
}
