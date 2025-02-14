package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas"
)

// GetParams returns the total set of fees parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params devgas.ModuleParams) {
	params, _ = k.ModuleParams.Get(ctx)
	return params.Sanitize()
}
