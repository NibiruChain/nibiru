package upgrades

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/NibiruChain/nibiru/v2/x/nutil/asset"
)

// - Deprecate the x/oracle module and wipe  its state.
// - Make the oracle ABCI hook inert in `x/oracle/abci.go` so the module no
// longer runs exchange-rate updates, miss-counter processing, or any oracle
// slashing/jailing logic during block execution.
func runUpgrade2_12_0(nibiru *keepers.PublicKeepers, ctx sdk.Context) error {
	params, err := nibiru.OracleKeeper.ModuleParams.Get(ctx)
	if err != nil {
		return fmt.Errorf("get oracle params: %w", err)
	}

	params.Whitelist = []asset.Pair{}
	nibiru.OracleKeeper.ModuleParams.Set(ctx, params)

	for _, pair := range nibiru.OracleKeeper.WhitelistedPairs.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
		nibiru.OracleKeeper.WhitelistedPairs.Delete(ctx, pair)
	}

	for _, pair := range nibiru.OracleKeeper.ExchangeRateMap.Iterate(ctx, collections.Range[asset.Pair]{}).Keys() {
		_ = nibiru.OracleKeeper.ExchangeRateMap.Delete(ctx, pair)
	}

	// This upgrade is too heavy for the archive nodes. Disable it.
	//for _, key := range nibiru.OracleKeeper.PriceSnapshots.Iterate(
	//	ctx,
	//	collections.PairRange[asset.Pair, time.Time]{},
	//).Keys() {
	//	_ = nibiru.OracleKeeper.PriceSnapshots.Delete(ctx, key)
	//}

	for _, valAddr := range nibiru.OracleKeeper.Prevotes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys() {
		_ = nibiru.OracleKeeper.Prevotes.Delete(ctx, valAddr)
	}

	for _, valAddr := range nibiru.OracleKeeper.Votes.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys() {
		_ = nibiru.OracleKeeper.Votes.Delete(ctx, valAddr)
	}

	for _, valAddr := range nibiru.OracleKeeper.MissCounters.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Keys() {
		_ = nibiru.OracleKeeper.MissCounters.Delete(ctx, valAddr)
	}

	return nil
}
