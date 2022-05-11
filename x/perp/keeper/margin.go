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
	addedMargin sdk.Int,
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
			sdk.NewCoin(common.StableDenom, addedMargin),
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
) (sdk.Int, error) {
	position, err := k.Positions().Get(ctx, pair, trader) // TODO(mercilex): inefficient position get
	if err != nil {
		return sdk.Int{}, err
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
		return sdk.Int{}, err
	}

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		/* pair */ pair,
		/* oldPosition */ position,
		/* marginDelta */ unrealizedPnL,
	)
	if err != nil {
		return sdk.Int{}, err
	}

	marginRatio := remaining.margin.Sub(remaining.badDebt).Quo(positionNotional)
	return marginRatio, err
}

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

type Remaining struct {
	// margin sdk.Int: amount of quote token (y) backing the position.
	margin sdk.Int

	/* badDebt sdk.Int: Bad debt (margin units) cleared by the PerpEF during the tx.
	   Bad debt is negative net margin past the liquidation point of a position. */
	badDebt sdk.Int

	/* fundingPayment sdk.Dec: A funding payment made or received by the trader on
	    the current position. 'fundingPayment' is positive if 'owner' is the sender
		and negative if 'owner' is the receiver of the payment. Its magnitude is
		abs(vSize * fundingRate). Funding payments act to converge the mark price
		(vPrice) and index price (average price on major exchanges). */
	fPayment sdk.Int

	/* latestCPF: latest cumulative premium fraction */
	latestCPF sdk.Int
}

// TODO test: CalcRemainMarginWithFundingPayment | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) CalcRemainMarginWithFundingPayment(
	ctx sdk.Context, pair common.TokenPair,
	oldPosition *types.Position, marginDelta sdk.Int,
) (remaining Remaining, err error) {
	remaining.latestCPF, err = k.GetLatestCumulativePremiumFraction(ctx, pair)
	if err != nil {
		return
	}

	if oldPosition.Size_.IsZero() {
		remaining.fPayment = remaining.latestCPF.
			Sub(oldPosition.LastUpdateCumulativePremiumFraction).
			Mul(oldPosition.Size_)
	} else {
		remaining.fPayment = sdk.ZeroInt()
	}

	signedRemainMargin := marginDelta.Sub(remaining.fPayment).Add(oldPosition.Margin)

	if signedRemainMargin.IsNegative() {
		// the remaining margin is negative, liquidators didn't do their job
		// and we have negative margin that must come out of the ecosystem fund
		remaining.badDebt = signedRemainMargin.Abs()
	} else {
		remaining.badDebt = sdk.ZeroInt()
		remaining.margin = signedRemainMargin.Abs()
	}

	return remaining, err
}
