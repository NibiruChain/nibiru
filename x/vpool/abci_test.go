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
)

func TestTWAPriceUpdates(t *testing.T) {
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
		sdk.OneDec(),
		sdk.OneDec(),
		sdk.OneDec(),
		sdk.OneDec(),
		sdk.OneDec(),
		sdk.NewDec(10),
	)

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	twap, err := vpoolKeeper.GetCurrentTWAP(ctx, common.PairBTCStable)
	require.NoError(t, err)
	assert.EqualValues(t, sdk.OneDec(), twap.Price)
}
