package keeper

import (
	"errors"
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	pooltypes "github.com/NibiruChain/nibiru/x/vpool/types"

	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/* TODO tests | These _ vars are here to pass the golangci-lint for unused methods.
They also serve as a reminder of which functions still need MVP unit or
integration tests */
var (
	_ = Keeper.swapInput
	_ = Keeper.closePosition
	_ = Keeper.increasePosition
	_ = Keeper.reducePosition
	_ = Keeper.updateOpenInterestNotional
	_ = Keeper.closeAndOpenReversePosition
	_ = Keeper.openReversePosition
	_ = Keeper.openPosition
	_ = Keeper.transferFee
)

// TODO test: openPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) openPosition(
	ctx sdk.Context, vamm types.IVirtualPool, pair common.TokenPair, side types.Side, trader string,
	quoteAssetAmount, leverage, baseAssetAmountLimit sdk.Int,
) error {
	// TODO(mercilex): missing checks
	params := k.GetParams(ctx)

	position, err := k.Positions().Get(ctx, pair, trader)
	positionExists := errors.Is(err, errNotFound)

	var positionResp *types.PositionResp
	switch {
	// increase position case
	case !positionExists,
		position.Size_.IsPositive() && side == types.Side_BUY,
		position.Size_.IsNegative() && side == types.Side_SELL:
		positionResp, err = k.increasePosition(
			ctx, vamm, pair, side, trader,
			quoteAssetAmount.Mul(leverage),
			baseAssetAmountLimit,
			leverage)
		if err != nil {
			return err
		}

	// everything else decreases the position
	default:
		positionResp, err = k.openReversePosition(
			ctx, vamm, pair, side, trader,
			quoteAssetAmount, leverage, baseAssetAmountLimit, false)
		if err != nil {
			return err
		}
	}

	// update position in state
	k.Positions().Set(ctx, pair, trader, positionResp.Position)

	if !positionExists && !positionResp.Position.Size_.IsZero() {
		marginRatio, err := k.GetMarginRatio(ctx, vamm, pair, trader)
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
			ctx, traderAddr, types.VaultModuleAccount,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), positionResp.MarginToVault)))
		if err != nil {
			return err
		}
	case positionResp.MarginToVault.IsNegative():
		err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.VaultModuleAccount, traderAddr,
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

	return ctx.EventManager().EmitTypedEvent(&types.PositionChangedEvent{
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
	ctx sdk.Context, vamm types.IVirtualPool, pair common.TokenPair, side types.Side, trader string,
	openNotional sdk.Int, minPositionSize sdk.Int, leverage sdk.Int) (
	positionResp *types.PositionResp, err error) {
	positionResp = new(types.PositionResp)

	oldPosition, err := k.Positions().Get(ctx, pair, trader) // TODO(mercilex) we already have the info from the caller
	if err != nil {
		panic(err)
	}

	positionResp.ExchangedPositionSize, err = k.swapInput(ctx, pair, side, openNotional, minPositionSize, false)
	if err != nil {
		return nil, err
	}

	newSize := oldPosition.Size_.Add(positionResp.ExchangedPositionSize)

	err = k.updateOpenInterestNotional(ctx, pair, openNotional, trader)
	if err != nil {
		return nil, err
	}

	// check if trader is not in whitelist to check max position size
	if !k.Whitelist().IsWhitelisted(ctx, trader) {
		maxHoldingBaseAsset, err := k.VpoolKeeper.GetMaxHoldingBaseAsset(ctx, pair)
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
		pair,
		oldPosition,
		increaseMarginRequirement,
	)
	if err != nil {
		return nil, err
	}

	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		vamm,
		pair,
		trader,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedQuoteAssetAmount = openNotional
	positionResp.UnrealizedPnlAfter = unrealizedPnL
	positionResp.MarginToVault = increaseMarginRequirement
	positionResp.FundingPayment = fundingPayment
	positionResp.Position = &types.Position{
		Address:                             trader,
		Pair:                                pair.String(),
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
func (k Keeper) updateOpenInterestNotional(ctx sdk.Context, pair common.TokenPair, amount sdk.Int, trader string) error {
	maxOpenInterest, err := k.VpoolKeeper.GetOpenInterestNotionalCap(ctx, pair)
	if err != nil {
		return err
	}
	if maxOpenInterest.IsZero() {
		return nil
	}

	pairMetadata, err := k.PairMetadata().Get(ctx, pair.String())
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
	ctx sdk.Context, pair common.TokenPair,
	oldPosition *types.Position, marginDelta sdk.Int,
) (remainMargin sdk.Int, badDebt sdk.Int, fundingPayment sdk.Int,
	latestCumulativePremiumFraction sdk.Int, err error) {
	latestCumulativePremiumFraction, err = k.getLatestCumulativePremiumFraction(ctx, pair)
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
func (k Keeper) getLatestCumulativePremiumFraction(ctx sdk.Context, pair common.TokenPair) (sdk.Int, error) {
	pairMetadata, err := k.PairMetadata().Get(ctx, pair.String())
	if err != nil {
		return sdk.Int{}, err
	}
	// this should never fail
	return pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1], nil
}

// TODO test: getPositionNotionalAndUnrealizedPnL | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) getPositionNotionalAndUnrealizedPnL(
	ctx sdk.Context, vamm types.IVirtualPool, pair common.TokenPair,
	trader string, pnlCalcOption types.PnLCalcOption,
) (
	positionNotional, unrealizedPnL sdk.Int, err error) {
	position, err := k.Positions().Get(ctx, pair, trader) // tODO(mercilex): inefficient refetch
	if err != nil {
		return
	}

	positionSizeAbs := position.Size_.Abs()
	if positionSizeAbs.IsZero() {
		return sdk.ZeroInt(), sdk.ZeroInt(), nil
	}

	isShortPosition := position.Size_.IsNegative()
	var dir types.VirtualPoolDirection
	switch isShortPosition {
	case true:
		dir = types.VirtualPoolDirection_RemoveFromAMM
	default:
		dir = types.VirtualPoolDirection_AddToAMM
	}

	switch pnlCalcOption {
	case types.PnLCalcOption_TWAP:
		positionNotional, err = vamm.GetOutputTWAP(ctx, dir, positionSizeAbs)
		if err != nil {
			return
		}
	case types.PnLCalcOption_SPOT_PRICE:
		positionNotional, err = vamm.GetOutputPrice(ctx, dir, positionSizeAbs)
		if err != nil {
			return
		}
	case types.PnLCalcOption_ORACLE:
		oraclePrice, err2 := vamm.GetUnderlyingPrice(ctx)
		if err2 != nil {
			err = err2
			return
		}
		// TODO: Replace price variables with sdk.Dec instead of sdk.Int
		positionNotional = positionSizeAbs.Mul(oraclePrice.TruncateInt())
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
	ctx sdk.Context, vamm types.IVirtualPool, pair common.TokenPair, side types.Side, trader string,
	quoteAssetAmount sdk.Int, leverage sdk.Int, baseAssetAmountLimit sdk.Int,
	canOverFluctuationLimit bool,
) (positionResp *types.PositionResp, err error) {
	openNotional := quoteAssetAmount.Mul(leverage)
	oldPositionNotional, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		vamm,
		pair,
		trader,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return nil, err
	}

	switch oldPositionNotional.GT(openNotional) {
	// position reduction
	case true:
		return k.reducePosition(
			ctx,
			vamm,
			pair,
			side,
			trader,
			openNotional,
			oldPositionNotional,
			baseAssetAmountLimit,
			unrealizedPnL,
			canOverFluctuationLimit,
		)
	// close and reverse
	default:
		return k.closeAndOpenReversePosition(ctx, vamm, pair, side, trader, quoteAssetAmount, leverage, baseAssetAmountLimit)
	}
}

// TODO test: reducePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) reducePosition(
	ctx sdk.Context, vamm types.IVirtualPool, pair common.TokenPair, side types.Side, trader string,
	openNotional, oldPositionNotional, baseAssetAmountLimit, unrealizedPnL sdk.Int,
	canOverFluctuationLimit bool,
) (positionResp *types.PositionResp, err error) {
	positionResp = new(types.PositionResp)

	err = k.updateOpenInterestNotional(ctx, pair, openNotional.MulRaw(-1), trader)
	if err != nil {
		return nil, err
	}
	var oldPosition *types.Position
	oldPosition, err = k.Positions().Get(ctx, pair, trader)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedPositionSize, err = k.swapInput(
		ctx, pair, side, openNotional, baseAssetAmountLimit, canOverFluctuationLimit,
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
		k.calcRemainMarginWithFundingPayment(ctx, pair, oldPosition, positionResp.RealizedPnl)
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

	positionResp.Position = &types.Position{
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
	ctx sdk.Context, amm types.IVirtualPool, pair common.TokenPair, side types.Side, trader string,
	quoteAssetAmount, leverage, baseAssetAmountLimit sdk.Int,
) (positionResp *types.PositionResp, err error) {
	positionResp = new(types.PositionResp)

	closePositionResp, err := k.closePosition(ctx, amm, pair, trader, sdk.ZeroInt())
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

		var increasePositionResp *types.PositionResp
		increasePositionResp, err = k.increasePosition(
			ctx, amm, pair, side, trader, openNotional, updatedBaseAssetAmountLimit, leverage)
		if err != nil {
			return nil, err
		}
		positionResp = &types.PositionResp{
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
func (k Keeper) closePosition(ctx sdk.Context, vamm types.IVirtualPool, pair common.TokenPair, trader string, quoteAssetAmountLimit sdk.Int) (
	positionResp *types.PositionResp, err error) {
	positionResp = new(types.PositionResp)

	oldPosition, err := k.Positions().Get(ctx, pair, trader)
	if err != nil {
		return nil, err
	}
	if oldPosition.Size_.IsZero() {
		return nil, fmt.Errorf("zero position size")
	}
	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(ctx, vamm, pair, trader, types.PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	remainMargin, badDebt, fundingPayment, _, err := k.calcRemainMarginWithFundingPayment(ctx, pair, oldPosition, unrealizedPnL)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedPositionSize = oldPosition.Size_.MulRaw(-1)
	positionResp.RealizedPnl = unrealizedPnL
	positionResp.BadDebt = badDebt
	positionResp.FundingPayment = fundingPayment
	positionResp.MarginToVault = remainMargin.MulRaw(-1)

	var vammDir types.VirtualPoolDirection
	switch oldPosition.Size_.GTE(sdk.ZeroInt()) {
	case true:
		vammDir = types.VirtualPoolDirection_AddToAMM
	case false:
		vammDir = types.VirtualPoolDirection_RemoveFromAMM
	}
	positionResp.ExchangedQuoteAssetAmount, err = vamm.SwapOutput(ctx, vammDir, oldPosition.Size_.Abs(), quoteAssetAmountLimit)
	if err != nil {
		return nil, err
	}

	err = k.updateOpenInterestNotional(ctx, pair, unrealizedPnL.Add(badDebt).Add(oldPosition.OpenNotional).MulRaw(-1), trader)
	if err != nil {
		return nil, err
	}

	err = k.ClearPosition(ctx, pair, trader)
	if err != nil {
		return nil, err
	}

	return positionResp, nil
}

// TODO test: transferFee | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) transferFee(
	ctx sdk.Context, trader sdk.AccAddress, vamm types.IVirtualPool,
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
		err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, trader, types.PerpEFModuleAccount,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), spread)))
		if err != nil {
			return sdk.Int{}, err
		}
	}
	if hasToll {
		err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, trader, types.FeePoolModuleAccount,
			sdk.NewCoins(sdk.NewCoin(vamm.QuoteTokenDenom(), toll)))
		if err != nil {
			return sdk.Int{}, err
		}
	}

	return toll.Add(spread), nil
}

// TODO test: getPreferencePositionNotionalAndUnrealizedPnL
func (k Keeper) getPreferencePositionNotionalAndUnrealizedPnL(
	ctx sdk.Context,
	vamm types.IVirtualPool,
	pair common.TokenPair,
	trader string,
	pnLPreferenceOption types.PnLPreferenceOption,
) (sdk.Int, sdk.Int, error) {
	// TODO(mercilex): maybe inefficient get position notional and unrealized pnl
	spotPositionNotional, spotPricePnl, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		vamm,
		pair,
		trader,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return sdk.Int{}, sdk.Int{}, err
	}

	twapPositionNotional, twapPricePnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		vamm,
		pair,
		trader,
		types.PnLCalcOption_TWAP,
	)
	if err != nil {
		return sdk.Int{}, sdk.Int{}, err
	}

	// todo(mercilex): logic can be simplified here but keeping it for now as perp reference
	switch pnLPreferenceOption {
	// if MAX PNL
	case types.PnLPreferenceOption_MAX:
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
	case types.PnLPreferenceOption_MIN:
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
// TODO: Check Can Over Fluctuation Limit
func (k Keeper) swapInput(ctx sdk.Context, pair common.TokenPair,
	side types.Side, inputAmount sdk.Int, minOutputAmount sdk.Int, canOverFluctuationLimit bool) (sdk.Int, error) {
	var vammDir pooltypes.Direction
	switch side {
	case types.Side_BUY:
		vammDir = pooltypes.Direction_ADD_TO_AMM
	case types.Side_SELL:
		vammDir = pooltypes.Direction_REMOVE_FROM_AMM
	default:
		panic("invalid side")
	}

	outputAmount, err := k.VpoolKeeper.SwapInput(ctx, pair, vammDir, inputAmount, minOutputAmount)
	if err != nil {
		return sdk.Int{}, err
	}

	switch vammDir {
	case types.VirtualPoolDirection_AddToAMM:
		return outputAmount, nil
	case types.VirtualPoolDirection_RemoveFromAMM:
		inverseSign := outputAmount.MulRaw(-1)
		return inverseSign, nil
	default:
		panic("invalid side")
	}
}
