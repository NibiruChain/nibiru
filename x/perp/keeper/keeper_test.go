package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func Test_TrivialPass(t *testing.T) {
	require.True(t, sdk.ZeroInt().IsZero())
}
