package types

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// ChainAccessConfigs chain settings
type ChainAccessConfigs struct {
	Upload      AccessConfig
	Instantiate AccessConfig
}

// NewChainAccessConfigs constructor
func NewChainAccessConfigs(upload, instantiate AccessConfig) ChainAccessConfigs {
	return ChainAccessConfigs{Upload: upload, Instantiate: instantiate}
}

type AuthorizationPolicyAction uint64

const (
	_ AuthorizationPolicyAction = iota
	AuthZActionInstantiate
	AuthZActionMigrateContract
)

// AuthorizationPolicy is an abstract authorization ruleset defined as an extension point that can be customized by
// chains
type AuthorizationPolicy interface {
	CanCreateCode(chainConfigs ChainAccessConfigs, actor sdk.AccAddress, contractConfig AccessConfig) bool
	CanInstantiateContract(c AccessConfig, actor sdk.AccAddress) bool
	CanModifyContract(admin, actor sdk.AccAddress) bool
	CanModifyCodeAccessConfig(creator, actor sdk.AccAddress, isSubset bool) bool
	// SubMessageAuthorizationPolicy returns authorization policy to be used for submessages. Must never be nil
	SubMessageAuthorizationPolicy(entrypoint AuthorizationPolicyAction) AuthorizationPolicy
}
