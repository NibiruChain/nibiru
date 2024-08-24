// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcoretypes "github.com/ethereum/go-ethereum/core/types"
)

// BeginBlock hook for the EVM module.
func (k *Keeper) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock also retrieves the bloom filter value from the transient store and commits it to the
func (k *Keeper) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	bloom := gethcoretypes.BytesToBloom(k.EvmState.GetBlockBloomTransient(ctx).Bytes())
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventBlockBloom{
		Bloom: string(bloom.Bytes()),
	})
	// The bloom logic doesn't update the validator set.
	return []abci.ValidatorUpdate{}
}
