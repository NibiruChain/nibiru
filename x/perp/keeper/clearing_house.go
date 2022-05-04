package keeper

import (
	"errors"
	"fmt"

	v1 "github.com/NibiruChain/nibiru/x/perp/types/v1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ = Keeper.openPosition
	_ = Keeper.increasePosition
	_ = swapInput
	_ = Keeper.transferFee
)

// TODO test: openPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) openPosition(
	ctx sdk.Context, vamm v1.VirtualPool, side v1.Side, trader string,
	quoteAssetAmount, leverage, baseAssetAmountLimit sdk.Int,
) error {
	// TODO(mercilex): missing checks
	params := k.GetParams(ctx)

	position, err := k.Positions().Get(ctx, vamm.Pair(), trader)
	positionExists := errors.Is(err, errNotFound)

	var positionResp *v1.PositionResp
	switch {
	// increase position case
	case !positionExists,
		position.Size_.IsPositive() && side == v1.Side_Side_BUY,
		position.Size_.IsNegative() && side == v1.Side_Side_SELL:
		positionResp, err = k.increasePosition(
			ctx, vamm, side, trader,
			quoteAssetAmount.Mul(leverage),
			baseAssetAmountLimit,
			leverage)
		if err != nil {
			return err
		}

	// everything else decreases the position
	default:
		positionResp, err = k.openReversePosition(
			ctx, vamm, side, trader,
			quoteAssetAmount, leverage, baseAssetAmountLimit, false)
		if err != nil {
			return err
		}
	}

	// update position in state
	k.Positions().Set(ctx, positionResp.Position)

	if !positionExists && !positionResp.Position.Size_.IsZero() {
		marginRatio, err := k.GetMarginRatio(ctx, vamm, trader)
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
		err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, traderAddr, v1.VaultModuleAccount,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), positionResp.MarginToVault)))
		if err != nil {
			return err
		}
	case positionResp.MarginToVault.IsNegative():
		err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, v1.VaultModuleAccount, traderAddr,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), positionResp.MarginToVault.Abs())))
		if err != nil {
			return err
		}
	}

	transferredFee, err := k.transferFee(ctx, traderAddr, vamm, positionResp.ExchangedQuoteAssetAmount)
	if err != nil {
		return err
	}

	spotPrice, err := vamm.GetSpotPrice(ctx)
	if err != nil {
		return err
	}

	return ctx.EventManager().EmitTypedEvent(&v1.PositionChangedEvent{
		Trader:                trader,
		Pair:                  vamm.Pair(),
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

// TODO test: increasePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) increasePosition(
	ctx sdk.Context, vamm v1.VirtualPool, side v1.Side, trader string,
	openNotional sdk.Int, minPositionSize sdk.Int, leverage sdk.Int) (
	positionResp *v1.PositionResp, err error) {
	positionResp = new(v1.PositionResp)

	oldPosition, err := k.Positions().Get(ctx, vamm.Pair(), trader) // TODO(mercilex) we already have the info from the caller
	if err != nil {
		panic(err)
	}

	positionResp.ExchangedPositionSize, err = swapInput(ctx, vamm, side, openNotional, minPositionSize, false)
	if err != nil {
		return nil, err
	}

	newSize := oldPosition.Size_.Add(positionResp.ExchangedPositionSize)

	err = k.updateOpenInterestNotional(ctx, vamm, openNotional, trader)
	if err != nil {
		return nil, err
	}

	// check if trader is not in whitelist to check max position size
	if !k.Whitelist().IsWhitelisted(ctx, trader) {
		maxHoldingBaseAsset, err := vamm.GetMaxHoldingBaseAsset(ctx)
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
		vamm,
		oldPosition,
		increaseMarginRequirement,
	)
	if err != nil {
		return nil, err
	}

	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, vamm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedQuoteAssetAmount = openNotional
	positionResp.UnrealizedPnlAfter = unrealizedPnL
	positionResp.MarginToVault = increaseMarginRequirement
	positionResp.FundingPayment = fundingPayment
	positionResp.Position = &v1.Position{
		Address:                             trader,
		Pair:                                vamm.Pair(),
		Size_:                               newSize,
		Margin:                              remainMargin,
		OpenNotional:                        oldPosition.OpenNotional.Add(positionResp.ExchangedQuoteAssetAmount),
		LastUpdateCumulativePremiumFraction: latestCumulativePremiumFraction,
		LiquidityHistoryIndex:               oldPosition.LiquidityHistoryIndex,
		BlockNumber:                         ctx.BlockHeight(),
	}

	return
}

// TODO test: updateOpenInterestNotional | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) updateOpenInterestNotional(ctx sdk.Context, vamm v1.VirtualPool, amount sdk.Int, trader string) error {
	maxOpenInterest, err := vamm.GetOpenInterestNotionalCap(ctx)
	if err != nil {
		return err
	}
	if maxOpenInterest.IsZero() {
		return nil
	}

	pairMetadata, err := k.PairMetadata().Get(ctx, vamm.Pair())
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

// TODO test: calcRemainMarginWithFundingPayment | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) calcRemainMarginWithFundingPayment(
	ctx sdk.Context, vamm v1.VirtualPool,
	oldPosition *v1.Position, marginDelta sdk.Int,
) (remainMargin sdk.Int, badDebt sdk.Int, fundingPayment sdk.Int,
	latestCumulativePremiumFraction sdk.Int, err error) {
	latestCumulativePremiumFraction, err = k.getLatestCumulativePremiumFraction(ctx, vamm)
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

// TODO test: getLatestCumulativePremiumFraction | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) getLatestCumulativePremiumFraction(ctx sdk.Context, vamm v1.VirtualPool) (sdk.Int, error) {
	pairMetadata, err := k.PairMetadata().Get(ctx, vamm.Pair())
	if err != nil {
		return sdk.Int{}, err
	}
	// this should never fail
	return pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1], nil
}

// TODO test: getPositionNotionalAndUnrealizedPnL | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) getPositionNotionalAndUnrealizedPnL(ctx sdk.Context, vamm v1.VirtualPool,
	trader string, pnlCalcOption v1.PnLCalcOption) (
	positionNotional, unrealizedPnL sdk.Int, err error) {
	position, err := k.Positions().Get(ctx, vamm.Pair(), trader) // tODO(mercilex): inefficient refetch
	if err != nil {
		return
	}

	positionSizeAbs := position.Size_.Abs()
	if positionSizeAbs.IsZero() {
		return sdk.ZeroInt(), sdk.ZeroInt(), nil
	}

	isShortPosition := position.Size_.IsNegative()
	var dir v1.VirtualPoolDirection
	switch isShortPosition {
	case true:
		dir = v1.VirtualPoolDirection_RemoveFromAMM
	default:
		dir = v1.VirtualPoolDirection_AddToAMM
	}

	switch pnlCalcOption {
	case v1.PnLCalcOption_PnLCalcOption_TWAP:
		positionNotional, err = vamm.GetOutputTWAP(ctx, dir, positionSizeAbs)
		if err != nil {
			return
		}
	case v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE:
		positionNotional, err = vamm.GetOutputPrice(ctx, dir, positionSizeAbs)
		if err != nil {
			return
		}
	case v1.PnLCalcOption_PnLCalcOption_ORACLE:
		oraclePrice, err2 := vamm.GetUnderlyingPrice(ctx)
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

// TODO test: openReversePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) openReversePosition(
	ctx sdk.Context, vamm v1.VirtualPool, side v1.Side, trader string,
	quoteAssetAmount sdk.Int, leverage sdk.Int, baseAssetAmountLimit sdk.Int,
	canOverFluctuationLimit bool,
) (positionResp *v1.PositionResp, err error) {
	openNotional := quoteAssetAmount.Mul(leverage)
	oldPositionNotional, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, vamm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	switch oldPositionNotional.GT(openNotional) {
	// position reduction
	case true:
		return k.reducePosition(
			ctx, vamm, side, trader,
			openNotional, oldPositionNotional, baseAssetAmountLimit, unrealizedPnL,
			canOverFluctuationLimit)
	// close and reverse
	default:
		return k.closeAndOpenReversePosition(ctx, vamm, side, trader, quoteAssetAmount, leverage, baseAssetAmountLimit)
	}
}

// TODO test: reducePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) reducePosition(
	ctx sdk.Context, vamm v1.VirtualPool, side v1.Side, trader string,
	openNotional, oldPositionNotional, baseAssetAmountLimit, unrealizedPnL sdk.Int,
	canOverFluctuationLimit bool,
) (positionResp *v1.PositionResp, err error) {
	positionResp = new(v1.PositionResp)

	err = k.updateOpenInterestNotional(ctx, vamm, openNotional.MulRaw(-1), trader)
	if err != nil {
		return nil, err
	}
	var oldPosition *v1.Position
	oldPosition, err = k.Positions().Get(ctx, vamm.Pair(), trader)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedPositionSize, err = swapInput(
		ctx, vamm, side, openNotional, baseAssetAmountLimit, canOverFluctuationLimit,
	)
	if err != nil {
		return nil, err
	}

	if !oldPosition.Size_.IsZero() {
		var realizedPnL = unrealizedPnL.Mul(positionResp.ExchangedPositionSize.Abs()).Quo(oldPosition.Size_.Abs())
		positionResp.RealizedPnl = realizedPnL
	}
	var remainMargin, latestCumulativePremiumFraction sdk.Int
	remainMargin, positionResp.BadDebt, positionResp.FundingPayment, latestCumulativePremiumFraction, err =
		k.calcRemainMarginWithFundingPayment(ctx, vamm, oldPosition, positionResp.RealizedPnl)
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
		Pair:                                vamm.Pair(),
		Size_:                               oldPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                              remainMargin,
		OpenNotional:                        remainOpenNotional.Abs(),
		LastUpdateCumulativePremiumFraction: latestCumulativePremiumFraction,
		LiquidityHistoryIndex:               oldPosition.LiquidityHistoryIndex,
		BlockNumber:                         ctx.BlockHeight(),
	}
	return positionResp, nil
}

// TODO test: closeAndOpenReversePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) closeAndOpenReversePosition(
	ctx sdk.Context, amm v1.VirtualPool, side v1.Side, trader string,
	quoteAssetAmount, leverage, baseAssetAmountLimit sdk.Int,
) (positionResp *v1.PositionResp, err error) {
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

// TODO test: closePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) closePosition(ctx sdk.Context, vamm v1.VirtualPool, trader string, quoteAssetAmountLimit sdk.Int) (
	positionResp *v1.PositionResp, err error) {
	positionResp = new(v1.PositionResp)

	oldPosition, err := k.Positions().Get(ctx, vamm.Pair(), trader)
	if err != nil {
		return nil, err
	}
	if oldPosition.Size_.IsZero() {
		return nil, fmt.Errorf("zero position size")
	}
	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, vamm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	remainMargin, badDebt, fundingPayment, _, err := k.calcRemainMarginWithFundingPayment(ctx, vamm, oldPosition, unrealizedPnL)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedPositionSize = oldPosition.Size_.MulRaw(-1)
	positionResp.RealizedPnl = unrealizedPnL
	positionResp.BadDebt = badDebt
	positionResp.FundingPayment = fundingPayment
	positionResp.MarginToVault = remainMargin.MulRaw(-1)

	var vammDir v1.VirtualPoolDirection
	switch oldPosition.Size_.GTE(sdk.ZeroInt()) {
	case true:
		vammDir = v1.VirtualPoolDirection_AddToAMM
	case false:
		vammDir = v1.VirtualPoolDirection_RemoveFromAMM
	}
	positionResp.ExchangedQuoteAssetAmount, err = vamm.SwapOutput(ctx, vammDir, oldPosition.Size_.Abs(), quoteAssetAmountLimit)
	if err != nil {
		return nil, err
	}

	err = k.updateOpenInterestNotional(ctx, vamm, unrealizedPnL.Add(badDebt).Add(oldPosition.OpenNotional).MulRaw(-1), trader)
	if err != nil {
		return nil, err
	}

	err = k.clearPosition(ctx, vamm, trader)
	if err != nil {
		return nil, err
	}

	return positionResp, nil
}

// TODO test: clearPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) clearPosition(ctx sdk.Context, vamm v1.VirtualPool, trader string) error {
	return k.Positions().Update(ctx, &v1.Position{
		Address:                             trader,
		Pair:                                vamm.Pair(),
		Size_:                               sdk.ZeroInt(),
		Margin:                              sdk.ZeroInt(),
		OpenNotional:                        sdk.ZeroInt(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroInt(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	})
}

// TODO test: transferFee | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) transferFee(
	ctx sdk.Context, trader sdk.AccAddress, vamm v1.VirtualPool,
	positionNotional sdk.Int,
) (sdk.Int, error) {
	toll, spread, err := vamm.CalcFee(positionNotional)
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
		err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, trader, v1.PerpEFModuleAccount,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), spread)))
		if err != nil {
			return sdk.Int{}, err
		}
	}
	if hasToll {
		err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, trader, v1.FeePoolModuleAccount,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), toll)))
		if err != nil {
			return sdk.Int{}, err
		}
	}

	return toll.Add(spread), nil
}

// TODO test: getPreferencePositionNotionalAndUnrealizedPnL
func (k Keeper) getPreferencePositionNotionalAndUnrealizedPnL(ctx sdk.Context, vamm v1.VirtualPool, trader string, pnLPreferenceOption v1.PnLPreferenceOption) (sdk.Int, sdk.Int, error) {
	// TODO(mercilex): maybe inefficient get position notional and unrealized pnl
	spotPositionNotional, spotPricePnl, err := k.getPositionNotionalAndUnrealizedPnL(ctx, vamm, trader, v1.PnLCalcOption_PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return sdk.Int{}, sdk.Int{}, err
	}

	twapPositionNotional, twapPricePnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, vamm, trader, v1.PnLCalcOption_PnLCalcOption_TWAP)
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

// TODO test: swapInput | https://github.com/NibiruChain/nibiru/issues/299
func swapInput(ctx sdk.Context, vamm v1.VirtualPool,
	side v1.Side, inputAmount sdk.Int, minOutputAmount sdk.Int, canOverFluctuationLimit bool) (sdk.Int, error) {
	var vammDir v1.VirtualPoolDirection
	switch side {
	case v1.Side_Side_BUY:
		vammDir = v1.VirtualPoolDirection_AddToAMM
	case v1.Side_Side_SELL:
		vammDir = v1.VirtualPoolDirection_RemoveFromAMM
	default:
		panic("invalid side")
	}

	outputAmount, err := vamm.SwapInput(ctx, vammDir, inputAmount, minOutputAmount, canOverFluctuationLimit)
	if err != nil {
		return sdk.Int{}, err
	}

	switch vammDir {
	case v1.VirtualPoolDirection_AddToAMM:
		return outputAmount, nil
	case v1.VirtualPoolDirection_RemoveFromAMM:
		inverseSign := outputAmount.MulRaw(-1)
		return inverseSign, nil
	default:
		panic("invalid side")
	}
}

