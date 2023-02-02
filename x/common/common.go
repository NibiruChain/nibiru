package common

import (
	"github.com/holiman/uint256"
)

const (
	ModuleName                = "common"
	TreasuryPoolModuleAccount = "treasury_pool"
	PairSeparator             = ":"
)

var (
	APrecision = uint256.NewInt().SetUint64(1)
	// Precision for int representation in sdk.Int objects
	Precision = int64(1_000_000)
)
