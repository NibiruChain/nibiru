package keeper

import (
	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// wasmDeploymentOperation identifies the bytecode lifecycle operation checked
// by the temporary deployment guard. The operation also selects any existing
// action-specific governance authorization that remains valid.
type wasmDeploymentOperation string

const (
	wasmDeploymentUpload      wasmDeploymentOperation = "code upload"
	wasmDeploymentInstantiate wasmDeploymentOperation = "contract instantiation"
	wasmDeploymentMigrate     wasmDeploymentOperation = "contract migration"
)

// wasmDeployerGuard limits ordinary deployment operations on one exact chain.
// Keeping the chain ID and root source in the keeper makes the policy apply to
// message-server, contract-submessage, authz, and direct keeper callers.
//
// NOTE: This v2.20 mainnet policy is temporary. Remove or replace it in the
// planned follow-up upgrade.
type wasmDeployerGuard struct {
	chainID        string
	sudoRootSource types.SudoRootSource
}

// requireActorIsAuthedDeployer applies the temporary deployment policy before
// a caller can compile, instantiate, or migrate Wasm code. It is a no-op when
// the guard is not configured or the context has another chain ID.
//
// Existing governance authorization bypasses the root comparison. Ordinary
// calls resolve the current x/sudo root on every attempt, so root rotation takes
// effect without copying an address into Wasm state. A lookup failure rejects
// the operation. Passing this guard does not bypass later Wasm access or admin
// checks.
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

	root, err := guard.sudoRootSource.GetRootAddr(ctx)
	if err != nil {
		return sdkioerrors.Wrapf(err, "get x/sudo root for Wasm %s", operation)
	}
	if root.Equals(actor) {
		return nil
	}

	return sdkioerrors.Wrapf(
		sdkerrors.ErrUnauthorized,
		"Wasm %s requires x/sudo root %q; actor %q is not authorized",
		operation,
		root.String(),
		actor.String(),
	)
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
	case wasmDeploymentInstantiate:
		return policy.action == types.AuthZActionInstantiate
	case wasmDeploymentMigrate:
		return policy.action == types.AuthZActionMigrateContract
	default:
		return false
	}
}
