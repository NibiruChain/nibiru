package keeper

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"

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
	return getTraderDiscountOrRebate(
		ctx,
		k.GlobalDiscounts,
		k.TraderDiscounts,
		trader,
		volume,
	)
}

// GetTraderRebate will check if the trader has a custom rebate for the given volume.
// If it does not have a custom rebate, it will return the global rebate for the given volume.
// The rebate is the nearest left entry of the trader volume.
func (k Keeper) GetTraderRebate(ctx sdk.Context, trader sdk.AccAddress, volume math.Int) (math.LegacyDec, bool) {
	return getTraderDiscountOrRebate(
		ctx,
		k.GlobalRebates,
		k.TraderRebates,
		trader,
		volume,
	)
}

// applyDiscountAndRebate applies the discount and rebate to the given exchange fee ratio.
// It updates the current epoch trader volume.
// It returns the new exchange fee ratio.
func (k Keeper) applyDiscountAndRebate(
	ctx sdk.Context,
	pair asset.Pair,
	trader sdk.AccAddress,
	positionNotional math.LegacyDec,
	feeRatio sdk.Dec,
) (sdk.Dec, error) {
	// update user volume
	dnrEpoch, err := k.DnREpoch.Get(ctx)
	if err != nil {
		return feeRatio, err
	}
	k.IncreaseTraderVolume(ctx, dnrEpoch, trader, positionNotional.Abs().TruncateInt())

	// get past epoch volume
	pastVolume := k.GetTraderVolumeLastEpoch(ctx, dnrEpoch, trader)
	// if the trader has no volume for the last epoch, we return the provided fee ratios.
	if pastVolume.IsZero() {
		return feeRatio, nil
	}

	// apply rebates
	err = k.applyRebates(ctx, trader, pair, pastVolume, positionNotional)
	if err != nil {
		return feeRatio, err
	}

	// try to apply discount
	discountedFeeRatio, hasDiscount := k.GetTraderDiscount(ctx, trader, pastVolume)
	// if the trader does not have any discount, we return the provided fee ratios.
	if !hasDiscount {
		return feeRatio, nil
	}
	// return discounted fee ratios
	return discountedFeeRatio, nil
}

// applyRebates calculates the rebate based on the last epoch volume.
// then estimates how much the rebate would be in the quote asset.
// it converts the quote asset to the bond denom, and mints it to the trader.
func (k Keeper) applyRebates(
	ctx sdk.Context,
	trader sdk.AccAddress,
	pair asset.Pair,
	volume math.Int,
	positionNotional math.LegacyDec,
) error {
	// we calculate how much the rebate is, if zero. we skip.
	rebate, hasRebate := k.GetTraderRebate(ctx, trader, volume)
	if !hasRebate {
		return nil
	}
	rebateCoins := sdk.NewCoin(pair.QuoteDenom(), positionNotional.Abs().Mul(rebate).TruncateInt())
	bondRebateCoin, err := k.coinToBondEquivalent(ctx, rebateCoins)
	if err != nil {
		// log and skip.
		ctx.Logger().Error("error converting rebate to bond equivalent", "error", err)
		return nil
	}
	bondRebateCoins := sdk.NewCoins(bondRebateCoin)
	err = k.BankKeeper.MintCoins(ctx, types.ModuleName, bondRebateCoins)
	if err != nil {
		return err
	}
	err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, trader, bondRebateCoins)
	if err != nil {
		return err
	}
	return nil
}

// coinToBondEquivalent returns the amount of bond coins required to buy the given amount of denom coins.
func (k Keeper) coinToBondEquivalent(ctx sdk.Context, denomCoin sdk.Coin) (coin sdk.Coin, err error) {
	// find the bond denom price.
	bondDenom := k.StakingKeeper.BondDenom(ctx)
	// the amount required to buy itself is itself.
	if bondDenom == denomCoin.Denom {
		return denomCoin, nil
	}
	// find the bond denom price.
	bondDenomPrice, err := k.OracleKeeper.GetExchangeRate(ctx, asset.NewPair(bondDenom, denoms.USD))
	if err != nil {
		return coin, err
	}
	// find the denom price.
	denomPrice, err := k.OracleKeeper.GetExchangeRate(ctx, asset.NewPair(denomCoin.Denom, denoms.USD))
	if err != nil {
		return coin, err
	}
	// amt * denomPrice / bondDenomPrice
	bondEquivalentAmount := denomCoin.Amount.ToLegacyDec().Mul(denomPrice).Quo(bondDenomPrice).TruncateInt()
	return sdk.NewCoin(bondDenom, bondEquivalentAmount), nil
}

// getTraderDiscountOrRebate will check if the trader has a custom discount or rebate for the given volume.
// It takes the globalMap which either represents the global discounts or rebates.
// It takes the tradersMap which either represents the custom discounts or rebates.
// It takes the trader and its volume to provide the discount or rebate ratio.
// Returns the discount or rebate ratio and a boolean indicating if the trader has a discount or rebate.
func getTraderDiscountOrRebate(
	ctx sdk.Context,
	globalMap collections.Map[math.Int, math.LegacyDec],
	tradersMap collections.Map[collections.Pair[sdk.AccAddress, math.Int], math.LegacyDec],
	trader sdk.AccAddress,
	volume math.Int,
) (math.LegacyDec, bool) {
	// we try to see if the trader has a custom discount or rebate.
	traderRng := collections.PairRange[sdk.AccAddress, math.Int]{}.
		Prefix(trader).
		EndInclusive(volume).
		Descending()

	custom := tradersMap.Iterate(ctx, traderRng)
	defer custom.Close()

	if custom.Valid() {
		return custom.Value(), true
	}

	// if it does not have a custom discount or rebate we try with global ones
	globalRng := collections.Range[math.Int]{}.
		EndInclusive(volume).
		Descending()

	global := globalMap.Iterate(ctx, globalRng)
	defer global.Close()

	if global.Valid() {
		return global.Value(), true
	}
	return math.LegacyZeroDec(), false
}
