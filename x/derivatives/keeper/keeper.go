package keeper

import (
	"errors"
	"fmt"

	v1 "github.com/MatrixDao/matrix/x/derivatives/types/v1"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

type Keeper struct {
	key sdk.StoreKey
	cdc codec.BinaryCodec

	bk bankkeeper.Keeper
}

type VirtualAMMDirection uint8

const (
	VirtualAMMDirection_AddToAMM = iota
	VirtualAMMDirection_RemoveFromAMM
)

type VirtualAMM interface {
	Pair() string
	QuoteTokenDenom() string
	SwapInput(ctx sdk.Context, ammDir VirtualAMMDirection, inputAmount, minOutputAmount sdk.Int, canOverFluctuationLimit bool) (sdk.Int, error)
	SwapOutput(ctx sdk.Context, dir VirtualAMMDirection, abs sdk.Int, limit sdk.Int) (sdk.Int, error)
	GetOpenInterestNotionalCap(ctx sdk.Context) (sdk.Int, error)
	GetMaxHoldingBaseAsset(ctx sdk.Context) (sdk.Int, error)
	GetOutputTWAP(ctx sdk.Context, dir VirtualAMMDirection, abs sdk.Int) (sdk.Int, error)
	GetOutputPrice(ctx sdk.Context, dir VirtualAMMDirection, abs sdk.Int) (sdk.Int, error)
	GetUnderlyingPrice(ctx sdk.Context) (sdk.Int, error)
	GetSpotPrice(ctx sdk.Context) (sdk.Int, error)
	CalcFee(notional sdk.Int) (sdk.Int, sdk.Int, error)
}

// TODO test: openPosition
func (k Keeper) openPosition(
	ctx sdk.Context, amm VirtualAMM, side v1.Side, trader string,
	quoteAssetAmount, leverage, baseAssetAmountLimit sdk.Int,
) error {
	// TODO(mercilex): missing checks
	params, err := k.Params().Get(ctx)
	if err != nil {
		panic(err)
	}

	position, err := k.Positions().Get(ctx, amm.Pair(), trader)
	positionExists := errors.Is(err, errNotFound)

	var positionResp *v1.PositionResp
	switch {
	// increase position case
	case !positionExists,
		position.Size_.IsPositive() && side == v1.Side_Side_BUY,
		position.Size_.IsNegative() && side == v1.Side_Side_SELL:
		positionResp, err = k.increasePosition(
			ctx, amm, side, trader,
			quoteAssetAmount.Mul(leverage),
			baseAssetAmountLimit,
			leverage)
		if err != nil {
			return err
		}

	// everything else decreases the position
	default:
		positionResp, err = k.openReversePosition(
			ctx, amm, side, trader,
			quoteAssetAmount, leverage, baseAssetAmountLimit, false)
		if err != nil {
			return err
		}
	}

	// update position in state
	k.Positions().Set(ctx, positionResp.Position)

	if !positionExists && !positionResp.Position.Size_.IsZero() {
		marginRatio, err := k.getMarginRatio(ctx, amm, trader)
		if err != nil {
			return err
		}
		if err = requireMoreMarginRatio(marginRatio, params.MaintenanceMarginRatio, true); err != nil {
			// TODO(mercilex): should panic? it's a require
			return err
		}
	}

	if positionResp.BadDebt.IsZero() {
		return fmt.Errorf("bad debt")
	}

	// transfer trader <=> vault
	traderAddr, err := sdk.AccAddressFromBech32(trader) // should fail at validate basic
	if err != nil {
		panic(err)
	}
	switch {
	case positionResp.MarginToVault.IsPositive():
		err = k.bk.SendCoinsFromAccountToModule(
			ctx, traderAddr, v1.VaultModuleAccount,
			sdk.NewCoins(sdk.NewCoin(amm.QuoteTokenDenom(), positionResp.MarginToVault)))
		if err != nil {
			return err
		}
	case positionResp.MarginToVault.IsNegative():
		err = k.bk.SendCoinsFromModuleToAccount(ctx, v1.VaultModuleAccount, traderAddr,
			sdk.NewCoins(sdk.NewCoin(amm.QuoteTokenDenom(), positionResp.MarginToVault.Abs())))
		if err != nil {
			return err
		}
	}

	transferredFee, err := k.transferFee(ctx, traderAddr, amm, positionResp.ExchangedQuoteAssetAmount)
	if err != nil {
		return err
	}

	spotPrice, err := amm.GetSpotPrice(ctx)
	if err != nil {
		return err
	}

	return ctx.EventManager().EmitTypedEvent(&v1.PositionChangedEvent{
		Trader:                trader,
		Pair:                  amm.Pair(),
		Margin:                positionResp.Position.Margin,
		PositionNotional:      positionResp.ExchangedPositionSize,
		ExchangedPositionSize: positionResp.ExchangedPositionSize,
		Fee:                   transferredFee,
		PositionSizeAfter:     positionResp.Position.Size_,
		RealizedPnl:           positionResp.RealizedPnl,
		UnrealizedPnlAfter:    positionResp.UnrealizedPnlAfter,
		BadDebt:               positionResp.BadDebt,
		LiquidationPenalty:    sdk.ZeroInt(),
		SpotPrice:             spotPrice,
		FundingPayment:        positionResp.FundingPayment,
	})
}

// TODO test: increasePosition
func (k Keeper) increasePosition(
	ctx sdk.Context, amm VirtualAMM, side v1.Side, trader string,
	openNotional sdk.Int, minPositionSize sdk.Int, leverage sdk.Int) (
	positionResp *v1.PositionResp, err error) {

	positionResp = new(v1.PositionResp)

	oldPosition, err := k.Positions().Get(ctx, amm.Pair(), trader) // TODO(mercilex) we already have the info from the caller
	if err != nil {
		panic(err)
	}

	positionResp.ExchangedPositionSize, err = swapInput(ctx, amm, side, openNotional, minPositionSize, false)
	if err != nil {
		return nil, err
	}

	newSize := oldPosition.Size_.Add(positionResp.ExchangedPositionSize)

	err = k.updateOpenInterestNotional(ctx, amm, openNotional, trader)
	if err != nil {
		return nil, err
	}

	// check if trader is not in whitelist to check max position size
	if !k.Whitelist().IsWhitelisted(ctx, trader) {
		maxHoldingBaseAsset, err := amm.GetMaxHoldingBaseAsset(ctx)
		if err != nil {
			return nil, err
		}

		if maxHoldingBaseAsset.IsPositive() && maxHoldingBaseAsset.LT(newSize.Abs()) {
			return nil, fmt.Errorf("hit position size upper bound")
		}
	}

	increaseMarginRequirement := openNotional.Quo(leverage)

	remainMargin, _, fundingPayment, latestCumulativePremiumFraction, err := k.calcRemainMarginWithFundingPayment(
		ctx,
		amm,
		oldPosition,
		increaseMarginRequirement,
	)
	if err != nil {
		return nil, err
	}

	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, amm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedQuoteAssetAmount = openNotional
	positionResp.UnrealizedPnlAfter = unrealizedPnL
	positionResp.MarginToVault = increaseMarginRequirement
	positionResp.FundingPayment = fundingPayment
	positionResp.Position = &v1.Position{
		Address:                             trader,
		Pair:                                amm.Pair(),
		Size_:                               newSize,
		Margin:                              remainMargin,
		OpenNotional:                        oldPosition.OpenNotional.Add(positionResp.ExchangedQuoteAssetAmount),
		LastUpdateCumulativePremiumFraction: latestCumulativePremiumFraction,
		LiquidityHistoryIndex:               oldPosition.LiquidityHistoryIndex,
		BlockNumber:                         ctx.BlockHeight(),
	}

	return
}

// TODO test: updateOpenInterestNotional
func (k Keeper) updateOpenInterestNotional(ctx sdk.Context, amm VirtualAMM, amount sdk.Int, trader string) error {
	maxOpenInterest, err := amm.GetOpenInterestNotionalCap(ctx)
	if err != nil {
		return err
	}
	if maxOpenInterest.IsZero() {
		return nil
	}

	pairMetadata, err := k.PairMetadata().Get(ctx, amm.Pair())
	if err != nil {
		return err
	}
	updatedOpenInterestNotional := amount.Add(*pairMetadata.OpenInterestNotional)

	if updatedOpenInterestNotional.IsNegative() {
		updatedOpenInterestNotional = sdk.ZeroInt()
	}

	if amount.IsPositive() {
		if updatedOpenInterestNotional.GT(maxOpenInterest) && !k.Whitelist().IsWhitelisted(ctx, trader) {
			return fmt.Errorf("over limit")
		}
	}

	// update pair metadata
	pairMetadata.OpenInterestNotional = &updatedOpenInterestNotional
	k.PairMetadata().Set(ctx, pairMetadata)

	return nil
}

// TODO test: calcRemainMarginWithFundingPayment
func (k Keeper) calcRemainMarginWithFundingPayment(
	ctx sdk.Context, amm VirtualAMM,
	oldPosition *v1.Position, marginDelta sdk.Int) (
	remainMargin sdk.Int, badDebt sdk.Int, fundingPayment sdk.Int, latestCumulativePremiumFraction sdk.Int, err error) {

	latestCumulativePremiumFraction, err = k.getLatestCumulativePremiumFraction(ctx, amm)
	if err != nil {
		return
	}

	if !oldPosition.Size_.IsZero() { // TODO(mercilex): what if this does evaluate to false?
		fundingPayment = latestCumulativePremiumFraction.
			Sub(oldPosition.LastUpdateCumulativePremiumFraction).
			Mul(oldPosition.Size_)
	}

	signedRemainMargin := marginDelta.Sub(fundingPayment).Add(oldPosition.Margin)
	switch signedRemainMargin.IsNegative() {
	case true:
		badDebt = signedRemainMargin.Abs()
	case false:
		badDebt = sdk.ZeroInt()
		remainMargin = signedRemainMargin.Abs()
	}

	return
}

// TODO test: getLatestCumulativePremiumFraction
func (k Keeper) getLatestCumulativePremiumFraction(ctx sdk.Context, amm VirtualAMM) (sdk.Int, error) {
	pairMetadata, err := k.PairMetadata().Get(ctx, amm.Pair())
	if err != nil {
		return sdk.Int{}, err
	}
	// this should never fail
	return pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1], nil
}

// TODO test: getPositionNotionalAndUnrealizedPnL
func (k Keeper) getPositionNotionalAndUnrealizedPnL(ctx sdk.Context, amm VirtualAMM,
	trader string, pnlCalcOption v1.PnLCalcOption) (
	positionNotional, unrealizedPnL sdk.Int, err error) {

	position, err := k.Positions().Get(ctx, amm.Pair(), trader) // tODO(mercilex): inefficient refetch
	if err != nil {
		return
	}

	positionSizeAbs := position.Size_.Abs()
	if positionSizeAbs.IsZero() {
		return sdk.ZeroInt(), sdk.ZeroInt(), nil
	}

	isShortPosition := position.Size_.IsNegative()
	var dir VirtualAMMDirection
	switch isShortPosition {
	case true:
		dir = VirtualAMMDirection_RemoveFromAMM
	default:
		dir = VirtualAMMDirection_AddToAMM
	}

	switch pnlCalcOption {
	case v1.PnLCalcOption_PnLCalcOption_TWAP:
		positionNotional, err = amm.GetOutputTWAP(ctx, dir, positionSizeAbs)
		if err != nil {
			return
		}
	case v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE:
		positionNotional, err = amm.GetOutputPrice(ctx, dir, positionSizeAbs)
		if err != nil {
			return
		}
	case v1.PnLCalcOption_PnLCalcOption_ORACLE:
		oraclePrice, err2 := amm.GetUnderlyingPrice(ctx)
		if err2 != nil {
			err = err2
			return
		}
		positionNotional = positionSizeAbs.Mul(oraclePrice)
	default:
		panic("unrecognized pnl calc option: " + pnlCalcOption.String())
	}

	switch isShortPosition {
	case true:
		unrealizedPnL = position.OpenNotional.Sub(positionNotional)
	case false:
		unrealizedPnL = positionNotional.Sub(position.OpenNotional)
	}

	return
}

// TODO test: openReversePosition
func (k Keeper) openReversePosition(
	ctx sdk.Context, amm VirtualAMM, side v1.Side, trader string,
	quoteAssetAmount sdk.Int, leverage sdk.Int, baseAssetAmountLimit sdk.Int, canOverFluctuationLimit bool) (positionResp *v1.PositionResp, err error) {
	positionResp = new(v1.PositionResp)

	openNotional := quoteAssetAmount.Mul(leverage)
	oldPositionNotional, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, amm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	switch oldPositionNotional.GT(openNotional) {
	// position reduction
	case true:
		return k.reducePosition(
			ctx, amm, side, trader,
			openNotional, oldPositionNotional, baseAssetAmountLimit, unrealizedPnL,
			canOverFluctuationLimit)
	// close and reverse
	default:
		return k.closeAndOpenReversePosition(ctx, amm, side, trader, quoteAssetAmount, leverage, baseAssetAmountLimit)
	}
}

// TODO test: reducePosition
func (k Keeper) reducePosition(
	ctx sdk.Context, amm VirtualAMM, side v1.Side, trader string,
	openNotional, oldPositionNotional, baseAssetAmountLimit, unrealizedPnL sdk.Int,
	canOverFluctuationLimit bool) (positionResp *v1.PositionResp, err error) {

	positionResp = new(v1.PositionResp)

	err = k.updateOpenInterestNotional(ctx, amm, openNotional.MulRaw(-1), trader)
	if err != nil {
		return nil, err
	}
	var oldPosition *v1.Position
	oldPosition, err = k.Positions().Get(ctx, amm.Pair(), trader)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedPositionSize, err = swapInput(ctx, amm, side, openNotional, baseAssetAmountLimit, canOverFluctuationLimit)
	if err != nil {
		return nil, err
	}

	if !oldPosition.Size_.IsZero() {
		var realizedPnL = unrealizedPnL.Mul(positionResp.ExchangedPositionSize.Abs()).Quo(oldPosition.Size_.Abs())
		positionResp.RealizedPnl = realizedPnL
	}
	var remainMargin, latestCumulativePremiumFraction sdk.Int
	remainMargin, positionResp.BadDebt, positionResp.FundingPayment, latestCumulativePremiumFraction, err =
		k.calcRemainMarginWithFundingPayment(ctx, amm, oldPosition, positionResp.RealizedPnl)
	if err != nil {
		return nil, err
	}

	positionResp.UnrealizedPnlAfter = unrealizedPnL.Sub(positionResp.RealizedPnl)
	positionResp.ExchangedQuoteAssetAmount = openNotional

	var remainOpenNotional sdk.Int
	switch oldPosition.Size_.IsPositive() {
	case true:
		remainOpenNotional = oldPositionNotional.Sub(positionResp.ExchangedQuoteAssetAmount).Sub(positionResp.UnrealizedPnlAfter)
	case false:
		remainOpenNotional = positionResp.UnrealizedPnlAfter.Add(oldPositionNotional).Sub(positionResp.ExchangedQuoteAssetAmount)
	}

	if remainOpenNotional.LTE(sdk.ZeroInt()) {
		panic("value of open notional <= 0")
	}

	positionResp.Position = &v1.Position{
		Address:                             trader,
		Pair:                                amm.Pair(),
		Size_:                               oldPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                              remainMargin,
		OpenNotional:                        remainOpenNotional.Abs(),
		LastUpdateCumulativePremiumFraction: latestCumulativePremiumFraction,
		LiquidityHistoryIndex:               oldPosition.LiquidityHistoryIndex,
		BlockNumber:                         ctx.BlockHeight(),
	}
	return positionResp, nil
}

// TODO test: closeAndOpenReversePosition
func (k Keeper) closeAndOpenReversePosition(ctx sdk.Context, amm VirtualAMM, side v1.Side, trader string,
	quoteAssetAmount, leverage, baseAssetAmountLimit sdk.Int) (positionResp *v1.PositionResp, err error) {

	positionResp = new(v1.PositionResp)

	closePositionResp, err := k.closePosition(ctx, amm, trader, sdk.ZeroInt())
	if err != nil {
		return nil, err
	}

	if closePositionResp.BadDebt.LTE(sdk.ZeroInt()) {
		return nil, fmt.Errorf("underwater position")
	}

	openNotional := quoteAssetAmount.Mul(leverage).Sub(closePositionResp.ExchangedQuoteAssetAmount)

	switch openNotional.Quo(leverage).IsZero() {
	case true:
		positionResp = closePositionResp
	case false:
		var updatedBaseAssetAmountLimit sdk.Int
		if baseAssetAmountLimit.GT(closePositionResp.ExchangedPositionSize) {
			updatedBaseAssetAmountLimit = baseAssetAmountLimit.Sub(closePositionResp.ExchangedPositionSize.Abs())
		}

		var increasePositionResp *v1.PositionResp
		increasePositionResp, err = k.increasePosition(
			ctx, amm, side, trader, openNotional, updatedBaseAssetAmountLimit, leverage)
		if err != nil {
			return nil, err
		}
		positionResp = &v1.PositionResp{
			Position:                  increasePositionResp.Position,
			ExchangedQuoteAssetAmount: closePositionResp.ExchangedQuoteAssetAmount.Add(increasePositionResp.ExchangedQuoteAssetAmount),
			BadDebt:                   closePositionResp.BadDebt.Add(increasePositionResp.BadDebt),
			ExchangedPositionSize:     closePositionResp.ExchangedPositionSize.Add(increasePositionResp.ExchangedPositionSize),
			FundingPayment:            closePositionResp.FundingPayment.Add(increasePositionResp.FundingPayment),
			RealizedPnl:               closePositionResp.RealizedPnl.Add(increasePositionResp.RealizedPnl),
			MarginToVault:             closePositionResp.MarginToVault.Add(increasePositionResp.MarginToVault),
			UnrealizedPnlAfter:        sdk.ZeroInt(),
		}
	}

	return positionResp, nil
}

// TODO test: closePosition
func (k Keeper) closePosition(ctx sdk.Context, amm VirtualAMM, trader string, quoteAssetAmountLimit sdk.Int) (
	positionResp *v1.PositionResp, err error) {

	positionResp = new(v1.PositionResp)

	oldPosition, err := k.Positions().Get(ctx, amm.Pair(), trader)
	if err != nil {
		return nil, err
	}
	if oldPosition.Size_.IsZero() {
		return nil, fmt.Errorf("zero position size")
	}
	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, amm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	remainMargin, badDebt, fundingPayment, _, err := k.calcRemainMarginWithFundingPayment(ctx, amm, oldPosition, unrealizedPnL)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedPositionSize = oldPosition.Size_.MulRaw(-1)
	positionResp.RealizedPnl = unrealizedPnL
	positionResp.BadDebt = badDebt
	positionResp.FundingPayment = fundingPayment
	positionResp.MarginToVault = remainMargin.MulRaw(-1)

	var ammDir VirtualAMMDirection
	switch oldPosition.Size_.GTE(sdk.ZeroInt()) {
	case true:
		ammDir = VirtualAMMDirection_AddToAMM
	case false:
		ammDir = VirtualAMMDirection_RemoveFromAMM
	}
	positionResp.ExchangedQuoteAssetAmount, err = amm.SwapOutput(ctx, ammDir, oldPosition.Size_.Abs(), quoteAssetAmountLimit)
	if err != nil {
		return nil, err
	}

	err = k.updateOpenInterestNotional(ctx, amm, unrealizedPnL.Add(badDebt).Add(oldPosition.OpenNotional).MulRaw(-1), trader)
	if err != nil {
		return nil, err
	}

	err = k.clearPosition(ctx, amm, trader)
	if err != nil {
		return nil, err
	}

	return positionResp, nil
}

// TODO test: clearPosition
func (k Keeper) clearPosition(ctx sdk.Context, amm VirtualAMM, trader string) error {
	return k.Positions().Update(ctx, &v1.Position{
		Address:                             trader,
		Pair:                                amm.Pair(),
		Size_:                               sdk.ZeroInt(),
		Margin:                              sdk.ZeroInt(),
		OpenNotional:                        sdk.ZeroInt(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroInt(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	})
}

// TODO test: transferFee
func (k Keeper) transferFee(ctx sdk.Context, trader sdk.AccAddress, amm VirtualAMM, positionNotional sdk.Int) (sdk.Int, error) {
	toll, spread, err := amm.CalcFee(positionNotional)
	if err != nil {
		return sdk.Int{}, err
	}

	hasToll := toll.IsPositive()
	hasSpread := spread.IsPositive()

	if !hasToll && hasSpread {
		// TODO(mercilex): what's the meaning of returning sdk.Int if both evaluate to false, should this happen?
		return sdk.Int{}, nil
	}

	if hasSpread {
		err = k.bk.SendCoinsFromAccountToModule(ctx, trader, v1.InsuranceFundModuleAccount,
			sdk.NewCoins(sdk.NewCoin(amm.QuoteTokenDenom(), spread)))
		if err != nil {
			return sdk.Int{}, err
		}
	}
	if hasToll {
		err = k.bk.SendCoinsFromAccountToModule(ctx, trader, v1.FeePoolModuleAccount,
			sdk.NewCoins(sdk.NewCoin(amm.QuoteTokenDenom(), toll)))
	}

	return toll.Add(spread), nil
}

// TODO test: getMarginRatio
func (k Keeper) getMarginRatio(ctx sdk.Context, amm VirtualAMM, trader string) (sdk.Int, error) {
	position, err := k.Positions().Get(ctx, amm.Pair(), trader) // TODO(mercilex): inefficient position get
	if err != nil {
		return sdk.Int{}, err
	}

	if position.Size_.IsZero() {
		panic("position with zero size") // tODO(mercilex): panic or error? this is a require
	}

	unrealizedPnL, positionNotional, err := k.getPreferencePositionNotionalAndUnrealizedPnL(ctx, amm, trader, v1.PnLPreferenceOption_PnLPreferenceOption_MAX)
	if err != nil {
		return sdk.Int{}, err
	}

	return k._getMarginRatio(ctx, amm, position, unrealizedPnL, positionNotional)
}

// TODO test: _getMarginRatio
func (k Keeper) _getMarginRatio(ctx sdk.Context, amm VirtualAMM, position *v1.Position, unrealizedPnL, positionNotional sdk.Int) (sdk.Int, error) {
	// todo(mercilex): maybe inefficient re-get
	remainMargin, badDebt, _, _, err := k.calcRemainMarginWithFundingPayment(ctx, amm, position, unrealizedPnL)
	if err != nil {
		return sdk.Int{}, err
	}

	return remainMargin.Sub(badDebt).Quo(positionNotional), nil
}

// TODO test: getPreferencePositionNotionalAndUnrealizedPnL
func (k Keeper) getPreferencePositionNotionalAndUnrealizedPnL(ctx sdk.Context, amm VirtualAMM, trader string, pnLPreferenceOption v1.PnLPreferenceOption) (sdk.Int, sdk.Int, error) {
	// TODO(mercilex): maybe inefficient get position notional and unrealized pnl
	spotPositionNotional, spotPricePnl, err := k.getPositionNotionalAndUnrealizedPnL(ctx, amm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return sdk.Int{}, sdk.Int{}, err
	}

	twapPositionNotional, twapPricePnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, amm, trader, v1.PnLCalcOption_PnLCalcOption_TWAP)
	if err != nil {
		return sdk.Int{}, sdk.Int{}, err
	}

	// todo(mercilex): logic can be simplified here but keeping it for now as perp reference
	switch pnLPreferenceOption {
	// if MAX PNL
	case v1.PnLPreferenceOption_PnLPreferenceOption_MAX:
		// spotPNL > twapPnL
		switch spotPricePnl.GT(twapPricePnL) {
		// true: spotPNL > twapPNL -> return spot pnl, spot position notional
		case true:
			return spotPricePnl, spotPositionNotional, nil
		// false: spotPNL <= twapPNL -> return twapPNL twapPositionNotional
		default:
			return twapPricePnL, twapPositionNotional, nil
		}
	// if min PNL
	case v1.PnLPreferenceOption_PnLPreferenceOption_MIN:
		switch spotPricePnl.GT(twapPricePnL) {
		// true: spotPNL > twapPNL -> return twapPNL, twapPositionNotional
		case true:
			return twapPricePnL, twapPositionNotional, nil
		// false: spotPNL <= twapPNL -> return spotPNL, spotPositionNotional
		default:
			return spotPricePnl, spotPositionNotional, nil
		}
	default:
		panic("invalid pnl preference option " + pnLPreferenceOption.String())
	}
}

// TODO test: swapInput
func swapInput(ctx sdk.Context, amm VirtualAMM,
	side v1.Side, inputAmount sdk.Int, minOutputAmount sdk.Int, canOverFluctuationLimit bool) (sdk.Int, error) {

	var ammDir VirtualAMMDirection
	switch side {
	case v1.Side_Side_BUY:
		ammDir = VirtualAMMDirection_AddToAMM
	case v1.Side_Side_SELL:
		ammDir = VirtualAMMDirection_RemoveFromAMM
	default:
		panic("invalid side")
	}

	outputAmount, err := amm.SwapInput(ctx, ammDir, inputAmount, minOutputAmount, canOverFluctuationLimit)
	if err != nil {
		return sdk.Int{}, err
	}

	switch ammDir {
	case VirtualAMMDirection_AddToAMM:
		return outputAmount, nil
	case VirtualAMMDirection_RemoveFromAMM:
		inverseSign := outputAmount.MulRaw(-1)
		return inverseSign, nil
	default:
		panic("invalid side")
	}

}

/*
function requireMoreMarginRatio(
        SignedDecimal.signedDecimal memory _marginRatio,
        Decimal.decimal memory _baseMarginRatio,
        bool _largerThanOrEqualTo
    ) private pure {
        int256 remainingMarginRatio = _marginRatio.subD(_baseMarginRatio).toInt();
        require(
            _largerThanOrEqualTo ? remainingMarginRatio >= 0 : remainingMarginRatio < 0,
            "Margin ratio not meet criteria"
        );
    }
*/

// TODO test: requireMoreMarginRatio
func requireMoreMarginRatio(marginRatio, baseMarginRatio sdk.Int, largerThanOrEqualTo bool) error {
	// TODO(mercilex): look at this and make sure it's legit compared ot the counterparty above ^
	remainMarginRatio := marginRatio.Sub(baseMarginRatio)
	switch largerThanOrEqualTo {
	case true:
		if !remainMarginRatio.GTE(sdk.ZeroInt()) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	default:
		if remainMarginRatio.LT(sdk.ZeroInt()) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	}

	return nil
}
