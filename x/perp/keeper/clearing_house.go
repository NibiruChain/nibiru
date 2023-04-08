package keeper

import (
	"errors"
	"fmt"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

/*
OpenPosition opens a position on the selected pair.

args:
  - ctx: cosmos-sdk context
  - pair: the pair where the position will be opened
  - side: whether the position in the BUY or SELL direction
  - traderAddr: the address of the trader who opens the position
  - quoteAssetAmount: the amount of quote asset
  - leverage: the amount of leverage to take, as sdk.Dec
  - baseAmtLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units

ret:
  - positionResp: contains the result of the open position and the new position
  - err: error
*/
func (k Keeper) OpenPosition(
	ctx sdk.Context,
	pair asset.Pair,
	side types.Side,
	traderAddr sdk.AccAddress,
	quoteAssetAmount sdk.Int,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (positionResp *types.PositionResp, err error) {
	vpool, err := k.VpoolKeeper.GetPool(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	err = k.checkOpenPositionRequirements(vpool, quoteAssetAmount, leverage)
	if err != nil {
		return nil, err
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	isNewPosition := errors.Is(err, collections.ErrNotFound)
	if isNewPosition {
		position = types.ZeroPosition(ctx, pair, traderAddr)
	} else if err != nil && !isNewPosition {
		return nil, err
	}

	sameSideLong := position.Size_.IsPositive() && side == types.Side_BUY
	sameSideShort := position.Size_.IsNegative() && side == types.Side_SELL

	var updatedVpool perpammtypes.Vpool
	var openSideMatchesPosition = sameSideLong || sameSideShort
	if isNewPosition || openSideMatchesPosition {
		updatedVpool, positionResp, err = k.increasePosition(
			ctx,
			vpool,
			position,
			side,
			/* openNotional */ leverage.MulInt(quoteAssetAmount),
			/* minPositionSize */ baseAmtLimit,
			/* leverage */ leverage)
		if err != nil {
			return nil, err
		}
	} else {
		updatedVpool, positionResp, err = k.openReversePosition(
			ctx,
			vpool,
			position,
			/* quoteAssetAmount */ quoteAssetAmount.ToDec(),
			/* leverage */ leverage,
			/* baseAmtLimit */ baseAmtLimit,
		)
		if err != nil {
			return nil, err
		}
	}

	if err = k.afterPositionUpdate(ctx, updatedVpool, traderAddr, *positionResp); err != nil {
		return nil, err
	}

	return positionResp, nil
}

// checkOpenPositionRequirements checks the minimum requirements to open a position.
//
// - Checks that the Vpool exists.
// - Checks that quote asset is not zero.
// - Checks that leverage is not zero.
// - Checks that leverage is below requirement.
func (k Keeper) checkOpenPositionRequirements(vpool perpammtypes.Vpool, quoteAssetAmount sdk.Int, leverage sdk.Dec) error {
	if quoteAssetAmount.IsZero() {
		return types.ErrQuoteAmountIsZero
	}

	if leverage.IsZero() {
		return types.ErrLeverageIsZero
	}

	if leverage.GT(vpool.Config.MaxLeverage) {
		return types.ErrLeverageIsTooHigh
	}

	return nil
}

// afterPositionUpdate is called when a position has been updated.
func (k Keeper) afterPositionUpdate(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	traderAddr sdk.AccAddress,
	positionResp types.PositionResp,
) (err error) {
	pair := vpool.Pair
	if !positionResp.Position.Size_.IsZero() {
		k.Positions.Insert(ctx, collections.Join(pair, traderAddr), *positionResp.Position)
	}

	if !positionResp.BadDebt.IsZero() {
		return fmt.Errorf("bad debt must be zero to prevent attacker from leveraging it")
	}

	if !positionResp.Position.Size_.IsZero() {
		marginRatio, err := k.GetMarginRatio(
			ctx,
			vpool,
			*positionResp.Position,
			types.MarginCalculationPriceOption_MAX_PNL,
		)
		if err != nil {
			return err
		}

		maintenanceMarginRatio := vpool.Config.MaintenanceMarginRatio
		if err = validateMarginRatio(marginRatio, maintenanceMarginRatio, true); err != nil {
			return types.ErrMarginRatioTooLow
		}
	}

	// transfer trader <=> vault
	marginToVault := positionResp.MarginToVault.RoundInt()
	switch {
	case marginToVault.IsPositive():
		coinToSend := sdk.NewCoin(pair.QuoteDenom(), marginToVault)
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(coinToSend)); err != nil {
			return err
		}
	case marginToVault.IsNegative():
		if err = k.Withdraw(ctx, pair.QuoteDenom(), traderAddr, marginToVault.Abs()); err != nil {
			return err
		}
	}

	transferredFee, err := k.transferFee(ctx, pair, traderAddr, positionResp.ExchangedNotionalValue)
	if err != nil {
		return err
	}

	markPrice, err := k.VpoolKeeper.GetMarkPrice(ctx, pair)
	if err != nil {
		return err
	}

	// calculate positionNotional (it's different depends on long or short side)
	// long: unrealizedPnl = positionNotional - openNotional => positionNotional = openNotional + unrealizedPnl
	// short: unrealizedPnl = openNotional - positionNotional => positionNotional = openNotional - unrealizedPnl
	positionNotional := sdk.ZeroDec()
	if positionResp.Position.Size_.IsPositive() {
		positionNotional = positionResp.Position.OpenNotional.Add(positionResp.UnrealizedPnlAfter)
	} else if positionResp.Position.Size_.IsNegative() {
		positionNotional = positionResp.Position.OpenNotional.Sub(positionResp.UnrealizedPnlAfter)
	}

	return ctx.EventManager().EmitTypedEvent(&types.PositionChangedEvent{
		TraderAddress:      traderAddr.String(),
		Pair:               pair,
		Margin:             sdk.NewCoin(pair.QuoteDenom(), positionResp.Position.Margin.RoundInt()),
		PositionNotional:   positionNotional,
		ExchangedNotional:  positionResp.ExchangedNotionalValue,
		ExchangedSize:      positionResp.ExchangedPositionSize,
		TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), transferredFee),
		PositionSize:       positionResp.Position.Size_,
		RealizedPnl:        positionResp.RealizedPnl,
		UnrealizedPnlAfter: positionResp.UnrealizedPnlAfter,
		BadDebt:            sdk.NewCoin(pair.QuoteDenom(), positionResp.BadDebt.RoundInt()),
		MarkPrice:          markPrice,
		FundingPayment:     positionResp.FundingPayment,
		BlockHeight:        ctx.BlockHeight(),
		BlockTimeMs:        ctx.BlockTime().UnixMilli(),
	})
}

/*
increases a position by increasedNotional amount in margin units.
Calculates the amount of margin required given the leverage parameter.
Recalculates the remaining margin after applying a funding payment.
Does not realize PnL.

For example, a long position with position notional value of 150 NUSD and unrealized PnL of 50 NUSD
could increase their position by 30 NUSD using 10x leverage.
This would be:
  - 3 NUSD as margin requirement
  - new open notional value of 130 NUSD
  - new position notional value of 150 NUSD
  - unrealized PnL remains unchanged at 50 NUSD
  - remaining margin is calculated by applying the funding payment

args:
  - ctx: cosmos-sdk context
  - currentPosition: the current position
  - side: whether the position is increasing in the BUY or SELL direction
  - increasedNotional: the notional value to increase the position by, in margin units
  - baseAmtLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units
  - leverage: the amount of leverage to take, as sdk.Dec

ret:
  - positionResp: contains the result of the increase position and the new position
  - err: error
*/
func (k Keeper) increasePosition(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	currentPosition types.Position,
	side types.Side,
	increasedNotional sdk.Dec,
	baseAmtLimit sdk.Dec,
	leverage sdk.Dec,
) (updatedVpool perpammtypes.Vpool, positionResp *types.PositionResp, err error) {
	positionResp = &types.PositionResp{}

	updatedVpool, positionResp.ExchangedPositionSize, err = k.swapQuoteForBase(
		ctx,
		vpool,
		side,
		increasedNotional,
		baseAmtLimit,
		/* skipFluctuationLimitCheck */ false,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	increaseMarginRequirement := increasedNotional.Quo(leverage)

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		currentPosition,
		increaseMarginRequirement,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionNotional, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		updatedVpool,
		currentPosition,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp.ExchangedNotionalValue = increasedNotional
	positionResp.PositionNotional = positionNotional.Add(increasedNotional)
	positionResp.UnrealizedPnlAfter = unrealizedPnL
	positionResp.RealizedPnl = sdk.ZeroDec()
	positionResp.MarginToVault = increaseMarginRequirement
	positionResp.FundingPayment = remaining.FundingPayment
	positionResp.BadDebt = remaining.BadDebt
	positionResp.Position = &types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                          remaining.Margin,
		OpenNotional:                    currentPosition.OpenNotional.Add(increasedNotional),
		LatestCumulativePremiumFraction: remaining.LatestCumulativePremiumFraction,
		BlockNumber:                     ctx.BlockHeight(),
	}

	return updatedVpool, positionResp, nil
}

// TODO test: openReversePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) openReversePosition(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	currentPosition types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (updatedVpool perpammtypes.Vpool, positionResp *types.PositionResp, err error) {
	notionalToDecreaseBy := leverage.Mul(quoteAssetAmount)
	currentPositionNotional, _, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		vpool,
		currentPosition,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	if currentPositionNotional.GT(notionalToDecreaseBy) {
		// position reduction
		return k.decreasePosition(
			ctx,
			vpool,
			currentPosition,
			notionalToDecreaseBy,
			baseAmtLimit,
			/* skipFluctuationLimitCheck */ false,
		)
	} else {
		// close and reverse
		return k.closeAndOpenReversePosition(
			ctx,
			vpool,
			currentPosition,
			quoteAssetAmount,
			leverage,
			baseAmtLimit,
		)
	}
}

/*
Decreases a position by decreasedNotional amount in margin units.
Realizes PnL and calculates remaining margin after applying a funding payment.

For example, a long position with position notional value of 150 NUSD and PnL of 50 NUSD
could decrease their position by 30 NUSD. This would realize a PnL of 10 NUSD (50NUSD * 30/150)
and update their margin (old margin + realized PnL - funding payment).
Their new position notional value would be 120 NUSD and their position size would
shrink by 20%.

args:
  - ctx: cosmos-sdk context
  - currentPosition: the current position
  - decreasedNotional: the notional value to decrease the position by, in margin units
  - baseAmtLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units
  - skipFluctuationLimitCheck: whether or not the position change can go over the fluctuation limit

ret:
  - positionResp: contains the result of the decrease position and the new position
  - err: error
*/
func (k Keeper) decreasePosition(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	currentPosition types.Position,
	decreasedNotional sdk.Dec,
	baseAmtLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedVpool perpammtypes.Vpool, positionResp *types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return perpammtypes.Vpool{}, nil, fmt.Errorf("current position size is zero, nothing to decrease")
	}

	positionResp = &types.PositionResp{
		MarginToVault: sdk.ZeroDec(),
	}

	currentPositionNotional, currentUnrealizedPnL, err := k.
		getPositionNotionalAndUnrealizedPnL(
			ctx,
			vpool,
			currentPosition,
			types.PnLCalcOption_SPOT_PRICE,
		)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	var sideToTake types.Side
	// flipped since we are going against the current position
	if currentPosition.Size_.IsPositive() {
		sideToTake = types.Side_SELL
	} else {
		sideToTake = types.Side_BUY
	}

	updatedVpool, positionResp.ExchangedPositionSize, err = k.swapQuoteForBase(
		ctx,
		vpool,
		sideToTake,
		decreasedNotional,
		baseAmtLimit,
		skipFluctuationLimitCheck,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp.RealizedPnl = currentUnrealizedPnL.Mul(
		positionResp.ExchangedPositionSize.Abs().
			Quo(currentPosition.Size_.Abs()),
	)

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		currentPosition,
		positionResp.RealizedPnl,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp.BadDebt = remaining.BadDebt
	positionResp.FundingPayment = remaining.FundingPayment
	positionResp.UnrealizedPnlAfter = currentUnrealizedPnL.Sub(positionResp.RealizedPnl)
	positionResp.ExchangedNotionalValue = decreasedNotional
	positionResp.PositionNotional = currentPositionNotional.Sub(decreasedNotional)

	// calculate openNotional (it's different depends on long or short side)
	// long: unrealizedPnl = positionNotional - openNotional => openNotional = positionNotional - unrealizedPnl
	// short: unrealizedPnl = openNotional - positionNotional => openNotional = positionNotional + unrealizedPnl
	// positionNotional = oldPositionNotional - notionalValueToDecrease
	var remainOpenNotional sdk.Dec
	if currentPosition.Size_.IsPositive() {
		remainOpenNotional = positionResp.PositionNotional.Sub(positionResp.UnrealizedPnlAfter)
	} else {
		remainOpenNotional = positionResp.PositionNotional.Add(positionResp.UnrealizedPnlAfter)
	}

	if remainOpenNotional.IsNegative() {
		return perpammtypes.Vpool{}, nil, fmt.Errorf("value of open notional < 0")
	}

	positionResp.Position = &types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                          remaining.Margin,
		OpenNotional:                    remainOpenNotional,
		LatestCumulativePremiumFraction: remaining.LatestCumulativePremiumFraction,
		BlockNumber:                     ctx.BlockHeight(),
	}

	return updatedVpool, positionResp, nil
}

/*
Closes a position and realizes PnL and funding payments.
Opens a position in the opposite direction if there is notional value remaining.
Errors out if the provided notional value is not greater than the existing position's notional value.
Errors out if there is bad debt.

args:
  - ctx: cosmos-sdk context
  - existingPosition: current position
  - quoteAssetAmount: the amount of notional value to move by. Must be greater than the existingPosition's notional value.
  - leverage: the amount of leverage to take
  - baseAmtLimit: limit on the base asset movement to ensure trader doesn't get screwed
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - positionResp: response object containing information about the position change
  - err: error
*/
func (k Keeper) closeAndOpenReversePosition(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	existingPosition types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (updatedVpool perpammtypes.Vpool, positionResp *types.PositionResp, err error) {
	trader, err := sdk.AccAddressFromBech32(existingPosition.TraderAddress)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	updatedVpool, closePositionResp, err := k.closePositionEntirely(
		ctx,
		vpool,
		existingPosition,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ false,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	if closePositionResp.BadDebt.IsPositive() {
		return perpammtypes.Vpool{}, nil, fmt.Errorf("underwater position")
	}

	reverseNotionalValue := leverage.Mul(quoteAssetAmount)
	remainingReverseNotionalValue := reverseNotionalValue.Sub(
		closePositionResp.ExchangedNotionalValue)

	var increasePositionResp *types.PositionResp
	if remainingReverseNotionalValue.IsNegative() {
		// should never happen as this should also be checked in the caller
		return perpammtypes.Vpool{}, nil, fmt.Errorf(
			"provided quote asset amount and leverage not large enough to close position. need %s but got %s",
			closePositionResp.ExchangedNotionalValue.String(), reverseNotionalValue.String())
	} else if remainingReverseNotionalValue.IsPositive() {
		updatedBaseAmtLimit := baseAmtLimit
		if baseAmtLimit.IsPositive() {
			updatedBaseAmtLimit = baseAmtLimit.Sub(closePositionResp.ExchangedPositionSize.Abs())
		}
		if updatedBaseAmtLimit.IsNegative() {
			return perpammtypes.Vpool{}, nil, fmt.Errorf(
				"position size changed by greater than the specified base limit: %s",
				baseAmtLimit.String(),
			)
		}

		var sideToTake types.Side
		// flipped since we are going against the current position
		if existingPosition.Size_.IsPositive() {
			sideToTake = types.Side_SELL
		} else {
			sideToTake = types.Side_BUY
		}

		newPosition := types.ZeroPosition(
			ctx,
			existingPosition.Pair,
			trader,
		)
		updatedVpool, increasePositionResp, err = k.increasePosition(
			ctx,
			updatedVpool,
			newPosition,
			sideToTake,
			remainingReverseNotionalValue,
			updatedBaseAmtLimit,
			leverage,
		)
		if err != nil {
			return perpammtypes.Vpool{}, nil, err
		}

		positionResp = &types.PositionResp{
			Position:               increasePositionResp.Position,
			PositionNotional:       increasePositionResp.PositionNotional,
			ExchangedNotionalValue: closePositionResp.ExchangedNotionalValue.Add(increasePositionResp.ExchangedNotionalValue),
			BadDebt:                closePositionResp.BadDebt.Add(increasePositionResp.BadDebt),
			ExchangedPositionSize:  closePositionResp.ExchangedPositionSize.Add(increasePositionResp.ExchangedPositionSize),
			FundingPayment:         closePositionResp.FundingPayment.Add(increasePositionResp.FundingPayment),
			RealizedPnl:            closePositionResp.RealizedPnl.Add(increasePositionResp.RealizedPnl),
			MarginToVault:          closePositionResp.MarginToVault.Add(increasePositionResp.MarginToVault),
			UnrealizedPnlAfter:     sdk.ZeroDec(),
		}
	} else {
		// case where remaining open notional == 0
		positionResp = closePositionResp
	}

	return updatedVpool, positionResp, nil
}

/*
Closes a position and realizes PnL and funding payments.
Does not error out if there is bad debt, that is for callers to decide.

args:
  - ctx: cosmos-sdk context
  - currentPosition: current position
  - quoteAssetAmountLimit: a limit on quote asset to ensure trader doesn't get screwed
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - positionResp: response object containing information about the position change
  - err: error
*/
func (k Keeper) closePositionEntirely(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	currentPosition types.Position,
	quoteAssetAmountLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedVpool perpammtypes.Vpool, positionResp *types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return perpammtypes.Vpool{}, nil, fmt.Errorf("zero position size")
	}

	trader, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp = &types.PositionResp{
		UnrealizedPnlAfter:    sdk.ZeroDec(),
		ExchangedPositionSize: currentPosition.Size_.Neg(),
		PositionNotional:      sdk.ZeroDec(),
	}

	// calculate unrealized PnL
	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		vpool,
		currentPosition,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp.RealizedPnl = unrealizedPnL
	// calculate remaining margin with funding payment
	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx, currentPosition, unrealizedPnL)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp.BadDebt = remaining.BadDebt
	positionResp.FundingPayment = remaining.FundingPayment
	positionResp.MarginToVault = remaining.Margin.Neg()

	var sideToTake types.Side
	// flipped since we are going against the current position
	if currentPosition.Size_.IsPositive() {
		sideToTake = types.Side_SELL
	} else {
		sideToTake = types.Side_BUY
	}
	updatedVpool, exchangedNotionalValue, err := k.swapBaseForQuote(
		ctx,
		vpool,
		sideToTake,
		currentPosition.Size_.Abs(),
		quoteAssetAmountLimit,
		skipFluctuationLimitCheck,
	)
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	positionResp.ExchangedNotionalValue = exchangedNotionalValue
	positionResp.Position = &types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.ZeroDec(),
		OpenNotional:                    sdk.ZeroDec(),
		LatestCumulativePremiumFraction: remaining.LatestCumulativePremiumFraction,
		BlockNumber:                     ctx.BlockHeight(),
	}

	err = k.Positions.Delete(ctx, collections.Join(currentPosition.Pair, trader))
	if err != nil {
		return perpammtypes.Vpool{}, nil, err
	}

	return updatedVpool, positionResp, nil
}

/*
ClosePosition closes a position entirely and transfers the remaining margin back to the user.
Errors if the position has bad debt.

args:
  - ctx: the cosmos-sdk context
  - pair: the trading pair
  - traderAddr: the trader's address

ret:
  - positionResp: the response containing the updated position and applied funding payment, bad debt, PnL
  - err: error if any
*/
func (k Keeper) ClosePosition(ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress) (*types.PositionResp, error) {
	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	vpool, err := k.VpoolKeeper.GetPool(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	updatedVpool, positionResp, err := k.closePositionEntirely(
		ctx,
		vpool,
		position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ false,
	)
	if err != nil {
		return nil, err
	}

	if positionResp.BadDebt.IsPositive() {
		return nil, fmt.Errorf("underwater position")
	}

	if err = k.afterPositionUpdate(
		ctx,
		updatedVpool,
		traderAddr,
		*positionResp,
	); err != nil {
		return nil, err
	}

	return positionResp, nil
}

func (k Keeper) transferFee(
	ctx sdk.Context,
	pair asset.Pair,
	trader sdk.AccAddress,
	positionNotional sdk.Dec,
) (fees sdk.Int, err error) {
	params := k.GetParams(ctx)
	feeToFeePool := params.FeePoolFeeRatio.Mul(positionNotional).RoundInt()
	if feeToFeePool.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ types.FeePoolModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToFeePool,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	feeToEcosystemFund := params.EcosystemFundFeeRatio.Mul(positionNotional).RoundInt()
	if feeToEcosystemFund.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ types.PerpEFModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToEcosystemFund,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	return feeToFeePool.Add(feeToEcosystemFund), nil
}

/*
Trades quoteAssets in exchange for baseAssets.
The quote asset is a stablecoin like NUSD.
The base asset is a crypto asset like BTC or ETH.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - quoteAssetAmount: the amount of quote asset being traded
  - baseAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to check if the swapped amount is over the fluctuation limit. Currently unused.

ret:
  - baseAssetAmount: the amount of base asset swapped
  - err: error
*/
func (k Keeper) swapQuoteForBase(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	side types.Side,
	quoteAssetAmount sdk.Dec,
	baseAssetLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedVpool perpammtypes.Vpool, baseAssetAmount sdk.Dec, err error) {
	var quoteAssetDirection perpammtypes.Direction
	if side == types.Side_BUY {
		quoteAssetDirection = perpammtypes.Direction_ADD_TO_POOL
	} else {
		// side == types.Side_SELL
		quoteAssetDirection = perpammtypes.Direction_REMOVE_FROM_POOL
	}

	updatedVpool, baseAssetAmount, err = k.VpoolKeeper.SwapQuoteForBase(
		ctx, vpool, quoteAssetDirection, quoteAssetAmount, baseAssetLimit, skipFluctuationLimitCheck)
	if err != nil {
		return perpammtypes.Vpool{}, sdk.Dec{}, err
	}
	if side == types.Side_SELL {
		baseAssetAmount = baseAssetAmount.Neg()
	}
	k.OnSwapEnd(ctx, vpool.Pair, quoteAssetAmount, baseAssetAmount)
	return updatedVpool, baseAssetAmount, nil
}

/*
Trades baseAssets in exchange for quoteAssets.
The base asset is a crypto asset like BTC.
The quote asset is a stablecoin like NUSD.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - baseAssetAmount: the amount of quote asset being traded
  - quoteAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - quoteAssetAmount: the amount of quote asset swapped
  - err: error
*/
func (k Keeper) swapBaseForQuote(
	ctx sdk.Context,
	vpool perpammtypes.Vpool,
	side types.Side,
	baseAssetAmount sdk.Dec,
	quoteAssetLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedVpool perpammtypes.Vpool, baseAmount sdk.Dec, err error) {
	var baseAssetDirection perpammtypes.Direction
	if side == types.Side_SELL {
		baseAssetDirection = perpammtypes.Direction_ADD_TO_POOL
	} else {
		// side == types.Side_BUY
		baseAssetDirection = perpammtypes.Direction_REMOVE_FROM_POOL
	}

	updatedVpool, quoteAssetAmount, err := k.VpoolKeeper.SwapBaseForQuote(
		ctx, vpool, baseAssetDirection, baseAssetAmount, quoteAssetLimit, skipFluctuationLimitCheck)
	if err != nil {
		return perpammtypes.Vpool{}, sdk.Dec{}, err
	}

	if side == types.Side_SELL {
		baseAssetAmount = baseAssetAmount.Neg()
	}

	k.OnSwapEnd(ctx, vpool.Pair, quoteAssetAmount, baseAssetAmount)
	return updatedVpool, quoteAssetAmount, err
}

// OnSwapEnd recalculates perp metrics for a particular pair.
func (k Keeper) OnSwapEnd(
	ctx sdk.Context,
	pair asset.Pair,
	quoteAssetAmount sdk.Dec,
	baseAssetAmount sdk.Dec,
) {
	// Update Metrics
	metrics := k.Metrics.GetOr(ctx, pair, types.Metrics{
		Pair:        pair,
		NetSize:     sdk.ZeroDec(),
		VolumeQuote: sdk.ZeroDec(),
		VolumeBase:  sdk.ZeroDec(),
	})
	metrics.NetSize = metrics.NetSize.Add(baseAssetAmount)
	metrics.VolumeBase = metrics.VolumeBase.Add(baseAssetAmount.Abs())
	metrics.VolumeQuote = metrics.VolumeQuote.Add(quoteAssetAmount.Abs())
	k.Metrics.Insert(ctx, pair, metrics)
}
