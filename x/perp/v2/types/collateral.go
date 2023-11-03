package types

import (
	sdkerrors "cosmossdk.io/errors"

	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// Errors
var ErrInvalidCollateral = sdkerrors.Register("collateral", 1, "invalid token factory")

// Default collateral used for testing only.
var DefaultTestingCollateralNotForProd = tftypes.TFDenom{
	Creator:  "cosmos15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hesh45zsv",
	Subdenom: "unusd",
}
var NibiTestingCollateralNotForProd = tftypes.TFDenom{
	Creator:  "nibi14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssa9gcs",
	Subdenom: "unusd",
}
