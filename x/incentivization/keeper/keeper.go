package keeper

import (
	dexkeeper "github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	lockupkeeper "github.com/NibiruChain/nibiru/x/lockup/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"time"
)

const (
	// MinLockupDuration defines the lockup minimum time
	// TODO(mercilex): maybe module param
	MinLockupDuration = 24 * time.Hour
	// MinEpochs defines the minimum number of epochs
	// TODO(mercilex): maybe module param
	MinEpochs int64 = 7
)

type Keeper struct {
	cdc      codec.Codec
	storeKey sdk.StoreKey

	bk bankkeeper.Keeper
	dk dexkeeper.Keeper
	lk lockupkeeper.LockupKeeper
}

func (k Keeper) CreateIncentivizationProgram(
	ctx sdk.Context,
	lpDenom string, minLockupDuration time.Duration, starTime time.Time, epochs int64) (*types.IncentivizationProgram, error) {

	// TODO(mercilex): assert lp denom from dex keeper

	if epochs < MinEpochs {
		return nil, types.ErrEpochsTooLow.Wrapf("%d is lower than minimum allowed %d", epochs, MinEpochs)
	}

	if minLockupDuration < MinLockupDuration {
		return nil, types.ErrMinLockupDurationTooLow.Wrapf("%s is lower than minimum allowed %s", minLockupDuration, MinLockupDuration)
	}

	if ctx.BlockTime().Before(starTime) {
		return nil, types.ErrStartTimeInPast.Wrapf("current time %s, got: %s", ctx.BlockTime(), starTime)
	}

	panic("impl")

}

// Distribute distributes incentivization rewards to accounts
// that meet incentivization program criteria.
func (k Keeper) Distribute(ctx sdk.Context) error {
	panic("impl")
}
