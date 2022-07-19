package keeper

import (
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestMsgServer_CreatePool(t *testing.T) {
	poolCreator := sample.AccAddress()

	msg := &types.MsgCreatePool{
		Sender:                poolCreator.String(),
		Pair:                  "BTC:USD",
		TradeLimitRatio:       sdk.MustNewDecFromStr("0.2"),
		QuoteAssetReserve:     sdk.NewDec(1_000_000),
		BaseAssetReserve:      sdk.NewDec(100),
		FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
		MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.05"),
	}

	priceFeed := mock.NewMockPricefeedKeeper(gomock.NewController(t))
	priceFeed.EXPECT().
		IsActivePair(gomock.Any(), gomock.Eq(msg.Pair)).
		Return(true)

	vpoolKeeper, ctx := VpoolKeeper(t, priceFeed)
	require.NoError(t, vpoolKeeper.Whitelist(ctx).Add(poolCreator))

	s := NewMsgServer(vpoolKeeper)

	_, err := s.CreatePool(sdk.WrapSDKContext(ctx), msg)
	require.NoError(t, err)
}

func TestQueryReserveAssets(t *testing.T) {
	t.Log("initialize vpoolkeeper")
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)
	queryServer := NewQuerier(vpoolKeeper)

	t.Log("initialize vpool")
	pool := types.NewPool(
		/* pair */ BTCNusdPair,
		/* tradeLimitRatio */ sdk.ZeroDec(),
		/* quoteAmount */ sdk.NewDec(1_000_000),
		/* baseAmount */ sdk.NewDec(1000),
		/* fluctuationLimitRatio */ sdk.ZeroDec(),
		/* maxOracleSpreadRatio */ sdk.ZeroDec(),
	)
	vpoolKeeper.savePool(ctx, pool)

	t.Log("query reserve assets")
	resp, err := queryServer.ReserveAssets(
		sdk.WrapSDKContext(ctx),
		&types.QueryReserveAssetsRequest{
			Pair: "BTC:NUSD",
		},
	)

	t.Log("assert reserve assets")
	require.NoError(t, err)
	assert.EqualValues(t, pool.QuoteAssetReserve, resp.QuoteAssetReserve)
	assert.EqualValues(t, pool.BaseAssetReserve, resp.BaseAssetReserve)
}
