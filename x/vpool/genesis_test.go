package vpool_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/vpool"
)

func TestGenesis(t *testing.T) {
	vpools := []types.Vpool{
		{
			Pair:              asset.MustNewPair("BTC:NUSD"),
			BaseAssetReserve:  sdk.NewDec(1 * common.TO_MICRO),      // 1
			QuoteAssetReserve: sdk.NewDec(30_000 * common.TO_MICRO), // 30,000
			SqrtDepth:         common.MustSqrtDec(sdk.NewDec(30_000 * common.TO_MICRO * common.TO_MICRO)),
			Config: types.VpoolConfig{
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
			Pair:              asset.MustNewPair("ETH:NUSD"),
			BaseAssetReserve:  sdk.NewDec(2 * common.TO_MICRO),      // 2
			QuoteAssetReserve: sdk.NewDec(60_000 * common.TO_MICRO), // 60,000
			SqrtDepth:         common.MustSqrtDec(sdk.NewDec(2 * 60_000 * common.TO_MICRO * common.TO_MICRO)),
			Config: types.VpoolConfig{
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
		Vpools: vpools,
	}

	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
	k := nibiruApp.VpoolKeeper

	vpool.InitGenesis(ctx, k, genesisState)

	for _, vp := range vpools {
		require.True(t, k.ExistsPool(ctx, vp.Pair))
	}

	exportedGenesis := vpool.ExportGenesis(ctx, k)
	require.Len(t, exportedGenesis.Vpools, 2)

	iter := k.ReserveSnapshots.Iterate(
		ctx,
		collections.PairRange[asset.Pair, time.Time]{})
	defer iter.Close()

	require.Len(t, iter.Values(), 2)

	for _, pool := range genesisState.Vpools {
		require.Contains(t, exportedGenesis.Vpools, pool)
	}
}
