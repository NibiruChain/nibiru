package types_test

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/require"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestTryNewCollateral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tokenStr string
		err      error
	}{
		{
			"no token factory data",
			"tf",
			types.ErrInvalidCollateral,
		},
		{
			"no token factory data",
			"tf/abc/unusd",
			fmt.Errorf("decoding bech32 failed:"),
		},
		{
			"no token factory data",
			"tf/abc/unusd",
			fmt.Errorf("decoding bech32 failed:"),
		},
		{
			"happy path",
			"tf/cosmos15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hesh45zsv/unusd",
			nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.TryNewCollateral(tc.tokenStr)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCollateralTFDenom(t *testing.T) {
	collateralString := "tf/cosmos15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hesh45zsv/unusd"
	collateral := types.MustNewCollateral(collateralString)
	require.Equal(t, collateral.GetTFDenom(), collateralString)
}

func TestMustNewCollateral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tokenStr    string
		shouldPanic bool
	}{
		{
			"no token factory data",
			"tf",
			true,
		},
		{
			"invalid bech32",
			"tf/abc/unusd",
			true,
		},
		{
			"happy path",
			"tf/cosmos15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hesh45zsv/unusd",
			false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				require.Panics(t, func() { types.MustNewCollateral(tc.tokenStr) })
			} else {
				require.NotPanics(t, func() { types.MustNewCollateral(tc.tokenStr) })
			}
		})
	}
}

func TestUpdateContractAddress(t *testing.T) {
	collateral := types.MustNewCollateral("tf/cosmos15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hesh45zsv/unusd")
	collateral, err := collateral.UpdatedContractAddress("cosmos168ctmpyppk90d34p3jjy658zf5a5l3w8wk35wht6ccqj4mr0yv8skhnwe8")
	require.NoError(t, err)

	require.Equal(t, collateral.ContractAddress, "cosmos168ctmpyppk90d34p3jjy658zf5a5l3w8wk35wht6ccqj4mr0yv8skhnwe8")
	require.True(t, collateral.Equal(types.MustNewCollateral("tf/cosmos168ctmpyppk90d34p3jjy658zf5a5l3w8wk35wht6ccqj4mr0yv8skhnwe8/unusd")))
}
