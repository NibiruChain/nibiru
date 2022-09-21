package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// GetSnapshot returns the snapshot saved by counter num
func (k Keeper) GetSnapshot(ctx sdk.Context, pair common.AssetPair, blockHeight uint64) (
	snapshot types.ReserveSnapshot, err error,
) {
	bz := prefix.NewStore(
		ctx.KVStore(k.storeKey), types.SnapshotsKeyPrefix,
	).Get(types.GetSnapshotKey(pair, blockHeight))

	if bz == nil {
		return types.ReserveSnapshot{}, types.ErrNoLastSnapshotSaved.
			Wrap(fmt.Sprintf("snapshot at blockHeight %d was not found", blockHeight))
	}

	k.codec.MustUnmarshal(bz, &snapshot)

	return snapshot, nil
}

func (k Keeper) SaveSnapshot(
	ctx sdk.Context,
	snapshot types.ReserveSnapshot,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SnapshotsKeyPrefix)
	store.Set(types.GetSnapshotKey(snapshot.Pair, uint64(snapshot.BlockNumber)), k.codec.MustMarshal(&snapshot))
}

// GetLatestReserveSnapshot returns the last snapshot that was saved
func (k Keeper) GetLatestReserveSnapshot(ctx sdk.Context, pair common.AssetPair) (
	snapshot types.ReserveSnapshot, err error,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SnapshotsKeyPrefix)
	iter := store.ReverseIterator(types.GetSnapshotKey(pair, 0), types.GetSnapshotKey(pair, uint64(ctx.BlockHeight()+1)))

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		k.codec.MustUnmarshal(iter.Value(), &snapshot)
		return snapshot, nil
	}

	return types.ReserveSnapshot{}, types.ErrNoLastSnapshotSaved
}

// GetAllSnapshots returns all pools that exist.
func (k Keeper) GetAllSnapshots(ctx sdk.Context) []types.ReserveSnapshot {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.SnapshotsKeyPrefix)

	var snapshots []types.ReserveSnapshot
	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()

		var snapshot types.ReserveSnapshot
		k.codec.MustUnmarshal(bz, &snapshot)

		snapshots = append(snapshots, snapshot)
	}

	return snapshots
}

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
		pool := &types.VPool{
			Pair:                   snapshotPriceOpts.pair,
			TradeLimitRatio:        sdk.ZeroDec(), // unused
			QuoteAssetReserve:      snapshot.QuoteAssetReserve,
			BaseAssetReserve:       snapshot.BaseAssetReserve,
			FluctuationLimitRatio:  sdk.ZeroDec(), // unused
			MaxOracleSpreadRatio:   sdk.ZeroDec(), // unused
			MaintenanceMarginRatio: sdk.ZeroDec(), // unused
			MaxLeverage:            sdk.ZeroDec(), // unused
		}
		return pool.GetBaseAmountByQuoteAmount(snapshotPriceOpts.direction, snapshotPriceOpts.assetAmount)

	case types.TwapCalcOption_BASE_ASSET_SWAP:
		pool := &types.VPool{
			Pair:                   snapshotPriceOpts.pair,
			TradeLimitRatio:        sdk.ZeroDec(), // unused
			QuoteAssetReserve:      snapshot.QuoteAssetReserve,
			BaseAssetReserve:       snapshot.BaseAssetReserve,
			FluctuationLimitRatio:  sdk.ZeroDec(), // unused
			MaxOracleSpreadRatio:   sdk.ZeroDec(), // unused
			MaintenanceMarginRatio: sdk.ZeroDec(), // unused
			MaxLeverage:            sdk.ZeroDec(), // unused
		}
		return pool.GetQuoteAmountByBaseAmount(snapshotPriceOpts.direction, snapshotPriceOpts.assetAmount)
	}

	return sdk.ZeroDec(), nil
}
