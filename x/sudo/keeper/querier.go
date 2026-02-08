package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// Ensure the interface is properly implemented at compile time
var _ sudo.QueryServer = (*Keeper)(nil)

func (k Keeper) QuerySudoers(
	goCtx context.Context,
	req *sudo.QuerySudoersRequest,
) (resp *sudo.QuerySudoersResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sudoers, err := k.Sudoers.Get(ctx)

	return &sudo.QuerySudoersResponse{
		Sudoers: sudoers,
	}, err
}

func (k Keeper) GetZeroGasActors(ctx sdk.Context) sudo.ZeroGasActors {
	return k.ZeroGasActors.GetOr(ctx, sudo.DefaultZeroGasActors())
}

// GetZeroGasEvmContracts loads the ZeroGasActors state and returns a set (map) of
// EVM contract addresses that can be parsed from the zero-gas list, for O(1)
// lookup. Used by the EVM module via the evm.SudoKeeper interface without
// depending on x/sudo-specific types.
func (k Keeper) GetZeroGasEvmContracts(ctx sdk.Context) map[gethcommon.Address]struct{} {
	actors := k.GetZeroGasActors(ctx)

	evmAddrs := make(map[gethcommon.Address]struct{}, len(actors.AlwaysZeroGasContracts))
	for _, raw := range actors.AlwaysZeroGasContracts {
		addr, err := eth.NewEIP55AddrFromStr(raw)
		if err != nil {
			// Entries should already be validated, but skip defensively.
			continue
		}
		evmAddrs[addr.Address] = struct{}{}
	}

	return evmAddrs
}

func (k Keeper) QueryZeroGasActors(
	goCtx context.Context,
	_ *sudo.QueryZeroGasActorsRequest,
) (resp *sudo.QueryZeroGasActorsResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &sudo.QueryZeroGasActorsResponse{
		Actors: k.GetZeroGasActors(ctx),
	}, nil
}
