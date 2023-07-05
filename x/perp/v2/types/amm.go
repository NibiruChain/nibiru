package types

import (
	fmt "fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
)

func (amm AMM) Validate() error {
	if amm.BaseReserve.IsNil() || amm.BaseReserve.IsZero() {
		return fmt.Errorf("base reserves must be > 0")
	}

	if amm.QuoteReserve.IsNil() || amm.QuoteReserve.IsZero() {
		return fmt.Errorf("quote reserves must be > 0")
	}

	if amm.PriceMultiplier.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init price multiplier must be > 0")
	}

	if amm.SqrtDepth.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init sqrt depth must be > 0")
	}

	computedSqrtDepth, err := amm.ComputeSqrtDepth()
	if err != nil {
		return err
	}

	if !amm.SqrtDepth.IsPositive() {
		return ErrLiquidityDepth.Wrap(
			"liq depth must be positive. pool: " + amm.String())
	}

	if !amm.SqrtDepth.Sub(computedSqrtDepth).Abs().LTE(sdk.NewDec(1)) {
		return ErrLiquidityDepth.Wrap(
			"computed sqrt and current sqrt are mismatched. pool: " + amm.String())
	}

	return nil
}

// returns the amount of quote reserve equivalent to the amount of quote asset given
func (amm AMM) FromQuoteAssetToReserve(quoteAsset sdkmath.Uint) sdkmath.Uint {
	return sdkmath.NewUintFromBigInt(amm.PriceMultiplier.QuoInt64(int64(quoteAsset.Uint64())).BigInt())
}

// returns the amount of quote asset equivalent to the amount of quote reserve given
func (amm AMM) FromQuoteReserveToAsset(quoteReserve sdkmath.Uint) sdkmath.Uint {
	return sdkmath.NewUintFromBigInt(amm.PriceMultiplier.MulInt64(int64(quoteReserve.Uint64())).BigInt())
}

// Returns the amount of base reserve equivalent to the amount of quote reserve given
//
// args:
// - quoteReserveAmt: the amount of quote reserve before the trade, must be positive
// - dir: the direction of the trade
//
// returns:
// - baseReserveDelta: the amount of base reserve after the trade, always positive
// - err: error
//
// Throws an error if the final quote reserve is not positive
func (amm AMM) GetBaseReserveAmt(
	quoteReserveAmt sdkmath.Uint, // unsigned
	dir Direction,
) (baseReserveDelta sdkmath.Uint, err error) {
	invariant := amm.QuoteReserve.Mul(amm.BaseReserve) // x * y = k

	var quoteReservesAfter sdkmath.Uint
	if dir == Direction_LONG {
		quoteReservesAfter = amm.QuoteReserve.Add(quoteReserveAmt)
	} else {
		if quoteReserveAmt.GT(amm.QuoteReserve) {
			return sdkmath.ZeroUint(), ErrQuoteReserveAtZero.Wrapf(
				"quote reserves below zero after trying to swap %s quote reserves. there are only %s quote reserves in the pool",
				quoteReserveAmt.String(), amm.QuoteReserve.String(),
			)
		}

		quoteReservesAfter = amm.QuoteReserve.Sub(quoteReserveAmt)
	}

	baseReservesAfter := invariant.Quo(quoteReservesAfter)

	if baseReservesAfter.GT(amm.BaseReserve) {
		return baseReservesAfter.Sub(amm.BaseReserve), nil
	}

	return amm.BaseReserve.Sub(baseReservesAfter), nil
}

// returns the amount of quote reserve equivalent to the amount of base asset given
//
// args:
// - baseReserveAmt: the amount of base reserves to trade, must be positive
// - dir: the direction of the trade
//
// returns:
// - quoteReserveDelta: the amount of quote reserve after the trade
// - err: error
//
// NOTE: quoteReserveDelta is always positive
func (amm AMM) GetQuoteReserveAmt(
	baseReserveAmt sdkmath.Uint,
	dir Direction,
) (quoteReserveDelta sdkmath.Uint, err error) {
	invariant := amm.QuoteReserve.Mul(amm.BaseReserve) // x * y = k

	var baseReservesAfter sdkmath.Uint
	if dir == Direction_LONG {
		if baseReserveAmt.GT(amm.BaseReserve) {
			return sdkmath.ZeroUint(), ErrBaseReserveAtZero.Wrapf(
				"base reserves below zero after trying to swap %s base reserves. there are only %s base reserves in the pool",
				baseReserveAmt.String(), amm.BaseReserve.String(),
			)
		}

		baseReservesAfter = amm.BaseReserve.Sub(baseReserveAmt)
	} else {
		baseReservesAfter = amm.BaseReserve.Add(baseReserveAmt)
	}

	quoteReservesAfter := invariant.Quo(baseReservesAfter)

	if quoteReservesAfter.GT(amm.QuoteReserve) {
		return quoteReservesAfter.Sub(amm.QuoteReserve), nil
	}

	return amm.QuoteReserve.Sub(quoteReservesAfter), nil
}

// Returns the instantaneous mark price of the trading pair
func (amm AMM) MarkPrice() sdk.Dec {
	if amm.BaseReserve.IsNil() || amm.BaseReserve.IsZero() ||
		amm.QuoteReserve.IsNil() || amm.QuoteReserve.IsZero() {
		return sdk.ZeroDec()
	}

	return amm.PriceMultiplier.MulInt(
		sdkmath.NewIntFromUint64(
			amm.QuoteReserve.Quo(amm.BaseReserve).Uint64(),
		),
	)
}

// Returns the sqrt k of the reserves
func (amm AMM) ComputeSqrtDepth() (sqrtDepth sdk.Dec, err error) {
	mul := new(big.Int).Mul(amm.BaseReserve.BigInt(), amm.BaseReserve.BigInt())

	chopped := common.ChopPrecisionAndRound(mul)
	if chopped.BitLen() > common.MaxDecBitLen {
		return sdk.Dec{}, ErrLiquidityDepthOverflow
	}

	liqDepth := sdkmath.LegacyNewDecFromBigInt(amm.QuoteReserve.Mul(amm.BaseReserve).BigInt())
	return common.SqrtDec(liqDepth)
}

func (amm *AMM) WithPair(pair asset.Pair) *AMM {
	amm.Pair = pair
	return amm
}

func (amm *AMM) WithBaseReserve(baseReserve sdkmath.Uint) *AMM {
	amm.BaseReserve = baseReserve
	return amm
}

func (amm *AMM) WithQuoteReserve(quoteReserve sdkmath.Uint) *AMM {
	amm.QuoteReserve = quoteReserve
	return amm
}

func (amm *AMM) WithPriceMultiplier(priceMultiplier sdk.Dec) *AMM {
	amm.PriceMultiplier = priceMultiplier
	return amm
}

func (amm *AMM) WithTotalLong(totalLong sdkmath.Uint) *AMM {
	amm.TotalLong = totalLong
	return amm
}

func (amm *AMM) WithTotalShort(totalShort sdkmath.Uint) *AMM {
	amm.TotalShort = totalShort
	return amm
}

func (amm *AMM) WithSqrtDepth(sqrtDepth sdk.Dec) *AMM {
	amm.SqrtDepth = sqrtDepth
	return amm
}

// SwapQuoteAsset swaps base asset for quote asset
//
// args:
// - quoteAssetAmt: amount of base asset to swap, must be positive
// - dir: direction the user takes
//
// returns:
// - baseAssetDelta: amount of base asset received
// - err: error
//
// NOTE: baseAssetDelta is always positive
func (amm *AMM) SwapQuoteAsset(
	quoteAssetAmt sdkmath.Uint, // unsigned
	dir Direction,
) (baseAssetDelta sdkmath.Uint, err error) {
	quoteReserveAmt := amm.FromQuoteAssetToReserve(quoteAssetAmt)
	baseReserveDelta, err := amm.GetBaseReserveAmt(quoteReserveAmt, dir)
	if err != nil {
		return sdkmath.ZeroUint(), err
	}

	if dir == Direction_LONG {
		amm.QuoteReserve = amm.QuoteReserve.Add(quoteReserveAmt)
		amm.BaseReserve = amm.BaseReserve.Sub(baseReserveDelta)
		amm.TotalLong = amm.TotalLong.Add(baseReserveDelta)
	} else if dir == Direction_SHORT {
		amm.QuoteReserve = amm.QuoteReserve.Sub(quoteReserveAmt)
		amm.BaseReserve = amm.BaseReserve.Add(baseReserveDelta)
		amm.TotalShort = amm.TotalShort.Add(baseReserveDelta)
	}

	return baseReserveDelta, nil
}

// SwapBaseAsset swaps base asset for quote asset
//
// args:
//   - baseAssetAmt: amount of base asset to swap, must be positive
//   - dir: direction of swap
//
// returns:
//   - quoteAssetDelta: amount of quote asset received
//   - err: error if any
//
// Note: quoteAssetDelta is always positive
func (amm *AMM) SwapBaseAsset(baseAssetAmt sdkmath.Uint, dir Direction) (quoteAssetDelta sdkmath.Uint, err error) {
	quoteReserveDelta, err := amm.GetQuoteReserveAmt(baseAssetAmt, dir)
	if err != nil {
		return sdkmath.ZeroUint(), err
	}

	if dir == Direction_LONG {
		amm.QuoteReserve = amm.QuoteReserve.Add(quoteReserveDelta)
		amm.BaseReserve = amm.BaseReserve.Sub(baseAssetAmt)
		amm.TotalLong = amm.TotalLong.Add(baseAssetAmt)
	} else if dir == Direction_SHORT {
		amm.QuoteReserve = amm.QuoteReserve.Sub(quoteReserveDelta)
		amm.BaseReserve = amm.BaseReserve.Add(baseAssetAmt)
		amm.TotalShort = amm.TotalShort.Add(baseAssetAmt)
	}

	return amm.FromQuoteReserveToAsset(quoteReserveDelta), nil
}

/*
Bias returns the bias of the market in the base asset. It's the net amount of base assets for longs minus the net
amount of base assets for shorts.
*/
func (amm *AMM) Bias() (bias sdkmath.Int) {
	return sdkmath.NewIntFromUint64(amm.TotalLong.Uint64()).Sub(sdkmath.NewIntFromUint64(amm.TotalShort.Uint64()))
}

/*
CalcRepegCost provides the cost of re-pegging the pool to a new candidate peg multiplier.
*/
func (amm AMM) CalcRepegCost(newPriceMultiplier sdk.Dec) (cost sdkmath.Int, err error) {
	if !newPriceMultiplier.IsPositive() {
		return sdkmath.Int{}, ErrNonPositivePegMultiplier
	}

	bias := amm.Bias()

	if bias.IsZero() {
		return sdk.ZeroInt(), nil
	}

	var dir Direction
	if bias.IsPositive() {
		dir = Direction_SHORT
	} else {
		dir = Direction_LONG
	}

	biasInQuoteReserve, err := amm.GetQuoteReserveAmt(sdkmath.NewUintFromBigInt(bias.Abs().BigInt()), dir)
	if err != nil {
		return sdkmath.Int{}, err
	}

	costDec := (newPriceMultiplier.Sub(amm.PriceMultiplier)).MulInt64(int64(biasInQuoteReserve.Uint64()))

	if bias.IsNegative() {
		costDec = costDec.Neg()
	}

	return costDec.Ceil().TruncateInt(), nil
}

// returns the amount of quote assets the amm has to pay out if all longs and shorts close out their positions
// positive value means the amm has to pay out quote assets
// negative value means the amm has to receive quote assets
func (amm AMM) GetMarketValue() (sdkmath.Int, error) {
	bias := amm.Bias()

	if bias.IsZero() {
		return sdkmath.ZeroInt(), nil
	}

	var dir Direction
	if bias.IsPositive() {
		dir = Direction_SHORT
	} else {
		dir = Direction_LONG
	}

	marketValueInReserves, err := amm.GetQuoteReserveAmt(sdkmath.NewUintFromBigInt(bias.Abs().BigInt()), dir)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	marketValueInAssets := amm.FromQuoteReserveToAsset(marketValueInReserves)

	if bias.IsNegative() {
		return sdkmath.NewIntFromUint64(marketValueInAssets.Uint64()).Neg(), nil
	}

	return sdkmath.NewIntFromUint64(marketValueInAssets.Uint64()), nil
}

/*
CalcUpdateSwapInvariantCost returns the cost of updating the invariant of the pool
*/
func (amm AMM) CalcUpdateSwapInvariantCost(newSwapInvariant sdk.Dec) (sdkmath.Int, error) {
	if newSwapInvariant.IsNil() {
		return sdkmath.Int{}, ErrNilSwapInvariant
	}

	if !newSwapInvariant.IsPositive() {
		return sdkmath.Int{}, ErrNegativeSwapInvariant
	}

	marketValueBefore, err := amm.GetMarketValue()
	if err != nil {
		return sdkmath.Int{}, err
	}

	err = amm.UpdateSwapInvariant(newSwapInvariant)
	if err != nil {
		return sdkmath.Int{}, err
	}

	marketValueAfter, err := amm.GetMarketValue()
	if err != nil {
		return sdkmath.Int{}, err
	}

	cost := marketValueAfter.Sub(marketValueBefore)

	return cost.Ceil().TruncateInt(), nil
}

// UpdateSwapInvariant updates the swap invariant of the amm
func (amm *AMM) UpdateSwapInvariant(newSwapInvariant sdk.Dec) (err error) {
	// k = x * y
	// newK = (cx) * (cy) = c^2 xy = c^2 k
	// newPrice = (c y) / (c x) = y / x = price | unchanged price
	newSqrtDepth := common.MustSqrtDec(newSwapInvariant)
	multiplier := newSqrtDepth.Quo(amm.SqrtDepth)

	// Change the swap invariant while holding price constant.
	// Multiplying by the same factor to both of the reserves won't affect price.
	amm.SqrtDepth = newSqrtDepth
	amm.BaseReserve = amm.BaseReserve.Mul(multiplier)
	amm.QuoteReserve = amm.QuoteReserve.Mul(multiplier)

	return amm.Validate()
}
