// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/v2/eth"

	"github.com/NibiruChain/nibiru/v2/x/evm"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcoretypes "github.com/ethereum/go-ethereum/core/types"
)

// BeginBlock hook for the EVM module.
func (k *Keeper) BeginBlock(ctx context.Context) error {
	return nil
}

// EndBlock also retrieves the bloom filter value from the transient store and commits it to the
func (k *Keeper) EndBlock(c context.Context) error {
	ctx := sdk.UnwrapSDKContext(c).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bloom := gethcoretypes.BytesToBloom(k.EvmState.GetBlockBloomTransient(ctx).Bytes())
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventBlockBloom{
		Bloom: eth.BloomToHex(bloom),
	})
	// The bloom logic doesn't update the validator set.
	return nil
}
