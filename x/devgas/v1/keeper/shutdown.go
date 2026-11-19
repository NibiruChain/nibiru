package keeper

import (
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/collections"
)

// ClearFeeShareState deletes every registered fee share, along with its
// deployer and withdrawer index entries, and sets module param
// `enable_fee_share` to false. It returns the number of fee shares deleted.
//
// The function is the state cleanup for the x/devgas shutdown. The v2.24.0
// upgrade handler calls it.
//
// The function keeps the module params and the store key, so genesis export
// and the read-only queries still work after it runs. Running it again
// deletes nothing.
func (k Keeper) ClearFeeShareState(ctx sdk.Context) (deleted int, err error) {
	// Collect the keys before deleting so that deletion does not happen under
	// an open iterator.
	contracts := k.DevGasStore.Iterate(ctx, collections.Range[string]{}).Keys()
	for _, contract := range contracts {
		if err := k.DevGasStore.Delete(ctx, contract); err != nil {
			return deleted, fmt.Errorf(
				"delete fee share of contract %s: %w", contract, err,
			)
		}
		deleted++
	}

	params := k.GetParams(ctx)
	params.EnableFeeShare = false
	k.ModuleParams.Set(ctx, params)

	return deleted, nil
}
