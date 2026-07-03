package upgrades_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

// TestUpgrade2_15_0_SetsXOracleWasmPluginWhenConfigured runs the registered
// upgrade path, not the private helper, so the test covers the same handler
// wiring validators execute at the upgrade height. It asserts both the persisted
// params and the derived lookup index because EVM execution reads the latter.
func TestUpgrade2_15_0_SetsXOracleWasmPluginWhenConfigured(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()

	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_15_0))

	gotAddr, err := deps.App.EvmKeeper.EVMState().WasmPlugins.Get(ctx, evm.WasmPluginNameXOracle)
	require.NoError(t, err)
	require.Equal(t, upgrades.XOracleAdapterAddr_v2_15, gotAddr.String())

	params := deps.App.EvmKeeper.GetParams(ctx)
	require.Equal(t, []evm.WasmPlugin{
		{
			Name: evm.WasmPluginNameXOracle,
			Addr: upgrades.XOracleAdapterAddr_v2_15,
		},
	}, params.WasmPlugins)
}
