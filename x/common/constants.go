package common

import (
	"github.com/holiman/uint256"
)

const (
	TreasuryPoolModuleAccount = "treasury_pool"
	// Precision for int representation in sdk.Int objects
	Precision = int64(1_000_000)
)

var (
	APrecision = uint256.NewInt(1)
)
