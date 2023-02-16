package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(startingPoolNumber uint64, poolCreationFee sdk.Coins, whitelistedAssets []string) Params {
	return Params{
		StartingPoolNumber: startingPoolNumber,
		PoolCreationFee:    poolCreationFee,
		WhitelistedAsset:   whitelistedAssets,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		StartingPoolNumber: 1,
		PoolCreationFee:    sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1000*common.Precision)), // 1000 NIBI
		WhitelistedAsset: []string{
			denoms.NIBI,
			denoms.USDC,
			denoms.NUSD,
			denoms.USDT,
		},
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("StartingPoolNumber"), &p.StartingPoolNumber, validatePoolNumber),
		paramtypes.NewParamSetPair([]byte("PoolCreationFee"), &p.PoolCreationFee, validatePoolCreationFee),
		paramtypes.NewParamSetPair([]byte("WhitelistedAsset"), &p.WhitelistedAsset, func(value interface{}) error { return nil })}
}

func validatePoolNumber(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validatePoolCreationFee(i interface{}) error {
	v, ok := i.(sdk.Coins)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.Validate() != nil {
		return fmt.Errorf("invalid pool creation fee: %+v", i)
	}

	return nil
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validatePoolCreationFee(p.PoolCreationFee); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// GetWhitelistedAssetsAsMap returns the whitelisted assets as map, util for
// checking if a value is on the whitelist.
func (p Params) GetWhitelistedAssetsAsMap() map[string]bool {
	whitelistedAssets := make(map[string]bool)
	for _, a := range p.WhitelistedAsset {
		whitelistedAssets[a] = true
	}
	return whitelistedAssets
}
