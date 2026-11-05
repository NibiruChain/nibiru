package keeper

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/sudo"
	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// wasmDeploymentOperation identifies the bytecode lifecycle operation checked
// by the temporary deployment guard. The operation also selects any existing
// action-specific governance authorization that remains valid.
type wasmDeploymentOperation string

const (
	wasmDeploymentUpload  wasmDeploymentOperation = "code upload"
	wasmDeploymentMigrate wasmDeploymentOperation = "contract migration"
)

// wasmDeployerGuard restricts code upload and migration on one exact chain.
// All entrypoints resolve actor permissions through the central keeper guard.
type wasmDeployerGuard struct {
	chainID         string
	sudoPermissions types.SudoPermissionSource
}

// requireActorIsAuthedDeployer allows governance, root, or wasm_deployer members.
// Membership is read on each call, so revocation and root rotation take effect
// immediately. Wasm access and contract admin checks still run afterward.
func (k Keeper) requireActorIsAuthedDeployer(
	ctx sdk.Context,
	actor sdk.AccAddress,
	authz types.AuthorizationPolicy,
	operation wasmDeploymentOperation,
) error {
	guard := k.wasmDeployerGuard
	if guard == nil || ctx.ChainID() != guard.chainID {
		return nil
	}
	if govPolicyAllowsDeployment(authz, operation) {
		return nil
	}

	if err := guard.sudoPermissions.CheckPermissions(actor, ctx, sudo.RoleWasmDeployer); err != nil {
		return sdkerrors.ErrUnauthorized.Wrapf("Wasm %s: %s", operation, err)
	}
	return nil
}

// govPolicyAllowsDeployment recognizes the keeper's existing governance
// policies without widening the public AuthorizationPolicy interface. Full
// governance authorization permits every guarded operation. Partial
// authorization permits only the operation carried by that policy.
func govPolicyAllowsDeployment(
	authz types.AuthorizationPolicy,
	operation wasmDeploymentOperation,
) bool {
	switch policy := authz.(type) {
	case GovAuthorizationPolicy:
		return true
	case *GovAuthorizationPolicy:
		return policy != nil
	case PartialGovAuthorizationPolicy:
		return partialGovPolicyAllowsDeployment(policy, operation)
	case *PartialGovAuthorizationPolicy:
		return policy != nil && partialGovPolicyAllowsDeployment(*policy, operation)
	default:
		return false
	}
}

// partialGovPolicyAllowsDeployment maps propagated governance authorization to
// the guarded operations that already have matching authorization actions.
// Upload has no partial governance action and therefore requires full policy.
func partialGovPolicyAllowsDeployment(
	policy PartialGovAuthorizationPolicy,
	operation wasmDeploymentOperation,
) bool {
	switch operation {
	case wasmDeploymentMigrate:
		return policy.action == types.AuthZActionMigrateContract
	default:
		return false
	}
}
