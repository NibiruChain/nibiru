package keeper

import (
	"github.com/MatrixDao/matrix/x/stablecoin/types"
)

var _ types.QueryServer = Keeper{}
