package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestPrepaidBadDebtState(t *testing.T) {
	perpKeeper, _, ctx := getKeeper(t)

	t.Log("not found results in zero")
	amount := perpKeeper.PrepaidBadDebtState().Get(ctx, "foo")
	assert.EqualValues(t, sdk.ZeroInt(), amount)

	t.Log("set and get")
	perpKeeper.PrepaidBadDebtState().Set(ctx, "NUSD", sdk.NewInt(100))

	amount = perpKeeper.PrepaidBadDebtState().Get(ctx, "NUSD")
	assert.EqualValues(t, sdk.NewInt(100), amount)

	t.Log("increment and check")
	amount = perpKeeper.PrepaidBadDebtState().Increment(ctx, "NUSD", sdk.NewInt(50))
	assert.EqualValues(t, sdk.NewInt(150), amount)

	amount = perpKeeper.PrepaidBadDebtState().Get(ctx, "NUSD")
	assert.EqualValues(t, sdk.NewInt(150), amount)

	t.Log("decrement and check")
	amount = perpKeeper.PrepaidBadDebtState().Decrement(ctx, "NUSD", sdk.NewInt(75))
	assert.EqualValues(t, sdk.NewInt(75), amount)

	amount = perpKeeper.PrepaidBadDebtState().Get(ctx, "NUSD")
	assert.EqualValues(t, sdk.NewInt(75), amount)

	t.Log("decrement to below zero and check")
	amount = perpKeeper.PrepaidBadDebtState().Decrement(ctx, "NUSD", sdk.NewInt(1000))
	assert.EqualValues(t, sdk.ZeroInt(), amount)

	amount = perpKeeper.PrepaidBadDebtState().Get(ctx, "NUSD")
	assert.EqualValues(t, sdk.ZeroInt(), amount)
}

func TestPairMetadata_GetAll(t *testing.T) {
	pairMetadatas := []*types.PairMetadata{
		{
			Pair: "ubtc:unibi",
			CumulativePremiumFractions: []sdk.Dec{
				sdk.MustNewDecFromStr("1"),
			},
		},
		{
			Pair:                       "ueth:unibi",
			CumulativePremiumFractions: nil,
		},
	}

	perpKeeper, _, ctx := getKeeper(t)

	for _, m := range pairMetadatas {
		perpKeeper.PairMetadata().Set(ctx, m)
	}

	savedMetadata := perpKeeper.PairMetadata().GetAll(ctx)
	require.Len(t, savedMetadata, 2)

	for _, sm := range savedMetadata {
		require.Contains(t, pairMetadatas, sm)
	}
}
