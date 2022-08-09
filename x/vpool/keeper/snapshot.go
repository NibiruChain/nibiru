package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

/*
*
updateSnapshot updates the snapshot of the current vpool.
It creates a new one if the current block height is greater than the previous snapshot's block height.
Otherwise, it updates the latest snapshot in state.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair
  - quoteReserve: the amount of quote assets in the vpool
  - baseReserve: the amount of base assets in the vpool
*/
func (k Keeper) updateSnapshot(
	ctx sdk.Context,
	pair common.AssetPair,
	quoteReserve sdk.Dec,
	baseReserve sdk.Dec,
) error {
	lastSnapshot, lastCounter, err := k.getLatestReserveSnapshot(ctx, pair)
	if err != nil {
		return err
	}

	if ctx.BlockHeight() == lastSnapshot.BlockNumber {
		k.saveSnapshot(ctx, pair, lastCounter, quoteReserve, baseReserve)
	} else {
		newCounter := lastCounter + 1
		k.saveSnapshot(ctx, pair, newCounter, quoteReserve, baseReserve)
		k.saveSnapshotCounter(ctx, pair, newCounter)
	}

	return ctx.EventManager().EmitTypedEvent(&types.ReserveSnapshotSavedEvent{
		Pair:         pair.String(),
		QuoteReserve: quoteReserve,
		BaseReserve:  baseReserve,
	})
}

// getSnapshot returns the snapshot saved by counter num
func (k Keeper) getSnapshot(ctx sdk.Context, pair common.AssetPair, counter uint64) (
	snapshot types.ReserveSnapshot, err error,
) {
	bz := ctx.KVStore(k.storeKey).Get(types.GetSnapshotKey(pair, counter))
	if bz == nil {
		return types.ReserveSnapshot{}, types.ErrNoLastSnapshotSaved.
			Wrap(fmt.Sprintf("snapshot with counter %d was not found", counter))
	}

	k.codec.MustUnmarshal(bz, &snapshot)

	return snapshot, nil
}

func (k Keeper) saveSnapshot(
	ctx sdk.Context,
	pair common.AssetPair,
	counter uint64,
	quoteAssetReserve sdk.Dec,
	baseAssetReserve sdk.Dec,
) {
	snapshot := &types.ReserveSnapshot{
		BaseAssetReserve:  baseAssetReserve,
		QuoteAssetReserve: quoteAssetReserve,
		TimestampMs:       ctx.BlockTime().UnixMilli(),
		BlockNumber:       ctx.BlockHeight(),
	}

	ctx.KVStore(k.storeKey).Set(
		types.GetSnapshotKey(pair, counter),
		k.codec.MustMarshal(snapshot),
	)
}

// getSnapshotCounter returns the counter and if it has been found or not.
func (k Keeper) getSnapshotCounter(ctx sdk.Context, pair common.AssetPair) (
	snapshotCounter uint64, found bool,
) {
	bz := ctx.KVStore(k.storeKey).Get(types.GetSnapshotCounterKey(pair))
	if bz == nil {
		return uint64(0), false
	}

	return sdk.BigEndianToUint64(bz), true
}

func (k Keeper) saveSnapshotCounter(
	ctx sdk.Context,
	pair common.AssetPair,
	counter uint64,
) {
	ctx.KVStore(k.storeKey).Set(
		types.GetSnapshotCounterKey(pair),
		sdk.Uint64ToBigEndian(counter),
	)
}

// getLatestReserveSnapshot returns the last snapshot that was saved
func (k Keeper) getLatestReserveSnapshot(ctx sdk.Context, pair common.AssetPair) (
	snapshot types.ReserveSnapshot, counter uint64, err error,
) {
	counter, found := k.getSnapshotCounter(ctx, pair)
	if !found {
		return types.ReserveSnapshot{}, counter, types.ErrNoLastSnapshotSaved
	}

	snapshot, err = k.getSnapshot(ctx, pair, counter)
	if err != nil {
		return types.ReserveSnapshot{}, counter, types.ErrNoLastSnapshotSaved
	}

	return snapshot, counter, nil
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
		pool := types.NewPool(
			snapshotPriceOpts.pair,
			sdk.OneDec(),
			snapshot.QuoteAssetReserve,
			snapshot.BaseAssetReserve,
			sdk.OneDec(),
			sdk.MustNewDecFromStr("0.3"),
			sdk.MustNewDecFromStr("0.0625"),
		)
		return pool.GetBaseAmountByQuoteAmount(snapshotPriceOpts.direction, snapshotPriceOpts.assetAmount)

	case types.TwapCalcOption_BASE_ASSET_SWAP:
		pool := types.NewPool(
			snapshotPriceOpts.pair,
			sdk.OneDec(),
			snapshot.QuoteAssetReserve,
			snapshot.BaseAssetReserve,
			sdk.OneDec(),
			sdk.MustNewDecFromStr("0.3"),
			sdk.MustNewDecFromStr("0.0625"),
		)
		return pool.GetQuoteAmountByBaseAmount(snapshotPriceOpts.direction, snapshotPriceOpts.assetAmount)
	}

	return sdk.ZeroDec(), nil
}
