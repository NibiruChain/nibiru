package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// minimum number of assets a pool may have
	MinPoolAssets = 2
	// maximum number of assets a pool may have
	MaxPoolAssets = 8

	// the exponent of a pool display share compared to a pool base share (one pool display share = 10^18 pool base shares)
	DisplayPoolShareExponent = 18
)

var (
	// OneDisplayPoolShare represents one display pool share
	OneDisplayPoolShare = sdk.NewIntWithDecimal(1, DisplayPoolShareExponent)

	// InitPoolSharesSupply is the amount of new shares to initialize a pool with.
	InitPoolSharesSupply = OneDisplayPoolShare.MulRaw(100)
)
