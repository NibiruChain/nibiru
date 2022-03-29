package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/MatrixDao/matrix/x/vamm/types"
)

// addReserveSnapshot adds a snapshot of the current pool status and blocktime and blocknum.
func (k Keeper) addReserveSnapshot(ctx sdk.Context, pool *types.Pool) error {
	blockNumber := ctx.BlockHeight()
	lastSnapshot, err := k.getLastReserveSnapshot(ctx, pool.Pair)
	if err != nil {
		return err
	}

	if blockNumber == lastSnapshot.BlockNumber {
		err = k.updateSnapshot(ctx, pool)
		if err != nil {
			return fmt.Errorf("error updating snapshot: %w", err)
		}
	} else {
		err = k.saveReserveSnapshot(ctx, pool)
		if err != nil {
			return fmt.Errorf("error saving snapshot: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSnapshotSaved,
			sdk.NewAttribute(types.AttributeBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute(types.AttributeQuoteReserve, pool.QuoteAssetReserve),
			sdk.NewAttribute(types.AttributeBaseReserve, pool.BaseAssetReserve),
		),
	)

	return nil
}

// saveReserveSnapshot saves reserve snapshot and increments counter
func (k Keeper) saveReserveSnapshot(ctx sdk.Context, pool *types.Pool) error {
	counter, found := k.getSnapshotCounter(ctx, pool.Pair)
	if !found {
		counter = 1
	} else {
		counter = counter + 1
	}

	err := k.saveSnapshotInStore(ctx, pool, counter)
	if err != nil {
		return err
	}

	k.updateSnapshotCounter(ctx, pool.Pair, counter)

	return nil
}

// updateSnapshot saves the snapshot but does not increase the counter
func (k Keeper) updateSnapshot(ctx sdk.Context, pool *types.Pool) error {
	counter, found := k.getSnapshotCounter(ctx, pool.Pair)
	if !found {
		return fmt.Errorf("counter not found, probably is the first snapshot")
	}

	err := k.saveSnapshotInStore(ctx, pool, counter)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) saveSnapshotInStore(ctx sdk.Context, pool *types.Pool, counter int64) error {
	snapshot := &types.ReserveSnapshot{
		QuoteAssetReserve: pool.QuoteAssetReserve,
		BaseAssetReserve:  pool.BaseAssetReserve,
		Timestamp:         ctx.BlockTime().Unix(),
		BlockNumber:       ctx.BlockHeight(),
	}
	bz, err := k.codec.Marshal(snapshot)
	if err != nil {
		return err
	}

	store := k.getStore(ctx)
	store.Set(types.GetPoolReserveSnapshotKey(pool.Pair, counter), bz)

	return nil
}

// getSnapshotCounter returns the counter and if it has been found or not.
func (k Keeper) getSnapshotCounter(ctx sdk.Context, pair string) (int64, bool) {
	store := k.getStore(ctx)

	bz := store.
		Get(types.GetPoolReserveSnapshotCounter(pair))
	if bz == nil {
		return 0, false
	}

	sc := sdk.BigEndianToUint64(bz)

	return int64(sc), true
}

func (k Keeper) updateSnapshotCounter(ctx sdk.Context, pair string, counter int64) {
	store := k.getStore(ctx)

	store.Set(types.GetPoolReserveSnapshotCounter(pair), sdk.Uint64ToBigEndian(uint64(counter)))
}

// getLastReserveSnapshot returns the last snapshot that was saved
func (k Keeper) getLastReserveSnapshot(ctx sdk.Context, pair string) (types.ReserveSnapshot, error) {
	counter, found := k.getSnapshotCounter(ctx, pair)
	if !found {
		return types.ReserveSnapshot{}, types.ErrNoLastSnapshotSaved
	}

	return k.getSnapshotByCounter(ctx, pair, counter)
}

// getSnapshotByCounter returns the snapshot saved by counter num
func (k Keeper) getSnapshotByCounter(ctx sdk.Context, pair string, counter int64) (types.ReserveSnapshot, error) {
	store := k.getStore(ctx)
	bz := store.Get(types.GetPoolReserveSnapshotKey(pair, counter))
	if bz == nil {
		return types.ReserveSnapshot{}, types.ErrNoLastSnapshotSaved.
			Wrap(fmt.Sprintf("snapshot with counter %d was not found", counter))
	}

	var snapshot types.ReserveSnapshot
	err := k.codec.Unmarshal(bz, &snapshot)
	if err != nil {
		return types.ReserveSnapshot{}, fmt.Errorf("problem decoding snapshot")
	}

	return snapshot, nil
}
