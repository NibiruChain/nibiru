package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"

	types2 "github.com/NibiruChain/nibiru/x/vpool/types"

	"github.com/NibiruChain/nibiru/x/testutil/mock"
)

func TestCreatePool(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	vpoolKeeper.CreatePool(
		ctx,
		BTCNusdPair,
		sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdk.NewDec(10_000_000),       // 10 tokens
		sdk.NewDec(5_000_000),        // 5 tokens
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
	)

	exists := vpoolKeeper.ExistsPool(ctx, BTCNusdPair)
	require.True(t, exists)

	notExist := vpoolKeeper.ExistsPool(ctx, common.AssetPair{
		Token0: "BTC",
		Token1: "OTHER",
	})
	require.False(t, notExist)
}

func TestKeeper_GetAllPools(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	vpools := []*types2.Pool{
		{
			Pair:                  common.MustNewAssetPair("BTC:NUSD"),
			BaseAssetReserve:      sdk.NewDec(1_000_000),      // 1
			QuoteAssetReserve:     sdk.NewDec(30_000_000_000), // 30,000
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.88"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.20"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.20"),
		},
		{
			Pair:                  common.MustNewAssetPair("ETH:NUSD"),
			BaseAssetReserve:      sdk.NewDec(2_000_000),      // 1
			QuoteAssetReserve:     sdk.NewDec(60_000_000_000), // 30,000
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.77"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.30"),
		},
	}

	for _, vp := range vpools {
		vpoolKeeper.savePool(ctx, vp)
	}

	pools := vpoolKeeper.GetAllPools(ctx)
	require.Len(t, pools, 2)
	for _, pool := range pools {
		require.Contains(t, vpools, pool)
	}
}
