package perp

import (
	"time"

	"cosmossdk.io/math"
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
		k.SavePosition(ctx, p.Pair, p.Version, sdk.MustAccAddressFromBech32(p.Position.TraderAddress), p.Position)
	}

	for _, vol := range genState.TraderVolumes {
		k.TraderVolumes.Insert(
			ctx,
			collections.Join(sdk.MustAccAddressFromBech32(vol.Trader), vol.Epoch),
			vol.Volume,
		)
	}
	for _, globalDiscount := range genState.GlobalDiscount {
		k.GlobalDiscounts.Insert(
			ctx,
			globalDiscount.Volume,
			globalDiscount.Fee,
		)
	}

	for _, customDiscount := range genState.CustomDiscounts {
		k.TraderDiscounts.Insert(
			ctx,
			collections.Join(sdk.MustAccAddressFromBech32(customDiscount.Trader), customDiscount.Discount.Volume),
			customDiscount.Discount.Fee,
		)
	}

	if genState.CollateralDenom != "" {
		err := k.Admin.UnsafeChangeCollateralDenom(ctx, genState.CollateralDenom)
		if err != nil {
			panic(err)
		}
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
	pkv := k.Positions.Iterate(ctx, collections.PairRange[collections.Pair[asset.Pair, uint64], sdk.AccAddress]{}).KeyValues()
	for _, kv := range pkv {
		genesis.Positions = append(genesis.Positions, types.GenesisPosition{
			Pair:     kv.Key.K1().K1(),
			Version:  kv.Key.K1().K2(),
			Position: kv.Value,
		})
	}
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

	// export global discounts
	discounts := k.GlobalDiscounts.Iterate(ctx, collections.Range[math.Int]{})
	defer discounts.Close()
	for ; discounts.Valid(); discounts.Next() {
		genesis.GlobalDiscount = append(genesis.GlobalDiscount, types.GenesisState_Discount{
			Fee:    discounts.Value(),
			Volume: discounts.Key(),
		})
	}

	// export custom discounts
	customDiscounts := k.TraderDiscounts.Iterate(ctx, collections.PairRange[sdk.AccAddress, math.Int]{})
	defer customDiscounts.Close()

	for ; customDiscounts.Valid(); customDiscounts.Next() {
		key := customDiscounts.Key()
		genesis.CustomDiscounts = append(genesis.CustomDiscounts, types.GenesisState_CustomDiscount{
			Trader: key.K1().String(),
			Discount: &types.GenesisState_Discount{
				Fee:    sdk.Dec{},
				Volume: key.K2(),
			},
		})
	}

	collateral, err := k.Collateral.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.CollateralDenom = collateral
	return genesis
}
