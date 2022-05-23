package vpool

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TODO: https://github.com/NibiruChain/nibiru/issues/475
func TestGenesis(t *testing.T) {
	vpools := []*types.Pool{
		{
			Pair:                  "BTC:NUSD",
			BaseAssetReserve:      sdk.NewDec(1_000_000),      // 1
			QuoteAssetReserve:     sdk.NewDec(30_000_000_000), // 30,000
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.88"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.20"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.20"),
		},
		{
			Pair:                  "BTC:NUSD",
			BaseAssetReserve:      sdk.NewDec(2_000_000),      // 1
			QuoteAssetReserve:     sdk.NewDec(60_000_000_000), // 30,000
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.77"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.30"),
		},
	}

	genesisState := types.GenesisState{Vpools: vpools}

	nibiruApp, ctx := testutil.NewNibiruApp(true)
	k := nibiruApp.VpoolKeeper
	InitGenesis(ctx, k, genesisState)

	for _, vp := range vpools {
		require.True(t, k.ExistsPool(ctx, common.TokenPair(vp.Pair)))
	}

	exportedGenesis := ExportGenesis(ctx, k)

	require.Equal(t, vpools, exportedGenesis)
}
