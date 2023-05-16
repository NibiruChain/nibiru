package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

	if m.BlockNumber < 0 {
		return fmt.Errorf("invalid block number")
	}

	return nil
}

func (m *PairMetadata) Validate() error {
	if err := m.Pair.Validate(); err != nil {
		return err
	}

	if m.LatestCumulativePremiumFraction.IsNil() {
		return fmt.Errorf("invalid cumulative funding rate")
	}

	return nil
}

func (m *PrepaidBadDebt) Validate() error {
	return sdk.Coin{
		Denom:  m.Denom,
		Amount: m.Amount,
	}.Validate()
}
