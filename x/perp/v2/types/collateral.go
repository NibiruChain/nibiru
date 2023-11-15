package types

import (
	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// TestingCollateralDenomNUSD: For testing only
var TestingCollateralDenomNUSD string = tftypes.TFDenom{
	Creator:  "nibi14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssa9gcs",
	Subdenom: "unusd",
}.String()
