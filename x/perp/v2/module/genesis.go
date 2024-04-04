package perp

import (
	"time"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"

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

	k.DnREpochName.Set(ctx, genState.DnrEpochName)

	for _, m := range genState.Markets {
		k.SaveMarket(ctx, m)
	}

	for _, g := range genState.MarketLastVersions {
		k.MarketLastVersion.Set(ctx, g.Pair, types.MarketLastVersion{Version: g.Version})
	}

	for _, amm := range genState.Amms {
		pair := amm.Pair
		k.SaveAMM(ctx, amm)
		timestampMs := ctx.BlockTime().UnixMilli()
		k.ReserveSnapshots.Set(
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
		k.TraderVolumes.Set(
			ctx,
			collections.Join(sdk.MustAccAddressFromBech32(vol.Trader), vol.Epoch),
			vol.Volume,
		)
	}
	for _, globalDiscount := range genState.GlobalDiscount {
		k.GlobalDiscounts.Set(
			ctx,
			globalDiscount.Volume,
			globalDiscount.Fee,
		)
	}

	for _, customDiscount := range genState.CustomDiscounts {
		k.TraderDiscounts.Set(
			ctx,
			collections.Join(sdk.MustAccAddressFromBech32(customDiscount.Trader), customDiscount.Discount.Volume),
			customDiscount.Discount.Fee,
		)
	}

	if genState.CollateralDenom != "" {
		err := k.Sudo().UnsafeChangeCollateralDenom(ctx, genState.CollateralDenom)
		if err != nil {
			panic(err)
		}
	}

	for _, globalVolume := range genState.GlobalVolumes {
		k.GlobalVolumes.Set(
			ctx,
			globalVolume.Epoch,
			globalVolume.Volume,
		)
	}

	for _, rebateAlloc := range genState.RebatesAllocations {
		k.EpochRebateAllocations.Set(
			ctx,
			rebateAlloc.Epoch,
			types.DNRAllocation{
				Epoch:  rebateAlloc.Epoch,
				Amount: rebateAlloc.Amount,
			},
		)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	iterMarkets, err := k.Markets.Iterate(ctx, &collections.Range[collections.Pair[asset.Pair, uint64]]{})
	valuesMarkets, err := iterMarkets.Values()
	if err != nil {
		k.Logger(ctx).Error("failed getting max markets", "error", err)
		return nil
	}
	genesis.Markets = valuesMarkets
	if err != nil {
		k.Logger(ctx).Error("failed to get market values", "error", err)
		return nil
	}

	iterMarketLastVersion, err := k.MarketLastVersion.Iterate(ctx, &collections.Range[asset.Pair]{})
	if err != nil {
		k.Logger(ctx).Error("failed to get market last version", "error", err)
		return nil
	}
	kviterMarketLastVersion, err := iterMarketLastVersion.KeyValues()
	if err != nil {
		k.Logger(ctx).Error("failed to get market last version key values", "error", err)
		return nil
	}

	kv := kviterMarketLastVersion
	for _, kv := range kv {
		genesis.MarketLastVersions = append(genesis.MarketLastVersions, types.GenesisMarketLastVersion{
			Pair:    kv.Key,
			Version: kv.Value.Version,
		})
	}

	iterAmms, err := k.AMMs.Iterate(ctx, &collections.Range[collections.Pair[asset.Pair, uint64]]{})
	if err != nil {
		k.Logger(ctx).Error("failed to get amms", "error", err)
		return nil
	}
	valuesAmms, err := iterAmms.Values()
	if err != nil {
		k.Logger(ctx).Error("failed to get amm values", "error", err)
		return nil
	}
	genesis.Amms = valuesAmms

	iterPairRange, err := k.Positions.Iterate(ctx, &collections.PairRange[collections.Pair[asset.Pair, uint64], sdk.AccAddress]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate pair ranges", "error", err)
		return nil
	}
	kvPairRange, err := iterPairRange.KeyValues()
	if err != nil {
		k.Logger(ctx).Error("failed to get pair range key values", "error", err)
		return nil
	}

	pkv := kvPairRange
	for _, kv := range pkv {
		genesis.Positions = append(genesis.Positions, types.GenesisPosition{
			Pair:     kv.Key.K1().K1(),
			Version:  kv.Key.K1().K2(),
			Position: kv.Value,
		})
	}

	iterReserveSnapshots, err := k.ReserveSnapshots.Iterate(ctx, &collections.PairRange[asset.Pair, time.Time]{})
	if err != nil {
		k.Logger(ctx).Error("failed to get iterate reserve snapshots", "error", err)
		return nil
	}
	valuesReserveSnapshots, err := iterReserveSnapshots.Values()
	if err != nil {
		k.Logger(ctx).Error("failed to get reserve snapshots values", "error", err)
		return nil
	}

	genesis.ReserveSnapshots = valuesReserveSnapshots
	genesis.DnrEpoch, err = k.DnREpoch.Get(ctx)
	if err != nil {
		genesis.DnrEpoch = 0
	}
	genesis.DnrEpochName, err = k.DnREpochName.Get(ctx)
	if err != nil {
		genesis.DnrEpochName = ""
	}

	// export volumes
	volumes, err := k.TraderVolumes.Iterate(ctx, &collections.PairRange[sdk.AccAddress, uint64]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate trader volumes", "error", err)
		return nil
	}
	defer volumes.Close()
	for ; volumes.Valid(); volumes.Next() {
		key, _ := volumes.Key()
		value, _ := volumes.Value()
		genesis.TraderVolumes = append(genesis.TraderVolumes, types.GenesisState_TraderVolume{
			Trader: key.K1().String(),
			Epoch:  key.K2(),
			Volume: value,
		})
	}

	// export global discounts
	discounts, err := k.GlobalDiscounts.Iterate(ctx, &collections.Range[sdkmath.Int]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate discounts", "error", err)
		return nil
	}
	defer discounts.Close()
	for ; discounts.Valid(); discounts.Next() {
		key, _ := discounts.Key()
		value, _ := discounts.Value()
		genesis.GlobalDiscount = append(genesis.GlobalDiscount, types.GenesisState_Discount{
			Fee:    value,
			Volume: key,
		})
	}

	// export custom discounts
	customDiscounts, err := k.TraderDiscounts.Iterate(ctx, &collections.PairRange[sdk.AccAddress, sdkmath.Int]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate custom discounts", "error", err)
		return nil
	}
	defer customDiscounts.Close()

	for ; customDiscounts.Valid(); customDiscounts.Next() {
		key, _ := customDiscounts.Key()
		genesis.CustomDiscounts = append(genesis.CustomDiscounts, types.GenesisState_CustomDiscount{
			Trader: key.K1().String(),
			Discount: &types.GenesisState_Discount{
				Fee:    sdkmath.LegacyDec{},
				Volume: key.K2(),
			},
		})
	}

	collateral, err := k.Collateral.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.CollateralDenom = collateral

	// export global volumes
	globalVolumes, err := k.GlobalVolumes.Iterate(ctx, &collections.Range[uint64]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate global volumes", "error", err)
		return nil
	}
	defer globalVolumes.Close()
	for ; globalVolumes.Valid(); globalVolumes.Next() {
		key, _ := globalVolumes.Key()
		value, _ := globalVolumes.Value()
		genesis.GlobalVolumes = append(genesis.GlobalVolumes, types.GenesisState_GlobalVolume{
			Epoch:  key,
			Volume: value,
		})
	}

	// export rebates allocations
	iter, err := k.EpochRebateAllocations.Iterate(ctx, &collections.Range[uint64]{})
	if err != nil {
		k.Logger(ctx).Error("failed to iterate rebates allocations", "error", err)
		return nil
	}
	genesis.RebatesAllocations, err = iter.Values()

	return genesis
}
