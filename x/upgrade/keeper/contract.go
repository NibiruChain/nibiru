package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	contracttypes "github.com/NibiruChain/nibiru/v2/x/upgrade/types"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// The address marker is native bootstrap metadata, not a mirror of contract
// records. Historical contexts with no marker continue using their old state.
var contractAddressKey = []byte{0x04}

type TimeTarget = contracttypes.TimeTarget

type SudoKeeper interface {
	GetRootAddr(sdk.Context) (sdk.AccAddress, error)
	CheckPermissions(sdk.AccAddress, sdk.Context, string) error
	UpdateRoleMembers(context.Context, *sudo.MsgUpdateRoleMembers) (*sudo.MsgUpdateRoleMembersResponse, error)
}

func CreatorAddr() sdk.AccAddress {
	return authtypes.NewModuleAddress(contracttypes.ContractCreatorName)
}

// PredictContractAddr returns the initial Instantiate2 address for checksum,
// the stable module creator, and the contract salt. The bound [ContractBridge]
// performs the derivation with fixMsg=false, excluding genesis state. This
// predicts a fresh deployment; [Keeper.ContractAddr] identifies an existing
// deployment even after its Wasm code has been migrated.
func (k Keeper) PredictContractAddr(checksum []byte) sdk.AccAddress {
	return k.wasm.PredictContractAddress(checksum, CreatorAddr(), []byte(contracttypes.ContractSalt))
}

func (k *Keeper) BindWasm(wasm ContractBridge, permissions SudoKeeper, code func() []byte) {
	k.wasm, k.permissions, k.contractCode = wasm, permissions, code
}

func (k Keeper) ContractAddr(ctx sdk.Context) (sdk.AccAddress, bool) {
	bz := ctx.KVStore(k.storeKey).Get(contractAddressKey)
	if len(bz) == 0 {
		return nil, false
	}
	if len(bz) != wasmtypes.ContractAddrLen {
		panic("invalid upgrade contract address marker")
	}
	return sdk.AccAddress(bz), true
}
func (k Keeper) ContractActive(ctx sdk.Context) bool { _, ok := k.ContractAddr(ctx); return ok }

func (k Keeper) queryAt(ctx sdk.Context, addr sdk.AccAddress, request any, result any) error {
	if k.wasm == nil {
		return fmt.Errorf("upgrade Wasm keeper not bound")
	}
	msg, err := json.Marshal(request)
	if err != nil {
		return err
	}
	response, _, err := k.wasm.QuerySmartBounded(ctx, addr, msg, contracttypes.QueryGasLimit)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(response, result); err != nil {
		return fmt.Errorf("decode x-upgrade query: %w", err)
	}
	return nil
}

func (k Keeper) queryContract(ctx sdk.Context, request any, result any) error {
	addr, ok := k.ContractAddr(ctx)
	if !ok {
		return fmt.Errorf("upgrade contract is not active")
	}
	return k.queryAt(ctx, addr, request, result)
}

func (k Keeper) mustQuery(ctx sdk.Context, request any, result any) {
	if err := k.queryContract(ctx, request, result); err != nil {
		panic(fmt.Errorf("query x-upgrade: %w", err))
	}
}

func (k Keeper) sudoContract(ctx sdk.Context, request any) error {
	addr, ok := k.ContractAddr(ctx)
	if !ok || k.wasm == nil {
		return fmt.Errorf("upgrade contract is not active")
	}
	msg, err := json.Marshal(request)
	if err != nil {
		return err
	}
	_, err = k.wasm.Sudo(ctx, addr, msg)
	return err
}

func (k Keeper) mustSudo(ctx sdk.Context, request any) {
	if err := k.sudoContract(ctx, request); err != nil {
		panic(fmt.Errorf("sudo x-upgrade: %w", err))
	}
}

func (k Keeper) executeGovernance(ctx sdk.Context, request any) error {
	addr, ok := k.ContractAddr(ctx)
	if !ok || k.wasm == nil {
		return fmt.Errorf("upgrade contract is not active")
	}
	msg, err := json.Marshal(request)
	if err != nil {
		return err
	}
	authority, err := sdk.AccAddressFromBech32(k.authority)
	if err != nil {
		return err
	}
	_, err = k.wasm.Execute(ctx, addr, authority, msg, sdk.NewCoins())
	return err
}

func (k Keeper) TimeTarget(ctx sdk.Context) (*TimeTarget, error) {
	if !k.ContractActive(ctx) {
		return nil, nil
	}
	var target *TimeTarget
	err := k.queryContract(ctx, map[string]any{"time_target": struct{}{}}, &target)
	return target, err
}

func (k Keeper) SelectExecutionHeight(ctx sdk.Context) error {
	addr, ok := k.ContractAddr(ctx)
	if !ok {
		return nil
	}
	msg := []byte(`{"select_execution_height":{}}`)
	_, err := k.wasm.SudoWithErrorCause(ctx, addr, msg)
	return err
}

func (k Keeper) contractPlan(ctx sdk.Context) (upgradetypes.Plan, bool) {
	var p *contracttypes.Plan
	k.mustQuery(ctx, map[string]any{"current_plan": struct{}{}}, &p)
	if p == nil {
		return upgradetypes.Plan{}, false
	}
	return upgradetypes.Plan{Name: p.Name, Height: p.Height, Info: p.Info}, true
}

func (k Keeper) snapshotLegacy(ctx sdk.Context) contracttypes.Snapshot {
	snap := contracttypes.Snapshot{Completed: []contracttypes.CompletedUpgrade{}, ModuleVersions: []contracttypes.ModuleVersion{}}
	if p, ok := k.GetUpgradePlan(ctx); ok {
		snap.Plan = &contracttypes.Plan{Name: p.Name, Height: p.Height, Info: p.Info}
	}
	it := sdk.KVStorePrefixIterator(ctx.KVStore(k.storeKey), []byte{upgradetypes.DoneByte})
	for ; it.Valid(); it.Next() {
		name, height := parseDoneKey(it.Key())
		snap.Completed = append(snap.Completed, contracttypes.CompletedUpgrade{Name: name, Height: height})
	}
	it.Close()
	for _, v := range k.GetModuleVersions(ctx) {
		snap.ModuleVersions = append(snap.ModuleVersions, contracttypes.ModuleVersion{Name: v.Name, Version: v.Version})
	}
	if bz := ctx.KVStore(k.storeKey).Get([]byte{upgradetypes.ProtocolVersionByte}); bz != nil {
		if len(bz) != 8 {
			panic("invalid legacy protocol version")
		}
		v := binary.BigEndian.Uint64(bz)
		snap.ProtocolVersion = &v
	}
	return snap
}

func (k Keeper) deleteLegacy(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(upgradetypes.PlanKey())
	store.Delete([]byte{upgradetypes.ProtocolVersionByte})
	for _, prefix := range []byte{upgradetypes.DoneByte, upgradetypes.VersionMapByte} {
		it := sdk.KVStorePrefixIterator(store, []byte{prefix})
		var keys [][]byte
		for ; it.Valid(); it.Next() {
			keys = append(keys, bytes.Clone(it.Key()))
		}
		it.Close()
		for _, key := range keys {
			store.Delete(key)
		}
	}
}

func (k Keeper) findContract(ctx sdk.Context) (sdk.AccAddress, bool) {
	var found sdk.AccAddress
	k.wasm.IterateContractsByCreator(ctx, CreatorAddr(), func(addr sdk.AccAddress) bool { found = addr; return true })
	return found, found != nil
}

// ActivateContract performs the enabling upgrade after normal module migrations.
// Outer ApplyUpgrade owns the subsequent version, protocol and completion writes.
func (k *Keeper) ActivateContract(ctx sdk.Context) error {
	if k.ContractActive(ctx) {
		return nil
	}
	if k.wasm == nil || k.contractCode == nil {
		return fmt.Errorf("upgrade contract dependencies not bound")
	}
	cache, write := ctx.CacheContext()
	snap := k.snapshotLegacy(cache)
	addr, found := k.findContract(cache)
	if !found {
		code := k.contractCode()
		checksum := sha256.Sum256(code)
		expected := k.PredictContractAddr(checksum[:])
		codeID, _, err := k.wasm.CreateCode(cache, CreatorAddr(), code, &wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeNobody})
		if err != nil {
			return err
		}
		msg, err := json.Marshal(contracttypes.InstantiateMsg{GovernanceAuthority: k.authority, State: snap})
		if err != nil {
			return err
		}
		addr, _, err = k.wasm.Instantiate2(cache, codeID, CreatorAddr(), expected, msg, "x-upgrade", sdk.NewCoins(), []byte(contracttypes.ContractSalt), false)
		if err != nil {
			return err
		}
		if !addr.Equals(expected) {
			return fmt.Errorf("x-upgrade address differs from prediction")
		}
	}
	var actual contracttypes.Snapshot
	if err := k.queryAt(cache, addr, map[string]any{"snapshot": struct{}{}}, &actual); err != nil {
		return err
	}
	if !reflect.DeepEqual(snap, actual) {
		return fmt.Errorf("x-upgrade imported state differs from legacy snapshot")
	}
	if err := k.ensureDeployerRole(cache, addr); err != nil {
		return err
	}
	k.deleteLegacy(cache)
	cache.KVStore(k.storeKey).Set(contractAddressKey, addr)
	write()
	return nil
}

// InitContract runs after Wasm genesis. Imported contract state wins over binary
// defaults; a fresh chain is seeded once with the complete module version map.
func (k *Keeper) InitContract(ctx sdk.Context, versions module.VersionMap) error {
	if k.wasm == nil {
		return nil
	}
	if addr, found := k.findContract(ctx); found {
		var config struct {
			GovernanceAuthority string `json:"governance_authority"`
			Delay               uint64 `json:"activation_delay_blocks,string"`
		}
		if err := k.queryAt(ctx, addr, map[string]any{"config": struct{}{}}, &config); err != nil {
			return err
		}
		if config.GovernanceAuthority != k.authority || config.Delay < 2 || config.Delay > uint64(^uint64(0)>>1) {
			return fmt.Errorf("invalid imported x-upgrade configuration")
		}

		var snap contracttypes.Snapshot
		if err := k.queryAt(ctx, addr, map[string]any{"snapshot": struct{}{}}, &snap); err != nil {
			return err
		}
		k.deleteLegacy(ctx)
		ctx.KVStore(k.storeKey).Set(contractAddressKey, addr)
		if k.versionSetter != nil && snap.ProtocolVersion != nil {
			k.versionSetter.SetProtocolVersion(*snap.ProtocolVersion)
		}
		return nil
	}
	k.SetModuleVersionMap(ctx, versions)
	return k.ActivateContract(ctx)
}

func (k Keeper) ensureDeployerRole(ctx sdk.Context, addr sdk.AccAddress) error {
	if k.permissions == nil {
		return fmt.Errorf("upgrade sudo keeper not bound")
	}
	if k.permissions.CheckPermissions(addr, ctx, sudo.RoleWasmDeployer) == nil {
		return nil
	}
	root, err := k.permissions.GetRootAddr(ctx)
	if err != nil {
		return err
	}
	_, err = k.permissions.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: root.String(), Role: sudo.RoleWasmDeployer, Add: []string{addr.String()}})
	return err
}

func moduleRecords(vm module.VersionMap) []contracttypes.ModuleVersion {
	names := make([]string, 0, len(vm))
	for name := range vm {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]contracttypes.ModuleVersion, 0, len(names))
	for _, name := range names {
		result = append(result, contracttypes.ModuleVersion{Name: name, Version: vm[name]})
	}
	return result
}

// Decode Rust's Uint64 JSON strings while retaining the missing-value sentinel.
type stringUint64Result struct{ Target **uint64 }

func (v *stringUint64Result) UnmarshalJSON(bz []byte) error {
	if bytes.Equal(bz, []byte("null")) {
		*v.Target = nil
		return nil
	}
	var text string
	if err := json.Unmarshal(bz, &text); err != nil {
		return err
	}
	n, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return err
	}
	*v.Target = &n
	return nil
}

func (k Keeper) scheduleContract(ctx sdk.Context, plan upgradetypes.Plan) error {
	if err := plan.ValidateScheduleRequest(); err != nil {
		return err
	}
	request := contracttypes.ScheduleRequest{Name: plan.Name, Height: plan.Height, Info: plan.Info}
	if !plan.Time.IsZero() {
		seconds := plan.Time.Unix()
		if seconds < 0 || uint64(seconds) > (^uint64(0)-uint64(plan.Time.Nanosecond()))/1_000_000_000 {
			return fmt.Errorf("target time is outside Wasm timestamp range")
		}
		nanos := uint64(seconds)*1_000_000_000 + uint64(plan.Time.Nanosecond())
		request.TargetTime = &nanos
	}
	old, found := k.GetUpgradePlan(ctx)
	if err := k.executeGovernance(ctx, map[string]any{"schedule": map[string]any{"plan": request}}); err != nil {
		return err
	}
	if found {
		k.ClearIBCState(ctx, old.Height)
	}
	k.ClearSelectionFailure(ctx)
	return nil
}

// CancelUpgrade is the governance path. ClearUpgradePlan remains the trusted
// native lifecycle operation used after ApplyUpgrade and skip-height handling.
func (k Keeper) CancelUpgrade(ctx sdk.Context) error {
	if !k.ContractActive(ctx) {
		k.ClearUpgradePlan(ctx)
		return nil
	}
	old, found := k.GetUpgradePlan(ctx)
	if err := k.executeGovernance(ctx, map[string]any{"cancel": struct{}{}}); err != nil {
		return err
	}
	if found {
		k.ClearIBCState(ctx, old.Height)
	}
	k.ClearSelectionFailure(ctx)
	return nil
}

// WarmProtocolVersion restores only the native host field from canonical state.
// It writes no consensus state and is safe after loading a committed database.
func (k Keeper) WarmProtocolVersion(ctx sdk.Context) {
	if k.versionSetter != nil {
		k.versionSetter.SetProtocolVersion(k.getProtocolVersion(ctx))
	}
}
