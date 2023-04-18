package amm_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpamm "github.com/NibiruChain/nibiru/x/perp/amm"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestGenesis(t *testing.T) {
	markets := []types.Market{
		{
			Pair:         asset.MustNewPair("BTC:NUSD"),
			BaseReserve:  sdk.NewDec(1 * common.TO_MICRO),      // 1
			QuoteReserve: sdk.NewDec(30_000 * common.TO_MICRO), // 30,000
			SqrtDepth:    common.MustSqrtDec(sdk.NewDec(30_000 * common.TO_MICRO * common.TO_MICRO)),
			Config: types.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.88"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.20"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.20"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
			Bias:          sdk.NewDec(1 * common.TO_MICRO),
			PegMultiplier: sdk.OneDec(),
		},
		{
			Pair:         asset.MustNewPair("ETH:NUSD"),
			BaseReserve:  sdk.NewDec(2 * common.TO_MICRO),      // 2
			QuoteReserve: sdk.NewDec(60_000 * common.TO_MICRO), // 60,000
			SqrtDepth:    common.MustSqrtDec(sdk.NewDec(2 * 60_000 * common.TO_MICRO * common.TO_MICRO)),
			Config: types.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.77"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
			Bias:          sdk.NewDec(0),
			PegMultiplier: sdk.MustNewDecFromStr("0.2"),
		},
	}

	genesisState := types.GenesisState{
		Markets: markets,
	}

	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
	k := nibiruApp.PerpAmmKeeper

	perpamm.InitGenesis(ctx, k, genesisState)

	for _, vp := range markets {
		require.True(t, k.ExistsPool(ctx, vp.Pair))
	}

	exportedGenesis := perpamm.ExportGenesis(ctx, k)
	require.Len(t, exportedGenesis.Markets, 2)

	iter := k.ReserveSnapshots.Iterate(
		ctx,
		collections.PairRange[asset.Pair, time.Time]{})
	defer iter.Close()

	require.Len(t, iter.Values(), 2)

	for _, pool := range genesisState.Markets {
		require.Contains(t, exportedGenesis.Markets, pool)
	}
}
