package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestCreatePool(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	vpoolKeeper.CreatePool(
		ctx,
		common.PairBTCStable,
		sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdk.NewDec(10_000_000),       // 10 tokens
		sdk.NewDec(5_000_000),        // 5 tokens
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.0625"),
		sdk.MustNewDecFromStr("15"),
	)

	exists := vpoolKeeper.ExistsPool(ctx, common.PairBTCStable)
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

	vpools := []*types.Pool{
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
			BaseAssetReserve:       sdk.NewDec(2_000_000),      // 1
			QuoteAssetReserve:      sdk.NewDec(60_000_000_000), // 30,000
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.77"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}

	for _, vpool := range vpools {
		vpoolKeeper.savePool(ctx, vpool)
	}

	pools := vpoolKeeper.GetAllPools(ctx)
	require.Len(t, pools, 2)
	for _, pool := range pools {
		require.Contains(t, vpools, pool)
	}
}
