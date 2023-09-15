package keeper

import (
	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DnRGCFrequency is the frequency at which the DnR garbage collector runs.
const DnRGCFrequency = 1000

// IntValueEncoder instructs collections on how to encode a math.Int.
// TODO: move to collections.
var IntValueEncoder collections.ValueEncoder[math.Int] = intValueEncoder{}

type intValueEncoder struct{}

func (i intValueEncoder) Encode(value math.Int) []byte {
	v, err := value.Marshal()
	if err != nil {
		panic(err)
	}
	return v
}

func (i intValueEncoder) Decode(b []byte) math.Int {
	var v math.Int
	err := v.Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return v
}

func (i intValueEncoder) Stringify(value math.Int) string {
	return value.String()
}

func (i intValueEncoder) Name() string {
	return "math.Int"
}

// IncreaseTraderVolume adds the volume to the user's volume for the current epoch.
func (k Keeper) IncreaseTraderVolume(ctx sdk.Context, currentEpoch uint64, user sdk.AccAddress, volume math.Int) {
	currentVolume := k.TraderVolumes.GetOr(ctx, collections.Join(user, currentEpoch), math.ZeroInt())
	newVolume := currentVolume.Add(volume)
	k.TraderVolumes.Insert(ctx, collections.Join(user, currentEpoch), newVolume)
	k.gcUserVolume(ctx, user, currentEpoch)
}

// gcUserVolume deletes the un-needed user epochs.
func (k Keeper) gcUserVolume(ctx sdk.Context, user sdk.AccAddress, currentEpoch uint64) {
	// we do not want to do this always.
	if ctx.BlockHeight()%DnRGCFrequency != 0 {
		return
	}

	rng := collections.PairRange[sdk.AccAddress, uint64]{}.
		Prefix(user).                   // only iterate over the user's epochs.
		EndExclusive(currentEpoch - 1). // we want to preserve current and last epoch, as it's needed to compute DnR rewards.
		Descending()

	for _, key := range k.TraderVolumes.Iterate(ctx, rng).Keys() {
		err := k.TraderVolumes.Delete(ctx, key)
		if err != nil {
			panic(err)
		}
	}
}

// GetUserVolumeLastEpoch returns the user's volume for the last epoch.
// Returns zero if the user has no volume for the last epoch.
func (k Keeper) GetUserVolumeLastEpoch(ctx sdk.Context, user sdk.AccAddress) math.Int {
	currentEpoch, err := k.DnREpoch.Get(ctx)
	if err != nil {
		// a DnR epoch should always exist, otherwise it means the chain was not initialized properly.
		panic(err)
	}
	// if it's the first epoch, we do not have any user volume.
	if currentEpoch == 0 {
		return math.ZeroInt()
	}
	// return the user's volume for the last epoch, or zero.
	return k.TraderVolumes.GetOr(ctx, collections.Join(user, currentEpoch-1), math.ZeroInt())
}
