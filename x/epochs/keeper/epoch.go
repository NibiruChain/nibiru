package keeper

import (
	"errors"
	"fmt"
	"github.com/NibiruChain/collections"
	"time"

	"github.com/NibiruChain/nibiru/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetEpochInfo returns epoch info by identifier.
func (k Keeper) GetEpochInfo(ctx sdk.Context, identifier string) types.EpochInfo {
	epoch, err := k.Epochs.Get(ctx, identifier)
	if err != nil {
		panic(err)
	}

	return epoch
}

// EpochExists checks if the epoch exists
func (k Keeper) EpochExists(ctx sdk.Context, identifier string) bool {
	_, err := k.Epochs.Get(ctx, identifier)
	if errors.Is(err, collections.ErrNotFound) {
		return false
	} else if err != nil {
		panic(err)
	}

	return true
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
	k.Epochs.Insert(ctx, epoch.Identifier, epoch)
}

// DeleteEpochInfo delete epoch info.
func (k Keeper) DeleteEpochInfo(ctx sdk.Context, identifier string) {
	err := k.Epochs.Delete(ctx, identifier)
	if err != nil {
		panic(err)
	}
}

// IterateEpochInfo iterate through epochs.
func (k Keeper) IterateEpochInfo(
	ctx sdk.Context,
	fn func(index int64, epochInfo types.EpochInfo) (stop bool),
) {
	iterate := k.Epochs.Iterate(ctx, collections.Range[string]{})
	i := int64(0)

	for ; iterate.Valid(); iterate.Next() {
		epoch := iterate.Value()
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
