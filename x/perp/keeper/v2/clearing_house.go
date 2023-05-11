package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// TODO(k-yang): remove when we add OpenPosition
var _ = checkOpenPositionRequirements

// checkOpenPositionRequirements checks the minimum requirements to open a position.
//
// - Checks that quote asset is not zero.
// - Checks that leverage is not zero.
// - Checks that leverage is below requirement.
//
// args:
// - market: the market where the position will be opened
// - quoteAssetAmt: the amount of quote asset
// - leverage: the amount of leverage to take, as sdk.Dec
//
// returns:
// - error: if any of the requirements is not met
func checkOpenPositionRequirements(market v2types.Market, quoteAssetAmt sdk.Int, leverage sdk.Dec) error {
	if quoteAssetAmt.IsZero() {
		return v2types.ErrQuoteAmountIsZero
	}

	if leverage.IsZero() {
		return v2types.ErrLeverageIsZero
	}

	if leverage.GT(market.MaxLeverage) {
		return v2types.ErrLeverageIsTooHigh
	}

	return nil
}

// afterPositionUpdate is called when a position has been updated.
func (k Keeper) afterPositionUpdate(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	traderAddr sdk.AccAddress,
	positionResp v2types.PositionResp,
) (err error) {
	// check bad debt
	if !positionResp.BadDebt.IsZero() {
		return fmt.Errorf("bad debt must be zero to prevent attacker from leveraging it")
	}

	// check price fluctuation
	if err := k.checkPriceFluctuationLimitRatio(ctx, market, amm); err != nil {
		return err
	}

	if !positionResp.Position.Size_.IsZero() {
		k.Positions.Insert(ctx, collections.Join(market.Pair, traderAddr), *positionResp.Position)

		spotNotional, err := PositionNotionalSpot(amm, *positionResp.Position)
		if err != nil {
			return err
		}
		twapNotional, err := k.PositionNotionalTWAP(ctx, *positionResp.Position, market.TwapLookbackWindow)
		if err != nil {
			return err
		}
		positionNotional := sdk.MaxDec(spotNotional, twapNotional)

		marginRatio := MarginRatio(*positionResp.Position, positionNotional, market.LatestCumulativePremiumFraction)
		if marginRatio.LT(market.MaintenanceMarginRatio) {
			return v2types.ErrMarginRatioTooLow
		}
	}

	// transfer trader <=> vault
	marginToVault := positionResp.MarginToVault.RoundInt()
	switch {
	case marginToVault.IsPositive():
		coinToSend := sdk.NewCoin(market.Pair.QuoteDenom(), marginToVault)
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, traderAddr, v2types.VaultModuleAccount, sdk.NewCoins(coinToSend)); err != nil {
			return err
		}
	case marginToVault.IsNegative():
		if err = k.Withdraw(ctx, market, traderAddr, marginToVault.Abs()); err != nil {
			return err
		}
	}

	transferredFee, err := k.transferFee(ctx, market.Pair, traderAddr, positionResp.ExchangedNotionalValue)
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

	return ctx.EventManager().EmitTypedEvent(&v2types.PositionChangedEvent{
		TraderAddress:      traderAddr.String(),
		Pair:               market.Pair,
		Margin:             sdk.NewCoin(market.Pair.QuoteDenom(), positionResp.Position.Margin.RoundInt()),
		PositionNotional:   positionNotional,
		ExchangedNotional:  positionResp.ExchangedNotionalValue,
		ExchangedSize:      positionResp.ExchangedPositionSize,
		TransactionFee:     sdk.NewCoin(market.Pair.QuoteDenom(), transferredFee),
		PositionSize:       positionResp.Position.Size_,
		RealizedPnl:        positionResp.RealizedPnl,
		UnrealizedPnlAfter: positionResp.UnrealizedPnlAfter,
		BadDebt:            sdk.NewCoin(market.Pair.QuoteDenom(), positionResp.BadDebt.RoundInt()),
		FundingPayment:     positionResp.FundingPayment,
		BlockHeight:        ctx.BlockHeight(),
		BlockTimeMs:        ctx.BlockTime().UnixMilli(),
	})
}

// transfers the fee to the exchange fee pool
//
// args:
// - ctx: the cosmos-sdk context
// - pair: the trading pair
// - trader: the trader's address
// - positionNotional: the position's notional value
//
// returns:
// - fees: the fees to be transferred
// - err: error if any
func (k Keeper) transferFee(
	ctx sdk.Context,
	pair asset.Pair,
	trader sdk.AccAddress,
	positionNotional sdk.Dec,
) (fees sdk.Int, err error) {
	m, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	feeToExchangeFeePool := m.ExchangeFeeRatio.Mul(positionNotional).RoundInt()
	if feeToExchangeFeePool.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ v2types.FeePoolModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToExchangeFeePool,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	feeToEcosystemFund := m.EcosystemFundFeeRatio.Mul(positionNotional).RoundInt()
	if feeToEcosystemFund.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ v2types.PerpEFModuleAccount,
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

	return feeToExchangeFeePool.Add(feeToEcosystemFund), nil
}

// checks that the mark price of the pool does not violate the fluctuation limit
//
// args:
//   - ctx: the cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//
// returns:
//   - err: error if any
func (k Keeper) checkPriceFluctuationLimitRatio(ctx sdk.Context, market v2types.Market, amm v2types.AMM) error {
	if market.PriceFluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(amm.Pair).Descending())
	defer it.Close()

	if !it.Valid() {
		return fmt.Errorf("error getting last snapshot number for pair %s", amm.Pair)
	}

	snapshotMarkPrice := it.Value().Amm.MarkPrice()
	snapshotUpperLimit := snapshotMarkPrice.Mul(sdk.OneDec().Add(market.PriceFluctuationLimitRatio))
	snapshotLowerLimit := snapshotMarkPrice.Mul(sdk.OneDec().Sub(market.PriceFluctuationLimitRatio))

	if amm.MarkPrice().GT(snapshotUpperLimit) || snapshotMarkPrice.LT(snapshotLowerLimit) {
		return v2types.ErrOverFluctuationLimit
	}

	return nil
}

// ClosePosition closes a position entirely and transfers the remaining margin back to the user.
// Errors if the position has bad debt.
//
// args:
//   - ctx: the cosmos-sdk context
//   - pair: the pair of the position
//   - traderAddr: the address of the trader
//
// returns:
//   - positionResp: response object containing information about the position change
//   - err: error if any
func (k Keeper) ClosePosition(ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress) (*v2types.PositionResp, error) {
	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return nil, v2types.ErrPairNotFound
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return nil, err
	}

	updatedAMM, positionResp, err := k.closePositionEntirely(
		ctx,
		market,
		amm,
		position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
	)
	if err != nil {
		return nil, err
	}

	if positionResp.BadDebt.IsPositive() {
		return nil, fmt.Errorf("underwater position")
	}

	if err = k.afterPositionUpdate(
		ctx,
		market,
		*updatedAMM,
		traderAddr,
		*positionResp,
	); err != nil {
		return nil, err
	}

	return positionResp, nil
}

// Closes a position and realizes PnL and funding payments.
// Does not error out if there is bad debt, that is for callers to decide.
//
// args:
//   - ctx: cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//   - currentPosition: the existing position
//   - quoteAssetAmountLimit: the user-specified limit on the quote asset reserves
//   - skipFluctuationLimitCheck: whether to skip the fluctuation check
//
// returns:
//   - updatedAMM: updated AMM reserves
//   - positionResp: response object containing information about the position change
//   - err: error
func (k Keeper) closePositionEntirely(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	currentPosition v2types.Position,
	quoteAssetAmountLimit sdk.Dec,
) (updatedAMM *v2types.AMM, positionResp *v2types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return nil, nil, fmt.Errorf("zero position size")
	}

	trader, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return nil, nil, err
	}

	positionResp = &v2types.PositionResp{
		UnrealizedPnlAfter:    sdk.ZeroDec(),
		ExchangedPositionSize: currentPosition.Size_.Neg(),
		PositionNotional:      sdk.ZeroDec(),
	}

	// calculate unrealized PnL
	positionNotional, err := PositionNotionalSpot(amm, currentPosition)
	if err != nil {
		return nil, nil, err
	}
	unrealizedPnl := UnrealizedPnl(currentPosition, positionNotional)

	positionResp.RealizedPnl = unrealizedPnl
	// calculate remaining margin with funding payment
	fundingPayment := FundingPayment(currentPosition, market.LatestCumulativePremiumFraction)
	remainingMargin := currentPosition.Margin.Add(unrealizedPnl).Sub(fundingPayment)

	if remainingMargin.IsPositive() {
		positionResp.BadDebt = sdk.ZeroDec()
		positionResp.MarginToVault = remainingMargin.Neg()
	} else {
		positionResp.BadDebt = remainingMargin.Abs()
		positionResp.MarginToVault = sdk.ZeroDec()
	}

	positionResp.FundingPayment = fundingPayment

	var sideToTake v2types.Direction
	// flipped since we are going against the current position
	if currentPosition.Size_.IsPositive() {
		sideToTake = v2types.Direction_SHORT
	} else {
		sideToTake = v2types.Direction_LONG
	}
	updatedAMM, exchangedNotionalValue, err := k.SwapBaseAsset(
		ctx,
		market,
		amm,
		sideToTake,
		currentPosition.Size_.Abs(),
		quoteAssetAmountLimit,
	)
	if err != nil {
		return nil, nil, err
	}

	positionResp.ExchangedNotionalValue = exchangedNotionalValue
	positionResp.Position = &v2types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.ZeroDec(),
		OpenNotional:                    sdk.ZeroDec(),
		LatestCumulativePremiumFraction: market.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}

	err = k.Positions.Delete(ctx, collections.Join(currentPosition.Pair, trader))
	if err != nil {
		return nil, nil, err
	}

	return updatedAMM, positionResp, nil
}
