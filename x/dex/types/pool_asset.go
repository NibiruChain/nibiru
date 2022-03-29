package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validate validates the set of params
func (pa PoolAsset) ValidateWeight() error {
	if pa.Weight.LTE(sdk.ZeroInt()) {
		return fmt.Errorf("a token's weight in the pool must be greater than 0")
	}

	// TODO: Choose a value that is too large for weights
	// if asset.Weight >= (1 << 32) {
	// 	return fmt.Errorf("a token's weight in the pool must be less than 2^32")
	// }

	return nil
}
