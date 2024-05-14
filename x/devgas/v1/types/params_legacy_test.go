package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func TestLegacyParamKeyTable(t *testing.T) {
	require.IsType(t, paramtypes.KeyTable{}, ParamKeyTable())
	require.NotEmpty(t, ParamKeyTable())
}

func TestLegacyParamSetPairs(t *testing.T) {
	params := DefaultParams()
	require.NotEmpty(t, params.ParamSetPairs())
}

func TestLegacyParamsValidate(t *testing.T) {
	devShares := math.LegacyNewDecWithPrec(60, 2)
	acceptedDenoms := []string{"unibi"}

	testCases := []struct {
		name     string
		params   ModuleParams
		expError bool
	}{
		{"default", DefaultParams(), false},
		{
			"valid: enabled",
			NewParams(true, devShares, acceptedDenoms),
			false,
		},
		{
			"valid: disabled",
			NewParams(false, devShares, acceptedDenoms),
			false,
		},
		{
			"valid: 100% devs",
			ModuleParams{true, math.LegacyNewDecFromInt(math.NewInt(1)), acceptedDenoms},
			false,
		},
		{
			"empty",
			ModuleParams{},
			true,
		},
		{
			"invalid: share > 1",
			ModuleParams{true, math.LegacyNewDecFromInt(math.NewInt(2)), acceptedDenoms},
			true,
		},
		{
			"invalid: share < 0",
			ModuleParams{true, math.LegacyNewDecFromInt(math.NewInt(-1)), acceptedDenoms},
			true,
		},
		{
			"valid: all denoms allowed",
			ModuleParams{true, math.LegacyNewDecFromInt(math.NewInt(-1)), []string{}},
			true,
		},
	}
	for _, tc := range testCases {
		err := tc.params.Validate()

		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
		}
	}
}

func TestLegacyParamsValidateShares(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expError bool
	}{
		{"default", DefaultDeveloperShares, false},
		{"valid", math.LegacyNewDecFromInt(math.NewInt(1)), false},
		{"invalid - wrong type - bool", false, true},
		{"invalid - wrong type - string", "", true},
		{"invalid - wrong type - int64", int64(123), true},
		{"invalid - wrong type - math.Int", math.NewInt(1), true},
		{"invalid - is nil", nil, true},
		{"invalid - is negative", math.LegacyNewDecFromInt(math.NewInt(-1)), true},
		{"invalid - is > 1", math.LegacyNewDecFromInt(math.NewInt(2)), true},
	}
	for _, tc := range testCases {
		err := validateShares(tc.value)

		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
		}
	}
}

func TestLegacyParamsValidateBool(t *testing.T) {
	err := validateBool(DefaultEnableFeeShare)
	require.NoError(t, err)
	err = validateBool(true)
	require.NoError(t, err)
	err = validateBool(false)
	require.NoError(t, err)
	err = validateBool("")
	require.Error(t, err)
	err = validateBool(int64(123))
	require.Error(t, err)
}
