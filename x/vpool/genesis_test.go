package vpool_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestGenesis(t *testing.T) {
	vpools := []types.VPool{
		{
			Pair:                   common.MustNewAssetPair("BTC:NUSD"),
			BaseAssetReserve:       sdk.NewDec(1_000_000),      // 1
			QuoteAssetReserve:      sdk.NewDec(30_000_000_000), // 30,000
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.88"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.20"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.20"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		{
			Pair:                   common.MustNewAssetPair("ETH:NUSD"),
			BaseAssetReserve:       sdk.NewDec(2_000_000),      // 2
			QuoteAssetReserve:      sdk.NewDec(60_000_000_000), // 60,000
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.77"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}

	snapshots := []types.ReserveSnapshot{
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.NewDec(1_000_000),
			sdk.NewDec(60_000_000_000),
			time.UnixMilli(123456),
		),
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.NewDec(2_000_000),
			sdk.NewDec(50_000_000_000),
			time.UnixMilli(223456),
		),
		types.NewReserveSnapshot(
			common.Pair_ETH_NUSD,
			sdk.NewDec(1_000_000),
			sdk.NewDec(50_000_000_000),
			time.UnixMilli(223456),
		),
	}

	genesisState := types.GenesisState{
		Vpools:    vpools,
		Snapshots: snapshots,
	}

	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
	k := nibiruApp.VpoolKeeper
	vpool.InitGenesis(ctx, k, genesisState)

	for _, vp := range vpools {
		require.True(t, k.ExistsPool(ctx, vp.Pair))
	}

	exportedGenesis := vpool.ExportGenesis(ctx, k)
	require.Len(t, exportedGenesis.Vpools, 2)
	require.Len(t, exportedGenesis.Snapshots, 5) // 3 from imported + 2 created when creating a pool

	for _, pool := range genesisState.Vpools {
		require.Contains(t, exportedGenesis.Vpools, pool)
	}

	for _, snapshot := range genesisState.Snapshots {
		require.Contains(t, exportedGenesis.Snapshots, snapshot)
	}
}
