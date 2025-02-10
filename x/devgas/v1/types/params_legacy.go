package types

// TODO: Remove this and params_legacy_test.go after v0.47.x (v16) upgrade

import (
	"cosmossdk.io/math"
	params "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store key
var (
	DefaultEnableFeeShare  = true
	DefaultDeveloperShares = math.LegacyNewDecWithPrec(50, 2) // 50%
	// DefaultAllowedDenoms   = []string(nil)             // all allowed
	DefaultAllowedDenoms = []string{} // all allowed

	ParamStoreKeyEnableFeeShare  = []byte("EnableFeeShare")
	ParamStoreKeyDeveloperShares = []byte("DeveloperShares")
	ParamStoreKeyAllowedDenoms   = []byte("AllowedDenoms")
)

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&ModuleParams{})
}

// ParamSetPairs returns the parameter set pairs.
func (p *ModuleParams) ParamSetPairs() params.ParamSetPairs {
	return params.ParamSetPairs{
		params.NewParamSetPair(ParamStoreKeyEnableFeeShare, &p.EnableFeeShare, validateBool),
		params.NewParamSetPair(ParamStoreKeyDeveloperShares, &p.DeveloperShares, validateShares),
		params.NewParamSetPair(ParamStoreKeyAllowedDenoms, &p.AllowedDenoms, validateArray),
	}
}
