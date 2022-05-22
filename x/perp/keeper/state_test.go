package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestPrepaidBadDebtState(t *testing.T) {
	perpKeeper, _, ctx := getKeeper(t)

	t.Log("not found error")
	_, err := perpKeeper.PrepaidBadDebt().Get(ctx, "foo")
	assert.Error(t, err)

	t.Log("set and get")
	perpKeeper.PrepaidBadDebt().Set(ctx, "NUSD", sdk.NewInt(100))
	amount, err := perpKeeper.PrepaidBadDebt().Get(ctx, "NUSD")
	assert.NoError(t, err)
	assert.EqualValues(t, sdk.NewInt(100), amount)
}
