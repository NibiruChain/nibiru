package v2

import (
	fmt "fmt"

	"github.com/NibiruChain/nibiru/x/common/asset"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ZeroPosition(ctx sdk.Context, tokenPair asset.Pair, traderAddr sdk.AccAddress) Position {
	return Position{
		TraderAddress:                   traderAddr.String(),
		Pair:                            tokenPair,
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.ZeroDec(),
		OpenNotional:                    sdk.ZeroDec(),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}
}

func PositionsAreEqual(expected, actual *Position) error {
	if expected.Pair != actual.Pair {
		return fmt.Errorf("expected position pair %s, got %s", expected.Pair, actual.Pair)
	}

	if expected.TraderAddress != actual.TraderAddress {
		return fmt.Errorf("expected position trader address %s, got %s", expected.TraderAddress, actual.TraderAddress)
	}

	if !expected.Margin.Equal(actual.Margin) {
		return fmt.Errorf("expected position margin %s, got %s", expected.Margin, actual.Margin)
	}

	if !expected.OpenNotional.Equal(actual.OpenNotional) {
		return fmt.Errorf("expected position open notional %s, got %s", expected.OpenNotional, actual.OpenNotional)
	}

	if !expected.Size_.Equal(actual.Size_) {
		return fmt.Errorf("expected position size %s, got %s", expected.Size_, actual.Size_)
	}

	if expected.LastUpdatedBlockNumber != actual.LastUpdatedBlockNumber {
		return fmt.Errorf("expected position block number %d, got %d", expected.LastUpdatedBlockNumber, actual.LastUpdatedBlockNumber)
	}

	if !expected.LatestCumulativePremiumFraction.Equal(actual.LatestCumulativePremiumFraction) {
		return fmt.Errorf(
			"expected position latest cumulative premium fraction %s, got %s",
			expected.LatestCumulativePremiumFraction,
			actual.LatestCumulativePremiumFraction,
		)
	}
	return nil
}

type PositionResp struct {
	Position *Position
	// The amount of quote assets exchanged.
	ExchangedNotionalValue sdk.Dec
	// The amount of base assets exchanged.
	ExchangedPositionSize sdk.Dec
	// The amount of bad debt accrued during this position change.
	// Measured in absolute value of quote units.
	// If greater than zero, then the position change event will likely fail.
	BadDebt sdk.Dec
	// The funding payment applied on this position change.
	FundingPayment sdk.Dec
	// The amount of PnL realized on this position changed, measured in quote
	// units.
	RealizedPnl sdk.Dec
	// The unrealized PnL in the position after the position change.
	UnrealizedPnlAfter sdk.Dec
	// The amount of margin the trader has to give to the vault.
	// A negative value means the vault pays the trader.
	MarginToVault sdk.Dec
	// The position's notional value after the position change, measured in quote
	// units.
	PositionNotional sdk.Dec
}

type LiquidateResp struct {
	// Amount of bad debt created by the liquidation event
	BadDebt sdk.Int
	// Fee paid to the liquidator
	FeeToLiquidator sdk.Int
	// Fee paid to the Perp EF fund
	FeeToPerpEcosystemFund sdk.Int
	// Address of the liquidator
	Liquidator string
	// Position response from the close or open reverse position
	PositionResp *PositionResp
}

func (l *LiquidateResp) Validate() error {
	nilFieldError := fmt.Errorf("invalid liquidationOutput, must not have nil fields")

	// nil sdk.Int check
	for _, field := range []sdk.Int{
		l.FeeToLiquidator, l.FeeToPerpEcosystemFund} {
		if field.IsNil() {
			return nilFieldError
		}
	}

	// nil sdk.Int check
	for _, field := range []sdk.Int{l.BadDebt} {
		if field.IsNil() {
			return nilFieldError
		}
	}

	if _, err := sdk.AccAddressFromBech32(l.Liquidator); err != nil {
		return err
	}

	return nil
}
