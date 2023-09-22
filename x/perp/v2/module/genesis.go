package perp

import (
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	k.DnREpoch.Set(ctx, genState.DnrEpoch)

	for _, m := range genState.Markets {
		k.SaveMarket(ctx, m)
	}

	for _, g := range genState.MarketLastVersions {
		k.MarketLastVersion.Insert(ctx, g.Pair, types.MarketLastVersion{Version: g.Version})
	}

	for _, amm := range genState.Amms {
		pair := amm.Pair
		k.SaveAMM(ctx, amm)
		timestampMs := ctx.BlockTime().UnixMilli()
		k.ReserveSnapshots.Insert(
			ctx,
			collections.Join(pair, time.UnixMilli(timestampMs)),
			types.ReserveSnapshot{
				Amm:         amm,
				TimestampMs: timestampMs,
			},
		)
	}

	for _, p := range genState.Positions {
		k.Positions.Insert(
			ctx,
			collections.Join(p.Pair, sdk.MustAccAddressFromBech32(p.TraderAddress)),
			p,
		)
	}

	for _, vol := range genState.TraderVolumes {
		k.TraderVolumes.Insert(
			ctx,
			collections.Join(sdk.MustAccAddressFromBech32(vol.Trader), vol.Epoch),
			vol.Volume,
		)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Markets = k.Markets.Iterate(ctx, collections.Range[collections.Pair[asset.Pair, uint64]]{}).Values()

	kv := k.MarketLastVersion.Iterate(ctx, collections.Range[asset.Pair]{}).KeyValues()
	for _, kv := range kv {
		genesis.MarketLastVersions = append(genesis.MarketLastVersions, types.GenesisMarketLastVersion{
			Pair:    kv.Key,
			Version: kv.Value.Version,
		})
	}

	genesis.Amms = k.AMMs.Iterate(ctx, collections.Range[collections.Pair[asset.Pair, uint64]]{}).Values()
	genesis.Positions = k.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
	genesis.ReserveSnapshots = k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}).Values()
	genesis.DnrEpoch = k.DnREpoch.GetOr(ctx, 0)

	// export volumes
	volumes := k.TraderVolumes.Iterate(ctx, collections.PairRange[sdk.AccAddress, uint64]{})
	defer volumes.Close()
	for ; volumes.Valid(); volumes.Next() {
		key := volumes.Key()
		genesis.TraderVolumes = append(genesis.TraderVolumes, types.GenesisState_TraderVolume{
			Trader: key.K1().String(),
			Epoch:  key.K2(),
			Volume: volumes.Value(),
		})
	}

	return genesis
}
