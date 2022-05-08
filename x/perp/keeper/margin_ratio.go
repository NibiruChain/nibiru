package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO test: GetMarginRatio
func (k Keeper) GetMarginRatio(ctx sdk.Context, amm types.IVirtualPool, trader string) (sdk.Int, error) {
	position, err := k.Positions().Get(ctx, amm.Pair(), trader) // TODO(mercilex): inefficient position get
	if err != nil {
		return sdk.Int{}, err
	}

	if position.Size_.IsZero() {
		panic("position with zero size") // tODO(mercilex): panic or error? this is a require
	}

	unrealizedPnL, positionNotional, err := k.getPreferencePositionNotionalAndUnrealizedPnL(ctx, amm, trader, types.PnLPreferenceOption_MAX)
	if err != nil {
		return sdk.Int{}, err
	}

	return k._GetMarginRatio(ctx, amm, position, unrealizedPnL, positionNotional)
}

// TODO test: _GetMarginRatio
func (k Keeper) _GetMarginRatio(ctx sdk.Context, amm types.IVirtualPool, position *types.Position, unrealizedPnL, positionNotional sdk.Int) (sdk.Int, error) {
	// todo(mercilex): maybe inefficient re-get
	remainMargin, badDebt, _, _, err := k.calcRemainMarginWithFundingPayment(ctx, amm, position, unrealizedPnL)
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
