package app_test

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	upgradecontract "github.com/NibiruChain/nibiru/v2/x/upgrade/types"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func upgradeContractRecords(t *testing.T, chain *app.NibiruApp, ctx sdk.Context) map[string]string {
	t.Helper()
	addr, found := chain.UpgradeKeeper.ContractAddr(ctx)
	require.True(t, found)
	result := map[string]string{}
	chain.WasmKeeper.IterateContractState(ctx, addr, func(key, value []byte) bool {
		result[hex.EncodeToString(key)] = hex.EncodeToString(value)
		return false
	})
	return result
}

func queryUpgradeSnapshot(t *testing.T, chain *app.NibiruApp, ctx sdk.Context) upgradecontract.Snapshot {
	t.Helper()
	addr, found := chain.UpgradeKeeper.ContractAddr(ctx)
	require.True(t, found)
	bz, err := chain.WasmKeeper.QuerySmart(ctx, addr, []byte(`{"snapshot":{}}`))
	require.NoError(t, err)
	var state upgradecontract.Snapshot
	require.NoError(t, json.Unmarshal(bz, &state))
	return state
}

// Restore exported state without the fresh-chain helper, which intentionally
// adds another validator and is not appropriate for an existing chain.
func restoreUpgradeTestGenesis(t *testing.T, chain *app.NibiruApp, ctx sdk.Context) (*app.NibiruApp, sdk.Context) {
	t.Helper()
	genesis := chain.ModuleManager.ExportGenesis(ctx, chain.AppCodec())
	imported := app.NewNibiruApp(log.NewNopLogger(), tmdb.NewMemDB(), nil, true, sims.EmptyAppOptions{})
	stateBytes, err := json.Marshal(genesis)
	require.NoError(t, err)
	imported.InitChain(abci.RequestInitChain{
		Time: ctx.BlockTime(), InitialHeight: ctx.BlockHeight() + 1,
		ConsensusParams: chain.GetConsensusParams(ctx), AppStateBytes: stateBytes,
	})
	importedCtx := imported.NewContext(false, cmtproto.Header{
		Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime(),
	})
	return imported, importedCtx
}

func TestUpgradeHandlerCutsOverAllFamiliesBeforeTrailingWrites(t *testing.T) {
	chain, ctx := testapp.NewNibiruTestAppAndContextPreXConsensus()
	ctx = ctx.WithBlockHeight(77).WithBlockTime(time.Unix(1_800_000_000, 0).UTC())
	k := chain.UpgradeKeeper
	require.False(t, k.ContractActive(ctx))
	versions := k.GetModuleVersionMap(ctx)
	versions[upgradetypes.ModuleName] = 2
	versions["legacy-extra-module"] = 91
	k.SetModuleVersionMap(ctx, versions)
	store := ctx.KVStore(chain.GetKey(upgradetypes.StoreKey))
	for _, height := range []uint64{5, 7} {
		key := append([]byte{upgradetypes.DoneByte}, sdk.Uint64ToBigEndian(height)...)
		store.Set(append(key, []byte("duplicate-name")...), []byte{1})
	}
	protocol := make([]byte, 8)
	binary.BigEndian.PutUint64(protocol, 7)
	store.Set([]byte{upgradetypes.ProtocolVersionByte}, protocol)
	chain.SetProtocolVersion(7)
	untouchedIBC := upgradetypes.UpgradedClientKey(1234)
	store.Set(untouchedIBC, []byte("proof-visible bytes"))
	plan := upgradetypes.Plan{Name: "v2.22.0", Height: 77, Info: "enabling release"}
	require.NoError(t, k.ScheduleUpgrade(ctx, plan))

	// Run the actual registered application handler through ApplyUpgrade,
	// including its post-handler module/protocol/clear/completion operations.
	k.ApplyUpgrade(ctx, plan)
	require.True(t, k.ContractActive(ctx))
	require.Equal(t, uint64(8), chain.AppVersion())
	gotVersions := k.GetModuleVersionMap(ctx)
	require.Equal(t, uint64(3), gotVersions[upgradetypes.ModuleName])
	require.Equal(t, uint64(91), gotVersions["legacy-extra-module"])
	require.Equal(t, int64(5), k.GetDoneHeight(ctx, "duplicate-name"))
	require.Equal(t, int64(77), k.GetDoneHeight(ctx, plan.Name))
	_, found := k.GetUpgradePlan(ctx)
	require.False(t, found)
	name, height := k.GetLastCompletedUpgrade(ctx)
	require.Equal(t, plan.Name, name)
	require.Equal(t, int64(77), height)
	state := queryUpgradeSnapshot(t, chain, ctx)
	require.NotNil(t, state.ProtocolVersion)
	require.Equal(t, uint64(8), *state.ProtocolVersion)
	require.Len(t, state.Completed, 3)
	for _, family := range []byte{
		upgradetypes.PlanByte, upgradetypes.DoneByte,
		upgradetypes.VersionMapByte, upgradetypes.ProtocolVersionByte,
	} {
		iterator := sdk.KVStorePrefixIterator(store, []byte{family})
		require.False(t, iterator.Valid(), "retired family %d must be removed", family)
		iterator.Close()
	}
	require.Equal(t, []byte("proof-visible bytes"), store.Get(untouchedIBC))
}

func TestUpgradeContractStateSurvivesGenesisImport(t *testing.T) {
	chain, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithBlockHeight(9)
	k := chain.UpgradeKeeper
	k.SetModuleVersionMap(ctx, module.VersionMap{"extra-module": 91})
	k.SetUpgradeHandler("genesis-record", func(sdk.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
		return module.VersionMap{"handler-result": 37}, nil
	})
	plan := upgradetypes.Plan{Name: "genesis-record", Height: ctx.BlockHeight()}
	require.NoError(t, k.ScheduleUpgrade(ctx, plan))
	k.ApplyUpgrade(ctx, plan)
	require.NoError(t, k.ScheduleUpgrade(ctx, upgradetypes.Plan{
		Name: "future-request", Time: ctx.BlockTime().Add(24 * time.Hour), Info: "keep this intent",
	}))
	wantRecords := upgradeContractRecords(t, chain, ctx)
	wantVersions := k.GetModuleVersionMap(ctx)
	wantProtocol := chain.AppVersion()
	imported, importedCtx := restoreUpgradeTestGenesis(t, chain, ctx)
	require.Equal(t, wantRecords, upgradeContractRecords(t, imported, importedCtx))
	require.Equal(t, wantVersions, imported.UpgradeKeeper.GetModuleVersionMap(importedCtx))
	require.Equal(t, wantProtocol, imported.AppVersion())
	target, err := imported.UpgradeKeeper.TimeTarget(importedCtx)
	require.NoError(t, err)
	require.NotNil(t, target)
	require.Equal(t, "future-request", target.Name)
}

func TestUpgradeContractStateSurvivesAppRestart(t *testing.T) {
	db := tmdb.NewMemDB()
	chain, _ := testapp.NewNibiruTestAppWithDB(db, app.GenesisState{})
	ctx := testapp.NewContext(chain)
	chain.UpgradeKeeper.SetModuleVersionMap(ctx, module.VersionMap{"restart-marker": 99})
	require.NoError(t, chain.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{
		Name: "after-restart", Time: ctx.BlockTime().Add(time.Hour),
	}))
	want := upgradeContractRecords(t, chain, ctx)
	chain.EndBlock(abci.RequestEndBlock{Height: ctx.BlockHeight()})
	chain.Commit()
	header := cmtproto.Header{Height: chain.LastBlockHeight() + 1, Time: ctx.BlockTime().Add(time.Second)}
	restarted := app.NewNibiruApp(log.NewNopLogger(), db, nil, true, sims.EmptyAppOptions{})
	restarted.BeginBlock(abci.RequestBeginBlock{Header: header})
	restartedCtx := restarted.NewContext(false, header)
	require.Equal(t, want, upgradeContractRecords(t, restarted, restartedCtx))
	require.Equal(t, uint64(99), restarted.UpgradeKeeper.GetModuleVersionMap(restartedCtx)["restart-marker"])
}

// Failed contract instantiation must leave legacy state intact. A later block
// replay must not inherit any completion/version changes from an uncommitted
// successful attempt, even when the VM has already compiled that bytecode.
func TestUpgradeImportFailureAndUncommittedReplay(t *testing.T) {
	chain, ctx := testapp.NewNibiruTestAppAndContextPreXConsensus()
	ctx = ctx.WithBlockHeight(77).WithBlockTime(time.Unix(1_800_000_000, 0).UTC())
	k := chain.UpgradeKeeper
	versions := k.GetModuleVersionMap(ctx)
	versions[upgradetypes.ModuleName] = 2
	k.SetModuleVersionMap(ctx, versions)
	store := ctx.KVStore(chain.GetKey(upgradetypes.StoreKey))
	store.Set([]byte{upgradetypes.ProtocolVersionByte}, sdk.Uint64ToBigEndian(7))
	chain.SetProtocolVersion(7)
	plan := upgradetypes.Plan{Name: "v2.22.0", Height: 77}
	require.NoError(t, k.ScheduleUpgrade(ctx, plan))
	// Simulate malformed legacy input reaching the actual Rust import. The
	// keeper's ordinary setter disallows this empty prefix-store key.
	store.Set([]byte{upgradetypes.VersionMapByte}, sdk.Uint64ToBigEndian(1))
	rawNative := func() map[string]string {
		rows := map[string]string{}
		iter := store.Iterator(nil, nil)
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			rows[hex.EncodeToString(iter.Key())] = hex.EncodeToString(iter.Value())
		}
		return rows
	}
	codes := func() int {
		n := 0
		chain.WasmKeeper.IterateCodeInfos(ctx, func(_ uint64, _ wasmtypes.CodeInfo) bool { n++; return false })
		return n
	}
	before, codeCount := rawNative(), codes()
	err := k.ActivateContract(ctx)
	require.ErrorContains(t, err, "module name cannot be empty")
	require.False(t, k.ContractActive(ctx))
	require.Equal(t, before, rawNative())
	require.Equal(t, codeCount, codes(), "failed import must not publish code metadata")
	store.Delete([]byte{upgradetypes.VersionMapByte})

	// A process that dies before Commit loses its block cache. Model that
	// boundary explicitly rather than claiming a randomly timed SIGKILL did it.
	attempt, _ := ctx.CacheContext()
	k.ApplyUpgrade(attempt, plan)
	require.True(t, k.ContractActive(attempt))
	require.Equal(t, int64(77), k.GetDoneHeight(attempt, plan.Name))
	require.False(t, k.ContractActive(ctx))
	pending, found := k.GetUpgradePlan(ctx)
	require.True(t, found)
	require.Equal(t, plan, pending)
	require.Equal(t, uint64(7), binary.BigEndian.Uint64(store.Get([]byte{upgradetypes.ProtocolVersionByte})))

	replayed, commit := ctx.CacheContext()
	k.ApplyUpgrade(replayed, plan)
	commit()
	require.True(t, k.ContractActive(ctx))
	state := queryUpgradeSnapshot(t, chain, ctx)
	require.Equal(t, uint64(8), *state.ProtocolVersion)
	require.Len(t, state.Completed, 1)
	require.Equal(t, plan.Name, state.Completed[0].Name)
	require.Equal(t, int64(77), state.Completed[0].Height)
	require.Equal(t, uint64(8), chain.AppVersion())
}
