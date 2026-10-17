package keeper_test

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/NibiruChain/nibiru/v2/eth"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func TestRoleMembershipLifecycle(t *testing.T) {
	_, k, ctx := setup()
	root, eoa, stranger := testutil.NewAccAddress(), testutil.NewAccAddress(), testutil.NewAccAddress()
	contract := sdk.AccAddress(bytes.Repeat([]byte{7}, 32))
	k.Sudoers.Set(ctx, sudo.Sudoers{Root: root.String()})
	edit := &sudo.MsgUpdateRoleMembers{Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{eoa.String(), contract.String(), eoa.String()}}
	_, err := k.UpdateRoleMembers(ctx, edit)
	require.NoError(t, err)
	state, err := k.Sudoers.Get(ctx)
	require.NoError(t, err)
	require.Len(t, state.Roles, 1)
	require.Len(t, state.Roles[0].Members, 2)
	for _, actor := range []sdk.AccAddress{root, eoa, contract} {
		require.NoError(t, k.CheckPermissions(actor, ctx, sudo.RoleWasmDeployer))
	}
	require.NoError(t, k.CheckPermissions(root, ctx, "unassigned"))
	require.Error(t, k.CheckPermissions(eoa, ctx, ""))
	require.Error(t, k.CheckPermissions(eoa, ctx, "unassigned"))
	require.Error(t, k.CheckPermissions(stranger, ctx, sudo.RoleWasmDeployer))
	_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: eoa.String(), Role: sudo.RoleWasmDeployer, Add: []string{stranger.String()}})
	require.Error(t, err)
	_, err = k.EditZeroGasActors(ctx, &sudo.MsgEditZeroGasActors{Sender: eoa.String()})
	require.Error(t, err)
	_, err = k.EditSudoers(ctx, &sudo.MsgEditSudoers{Sender: eoa.String(), Action: string(sudo.EditWasmBlockHooksContract), Contracts: []string{contract.String()}})
	require.Error(t, err)
	_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: root.String(), Role: sudo.RoleWasmDeployer, Remove: []string{eoa.String()}})
	require.NoError(t, err)
	require.Error(t, k.CheckPermissions(eoa, ctx, sudo.RoleWasmDeployer))
	require.NoError(t, k.CheckPermissions(contract, ctx, sudo.RoleWasmDeployer))
	_, err = k.ChangeRoot(ctx, &sudo.MsgChangeRoot{Sender: root.String(), NewRoot: stranger.String()})
	require.NoError(t, err)
	require.Error(t, k.CheckPermissions(root, ctx, ""))
	require.NoError(t, k.CheckPermissions(stranger, ctx, ""))
	_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: stranger.String(), Role: sudo.RoleWasmDeployer, Remove: []string{contract.String()}})
	require.NoError(t, err)
	state, err = k.Sudoers.Get(ctx)
	require.NoError(t, err)
	require.Empty(t, state.Roles)
}

func TestRoleEditValidationIsAtomic(t *testing.T) {
	_, k, ctx := setup()
	root, member := testutil.NewAccAddress(), testutil.NewAccAddress()
	original := sudo.Sudoers{Root: root.String()}
	k.Sudoers.Set(ctx, original)
	for _, edit := range []*sudo.MsgUpdateRoleMembers{
		{Sender: root.String(), Role: "", Add: []string{member.String()}},
		{Sender: root.String(), Role: " spaced "},
		{Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{member.String(), "invalid"}},
		{Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{member.String()}, Remove: []string{member.String()}},
	} {
		_, err := k.UpdateRoleMembers(ctx, edit)
		require.Error(t, err)
		state, err := k.Sudoers.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, original, state)
	}
	for _, action := range []string{"add_contracts", "remove_contracts"} {
		_, err := k.EditSudoers(ctx, &sudo.MsgEditSudoers{Sender: root.String(), Action: action, Contracts: []string{member.String()}})
		require.ErrorContains(t, err, "retired")
	}
}

func TestMigrateLegacySudoStatePreservesHooksAndZeroGas(t *testing.T) {
	_, k, ctx := setup()
	root, former := testutil.NewAccAddress(), testutil.NewAccAddress()
	hook := sdk.AccAddress(bytes.Repeat([]byte{4}, 32)).String()
	zeroGas := sudo.ZeroGasActors{Senders: []string{former.String()}, Contracts: []string{hook}}
	k.ZeroGasActors.Set(ctx, zeroGas)
	k.WasmBlockHooksContract.Set(ctx, hook)
	// Exact old wire layout: root at tag 1, broad contracts at tag 2.
	legacy := protowire.AppendTag(nil, 1, protowire.BytesType)
	legacy = protowire.AppendString(legacy, root.String())
	legacy = protowire.AppendTag(legacy, 2, protowire.BytesType)
	legacy = protowire.AppendString(legacy, former.String())
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 0)
	rawStore := collections.Map[uint64, sudo.Sudoers](k.Sudoers).GetStore(ctx)
	rawStore.Set(key, legacy)
	require.NoError(t, k.Migrate1To2(ctx))
	state, err := k.Sudoers.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, root.String(), state.Root)
	require.Len(t, state.Roles, 2)
	require.NoError(t, k.CheckPermissions(former, ctx, sudo.RoleTFOper))
	require.NoError(t, k.CheckPermissions(former, ctx, sudo.RoleChainParams))
	require.NotEqual(t, legacy, rawStore.Get(key))
	require.Error(t, k.CheckPermissions(former, ctx, sudo.RoleWasmDeployer))
	require.NoError(t, k.CheckPermissions(root, ctx, ""))
	require.Equal(t, hook, k.WasmBlockHooksContract.GetOr(ctx, ""))
	require.Equal(t, zeroGas, k.ZeroGasActors.GetOr(ctx, sudo.ZeroGasActors{}))
	exported := k.ExportGenesis(ctx)
	require.NoError(t, exported.Validate())
	_, fresh, newCtx := setup()
	fresh.InitGenesis(newCtx, *exported)
	require.Equal(t, hook, fresh.WasmBlockHooksContract.GetOr(newCtx, ""))
	require.Equal(t, zeroGas, fresh.ZeroGasActors.GetOr(newCtx, sudo.ZeroGasActors{}))
}

func TestRoleGrantAndRevokeAcrossAddressFormats(t *testing.T) {
	root, member := testutil.NewAccAddress(), testutil.NewAccAddress()
	hex := eth.NibiruAddrToEthAddr(member).Hex()
	formats := []struct{ name, address string }{
		{"bech32", member.String()},
		{"uppercase-bech32", strings.ToUpper(member.String())},
		{"evm-checksum", hex},
		{"evm-lowercase", strings.ToLower(hex)},
	}
	for _, grant := range formats {
		for _, revoke := range formats {
			t.Run(grant.name+"/"+revoke.name, func(t *testing.T) {
				_, k, ctx := setup()
				k.Sudoers.Set(ctx, sudo.Sudoers{Root: root.String()})
				_, err := k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{
					Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{grant.address},
				})
				require.NoError(t, err)
				state, err := k.Sudoers.Get(ctx)
				require.NoError(t, err)
				require.Equal(t, []string{member.String()}, state.Roles[0].Members)
				require.NoError(t, k.CheckPermissions(member, ctx, sudo.RoleWasmDeployer))
				_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{
					Sender: root.String(), Role: sudo.RoleWasmDeployer, Remove: []string{revoke.address},
				})
				require.NoError(t, err)
				require.ErrorIs(t, k.CheckPermissions(member, ctx, sudo.RoleWasmDeployer), sudo.ErrUnauthorized)
			})
		}
	}
}

func TestRoleAddressAliasesDeduplicateAndRejectOverlap(t *testing.T) {
	_, k, ctx := setup()
	root, member := testutil.NewAccAddress(), testutil.NewAccAddress()
	contract := sdk.AccAddress(bytes.Repeat([]byte{9}, 32))
	aliases := []string{member.String(), strings.ToUpper(member.String()), eth.NibiruAddrToEthAddr(member).Hex()}
	k.Sudoers.Set(ctx, sudo.Sudoers{Root: root.String()})
	_, err := k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{
		Sender: root.String(), Role: sudo.RoleWasmDeployer,
		Add: append(append([]string{}, aliases...), strings.ToUpper(contract.String())),
	})
	require.NoError(t, err)
	original, err := k.Sudoers.Get(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{member.String(), contract.String()}, original.Roles[0].Members)
	require.NoError(t, k.CheckPermissions(contract, ctx, sudo.RoleWasmDeployer))
	for _, add := range aliases {
		for _, remove := range aliases {
			_, err := k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{
				Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{add}, Remove: []string{remove},
			})
			require.ErrorContains(t, err, "cannot be added and removed together")
			state, err := k.Sudoers.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, original, state)
		}
	}
	for _, invalid := range []string{"0x1234", "0x" + strings.Repeat("z", 40), "invalid"} {
		for _, edit := range []*sudo.MsgUpdateRoleMembers{
			{Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{member.String(), invalid}},
			{Sender: root.String(), Role: sudo.RoleWasmDeployer, Remove: []string{member.String(), invalid}},
		} {
			_, err := k.UpdateRoleMembers(ctx, edit)
			require.Error(t, err)
			state, err := k.Sudoers.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, original, state)
		}
	}
}

func TestGenesisNormalizesRoleAddressFormats(t *testing.T) {
	_, k, ctx := setup()
	root, member := testutil.NewAccAddress(), testutil.NewAccAddress()
	contract := sdk.AccAddress(bytes.Repeat([]byte{6}, 32))
	genesis := sudo.GenesisState{Sudoers: sudo.Sudoers{
		Root: root.String(), Roles: []sudo.RoleMembers{{Role: sudo.RoleWasmDeployer, Members: []string{
			strings.ToUpper(member.String()), eth.NibiruAddrToEthAddr(member).Hex(), member.String(),
			strings.ToUpper(contract.String()), contract.String(),
		}}},
	}}
	require.NoError(t, genesis.Validate())
	k.InitGenesis(ctx, genesis)
	state, err := k.Sudoers.Get(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{member.String(), contract.String()}, state.Roles[0].Members)
	require.NoError(t, k.CheckPermissions(member, ctx, sudo.RoleWasmDeployer))
	require.NoError(t, k.CheckPermissions(contract, ctx, sudo.RoleWasmDeployer))
	_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{
		Sender: root.String(), Role: sudo.RoleWasmDeployer, Remove: []string{eth.NibiruAddrToEthAddr(member).Hex()},
	})
	require.NoError(t, err)
	require.ErrorIs(t, k.CheckPermissions(member, ctx, sudo.RoleWasmDeployer), sudo.ErrUnauthorized)
	require.NoError(t, k.CheckPermissions(contract, ctx, sudo.RoleWasmDeployer))
	require.NoError(t, k.ExportGenesis(ctx).Validate())
}

func TestLegacyMigrationNormalizesAndRejectsInvalidState(t *testing.T) {
	root, eoa := testutil.NewAccAddress(), testutil.NewAccAddress()
	contract := sdk.AccAddress(bytes.Repeat([]byte{9}, 32))
	for _, members := range [][]string{
		nil,
		{root.String()},
		{root.String(), eoa.String(), strings.ToUpper(eoa.String()), eth.NibiruAddrToEthAddr(eoa).Hex(), contract.String()},
	} {
		_, k, ctx := setup()
		raw := protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), root.String())
		for _, member := range members {
			raw = protowire.AppendString(protowire.AppendTag(raw, 2, protowire.BytesType), member)
		}
		key := make([]byte, 8)
		store := collections.Map[uint64, sudo.Sudoers](k.Sudoers).GetStore(ctx)
		store.Set(key, raw)
		require.NoError(t, k.Migrate1To2(ctx))
		state, err := k.Sudoers.Get(ctx)
		require.NoError(t, err)
		normalized, err := sudo.NormalizeRoleMembers(members)
		require.NoError(t, err)
		if len(members) == 0 {
			require.Empty(t, state.Roles)
		} else {
			require.Len(t, state.Roles, 2)
			for _, role := range state.Roles {
				require.Equal(t, normalized, role.Members)
			}
		}
		require.Error(t, k.CheckPermissions(eoa, ctx, sudo.RoleWasmDeployer))
		exported := k.ExportGenesis(ctx)
		_, fresh, freshCtx := setup()
		fresh.InitGenesis(freshCtx, *exported)
		require.Equal(t, state, fresh.Sudoers.GetOr(freshCtx, sudo.Sudoers{}))
		for _, member := range normalized {
			if member != root.String() {
				continue
			}
			replacement := testutil.NewAccAddress()
			_, err := k.ChangeRoot(ctx, &sudo.MsgChangeRoot{Sender: root.String(), NewRoot: replacement.String()})
			require.NoError(t, err)
			require.NoError(t, k.CheckPermissions(root, ctx, sudo.RoleTFOper))
			require.NoError(t, k.CheckPermissions(root, ctx, sudo.RoleChainParams))
			require.Error(t, k.CheckPermissions(root, ctx, sudo.RoleWasmDeployer))
		}
	}
	for _, invalid := range [][]byte{
		{0x0a, 0xff},
		{0x08, 0x01},
		protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "invalid"),
		protowire.AppendString(protowire.AppendTag(protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), root.String()), 2, protowire.BytesType), "invalid"),
	} {
		_, k, ctx := setup()
		store := collections.Map[uint64, sudo.Sudoers](k.Sudoers).GetStore(ctx)
		key := make([]byte, 8)
		store.Set(key, invalid)
		require.Error(t, k.Migrate1To2(ctx))
		require.Equal(t, invalid, store.Get(key))
	}
}
