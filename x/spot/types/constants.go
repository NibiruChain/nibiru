package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// minimum number of assets a pool may have
	MinPoolAssets = 2
	// maximum number of assets a pool may have
	MaxPoolAssets = 2

	// Scaling factor for every weight. The pool weight is:
	// weight_in_MsgCreateBalancerPool * GuaranteedWeightPrecision
	//
	// This is done so that smooth weight changes have enough precision to actually be smooth.
	GuaranteedWeightPrecision int64 = 1 << 30

	// OneDisplayPoolShare represents one display pool share
	OneDisplayPoolShare = 1e6

	// InitPoolSharesSupply is the amount of new shares to initialize a pool with.
	InitPoolShareSupply = 100e6
)

var (
	// Pool creators can specify a weight in [1, MaxUserSpecifiedWeight)
	// for every token in the balancer pool.
	//
	// The weight used in the balancer equation is then creator-specified-weight * GuaranteedWeightPrecision.
	// This is done so that LBP's / smooth weight changes can actually happen smoothly,
	// without complex precision loss / edge effects.
	MaxUserSpecifiedWeight sdk.Int = sdk.NewIntFromUint64(1 << 20)
)
