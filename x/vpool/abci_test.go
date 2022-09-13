package vpool_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestSnapshotUpdates(t *testing.T) {
	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
	vpoolKeeper := nibiruApp.VpoolKeeper

	runBlock := func(duration time.Duration) {
		vpool.EndBlocker(ctx, nibiruApp.VpoolKeeper)
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
	}

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC)).WithBlockHeight(1)

	vpoolKeeper.CreatePool(
		ctx,
		common.PairBTCStable,
		sdk.OneDec(),
		sdk.NewDec(10),
		sdk.NewDec(10),
		sdk.NewDec(3),
		sdk.OneDec(),
		sdk.OneDec(),
		sdk.NewDec(10),
	)
	expectedSnapshot := types.ReserveSnapshot{
		BaseAssetReserve:  sdk.NewDec(10),
		QuoteAssetReserve: sdk.NewDec(10),
		TimestampMs:       ctx.BlockTime().UnixMilli(),
		BlockNumber:       ctx.BlockHeight(),
	}

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	snapshot, err := vpoolKeeper.GetSnapshot(ctx, common.PairBTCStable, uint64(ctx.BlockHeight()-1))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	t.Log("affect mark price")
	_, err = vpoolKeeper.SwapQuoteForBase(
		ctx,
		common.PairBTCStable,
		types.Direction_ADD_TO_POOL,
		sdk.NewDec(10),
		sdk.ZeroDec(),
		false,
	)
	require.NoError(t, err)
	expectedSnapshot = types.ReserveSnapshot{
		QuoteAssetReserve: sdk.NewDec(20),
		BaseAssetReserve:  sdk.NewDec(5),
		TimestampMs:       ctx.BlockTime().UnixMilli(),
		BlockNumber:       ctx.BlockHeight(),
	}

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	snapshot, err = vpoolKeeper.GetSnapshot(ctx, common.PairBTCStable, uint64(ctx.BlockHeight()-1))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)
}
