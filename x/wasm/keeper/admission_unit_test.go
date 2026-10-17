package keeper

import (
	"errors"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const guardedChainID = "guarded-chain"

// stubSudoPermissionSource records root reads so tests can prove that disabled and
// non-matching-chain guards do not touch sudo state.
type stubSudoPermissionSource struct {
	root    sdk.AccAddress
	err     error
	calls   int
	members map[string]bool
}

func (s *stubSudoPermissionSource) CheckPermissions(actor sdk.AccAddress, _ sdk.Context, role string) error {
	s.calls++
	if s.err != nil {
		return s.err
	}
	if s.root.Equals(actor) || (role == "wasm_deployer" && s.members[actor.String()]) {
		return nil
	}
	return sdkerrors.ErrUnauthorized.Wrapf("requires root %s or role %q; actor %s", s.root, role, actor)
}

// TestRequireActorIsAuthedDeployer covers chain scoping, live root lookup,
// governance bypasses, and fail-closed root lookup errors.
func TestRequireActorIsAuthedDeployer(t *testing.T) {
	root := DeterministicAccountAddress(t, 1)
	other := DeterministicAccountAddress(t, 2)
	rootErr := errors.New("sudo root unavailable")

	tests := map[string]struct {
		chainID      string
		guardEnabled bool
		rootErr      error
		actor        sdk.AccAddress
		authz        types.AuthorizationPolicy
		operation    wasmDeploymentOperation
		wantErr      error
		wantCalls    int
	}{
		"guard disabled": {
			chainID:   guardedChainID,
			actor:     other,
			authz:     DefaultAuthorizationPolicy{},
			operation: wasmDeploymentUpload,
		},
		"other chain": {
			chainID:      "other-chain",
			guardEnabled: true,
			actor:        other,
			authz:        DefaultAuthorizationPolicy{},
			operation:    wasmDeploymentUpload,
		},
		"sudo root": {
			chainID:      guardedChainID,
			guardEnabled: true,
			actor:        root,
			authz:        DefaultAuthorizationPolicy{},
			operation:    wasmDeploymentUpload,
			wantCalls:    1,
		},
		"non-root actor": {
			chainID:      guardedChainID,
			guardEnabled: true,
			actor:        other,
			authz:        DefaultAuthorizationPolicy{},
			operation:    wasmDeploymentMigrate,
			wantErr:      sdkerrors.ErrUnauthorized,
			wantCalls:    1,
		},
		"root lookup failure": {
			chainID:      guardedChainID,
			guardEnabled: true,
			rootErr:      rootErr,
			actor:        other,
			authz:        DefaultAuthorizationPolicy{},
			operation:    wasmDeploymentUpload,
			wantErr:      rootErr,
			wantCalls:    1,
		},
		"full governance policy": {
			chainID:      guardedChainID,
			guardEnabled: true,
			rootErr:      rootErr,
			actor:        other,
			authz:        GovAuthorizationPolicy{},
			operation:    wasmDeploymentUpload,
		},
		"matching partial migrate policy": {
			chainID:      guardedChainID,
			guardEnabled: true,
			rootErr:      rootErr,
			actor:        other,
			authz:        NewPartialGovAuthorizationPolicy(DefaultAuthorizationPolicy{}, types.AuthZActionMigrateContract),
			operation:    wasmDeploymentMigrate,
		},
		"partial policy does not authorize upload": {
			chainID:      guardedChainID,
			guardEnabled: true,
			actor:        other,
			authz:        NewPartialGovAuthorizationPolicy(DefaultAuthorizationPolicy{}, types.AuthZActionInstantiate),
			operation:    wasmDeploymentUpload,
			wantErr:      sdkerrors.ErrUnauthorized,
			wantCalls:    1,
		},
		"partial policy must match operation": {
			chainID:      guardedChainID,
			guardEnabled: true,
			actor:        other,
			authz:        NewPartialGovAuthorizationPolicy(DefaultAuthorizationPolicy{}, types.AuthZActionInstantiate),
			operation:    wasmDeploymentMigrate,
			wantErr:      sdkerrors.ErrUnauthorized,
			wantCalls:    1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			source := &stubSudoPermissionSource{root: root, err: test.rootErr}
			keeper := Keeper{}
			if test.guardEnabled {
				keeper.wasmDeployerGuard = &wasmDeployerGuard{
					chainID:         guardedChainID,
					sudoPermissions: source,
				}
			}

			err := keeper.requireActorIsAuthedDeployer(
				admissionTestContext(test.chainID),
				test.actor,
				test.authz,
				test.operation,
			)
			if test.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantErr.Error())
			}
			require.Equal(t, test.wantCalls, source.calls)
		})
	}
}

// TestRequireActorIsAuthedDeployerErrorText keeps rejected transactions useful
// to operators by naming the operation, required root, and rejected actor.
func TestRequireActorIsAuthedDeployerErrorText(t *testing.T) {
	root := DeterministicAccountAddress(t, 1)
	actor := DeterministicAccountAddress(t, 2)
	keeper := Keeper{wasmDeployerGuard: &wasmDeployerGuard{
		chainID:         guardedChainID,
		sudoPermissions: &stubSudoPermissionSource{root: root},
	}}

	err := keeper.requireActorIsAuthedDeployer(
		admissionTestContext(guardedChainID),
		actor,
		DefaultAuthorizationPolicy{},
		wasmDeploymentMigrate,
	)
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	require.ErrorContains(t, err, "contract migration")
	require.ErrorContains(t, err, root.String())
	require.ErrorContains(t, err, actor.String())
}

// TestGuardedKeeperEntrypointsRejectBeforeStateAccess uses an otherwise
// unusable keeper and inputs to prove that all three entrypoints reject before
// reading Wasm state or invoking the VM.
func TestGuardedKeeperEntrypointsRejectBeforeStateAccess(t *testing.T) {
	root := DeterministicAccountAddress(t, 1)
	actor := DeterministicAccountAddress(t, 2)
	source := &stubSudoPermissionSource{root: root}
	keeper := Keeper{wasmDeployerGuard: &wasmDeployerGuard{
		chainID:         guardedChainID,
		sudoPermissions: source,
	}}
	ctx := admissionTestContext(guardedChainID)
	authz := DefaultAuthorizationPolicy{}

	_, _, err := keeper.create(ctx, actor, []byte("not wasm"), nil, authz)
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)

	_, err = keeper.migrate(ctx, root, actor, 1, nil, authz)
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	require.Equal(t, 2, source.calls)
}

// TestWithWasmDeployerGuard verifies explicit configuration and rejects options
// that could silently disable or misconfigure the guard.
func TestWithWasmDeployerGuard(t *testing.T) {
	source := &stubSudoPermissionSource{}
	keeper := Keeper{}

	WithWasmDeployerGuard(guardedChainID, source).apply(&keeper)
	require.Equal(t, guardedChainID, keeper.wasmDeployerGuard.chainID)
	require.Same(t, source, keeper.wasmDeployerGuard.sudoPermissions)

	require.Panics(t, func() { WithWasmDeployerGuard("", source) })
	require.Panics(t, func() { WithWasmDeployerGuard(guardedChainID, nil) })
}

// admissionTestContext builds the minimum context needed to test admission
// without mounting stores. Any later keeper access would make the test fail.
func admissionTestContext(chainID string) sdk.Context {
	return sdk.NewContext(
		store.NewCommitMultiStore(nil),
		tmproto.Header{ChainID: chainID},
		false,
		log.NewNopLogger(),
	)
}
