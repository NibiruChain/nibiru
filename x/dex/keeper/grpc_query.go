package keeper

import (
	"github.com/MatrixDao/dex/x/dex/types"
)

var _ types.QueryServer = Keeper{}
