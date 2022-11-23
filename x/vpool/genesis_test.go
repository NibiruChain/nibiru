package vpool_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestGenesis(t *testing.T) {
	vpools := []types.Vpool{
		{
			Pair:              common.MustNewAssetPair("BTC:NUSD"),
			BaseAssetReserve:  sdk.NewDec(1_000_000),      // 1
			QuoteAssetReserve: sdk.NewDec(30_000_000_000), // 30,000
			Config: types.VpoolConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.88"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.20"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.20"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
		{
			Pair:              common.MustNewAssetPair("ETH:NUSD"),
			BaseAssetReserve:  sdk.NewDec(2_000_000),      // 2
			QuoteAssetReserve: sdk.NewDec(60_000_000_000), // 60,000
			Config: types.VpoolConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.77"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
	}

	genesisState := types.GenesisState{
		Vpools: vpools,
	}

	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
	k := nibiruApp.VpoolKeeper

	vpool.InitGenesis(ctx, k, genesisState)

	for _, vp := range vpools {
		require.True(t, k.ExistsPool(ctx, vp.Pair))
	}

	exportedGenesis := vpool.ExportGenesis(ctx, k)
	require.Len(t, exportedGenesis.Vpools, 2)

	iter := k.ReserveSnapshots.Iterate(
		ctx,
		collections.PairRange[common.AssetPair, time.Time]{})
	defer iter.Close()

	require.Len(t, iter.Values(), 2)

	for _, pool := range genesisState.Vpools {
		require.Contains(t, exportedGenesis.Vpools, pool)
	}
}
