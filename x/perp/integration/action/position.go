package action

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

// OpenPosition opens a position with the given parameters.
//
// responseCheckers are optional functions that can be used to check expected response.
func OpenPosition(
	account sdk.AccAddress,
	pair asset.Pair,
	side types.Side,
	margin sdk.Int,
	leverage sdk.Dec,
	baseLimit sdk.Dec,
	responseCheckers ...OpenPositionResponseChecker,
) testutil.Action {
	return &openPositionAction{
		Account:   account,
		Pair:      pair,
		Side:      side,
		Margin:    margin,
		Leverage:  leverage,
		BaseLimit: baseLimit,

		CheckResponse: responseCheckers,
	}
}

type OpenPositionResponseChecker func(resp *types.PositionResp) error

type openPositionAction struct {
	Account   sdk.AccAddress
	Pair      asset.Pair
	Side      types.Side
	Margin    sdk.Int
	Leverage  sdk.Dec
	BaseLimit sdk.Dec

	CheckResponse []OpenPositionResponseChecker
}

func (o openPositionAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	resp, err := app.PerpKeeper.OpenPosition(ctx, o.Pair, o.Side, o.Account, o.Margin, o.Leverage, o.BaseLimit)
	if err != nil {
		return ctx, err
	}

	if o.CheckResponse != nil {
		for _, check := range o.CheckResponse {
			err = check(resp)
			if err != nil {
				return ctx, err
			}
		}
	}

	return ctx, nil
}

// Open Position Response Checkers

// OpenPositionResp_PositionShouldBeEqual checks that the position included in the response is equal to the expected position response.
func OpenPositionResp_PositionShouldBeEqual(expected types.Position) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if err := types.PositionsAreEqual(&expected, actual.Position); err != nil {
			return err
		}

		return nil
	}
}

// OpenPositionResp_ExchangeNotionalValueShouldBeEqual checks that the exchanged notional value included in the response is equal to the expected value.
func OpenPositionResp_ExchangeNotionalValueShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.ExchangedNotionalValue.Equal(expected) {
			return fmt.Errorf("expected exchanged notional value %s, got %s", expected, actual.ExchangedNotionalValue)
		}

		return nil
	}
}

// OpenPositionResp_ExchangedPositionSizeShouldBeEqual checks that the exchanged position size included in the response is equal to the expected value.
func OpenPositionResp_ExchangedPositionSizeShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.ExchangedPositionSize.Equal(expected) {
			return fmt.Errorf("expected exchanged position size %s, got %s", expected, actual.ExchangedPositionSize)
		}

		return nil
	}
}

// OpenPositionResp_BadDebtShouldBeEqual checks that the bad debt included in the response is equal to the expected value.
func OpenPositionResp_BadDebtShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.BadDebt.Equal(expected) {
			return fmt.Errorf("expected bad debt %s, got %s", expected, actual.BadDebt)
		}

		return nil
	}
}

// OpenPositionResp_FundingPaymentShouldBeEqual checks that the funding payment included in the response is equal to the expected value.
func OpenPositionResp_FundingPaymentShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.FundingPayment.Equal(expected) {
			return fmt.Errorf("expected funding payment %s, got %s", expected, actual.FundingPayment)
		}

		return nil
	}
}

// OpenPositionResp_RealizedPnlShouldBeEqual checks that the realized pnl included in the response is equal to the expected value.
func OpenPositionResp_RealizedPnlShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.RealizedPnl.Equal(expected) {
			return fmt.Errorf("expected realized pnl %s, got %s", expected, actual.RealizedPnl)
		}

		return nil
	}
}

// OpenPositionResp_UnrealizedPnlAfterShouldBeEqual checks that the unrealized pnl after included in the response is equal to the expected value.
func OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.UnrealizedPnlAfter.Equal(expected) {
			return fmt.Errorf("expected unrealized pnl after %s, got %s", expected, actual.UnrealizedPnlAfter)
		}

		return nil
	}
}

// OpenPositionResp_MarginToVaultShouldBeEqual checks that the margin to vault included in the response is equal to the expected value.
func OpenPositionResp_MarginToVaultShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.MarginToVault.Equal(expected) {
			return fmt.Errorf("expected margin to vault %s, got %s", expected, actual.MarginToVault)
		}

		return nil
	}
}

// OpenPositionResp_PositionNotionalShouldBeEqual checks that the position notional included in the response is equal to the expected value.
func OpenPositionResp_PositionNotionalShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.PositionNotional.Equal(expected) {
			return fmt.Errorf("expected position notional %s, got %s", expected, actual.PositionNotional)
		}

		return nil
	}
}
