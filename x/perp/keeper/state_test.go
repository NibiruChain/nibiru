package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestPrepaidBadDebtState(t *testing.T) {
	perpKeeper, _, ctx := getKeeper(t)

	t.Log("not found results in zero")
	amount, err := perpKeeper.PrepaidBadDebtState().Get(ctx, "foo")
	assert.NoError(t, err)
	assert.EqualValues(t, sdk.ZeroInt(), amount)

	t.Log("set and get")
	perpKeeper.PrepaidBadDebtState().Set(ctx, "NUSD", sdk.NewInt(100))

	amount, err = perpKeeper.PrepaidBadDebtState().Get(ctx, "NUSD")
	assert.NoError(t, err)
	assert.EqualValues(t, sdk.NewInt(100), amount)

	t.Log("increment and check")
	amount, err = perpKeeper.PrepaidBadDebtState().Increment(ctx, "NUSD", sdk.NewInt(50))
	assert.NoError(t, err)
	assert.EqualValues(t, sdk.NewInt(150), amount)

	amount, err = perpKeeper.PrepaidBadDebtState().Get(ctx, "NUSD")
	assert.NoError(t, err)
	assert.EqualValues(t, sdk.NewInt(150), amount)
}
