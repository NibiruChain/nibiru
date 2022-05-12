package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	/* TODO tests | These _ vars are here to pass the golangci-lint for unused methods.
	They also serve as a reminder of which functions still need MVP unit or
	integration tests */
	_ = requireMoreMarginRatio
)

func (k Keeper) AddMargin(
	ctx sdk.Context,
	pair common.TokenPair,
	trader sdk.AccAddress,
	addedMargin sdk.Dec,
) (err error) {
	position, err := k.Positions().Get(ctx, pair, trader.String())
	if err != nil {
		return err
	}

	position.Margin = position.Margin.Add(addedMargin)

	if err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx,
		trader,
		types.ModuleName,
		sdk.NewCoins(
			sdk.NewCoin(common.StableDenom, addedMargin.TruncateInt()),
		),
	); err != nil {
		return err
	}

	k.Positions().Set(ctx, pair, trader.String(), position)

	return nil
}

// TODO test: GetMarginRatio
func (k Keeper) GetMarginRatio(
	ctx sdk.Context, pair common.TokenPair, trader string,
) (sdk.Dec, error) {
	position, err := k.Positions().Get(ctx, pair, trader) // TODO(mercilex): inefficient position get
	if err != nil {
		return sdk.Dec{}, err
	}

	if position.Size_.IsZero() {
		panic("position with zero size") // tODO(mercilex): panic or error? this is a require
	}

	unrealizedPnL, positionNotional, err := k.getPreferencePositionNotionalAndUnrealizedPnL(
		ctx,
		pair,
		trader,
		types.PnLPreferenceOption_MAX,
	)
	if err != nil {
		return sdk.Dec{}, err
	}

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		/* pair */ pair,
		/* oldPosition */ position,
		/* marginDelta */ unrealizedPnL,
	)
	if err != nil {
		return sdk.Dec{}, err
	}

	marginRatio := remaining.margin.Sub(remaining.badDebt).Quo(positionNotional)
	return marginRatio, err
}

/*
requireMoreMarginRatio checks if the marginRatio corresponding to the margin
backing a position is above or below the 'baseMarginRatio'.
If 'largerThanOrEqualTo' is true, 'marginRatio' must be >= 'baseMarginRatio'.

Args:
  marginRatio: Ratio of the value of the margin and corresponding position(s).
    marginRatio is defined as (margin + unrealizedPnL) / notional
  baseMarginRatio: Specifies the threshold value that 'marginRatio' must meet.
  largerThanOrEqualTo: Specifies whether 'marginRatio' should be larger or
    smaller than 'baseMarginRatio'.
*/
func requireMoreMarginRatio(marginRatio, baseMarginRatio sdk.Dec, largerThanOrEqualTo bool) error {
	switch largerThanOrEqualTo {
	case true:
		if !marginRatio.GTE(baseMarginRatio) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	default:
		if !marginRatio.LT(baseMarginRatio) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	}

	return nil
}

type Remaining struct {
	// margin sdk.Int: amount of quote token (y) backing the position.
	margin sdk.Dec

	/* badDebt sdk.Int: Bad debt (margin units) cleared by the PerpEF during the tx.
	   Bad debt is negative net margin past the liquidation point of a position. */
	badDebt sdk.Dec

	/* fundingPayment sdk.Dec: A funding payment made or received by the trader on
	    the current position. 'fundingPayment' is positive if 'owner' is the sender
		and negative if 'owner' is the receiver of the payment. Its magnitude is
		abs(vSize * fundingRate). Funding payments act to converge the mark price
		(vPrice) and index price (average price on major exchanges). */
	fPayment sdk.Dec

	/* latestCPF: latest cumulative premium fraction */
	latestCPF sdk.Dec
}

// TODO test: CalcRemainMarginWithFundingPayment | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) CalcRemainMarginWithFundingPayment(
	ctx sdk.Context, pair common.TokenPair,
	oldPosition *types.Position, marginDelta sdk.Dec,
) (remaining Remaining, err error) {
	remaining.latestCPF, err = k.getLatestCumulativePremiumFraction(ctx, pair)
	if err != nil {
		return
	}

	if oldPosition.Size_.IsZero() {
		remaining.fPayment = remaining.latestCPF.
			Sub(oldPosition.LastUpdateCumulativePremiumFraction).
			Mul(oldPosition.Size_)
	} else {
		remaining.fPayment = sdk.ZeroDec()
	}

	signedRemainMargin := marginDelta.Sub(remaining.fPayment).Add(oldPosition.Margin)

	if signedRemainMargin.IsNegative() {
		// the remaining margin is negative, liquidators didn't do their job
		// and we have negative margin that must come out of the ecosystem fund
		remaining.badDebt = signedRemainMargin.Abs()
	} else {
		remaining.badDebt = sdk.ZeroDec()
		remaining.margin = signedRemainMargin.Abs()
	}

	return remaining, err
}
