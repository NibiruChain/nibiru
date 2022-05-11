package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) updateReserve(
	ctx sdk.Context,
	pool *types.Pool,
	dir types.Direction,
	quoteAssetAmount sdk.Int,
	baseAssetAmount sdk.Int,
	skipFluctuationCheck bool,
) error {
	if dir == types.Direction_ADD_TO_POOL {
		pool.IncreaseQuoteAssetReserve(quoteAssetAmount)
		pool.DecreaseBaseAssetReserve(baseAssetAmount)
		// TODO baseAssetDeltaThisFunding
		// TODO totalPositionSize
		// TODO cumulativeNotional
	} else {
		pool.DecreaseQuoteAssetReserve(quoteAssetAmount)
		pool.IncreaseBaseAssetReserve(baseAssetAmount)
		// TODO baseAssetDeltaThisFunding
		// TODO totalPositionSize
		// TODO cumulativeNotional
	}

	// Check if its over Fluctuation Limit Ratio.
	if !skipFluctuationCheck {
		err := k.checkFluctuationLimitRatio(ctx, pool)
		if err != nil {
			return err
		}
	}

	err := k.addReserveSnapshot(ctx, pool)
	if err != nil {
		return fmt.Errorf("error creating snapshot: %w", err)
	}

	k.savePool(ctx, pool)

	return nil
}

// addReserveSnapshot adds a snapshot of the current pool status and blocktime and blocknum.
func (k Keeper) addReserveSnapshot(ctx sdk.Context, pool *types.Pool) error {
	lastSnapshot, lastCounter, err := k.getLatestReserveSnapshot(ctx, common.TokenPair(pool.Pair))
	if err != nil {
		return err
	}

	if ctx.BlockHeight() == lastSnapshot.BlockNumber {
		k.saveSnapshot(ctx, pool, lastCounter)
	} else {
		newCounter := lastCounter + 1
		k.saveSnapshot(ctx, pool, newCounter)
		k.saveSnapshotCounter(ctx, common.TokenPair(pool.Pair), newCounter)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSnapshotSaved,
			sdk.NewAttribute(types.AttributeBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute(types.AttributeQuoteReserve, pool.QuoteAssetReserve.String()),
			sdk.NewAttribute(types.AttributeBaseReserve, pool.BaseAssetReserve.String()),
		),
	)

	return nil
}

// getSnapshot returns the snapshot saved by counter num
func (k Keeper) getSnapshot(ctx sdk.Context, pair common.TokenPair, counter uint64) (
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
	pool *types.Pool,
	counter uint64,
) {
	snapshot := &types.ReserveSnapshot{
		BaseAssetReserve:  pool.BaseAssetReserve,
		QuoteAssetReserve: pool.QuoteAssetReserve,
		Timestamp:         ctx.BlockTime().Unix(),
		BlockNumber:       ctx.BlockHeight(),
	}
	bz := k.codec.MustMarshal(snapshot)
	ctx.KVStore(k.storeKey).Set(
		types.GetSnapshotKey(common.TokenPair(pool.Pair), counter),
		bz,
	)
}

// getSnapshotCounter returns the counter and if it has been found or not.
func (k Keeper) getSnapshotCounter(ctx sdk.Context, pair common.TokenPair) (
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
	pair common.TokenPair,
	counter uint64,
) {
	ctx.KVStore(k.storeKey).Set(
		types.GetSnapshotCounterKey(pair),
		sdk.Uint64ToBigEndian(counter),
	)
}

// getLatestReserveSnapshot returns the last snapshot that was saved
func (k Keeper) getLatestReserveSnapshot(ctx sdk.Context, pair common.TokenPair) (
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
