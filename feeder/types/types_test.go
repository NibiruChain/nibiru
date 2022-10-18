package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestValidatorSet(t *testing.T) {
	vs := ValidatorSet{}
	addr := sdk.ValAddress(testutil.AccAddress())
	require.False(t, vs.Has(addr))
	vs.Insert(addr)
	require.True(t, vs.Has(addr))
	vs.Remove(addr)
	require.False(t, vs.Has(addr))

	vs.Insert(addr)
	require.Panics(t, func() {
		vs.Insert(addr)
	})

	vs.Remove(addr)
	require.Panics(t, func() {
		vs.Remove(addr)
	})
}
