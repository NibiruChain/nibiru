package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_A(t *testing.T) {
	require.True(t, sdk.Int{}.IsZero())
}
