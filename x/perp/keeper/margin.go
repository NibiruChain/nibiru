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
func (k Keeper) GetMarginRatio(ctx sdk.Context, pair common.TokenPair, trader string) (sdk.Int, error) {
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

	remainMargin, badDebt, _, _, err := k.calcRemainMarginWithFundingPayment(ctx, pair, position, unrealizedPnL)
	if err != nil {
		return sdk.Int{}, err
	}

	return remainMargin.Sub(badDebt).Quo(positionNotional), nil
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
