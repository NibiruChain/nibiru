package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validates a PoolAsset amount and weights.
func (pa PoolAsset) Validate() error {
	if pa.Token.Amount.LTE(sdk.ZeroInt()) {
		return fmt.Errorf("can't add the zero or negative balance of token")
	}

	if pa.Weight.LTE(sdk.ZeroInt()) {
		return fmt.Errorf("a token's weight in the pool must be greater than 0")
	}

	if pa.Weight.GTE(MaxUserSpecifiedWeight.MulRaw(GuaranteedWeightPrecision)) {
		return fmt.Errorf("a token's weight in the pool must be less than 1^50")
	}

	return nil
}
