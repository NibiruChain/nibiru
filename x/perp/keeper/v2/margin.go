package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

/*
	AddMargin deleverages an existing position by adding margin (collateral)

to it. Adding margin increases the margin ratio of the corresponding position.
*/
func (k Keeper) AddMargin(
	ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress, margin sdk.Coin,
) (res *v2types.MsgAddMarginResponse, err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	remainingMargin := CalcRemainMarginWithFundingPayment(position, margin.Amount.ToDec(), market.LatestCumulativePremiumFraction)

	if !remainingMargin.BadDebtAbs.IsZero() {
		return nil, fmt.Errorf("failed to add margin; position has bad debt; consider adding more margin")
	}

	if err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx,
		/* from */ traderAddr,
		/* to */ types.VaultModuleAccount,
		/* amount */ sdk.NewCoins(margin),
	); err != nil {
		return nil, err
	}

	position.Margin = remainingMargin.MarginAbs
	position.LatestCumulativePremiumFraction = market.LatestCumulativePremiumFraction
	position.LastUpdatedBlockNumber = ctx.BlockHeight()
	k.Positions.Insert(ctx, collections.Join(position.Pair, traderAddr), position)

	positionNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return nil, err
	}
	unrealizedPnl := UnrealizedPnl(position, positionNotional)

	if err = ctx.EventManager().EmitTypedEvent(
		&types.PositionChangedEvent{
			Pair:               pair,
			TraderAddress:      traderAddr.String(),
			Margin:             sdk.NewCoin(pair.QuoteDenom(), position.Margin.RoundInt()),
			PositionNotional:   positionNotional,
			ExchangedNotional:  sdk.ZeroDec(),                                 // always zero when adding margin
			ExchangedSize:      sdk.ZeroDec(),                                 // always zero when adding margin
			TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
			PositionSize:       position.Size_,
			RealizedPnl:        sdk.ZeroDec(), // always zero when adding margin
			UnrealizedPnlAfter: unrealizedPnl,
			BadDebt:            sdk.NewCoin(pair.QuoteDenom(), remainingMargin.BadDebtAbs.RoundInt()), // always zero when adding margin
			FundingPayment:     remainingMargin.FundingPayment,
			MarkPrice:          amm.MarkPrice(),
			BlockHeight:        ctx.BlockHeight(),
			BlockTimeMs:        ctx.BlockTime().UnixMilli(),
		},
	); err != nil {
		return nil, err
	}

	return &v2types.MsgAddMarginResponse{
		FundingPayment: remainingMargin.FundingPayment,
		Position:       &position,
	}, nil
}

/*
	RemoveMargin further leverages an existing position by directly removing

the margin (collateral) that backs it from the vault. This also decreases the
margin ratio of the position.

Fails if the position goes underwater.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair
  - traderAddr: the trader's address
  - margin: the amount of margin to withdraw. Must be positive.

ret:
  - marginOut: the amount of margin removed
  - fundingPayment: the funding payment that was applied with this position interaction
  - err: error if any
*/
func (k Keeper) RemoveMargin(
	ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress, margin sdk.Coin,
) (marginOut sdk.Coin, fundingPayment sdk.Dec, position v2types.Position, err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, types.ErrPairNotFound
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, types.ErrPairNotFound
	}

	// ------------- RemoveMargin -------------
	position, err = k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, err
	}

	marginDelta := margin.Amount.Neg()
	remainingMargin := CalcRemainMarginWithFundingPayment(position, marginDelta.ToDec(), market.LatestCumulativePremiumFraction)
	if !remainingMargin.BadDebtAbs.IsZero() {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, types.ErrFailedRemoveMarginCanCauseBadDebt
	}

	position.Margin = remainingMargin.MarginAbs
	position.LatestCumulativePremiumFraction = market.LatestCumulativePremiumFraction

	freeCollateral, err := k.calcFreeCollateral(ctx, market, amm, position)
	if err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, err
	} else if !freeCollateral.IsPositive() {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, fmt.Errorf("not enough free collateral")
	}

	k.Positions.Insert(ctx, collections.Join(position.Pair, traderAddr), position)

	positionNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, err
	}
	unrealizedPnl := UnrealizedPnl(position, positionNotional)

	if err = k.Withdraw(ctx, market, traderAddr, margin.Amount); err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, err
	}

	if err = ctx.EventManager().EmitTypedEvent(
		&types.PositionChangedEvent{
			Pair:               pair,
			TraderAddress:      traderAddr.String(),
			Margin:             sdk.NewCoin(pair.QuoteDenom(), position.Margin.RoundInt()),
			PositionNotional:   positionNotional,
			ExchangedNotional:  sdk.ZeroDec(),                                 // always zero when removing margin
			ExchangedSize:      sdk.ZeroDec(),                                 // always zero when removing margin
			TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when removing margin
			PositionSize:       position.Size_,
			RealizedPnl:        sdk.ZeroDec(), // always zero when removing margin
			UnrealizedPnlAfter: unrealizedPnl,
			BadDebt:            sdk.NewCoin(pair.QuoteDenom(), remainingMargin.BadDebtAbs.RoundInt()), // always zero when removing margin
			FundingPayment:     remainingMargin.FundingPayment,
			MarkPrice:          amm.MarkPrice(),
			BlockHeight:        ctx.BlockHeight(),
			BlockTimeMs:        ctx.BlockTime().UnixMilli(),
		},
	); err != nil {
		return sdk.Coin{}, sdk.Dec{}, v2types.Position{}, err
	}

	return margin, remainingMargin.FundingPayment, position, nil
}

// Returns the margin ratio based on spot price.
func GetSpotMarginRatio(
	position v2types.Position,
	positionNotional sdk.Dec,
	marketLatestCumulativePremiumFraction sdk.Dec,
) (sdk.Dec, error) {
	if position.Size_.IsZero() || positionNotional.IsZero() {
		return sdk.ZeroDec(), nil
	}

	remaining := CalcRemainMarginWithFundingPayment(
		/* oldPosition */ position,
		/* marginDelta */ UnrealizedPnl(position, positionNotional),
		marketLatestCumulativePremiumFraction,
	)

	return remaining.MarginAbs.Sub(remaining.BadDebtAbs).Quo(positionNotional), nil
}

// Returns the margin ratio based on the max of twap price and spot price
func (k Keeper) GetMaxMarginRatio(
	ctx sdk.Context,
	amm v2types.AMM,
	position v2types.Position,
	twapLookbackWindow time.Duration,
	latestCumulativePremiumFraction sdk.Dec,
) (marginRatio sdk.Dec, err error) {
	if position.Size_.IsZero() {
		return sdk.ZeroDec(), nil
	}

	spotNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return sdk.Dec{}, err
	}
	twapNotional, err := k.PositionNotionalTWAP(ctx, position, twapLookbackWindow)
	if err != nil {
		return sdk.Dec{}, err
	}
	positionNotional := sdk.MaxDec(spotNotional, twapNotional)

	if positionNotional.IsZero() {
		return sdk.ZeroDec(), nil
	}

	remaining := CalcRemainMarginWithFundingPayment(
		/* oldPosition */ position,
		/* marginDelta */ UnrealizedPnl(position, positionNotional),
		latestCumulativePremiumFraction,
	)

	return remaining.MarginAbs.Sub(remaining.BadDebtAbs).Quo(positionNotional), nil
}

/*
validateMarginRatio checks if the marginRatio corresponding to the margin
backing a position is above or below the 'threshold'.
If 'largerThanOrEqualTo' is true, 'marginRatio' must be >= 'threshold'.

Args:
  - marginRatio: Ratio of the value of the margin and corresponding position(s).
    marginRatio is defined as (margin + unrealizedPnL) / notional
  - threshold: Specifies the threshold value that 'marginRatio' must meet.
    largerThanOrEqualTo: Specifies whether 'marginRatio' should be larger or
    smaller than 'threshold'.
*/
func validateMarginRatio(marginRatio, threshold sdk.Dec, largerThanOrEqualTo bool) error {
	if largerThanOrEqualTo {
		if !marginRatio.GTE(threshold) {
			return fmt.Errorf("%w: marginRatio: %s, threshold: %s",
				types.ErrMarginRatioTooLow, marginRatio, threshold)
		}
	} else {
		if !marginRatio.LT(threshold) {
			return fmt.Errorf("%w: marginRatio: %s, threshold: %s",
				types.ErrMarginRatioTooHigh, marginRatio, threshold)
		}
	}
	return nil
}
