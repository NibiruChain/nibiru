package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

// GetEpochInfo returns epoch info by identifier.
func (k Keeper) GetEpochInfo(ctx sdk.Context, identifier string) types.EpochInfo {
	epoch := types.EpochInfo{}
	store := ctx.KVStore(k.storeKey)
	b := store.Get(append(types.KeyPrefixEpoch, []byte(identifier)...))
	if b == nil {
		return epoch
	}
	err := proto.Unmarshal(b, &epoch)
	if err != nil {
		panic(err)
	}
	return epoch
}

// EpochExists checks if the epoch exists
func (k Keeper) EpochExists(ctx sdk.Context, identifier string) bool {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(append(types.KeyPrefixEpoch, []byte(identifier)...))

	return b != nil
}

// AddEpochInfo adds a new epoch info. Will return an error if the epoch fails validation,
// or re-uses an existing identifier.
// This method also sets the start time if left unset, and sets the epoch start height.
func (k Keeper) AddEpochInfo(ctx sdk.Context, epoch types.EpochInfo) error {
	if err := epoch.Validate(); err != nil {
		return err
	}

	if k.EpochExists(ctx, epoch.Identifier) {
		return fmt.Errorf("epoch with identifier %s already exists", epoch.Identifier)
	}

	// Initialize empty and default epoch values
	if epoch.StartTime.Equal(time.Time{}) {
		epoch.StartTime = ctx.BlockTime()
	}

	epoch.CurrentEpochStartHeight = ctx.BlockHeight()

	k.UpsertEpochInfo(ctx, epoch)

	return nil
}

// UpsertEpochInfo inserts the epoch if does not exist, and overwrites it if it does.
func (k Keeper) UpsertEpochInfo(ctx sdk.Context, epoch types.EpochInfo) {
	store := ctx.KVStore(k.storeKey)
	value, err := proto.Marshal(&epoch)
	if err != nil {
		panic(err)
	}
	store.Set(append(types.KeyPrefixEpoch, []byte(epoch.Identifier)...), value)
}

// DeleteEpochInfo delete epoch info.
func (k Keeper) DeleteEpochInfo(ctx sdk.Context, identifier string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(append(types.KeyPrefixEpoch, []byte(identifier)...))
}

// IterateEpochInfo iterate through epochs.
func (k Keeper) IterateEpochInfo(ctx sdk.Context, fn func(index int64, epochInfo types.EpochInfo) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, types.KeyPrefixEpoch)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		epoch := types.EpochInfo{}
		err := proto.Unmarshal(iterator.Value(), &epoch)
		if err != nil {
			panic(err)
		}
		stop := fn(i, epoch)

		if stop {
			break
		}
		i++
	}
}

// AllEpochInfos iterate through epochs to return all epochs info.
func (k Keeper) AllEpochInfos(ctx sdk.Context) []types.EpochInfo {
	var epochs []types.EpochInfo
	k.IterateEpochInfo(ctx, func(index int64, epochInfo types.EpochInfo) (stop bool) {
		epochs = append(epochs, epochInfo)
		return false
	})
	return epochs
}
