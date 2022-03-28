package keeper

import (
	ammtypes "github.com/MatrixDao/matrix/x/vamm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

func getSamplePool() *ammtypes.Pool {
	ratioLimit, _ := sdktypes.NewDecFromStr("0.9")

	pool := ammtypes.NewPool(
		UsdmPair,
		ratioLimit,
		sdktypes.NewInt(10_000_000),
		sdktypes.NewInt(5_000_000),
	)

	return pool
}
