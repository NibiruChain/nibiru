package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/stablecoin module sentinel errors
var (
	NoCoinFound            = sdkerrors.Register(ModuleName, 1, "No coin found")
	NotEnoughBalance       = sdkerrors.Register(ModuleName, 2, "Not enough balance")
	NoValidCollateralRatio = sdkerrors.Register(ModuleName, 3, "No valid collateral ratio, waiting for new prices")
)
