package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

// GetEpochInfo returns epoch info by identifier.
func (k Keeper) GetEpochInfo(ctx sdk.Context, identifier string) (epoch types.EpochInfo, err error) {
	epoch, err = k.Epochs.Get(ctx, identifier)
	if err != nil {
		err = fmt.Errorf("epoch with identifier %s not found", identifier)
		return
	}

	return
}

// EpochExists checks if the epoch exists
func (k Keeper) EpochExists(ctx sdk.Context, identifier string) bool {
	_, err := k.Epochs.Get(ctx, identifier)
	return err == nil
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

	k.Epochs.Set(ctx, epoch.Identifier, epoch)

	return nil
}

// DeleteEpochInfo delete epoch info.
func (k Keeper) DeleteEpochInfo(ctx sdk.Context, identifier string) (err error) {
	err = k.Epochs.Remove(ctx, identifier)
	return
}

// IterateEpochInfo iterate through epochs.
func (k Keeper) IterateEpochInfo(
	ctx sdk.Context,
	fn func(index int64, epochInfo types.EpochInfo) (stop bool),
) {
	iterate, _ := k.Epochs.Iterate(ctx, &collections.Range[string]{})
	i := int64(0)

	for ; iterate.Valid(); iterate.Next() {
		epoch, _ := iterate.Value()
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
