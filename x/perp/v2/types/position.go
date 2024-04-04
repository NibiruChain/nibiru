package types

import (
	sdkmath "cosmossdk.io/math"
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func ZeroPosition(ctx sdk.Context, tokenPair asset.Pair, traderAddr sdk.AccAddress) Position {
	return Position{
		TraderAddress:                   traderAddr.String(),
		Pair:                            tokenPair,
		Size_:                           sdkmath.LegacyZeroDec(),
		Margin:                          sdkmath.LegacyZeroDec(),
		OpenNotional:                    sdkmath.LegacyZeroDec(),
		LatestCumulativePremiumFraction: sdkmath.LegacyZeroDec(),
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
	Position Position
	// The amount of quote assets exchanged.
	ExchangedNotionalValue sdkmath.LegacyDec
	// The amount of base assets exchanged. Signed, positive represents long and negative represents short from the user's perspective.
	ExchangedPositionSize sdkmath.LegacyDec
	// The amount of bad debt accrued during this position change.
	// Measured in absolute value of quote units.
	// If greater than zero, then the position change event will likely fail.
	BadDebt sdkmath.LegacyDec
	// The funding payment applied on this position change.
	FundingPayment sdkmath.LegacyDec
	// The amount of PnL realized on this position changed, measured in quote
	// units.
	RealizedPnl sdkmath.LegacyDec
	// The unrealized PnL in the position after the position change.
	UnrealizedPnlAfter sdkmath.LegacyDec
	// The amount of margin the trader has to give to the vault.
	// A negative value means the vault pays the trader.
	MarginToVault sdkmath.LegacyDec
	// The position's notional value after the position change, measured in quote
	// units.
	PositionNotional sdkmath.LegacyDec
}

func (m *Position) Validate() error {
	if _, err := sdk.AccAddressFromBech32(m.TraderAddress); err != nil {
		return err
	}

	if err := m.Pair.Validate(); err != nil {
		return err
	}

	if m.Size_.IsZero() {
		return fmt.Errorf("zero size")
	}

	if m.Margin.IsNegative() || m.Margin.IsZero() {
		return fmt.Errorf("margin <= 0")
	}

	if m.OpenNotional.IsNegative() || m.OpenNotional.IsZero() {
		return fmt.Errorf("open notional <= 0")
	}

	if m.LastUpdatedBlockNumber < 0 {
		return fmt.Errorf("invalid block number")
	}

	return nil
}

func (position *Position) WithTraderAddress(value string) *Position {
	position.TraderAddress = value
	return position
}

func (position *Position) WithPair(value asset.Pair) *Position {
	position.Pair = value
	return position
}

func (position *Position) WithSize_(value sdkmath.LegacyDec) *Position {
	position.Size_ = value
	return position
}

func (position *Position) WithMargin(value sdkmath.LegacyDec) *Position {
	position.Margin = value
	return position
}

func (position *Position) WithOpenNotional(value sdkmath.LegacyDec) *Position {
	position.OpenNotional = value
	return position
}

func (position *Position) WithLatestCumulativePremiumFraction(value sdkmath.LegacyDec) *Position {
	position.LatestCumulativePremiumFraction = value
	return position
}

func (position *Position) WithLastUpdatedBlockNumber(value int64) *Position {
	position.LastUpdatedBlockNumber = value
	return position
}

func (p *Position) copy() *Position {
	return &Position{
		TraderAddress:                   p.TraderAddress,
		Pair:                            p.Pair,
		Size_:                           p.Size_,
		Margin:                          p.Margin,
		OpenNotional:                    p.OpenNotional,
		LatestCumulativePremiumFraction: p.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          p.LastUpdatedBlockNumber,
	}
}
