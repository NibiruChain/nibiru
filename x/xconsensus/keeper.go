package xconsensus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// ModuleName names the app module that refreshes the x/consensus store from
// x-consensus at the end of every block. It has no store of its own.
const ModuleName = "xconsensus"

// ContractLabel is the Wasm label of the x-consensus contract.
const ContractLabel = "x-consensus"

// ContractSalt is the fixed instantiate2 salt. With the fixed creator
// (CreatorAddr) it makes the address depend only on the code checksum (see
// PredictContractAddr), so it can be published before the contract exists.
// The app itself finds the contract through the creator index (see
// Keeper.ContractAddr), so a rebuild of the embedded Wasm, which changes the
// address, breaks nothing.
var ContractSalt = []byte("x-consensus")

// QueryGasLimit bounds the gas of the per-block params query, in SDK gas.
//
// Measured October 28, 2026 (TestQueryGasUsage): one params query on the
// embedded contract, unpinned, uses 66,013 gas and about 0.1 ms. Most of it
// is the fixed instance setup cost (InstantiateContractCosts); the contract
// itself does one storage read and a JSON encode. 1,000,000 gives 15 times
// headroom for a future contract version with more fields.
//
// The limit is also the worst case a broken contract can cost every block:
// a query that loops runs until the limit, which took about 0.12 s at
// 1,000,000 gas and 0.43 s at 3,000,000 (TestQuerySmartBounded, hackatom
// hashing loop). At the sub-second block times x-consensus exists for, a
// tighter limit keeps a bad migration from multiplying the block time while
// governance fixes it.
//
// It is a Go constant, not a param or node config, so every node meters the
// query the same way: a node-local limit could let some nodes refresh and
// others not, which would split the app hash.
const QueryGasLimit uint64 = 1_000_000

// CreatorAddr is the fixed creator of the x-consensus contract, the module
// address of ModuleName. No key controls it, so only this module can create
// contracts under it, which makes "first contract created by CreatorAddr" a
// safe way to find x-consensus on any chain (mainnet, testnets, localnet,
// tests) and after any rebuild of the embedded Wasm.
func CreatorAddr() sdk.AccAddress {
	return authtypes.NewModuleAddress(ModuleName)
}

// ParamStore is the x/consensus keeper as this module uses it: the persisted
// copy of the consensus params. baseapp reads the same store through its own
// ParamStore at BeginBlock, for every tx, and at EndBlock.
type ParamStore interface {
	Get(ctx sdk.Context) (*cmtproto.ConsensusParams, error)
	Set(ctx sdk.Context, cp *cmtproto.ConsensusParams)
}

// SudoKeeper is the x/sudo keeper as this module uses it: to grant
// x-consensus the wasm_deployer role through the role-editing message
// handler, the same path the v2.21 upgrade used to seed deployers.
type SudoKeeper interface {
	GetRootAddr(ctx sdk.Context) (sdk.AccAddress, error)
	CheckPermissions(actor sdk.AccAddress, ctx sdk.Context, role string) error
	UpdateRoleMembers(
		goCtx context.Context, msg *sudo.MsgUpdateRoleMembers,
	) (*sudo.MsgUpdateRoleMembersResponse, error)
}

// ErrNoStoredParams means the x/consensus store has no params to seed
// x-consensus from. Nibiru BFT always sends consensus params in InitChain and
// baseapp stores them before module genesis, so only test harnesses that call
// InitChain without params (for example the ibc-go testing package) hit this.
// The app then runs without the contract and keeps static params.
var ErrNoStoredParams = errors.New("x/consensus store has no consensus params")

// commitWaitUnset is the commitWait sentinel for "no chain value". Real
// values are non-negative durations in nanoseconds.
const commitWaitUnset int64 = -1

// Keeper wires x-consensus into the app. The contract is the only writer of
// consensus params; the x/consensus store is a deterministic copy that the
// module EndBlocker refreshes once per block (Refresh). Every reader, baseapp
// included, reads that store, so all nodes see the same params at the same
// height even right after a restart or state sync.
//
// The Keeper also holds two in-memory values that never decide state:
//   - the contract address, cached after the first lookup (it never changes
//     once the contract exists; an empty cache just looks it up again), and
//   - the commit wait for Nibiru BFT (CommitTimeoutFunc), which is liveness
//     only and is not in the x/consensus store because its proto has no field
//     for it.
//
// Use it through a pointer: it holds atomics.
type Keeper struct {
	wasm   *wasmkeeper.Keeper
	params ParamStore
	sudo   SudoKeeper

	// queryGasLimit is QueryGasLimit outside tests. Tests lower it to force
	// an out-of-gas query (see export_test.go).
	queryGasLimit uint64

	contract   atomic.Pointer[sdk.AccAddress]
	commitWait atomic.Int64
}

// NewKeeper returns a Keeper with the commit wait unset. wasm must be the
// app's Wasm keeper (by pointer, so later keeper options apply), params the
// x/consensus keeper, and sudo the x/sudo keeper.
func NewKeeper(
	wasm *wasmkeeper.Keeper, params ParamStore, sudo SudoKeeper,
) *Keeper {
	k := &Keeper{
		wasm:          wasm,
		params:        params,
		sudo:          sudo,
		queryGasLimit: QueryGasLimit,
	}
	k.commitWait.Store(commitWaitUnset)
	return k
}

// ContractAddr returns the x-consensus address, or false if the contract does
// not exist yet (before genesis or the v2.22 upgrade created it). It uses the
// in-memory cache when set and otherwise reads the Wasm creator index, which
// genesis import rebuilds, so it also works after export and import. A miss is
// not cached, so a later EnsureXConsensus is picked up.
func (k *Keeper) ContractAddr(ctx sdk.Context) (sdk.AccAddress, bool) {
	if addr := k.contract.Load(); addr != nil {
		return *addr, true
	}
	addr, found := k.findContract(ctx)
	if found {
		k.contract.Store(&addr)
	}
	return addr, found
}

// findContract returns the first contract created by CreatorAddr, without
// touching the cache. Only this module can create under that address, and
// EnsureXConsensus creates at most one, so the first is x-consensus.
func (k *Keeper) findContract(ctx sdk.Context) (sdk.AccAddress, bool) {
	var found sdk.AccAddress
	k.wasm.IterateContractsByCreator(ctx, CreatorAddr(),
		func(addr sdk.AccAddress) bool {
			found = addr
			return true
		})
	return found, found != nil
}

// EnsureXConsensus creates x-consensus if no contract by CreatorAddr exists,
// and in every case makes sure the contract holds the x/sudo wasm_deployer
// role (ensureDeployerRole). It is idempotent, so it is safe to call at every
// genesis (fresh or imported) and in the upgrade handler, and a chain whose
// contract was created before the role grant existed gets the role on the
// next call.
//
// It stores the embedded code (instantiation permission Nobody, so only
// governance can instantiate copies) and instantiates it with Instantiate2
// (fixed salt, fixMsg=false), seeded from the current x/consensus store with
// min_block_commit_duration_ms unset, so creating it changes no behavior. It
// goes through the governance permission keeper, so the mainnet Wasm deployer
// guard and the code upload policy pass for this module-owned creator.
//
// The contract is its own Wasm admin. Instantiate2 needs the admin before the
// address exists, so EnsureXConsensus passes PredictContractAddr and checks
// the result. The x/sudo root then migrates the contract, or moves its admin,
// by executing Migrate or UpdateAdmin on it, so those rights follow the live
// root just as param updates do, with no MsgUpdateAdmin after a root
// rotation. Governance stays the break-glass, since Nibiru's Wasm keeper lets
// a gov proposal migrate any contract regardless of admin
// (x/wasm/keeper/authz_policy.go:66).
//
// All writes happen in a cache context that is written only on success, so a
// failure leaves state untouched.
func (k *Keeper) EnsureXConsensus(ctx sdk.Context) error {
	cacheCtx, write := ctx.CacheContext()
	if addr, found := k.findContract(cacheCtx); found {
		if err := k.ensureDeployerRole(cacheCtx, addr); err != nil {
			return err
		}
		write()
		k.contract.Store(&addr)
		return nil
	}

	stored, err := k.params.Get(cacheCtx)
	if err != nil {
		return fmt.Errorf("read x/consensus params: %w", err)
	}
	if stored.Block == nil && stored.Evidence == nil && stored.Validator == nil {
		return ErrNoStoredParams
	}
	seed, err := ConsensusParamsFromProto(*stored)
	if err != nil {
		return fmt.Errorf("seed x-consensus from x/consensus params: %w", err)
	}
	initMsg, err := json.Marshal(InstantiateMsg{Params: seed})
	if err != nil {
		return err
	}

	gov := wasmkeeper.NewGovPermissionKeeper(k.wasm)
	codeID, checksum, err := gov.Create(cacheCtx, CreatorAddr(), XConsensusWasm,
		&wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeNobody})
	if err != nil {
		return fmt.Errorf("store x-consensus code: %w", err)
	}
	// The contract is its own Wasm admin. Instantiate2 sets the admin before
	// the address exists, so pass the address it is about to derive.
	admin := PredictContractAddr(checksum)
	addr, _, err := gov.Instantiate2(cacheCtx, codeID, CreatorAddr(), admin,
		initMsg, ContractLabel, sdk.NewCoins(), ContractSalt, false)
	if err != nil {
		return fmt.Errorf("instantiate x-consensus: %w", err)
	}
	if !addr.Equals(admin) {
		return fmt.Errorf(
			"x-consensus address %s differs from the predicted %s", addr, admin)
	}
	if err := k.ensureDeployerRole(cacheCtx, addr); err != nil {
		return err
	}

	write()
	k.contract.Store(&addr)
	k.commitWait.Store(commitWaitUnset)
	ctx.Logger().Info("created x-consensus", "module", ModuleName,
		"address", addr.String(), "code_id", codeID)
	return nil
}

// ensureDeployerRole adds addr, the x-consensus contract, to the x/sudo
// wasm_deployer role unless it already passes that role's check.
//
// Why: on mainnet the Wasm deployer guard (x/wasm/keeper/admission.go) checks
// the actor of every migration, and for ExecuteMsg.Migrate that actor is the
// contract, not the root. Without the role, the root could not migrate
// x-consensus through the contract on mainnet. The grant applies on every
// chain, where it is harmless without the guard, so behavior is the same on
// every network.
//
// Why this is safe: the contract can only send Migrate and UpdateAdmin for
// its own address, and only after the live x/sudo root check, so the role
// gives it no deployment power beyond migrating itself on the root's order.
// The role belongs to the contract address, not to its code. Any code the
// root later migrates x-consensus into inherits wasm_deployer, which on
// mainnet includes upload rights, so treat each x-consensus migration as a
// deployer grant. EnsureXConsensus also re-grants the role whenever it runs
// (today only at genesis and in v2.22), so a root that revoked it on purpose
// would see it return the next time a handler calls EnsureXConsensus.
//
// Rejected alternatives: exempting self-migration in the guard would weaken
// it for every self-admin contract, and a manual runbook grant would leave
// mainnet migration broken until someone remembered it.
//
// It goes through the role-editing message handler with the x/sudo root as
// sender, like the v2.21 deployer seed (app/upgrades/v2_21_0.go), so x/sudo
// normalizes the state and emits its usual event. The caller runs it in the
// same cache context as contract creation, so a failure rolls back both.
func (k *Keeper) ensureDeployerRole(
	ctx sdk.Context, addr sdk.AccAddress,
) error {
	if k.sudo.CheckPermissions(addr, ctx, sudo.RoleWasmDeployer) == nil {
		return nil
	}
	root, err := k.sudo.GetRootAddr(ctx)
	if err != nil {
		return fmt.Errorf("grant x-consensus the wasm_deployer role: "+
			"read x/sudo root: %w", err)
	}
	_, err = k.sudo.UpdateRoleMembers(sdk.WrapSDKContext(ctx),
		&sudo.MsgUpdateRoleMembers{
			Sender: root.String(),
			Role:   sudo.RoleWasmDeployer,
			Add:    []string{addr.String()},
		})
	if err != nil {
		return fmt.Errorf("grant x-consensus the wasm_deployer role: %w", err)
	}
	return nil
}

// PredictContractAddr returns the address Instantiate2 gives x-consensus for
// code with the given checksum: CreatorAddr, ContractSalt, and fixMsg=false,
// which leaves the instantiate message out of the derivation. It matches
// wasmkeeper.PredicableAddressGenerator, which clears the message when fixMsg
// is false before calling BuildContractAddressPredictable.
func PredictContractAddr(checksum []byte) sdk.AccAddress {
	return wasmkeeper.BuildContractAddressPredictable(
		checksum, CreatorAddr(), ContractSalt, []byte{})
}

// QueryParams reads the contract's params with a bounded, recovering smart
// query (wasmkeeper.Keeper.QuerySmartBounded) and decodes them. It never
// writes state and never panics.
func (k *Keeper) QueryParams(ctx sdk.Context) (ConsensusParams, error) {
	addr, found := k.ContractAddr(ctx)
	if !found {
		return ConsensusParams{}, errors.New("x-consensus contract not found")
	}
	req, err := json.Marshal(QueryParams())
	if err != nil {
		return ConsensusParams{}, err
	}
	resp, _, err := k.wasm.QuerySmartBounded(ctx, addr, req, k.queryGasLimit)
	if err != nil {
		return ConsensusParams{}, fmt.Errorf("query x-consensus params: %w", err)
	}
	var params ConsensusParams
	if err := json.Unmarshal(resp, &params); err != nil {
		return ConsensusParams{}, fmt.Errorf("decode x-consensus params: %w", err)
	}
	return params, nil
}

// Refresh copies the contract's params into the x/consensus store. The module
// EndBlocker calls it once per block, after every other EndBlocker, so baseapp
// returns the result to Nibiru BFT from this same EndBlock (applied at H+1),
// and BeginBlock and every tx of H+1 read the same values.
//
// It queries the contract (QueryParams), keeps Version from the stored params
// (the engine copies Version.App into block headers and it belongs to
// x/upgrade), validates the full struct with Nibiru BFT's own ValidateBasic
// plus the commit wait cap (ValidatedProto), and writes only if the encoded
// bytes differ, so an unchanged block writes nothing. On success it also sets
// the commit wait that CommitTimeoutFunc reports from the next height.
//
// On any error it changes nothing: the store keeps its last good params and
// the commit wait keeps its last good value. The caller logs the error. A
// contract mistake must never halt the chain.
func (k *Keeper) Refresh(ctx sdk.Context) error {
	params, err := k.QueryParams(ctx)
	if err != nil {
		return err
	}
	stored, err := k.params.Get(ctx)
	if err != nil {
		return fmt.Errorf("read x/consensus params: %w", err)
	}
	next, err := params.ValidatedProto(stored.Version)
	if err != nil {
		return err
	}
	storedBz, err := stored.Marshal()
	if err != nil {
		return err
	}
	nextBz, err := next.Marshal()
	if err != nil {
		return err
	}
	if !bytes.Equal(storedBz, nextBz) {
		k.params.Set(ctx, &next)
	}
	k.setCommitWait(params)
	return nil
}

// WarmCommitWait loads the commit wait from committed contract state without
// writing anything. The app calls it after loading the latest version and
// before Nibiru BFT starts, so the first height after a restart already uses
// the chain value instead of config.toml. Errors leave the wait unchanged.
func (k *Keeper) WarmCommitWait(ctx sdk.Context) error {
	params, err := k.QueryParams(ctx)
	if err != nil {
		return err
	}
	if _, err := params.ValidatedProto(nil); err != nil {
		return err
	}
	k.setCommitWait(params)
	return nil
}

// setCommitWait stores params' commit wait, or the unset sentinel.
func (k *Keeper) setCommitWait(params ConsensusParams) {
	wait, ok := params.MinBlockCommitDuration()
	if !ok {
		k.commitWait.Store(commitWaitUnset)
		return
	}
	k.commitWait.Store(int64(wait))
}

// CommitWait returns the chain commit wait and true, or false when unset.
// It only loads an atomic, so Nibiru BFT may call it under its consensus
// mutex. The value follows the last successful Refresh, EnsureXConsensus or
// WarmCommitWait.
func (k *Keeper) CommitWait() (time.Duration, bool) {
	v := k.commitWait.Load()
	if v == commitWaitUnset {
		return 0, false
	}
	return time.Duration(v), true
}

// ValidatedProto converts the contract params to the engine's proto type with
// the given version (copied from the store by the caller) and checks them the
// way Nibiru BFT will: ValidateBasic on the full params struct, since the
// engine halts on an invalid update, plus MaxMinBlockCommitDurationMs. The
// contract enforces the same rules; checking again here means a contract bug
// or a bad migration cannot halt the chain.
func (p ConsensusParams) ValidatedProto(
	version *cmtproto.VersionParams,
) (cmtproto.ConsensusParams, error) {
	if ms := p.MinBlockCommitDurationMs; ms != nil &&
		*ms > MaxMinBlockCommitDurationMs {
		return cmtproto.ConsensusParams{}, fmt.Errorf(
			"min_block_commit_duration_ms %d exceeds the %d ms cap",
			*ms, MaxMinBlockCommitDurationMs)
	}
	pb := p.ToProto(version)
	// bft/types ConsensusParamsFromProto dereferences Version; validation
	// does not look at it, so a nil version validates as the zero version.
	full := pb
	if full.Version == nil {
		full.Version = &cmtproto.VersionParams{}
	}
	if err := cmttypes.ConsensusParamsFromProto(full).ValidateBasic(); err != nil {
		return cmtproto.ConsensusParams{}, fmt.Errorf(
			"invalid x-consensus params: %w", err)
	}
	return pb, nil
}
