package types

import (
	sdkmath "cosmossdk.io/math"
)

const (
	// MinPoolAssets minimum number of assets a pool may have
	MinPoolAssets = 2
	// MaxPoolAssets maximum number of assets a pool may have
	MaxPoolAssets = 2

	// DisplayPoolShareExponent the exponent of a pool display share compared to a pool base share (one pool display share = 10^18 pool base shares)
	DisplayPoolShareExponent = 18

	// GuaranteedWeightPrecision Scaling factor for every weight. The pool weight is:
	// weight_in_MsgCreateBalancerPool * GuaranteedWeightPrecision
	//
	// This is done so that smooth weight changes have enough precision to actually be smooth.
	GuaranteedWeightPrecision int64 = 1 << 30
)

var (
	// OneDisplayPoolShare represents one display pool share
	OneDisplayPoolShare = sdkmath.NewIntWithDecimal(1, DisplayPoolShareExponent)

	// InitPoolSharesSupply is the amount of new shares to initialize a pool with.
	InitPoolSharesSupply = OneDisplayPoolShare.MulRaw(100)

	// MaxUserSpecifiedWeight Pool creators can specify a weight in [1, MaxUserSpecifiedWeight)
	// for every token in the balancer pool.
	//
	// The weight used in the balancer equation is then creator-specified-weight * GuaranteedWeightPrecision.
	// This is done so that LBP's / smooth weight changes can actually happen smoothly,
	// without complex precision loss / edge effects.
	MaxUserSpecifiedWeight = sdkmath.NewIntFromUint64(1 << 20)
)
