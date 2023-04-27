package v2

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (amm *AMM) Validate() error {
	if amm.BaseReserve.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init pool token supply must be > 0")
	}

	if amm.QuoteReserve.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init pool token supply must be > 0")
	}

	if amm.PriceMultiplier.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init price multiplier must be > 0")
	}

	if amm.SqrtDepth.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init sqrt depth must be > 0")
	}

	return nil
}
