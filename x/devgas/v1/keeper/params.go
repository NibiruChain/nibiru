package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

// GetParams returns the total set of fees parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.ModuleParams) {
	params, _ = k.ModuleParams.Get(ctx)
	return params.Sanitize()
}
