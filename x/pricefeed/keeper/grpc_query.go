package keeper

import (
	"github.com/MatrixDao/matrix/x/pricefeed/types"
)

var _ types.QueryServer = Keeper{}
