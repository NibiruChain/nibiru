package keeper

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

// DnRGCFrequency is the frequency at which the DnR garbage collector runs.
const DnRGCFrequency = 1000

// IntValueEncoder instructs collections on how to encode a math.Int as a value.
// TODO: move to collections.
var IntValueEncoder collections.ValueEncoder[math.Int] = intValueEncoder{}

// IntKeyEncoder instructs collections on how to encode a math.Int as a key.
// NOTE: unsafe to use as the first part of a composite key.
var IntKeyEncoder collections.KeyEncoder[math.Int] = intKeyEncoder{}

type intValueEncoder struct{}

func (intValueEncoder) Encode(value math.Int) []byte {
	return IntKeyEncoder.Encode(value)
}

func (intValueEncoder) Decode(b []byte) math.Int {
	_, got := IntKeyEncoder.Decode(b)
	return got
}

func (intValueEncoder) Stringify(value math.Int) string {
	return IntKeyEncoder.Stringify(value)
}

func (intValueEncoder) Name() string {
	return "math.Int"
}

type intKeyEncoder struct{}

const maxIntKeyLen = math.MaxBitLen / 8

func (intKeyEncoder) Encode(key math.Int) []byte {
	if key.IsNil() {
		panic("cannot encode invalid math.Int")
	}
	if key.IsNegative() {
		panic("cannot encode negative math.Int")
	}
	i := key.BigInt()

	be := i.Bytes()
	padded := make([]byte, maxIntKeyLen)
	copy(padded[maxIntKeyLen-len(be):], be)
	return padded
}

func (intKeyEncoder) Decode(b []byte) (int, math.Int) {
	if len(b) != maxIntKeyLen {
		panic("invalid key length")
	}
	i := new(big.Int).SetBytes(b)
	return maxIntKeyLen, math.NewIntFromBigInt(i)
}

func (intKeyEncoder) Stringify(key math.Int) string { return key.String() }

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

// GetTraderVolumeLastEpoch returns the user's volume for the last epoch.
// Returns zero if the user has no volume for the last epoch.
func (k Keeper) GetTraderVolumeLastEpoch(ctx sdk.Context, currentEpoch uint64, user sdk.AccAddress) math.Int {
	// if it's the first epoch, we do not have any user volume.
	if currentEpoch == 0 {
		return math.ZeroInt()
	}
	// return the user's volume for the last epoch, or zero.
	return k.TraderVolumes.GetOr(ctx, collections.Join(user, currentEpoch-1), math.ZeroInt())
}

// GetTraderDiscount will check if the trader has a custom discount for the given volume.
// If it does not have a custom discount, it will return the global discount for the given volume.
// The discount is the nearest left entry of the trader volume.
func (k Keeper) GetTraderDiscount(ctx sdk.Context, trader sdk.AccAddress, volume math.Int) (math.LegacyDec, bool) {
	// we try to see if the trader has a custom discount.
	customDiscountRng := collections.PairRange[sdk.AccAddress, math.Int]{}.
		Prefix(trader).
		EndInclusive(volume).
		Descending()

	customDiscount := k.TraderDiscounts.Iterate(ctx, customDiscountRng)
	defer customDiscount.Close()

	if customDiscount.Valid() {
		return customDiscount.Value(), true
	}

	// if it does not have a custom discount we try with global ones
	globalDiscountRng := collections.Range[math.Int]{}.
		EndInclusive(volume).
		Descending()

	globalDiscounts := k.GlobalDiscounts.Iterate(ctx, globalDiscountRng)
	defer globalDiscounts.Close()

	if globalDiscounts.Valid() {
		return globalDiscounts.Value(), true
	}
	return math.LegacyZeroDec(), false
}

// applyDiscountAndRebate applies the discount and rebate to the given exchange fee ratio.
// It updates the current epoch trader volume.
// It returns the new exchange fee ratio.
func (k Keeper) applyDiscountAndRebate(
	ctx sdk.Context,
	_ asset.Pair,
	trader sdk.AccAddress,
	positionNotional math.LegacyDec,
	exchangeFeeRatio sdk.Dec,
) (exFeeRatio sdk.Dec, err error) {
	// update user volume
	dnrEpoch, err := k.DnREpoch.Get(ctx)
	if err != nil {
		return
	}
	k.IncreaseTraderVolume(ctx, dnrEpoch, trader, positionNotional.Abs().TruncateInt())

	// get past epoch volume
	pastVolume := k.GetTraderVolumeLastEpoch(ctx, dnrEpoch, trader)
	// if the trader has no volume for the last epoch, we return the provided fee ratios.
	if pastVolume.IsZero() {
		return exchangeFeeRatio, nil
	}

	// try to apply discount
	discount, hasDiscount := k.GetTraderDiscount(ctx, trader, pastVolume)
	// if the trader does not have any discount, we return the provided fee ratios.
	if !hasDiscount {
		return exchangeFeeRatio, nil
	}
	// return discounted fee ratios
	return discount, nil
}
