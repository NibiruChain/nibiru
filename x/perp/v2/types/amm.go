package types

import (
	"math/big"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
)

func (amm AMM) Validate() error {
	if amm.BaseReserve.LTE(sdkmath.LegacyZeroDec()) {
		return ErrAmmBaseSupplyNonpositive
	}

	if amm.QuoteReserve.LTE(sdkmath.LegacyZeroDec()) {
		return ErrAmmQuoteSupplyNonpositive
	}

	if amm.PriceMultiplier.LTE(sdkmath.LegacyZeroDec()) {
		return ErrAmmNonPositivePegMult
	}

	if amm.SqrtDepth.LTE(sdkmath.LegacyZeroDec()) {
		return ErrAmmNonPositiveSwapInvariant
	}

	computedSqrtDepth, err := amm.ComputeSqrtDepth()
	if err != nil {
		return err
	}

	if !amm.SqrtDepth.Sub(computedSqrtDepth).Abs().LTE(sdkmath.LegacyOneDec()) {
		return ErrLiquidityDepth.Wrap(
			"computed sqrt and current sqrt are mismatched. pool: " + amm.String())
	}

	// Short positions borrow base asset, so if the base reserve is below the
	// quote reserve swapped to base, then the shorts can't close positions.
	_, err = amm.SwapBaseAsset(amm.TotalShort, Direction_LONG)
	if err != nil {
		return sdkerrors.Wrapf(ErrAmmBaseBorrowedTooHigh, "Base amount error, short exceed total base supply: %s", err.Error())
	}

	return nil
}

// ComputeSettlementPrice computes the uniform settlement price for the current AMM.
//
// Returns:
//   - price: uniform settlement price from several batched trades. In this case,
//     "price" is the result of closing all positions together and giving
//     all traders the same price.
//   - newAmm: The AMM that results from closing all positions together. Note that
//     this should have a bias, or skew, of 0.
//   - err: Errors if it's impossible to swap away the open interest bias.
func (amm AMM) ComputeSettlementPrice() (sdkmath.LegacyDec, AMM, error) {
	// bias: open interest (base) skew in the AMM.
	bias := amm.Bias()
	if bias.IsZero() {
		return amm.InstMarkPrice(), amm, nil
	}

	var dir Direction
	if bias.IsPositive() {
		dir = Direction_SHORT
	} else {
		dir = Direction_LONG
	}

	quoteAssetDelta, err := amm.SwapBaseAsset(bias.Abs(), dir)
	if err != nil {
		return sdkmath.LegacyDec{}, AMM{}, err
	}
	price := quoteAssetDelta.Abs().Quo(bias.Abs())

	return price, amm, err
}

// QuoteReserveToAsset converts quote reserves to assets\
func (amm AMM) QuoteReserveToAsset(quoteReserve sdkmath.LegacyDec) sdkmath.LegacyDec {
	return QuoteReserveToAsset(quoteReserve, amm.PriceMultiplier)
}

// QuoteAssetToReserve converts quote assets to reserves
func (amm AMM) QuoteAssetToReserve(quoteAssets sdkmath.LegacyDec) sdkmath.LegacyDec {
	return QuoteAssetToReserve(quoteAssets, amm.PriceMultiplier)
}

// QuoteAssetToReserve converts "quote assets" to "quote reserves". In this
// convention, "assets" are liquid funds that change hands, whereas reserves
// are simply a number field on the DAMM. The reason for this distinction is to
// account for the AMM.PriceMultiplier.
func QuoteAssetToReserve(quoteAsset, priceMult sdkmath.LegacyDec) sdkmath.LegacyDec {
	return quoteAsset.Quo(priceMult)
}

// QuoteReserveToAsset converts "quote reserves" to "quote assets". In this
// convention, "assets" are liquid funds that change hands, whereas reserves
// are simply a number field on the DAMM. The reason for this distinction is to
// account for the AMM.PriceMultiplier.
func QuoteReserveToAsset(quoteReserve, priceMult sdkmath.LegacyDec) sdkmath.LegacyDec {
	return quoteReserve.Mul(priceMult)
}

// GetBaseReserveAmt Returns the amount of base reserve equivalent to the amount of quote reserve given
//
// args:
// - quoteReserveAmt: the amount of quote reserve before the trade, must be positive
// - dir: the direction of the trade
//
// returns:
// - baseReserveDelta: the amount of base reserve after the trade, unsigned
// - err: error
//
// NOTE: baseReserveDelta is always positive
// Throws an error if input quoteReserveAmt is negative, or if the final quote reserve is not positive
func (amm AMM) GetBaseReserveAmt(
	quoteReserveAmt sdkmath.LegacyDec, // unsigned
	dir Direction,
) (baseReserveDelta sdkmath.LegacyDec, err error) {
	if quoteReserveAmt.IsNegative() {
		return sdkmath.LegacyDec{}, ErrInputQuoteAmtNegative
	}

	invariant := amm.QuoteReserve.Mul(amm.BaseReserve) // x * y = k

	var quoteReservesAfter sdkmath.LegacyDec
	if dir == Direction_LONG {
		quoteReservesAfter = amm.QuoteReserve.Add(quoteReserveAmt)
	} else {
		quoteReservesAfter = amm.QuoteReserve.Sub(quoteReserveAmt)
	}

	if !quoteReservesAfter.IsPositive() {
		return sdkmath.LegacyDec{}, ErrAmmNonpositiveReserves
	}

	baseReservesAfter := invariant.Quo(quoteReservesAfter)
	baseReserveDelta = baseReservesAfter.Sub(amm.BaseReserve).Abs()

	return baseReserveDelta, nil
}

// GetQuoteReserveAmt returns the amount of quote reserve equivalent to the amount of base asset given
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
	baseReserveAmt sdkmath.LegacyDec,
	dir Direction,
) (quoteReserveDelta sdkmath.LegacyDec, err error) {
	if baseReserveAmt.IsNegative() {
		return sdkmath.LegacyDec{}, ErrInputBaseAmtNegative
	}
	if baseReserveAmt.IsZero() {
		return sdkmath.LegacyZeroDec(), nil
	}

	invariant := amm.QuoteReserve.Mul(amm.BaseReserve) // x * y = k

	var baseReservesAfter sdkmath.LegacyDec
	if dir == Direction_LONG {
		baseReservesAfter = amm.BaseReserve.Sub(baseReserveAmt)
	} else {
		baseReservesAfter = amm.BaseReserve.Add(baseReserveAmt)
	}

	if !baseReservesAfter.IsPositive() {
		return sdkmath.LegacyDec{}, ErrAmmNonpositiveReserves.Wrapf(
			"base assets below zero (%s) after trying to swap %s base assets",
			baseReservesAfter.String(),
			baseReserveAmt.String(),
		)
	}

	quoteReservesAfter := invariant.Quo(baseReservesAfter)
	quoteReserveDelta = quoteReservesAfter.Sub(amm.QuoteReserve).Abs()

	return quoteReserveDelta, nil
}

// InstMarkPrice returns the instantaneous mark price of the trading pair.
// This is the price if the AMM has zero slippage, or equivalently, if there's
// infinite liquidity depth with the same ratio of reserves.
func (amm AMM) InstMarkPrice() sdkmath.LegacyDec {
	if amm.BaseReserve.IsNil() || amm.BaseReserve.IsZero() ||
		amm.QuoteReserve.IsNil() || amm.QuoteReserve.IsZero() {
		return sdkmath.LegacyZeroDec()
	}

	return amm.QuoteReserve.Quo(amm.BaseReserve).Mul(amm.PriceMultiplier)
}

// ComputeSqrtDepth returns the sqrt of the product of the reserves
func (amm AMM) ComputeSqrtDepth() (sqrtDepth sdkmath.LegacyDec, err error) {
	liqDepthBigInt := new(big.Int).Mul(
		amm.QuoteReserve.BigInt(), amm.BaseReserve.BigInt(),
	)
	chopped := common.ChopPrecisionAndRound(liqDepthBigInt)
	if chopped.BitLen() > common.MaxDecBitLen {
		return sdkmath.LegacyDec{}, ErrAmmLiquidityDepthOverflow
	}
	liqDepth := amm.QuoteReserve.Mul(amm.BaseReserve)
	return common.SqrtDec(liqDepth)
}

func (amm *AMM) WithPair(pair asset.Pair) *AMM {
	amm.Pair = pair
	return amm
}

func (amm *AMM) WithBaseReserve(baseReserve sdkmath.LegacyDec) *AMM {
	amm.BaseReserve = baseReserve
	return amm
}

func (amm *AMM) WithQuoteReserve(quoteReserve sdkmath.LegacyDec) *AMM {
	amm.QuoteReserve = quoteReserve
	return amm
}

func (amm *AMM) WithPriceMultiplier(priceMultiplier sdkmath.LegacyDec) *AMM {
	amm.PriceMultiplier = priceMultiplier
	return amm
}

func (amm *AMM) WithTotalLong(totalLong sdkmath.LegacyDec) *AMM {
	amm.TotalLong = totalLong
	return amm
}

func (amm *AMM) WithTotalShort(totalShort sdkmath.LegacyDec) *AMM {
	amm.TotalShort = totalShort
	return amm
}

func (amm *AMM) WithSqrtDepth(sqrtDepth sdkmath.LegacyDec) *AMM {
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
	quoteAssetAmt sdkmath.LegacyDec, // unsigned
	dir Direction,
) (baseAssetDelta sdkmath.LegacyDec, err error) {
	quoteReserveAmt := QuoteAssetToReserve(quoteAssetAmt, amm.PriceMultiplier)
	baseReserveDelta, err := amm.GetBaseReserveAmt(quoteReserveAmt, dir)
	if err != nil {
		return sdkmath.LegacyDec{}, err
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
//   - quoteAssetDelta: amount of quote asset received. Always positive
//   - err: error if any
func (amm *AMM) SwapBaseAsset(baseAssetAmt sdkmath.LegacyDec, dir Direction) (quoteAssetDelta sdkmath.LegacyDec, err error) {
	quoteReserveDelta, err := amm.GetQuoteReserveAmt(baseAssetAmt, dir)
	if err != nil {
		return sdkmath.LegacyDec{}, err
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

	return amm.QuoteReserveToAsset(quoteReserveDelta), nil
}

// Bias returns the bias, or open interest skew, of the market in the base
// units. Bias is the net amount of long perpetual contracts minus the net
// amount of shorts.
func (amm *AMM) Bias() (bias sdkmath.LegacyDec) {
	return amm.TotalLong.Sub(amm.TotalShort)
}

/*
CalcRepegCost provides the cost of re-pegging the pool to a new candidate peg multiplier.
*/
func (amm AMM) CalcRepegCost(newPriceMultiplier sdkmath.LegacyDec) (cost sdkmath.Int, err error) {
	if !newPriceMultiplier.IsPositive() {
		return sdkmath.Int{}, ErrAmmNonPositivePegMult
	}

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

	biasInQuoteReserve, err := amm.GetQuoteReserveAmt(bias.Abs(), dir)
	if err != nil {
		return sdkmath.Int{}, err
	}

	costDec := biasInQuoteReserve.Mul(newPriceMultiplier.Sub(amm.PriceMultiplier))

	if bias.IsNegative() {
		costDec = costDec.Neg()
	}

	return costDec.Ceil().TruncateInt(), nil
}

// GetMarketValue returns the amount of quote assets the amm has to pay out if all longs and shorts close out their positions
// positive value means the amm has to pay out quote assets
// negative value means the amm has to receive quote assets
func (amm AMM) GetMarketValue() (sdkmath.LegacyDec, error) {
	bias := amm.Bias()

	if bias.IsZero() {
		return sdkmath.LegacyZeroDec(), nil
	}

	var dir Direction
	if bias.IsPositive() {
		dir = Direction_SHORT
	} else {
		dir = Direction_LONG
	}

	marketValueInReserves, err := amm.GetQuoteReserveAmt(bias.Abs(), dir)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}

	if bias.IsNegative() {
		marketValueInReserves = marketValueInReserves.Neg()
	}

	return amm.QuoteReserveToAsset(marketValueInReserves), nil
}

/*
CalcUpdateSwapInvariantCost returns the cost of updating the invariant of the pool
*/
func (amm AMM) CalcUpdateSwapInvariantCost(newSwapInvariant sdkmath.LegacyDec) (sdkmath.Int, error) {
	if newSwapInvariant.IsNil() {
		return sdkmath.Int{}, ErrNilSwapInvariant
	}

	if !newSwapInvariant.IsPositive() {
		return sdkmath.Int{}, ErrAmmNonPositiveSwapInvariant
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
func (amm *AMM) UpdateSwapInvariant(newSwapInvariant sdkmath.LegacyDec) (err error) {
	// k = x * y
	// newK = (cx) * (cy) = c^2 xy = c^2 k
	// newPrice = (c y) / (c x) = y / x = price | unchanged price
	newSqrtDepth, err := common.SqrtDec(newSwapInvariant)
	if err != nil {
		return err
	}

	multiplier := newSqrtDepth.Quo(amm.SqrtDepth)
	updatedBaseReserve := amm.BaseReserve.Mul(multiplier)
	updatedQuoteReserve := amm.QuoteReserve.Mul(multiplier)

	newAmm := AMM{
		BaseReserve:     updatedBaseReserve,
		QuoteReserve:    updatedQuoteReserve,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       newSqrtDepth,
		TotalLong:       amm.TotalLong,
		TotalShort:      amm.TotalShort,
	}
	if err = newAmm.Validate(); err != nil {
		return err
	}

	// Change the swap invariant while holding price constant.
	// Multiplying by the same factor to both of the reserves won't affect price.
	amm.SqrtDepth = newSqrtDepth
	amm.BaseReserve = updatedBaseReserve
	amm.QuoteReserve = updatedQuoteReserve

	return nil
}
