// Copyright (c) 2023-2024 Nibi, Inc.
package evmstate

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcoretypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// BeginBlock hook for the EVM module.
func (k *Keeper) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock also retrieves the bloom filter value from the transient store and commits it to the
func (k *Keeper) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	bloom := gethcoretypes.BytesToBloom(k.EvmState.GetBlockBloomTransient(ctx).Bytes())
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventBlockBloom{
		Bloom: eth.BloomToHex(bloom),
	})

	// WeiBlockDelta should always be zero in a well-functioning program.
	//
	weiBlockDelta := k.Bank.WeiBlockDelta(ctx)
	if !weiBlockDelta.IsZero() {
		ctx.Logger().Error("nonzero wei block delta",
			"msg", fmt.Sprintf("block=%d, weiBlockDelta=%s", ctx.BlockHeight(), weiBlockDelta),
		)

		// Commit the current wei block delta to the store of committed wei
		// changes. This allows us to see non-zero wei block delta values from
		// prior blocks.
		committedDelta := k.EvmState.NetWeiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
		newCommittedDelta := committedDelta.Add(weiBlockDelta)
		k.EvmState.NetWeiBlockDelta.Set(ctx, newCommittedDelta)

		_ = ctx.EventManager().EmitTypedEvent(&evm.EventWeiBlockDelta{
			NetWeiBlockDelta: newCommittedDelta,
			WeiBlockDelta:    weiBlockDelta,
			BlockNumber:      0,
		})
	}

	// The bloom logic doesn't update the validator set.
	return []abci.ValidatorUpdate{}
}
