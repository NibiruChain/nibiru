package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

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

// GetZeroGasEvmContracts loads the ZeroGasActors state and returns the subset
// of entries that can be parsed as canonical EIP55 Ethereum contract
// addresses. This helper is used by the EVM module via the evm.SudoKeeper
// interface without depending on x/sudo-specific types.
func (k Keeper) GetZeroGasEvmContracts(ctx sdk.Context) []eth.EIP55Addr {
	actors := k.GetZeroGasActors(ctx)

	evmAddrs := make([]eth.EIP55Addr, 0, len(actors.AlwaysZeroGasContracts))
	for _, raw := range actors.AlwaysZeroGasContracts {
		addr, err := eth.NewEIP55AddrFromStr(raw)
		if err != nil {
			// Entries should already be validated, but skip defensively.
			continue
		}
		evmAddrs = append(evmAddrs, addr)
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
