package upgrades_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	nutiltestutil "github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

// TestUpgrade2_15_0_SetsXOracleWasmPluginWhenConfigured runs the registered
// upgrade path, not the private helper, so the test covers the same handler
// wiring validators execute at the upgrade height. It asserts both the persisted
// params and the derived lookup index because EVM execution reads the latter.
func TestUpgrade2_15_0_SetsXOracleWasmPluginWhenConfigured(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	oldAddrCfg := upgrades.AddrCfg_v2_15
	defer func() { upgrades.AddrCfg_v2_15 = oldAddrCfg }()

	wantAddr := deps.Sender.NibiruAddr
	upgrades.AddrCfg_v2_15.XOraclePluginAddr = wantAddr.String()

	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_15_0))

	gotAddr, err := deps.App.EvmKeeper.EVMState().WasmPlugins.Get(ctx, evm.WasmPluginNameXOracle)
	require.NoError(t, err)
	require.Equal(t, wantAddr.String(), gotAddr.String())

	params := deps.App.EvmKeeper.GetParams(ctx)
	require.Equal(t, []evm.WasmPlugin{
		{
			Name: evm.WasmPluginNameXOracle,
			Addr: wantAddr.String(),
		},
	}, params.WasmPlugins)
}

// TestUpgrade2_15_0_SkipsXOracleWasmPluginWhenUnset keeps the failure path
// non-halting, matching the v2.14 upgrade convention: the handler records a
// failure event for observability, then still lets module migrations run.
func TestUpgrade2_15_0_SkipsXOracleWasmPluginWhenUnset(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	oldAddrCfg := upgrades.AddrCfg_v2_15
	defer func() { upgrades.AddrCfg_v2_15 = oldAddrCfg }()

	upgrades.AddrCfg_v2_15.XOraclePluginAddr = ""

	eventsBeforeUpgrade := deps.Ctx().EventManager().Events()
	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_15_0))
	eventsInUpgrade := nutiltestutil.FilterNewEvents(eventsBeforeUpgrade, deps.Ctx().EventManager().Events())
	failureEvents := nutiltestutil.FindEventsOfType(eventsInUpgrade, "upgrade_failure")
	require.Len(t, failureEvents, 1)
	require.NoError(t, nutiltestutil.EventHasAttributeValue(
		failureEvents[0], "upgrade", "v2.15.0",
	))
	require.NoError(t, nutiltestutil.EventHasAttributeValue(
		failureEvents[0], "error", "x-oracle wasm plugin address not configured",
	))

	_, err := deps.App.EvmKeeper.EVMState().WasmPlugins.Get(ctx, evm.WasmPluginNameXOracle)
	require.Error(t, err)
	require.Empty(t, deps.App.EvmKeeper.GetParams(ctx).WasmPlugins)
}
