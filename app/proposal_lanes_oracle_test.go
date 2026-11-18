package app_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	xoracle "github.com/NibiruChain/nibiru/v2/x/oracle"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// instantiateXOracleAdapter stores and instantiates the embedded x-oracle
// adapter in mode and returns its address.
func instantiateXOracleAdapter(
	t testing.TB,
	deps *evmtest.TestDeps,
	mode xoracle.XOracleAdapterMode,
) sdk.AccAddress {
	t.Helper()
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	codeID, _, err := keeper.Create(deps.Ctx(), deps.Sender.NibiruAddr,
		xoracle.XOracleAdapterWasm,
		&wasm.AccessConfig{Permission: wasm.AccessTypeEverybody})
	require.NoError(t, err)
	initMsg, err := json.Marshal(xoracle.XOracleAdapterInstantiateMsg{
		Owner:          deps.Sender.NibiruAddr.String(),
		Mode:           mode,
		LegacyMappings: []xoracle.XOracleAdapterLegacyMapping{},
	})
	require.NoError(t, err)
	addr, _, err := keeper.Instantiate(deps.Ctx(), codeID,
		deps.Sender.NibiruAddr, deps.Sender.NibiruAddr, initMsg,
		"x-oracle adapter", sdk.Coins{})
	require.NoError(t, err)
	return addr
}

// setXOraclePlugin points the EVM params x-oracle Wasm plugin at adapter, or
// removes it when adapter is empty.
func setXOraclePlugin(t testing.TB, deps *evmtest.TestDeps, adapter string) {
	t.Helper()
	params := deps.EvmKeeper.GetParams(deps.Ctx())
	params.WasmPlugins = []evm.WasmPlugin{}
	if adapter != "" {
		params.WasmPlugins = []evm.WasmPlugin{
			{Name: evm.WasmPluginNameXOracle, Addr: adapter},
		}
	}
	require.NoError(t, deps.EvmKeeper.SetParams(deps.Ctx(), params))
}

// TestResolveSaiOracleAddr proves how PrepareProposal reads the Sai oracle
// address: from a proxy-mode adapter named by the x-oracle Wasm plugin. A
// missing plugin and a fixture-mode adapter select the fallback without an
// error log; a plugin naming an account with no contract makes the query
// fail, which also selects the fallback, as does a query that panics. The
// lookup runs on its own gas meter, so the caller's meter is not charged,
// even when nearly exhausted.
func TestResolveSaiOracleAddr(t *testing.T) {
	deps := evmtest.NewTestDeps()
	saiOracle := evmtest.NewEthPrivAcc().NibiruAddr.String()

	addr, err := deps.App.ResolveSaiOracleAddr(deps.Ctx())
	require.ErrorIs(t, err, app.ErrOracleAdapterNotConfigured, "no plugin")
	require.Empty(t, addr)

	fixture := instantiateXOracleAdapter(t, &deps,
		xoracle.XOracleAdapterFixtureMode())
	setXOraclePlugin(t, &deps, fixture.String())
	addr, err = deps.App.ResolveSaiOracleAddr(deps.Ctx())
	require.ErrorIs(t, err, app.ErrOracleAdapterNotConfigured, "fixture mode")
	require.Empty(t, addr)

	setXOraclePlugin(t, &deps, evmtest.NewEthPrivAcc().NibiruAddr.String())
	addr, err = deps.App.ResolveSaiOracleAddr(deps.Ctx())
	require.Error(t, err, "plugin without a contract")
	require.NotErrorIs(t, err, app.ErrOracleAdapterNotConfigured)
	require.Empty(t, addr)

	proxy := instantiateXOracleAdapter(t, &deps,
		xoracle.XOracleAdapterProxyModeOf(saiOracle))
	setXOraclePlugin(t, &deps, proxy.String())
	meter := sdk.NewGasMeter(10)
	addr, err = deps.App.ResolveSaiOracleAddr(deps.Ctx().WithGasMeter(meter))
	require.NoError(t, err)
	require.Equal(t, saiOracle, addr)
	require.Zero(t, meter.GasConsumed(), "caller's gas meter charged")

	// The Wasm VM panics on a context without a block time. The bounded
	// query turns that into an error, which selects the fallback.
	addr, err = deps.App.ResolveSaiOracleAddr(deps.Ctx().WithBlockTime(time.Time{}))
	require.ErrorContains(t, err, "panicked")
	require.NotErrorIs(t, err, app.ErrOracleAdapterNotConfigured)
	require.Empty(t, addr)
}

// TestEVMPrepareProposalOracleLaneFollowsSaiOracle proves that, with the Sai
// oracle address resolved from a proxy-mode adapter, the oracle lane holds
// every zero-gas call to the oracle contract regardless of message: price
// writes, expiration updates, and permission calls. A price write to another
// allowlisted contract is a zero-gas call, and an oracle call with a distinct
// fee payer stays in the zero-gas lane. With a fixture-mode adapter the same
// candidates fall back to the message-key rule, under which the keeper's
// price write to perp is an oracle-lane tx and promotes the keeper's earlier
// fill ahead of it.
func TestEVMPrepareProposalOracleLaneFollowsSaiOracle(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)
	payer := evmtest.NewEthPrivAcc().NibiruAddr

	keeperFill := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))
	expiry := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.admin, actors.oracle, updateExpirationMsg))
	priceToPerp := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, setPricesMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	sponsored := buildSequencedProposalWasmTx(t, &deps, payer, []uint64{0, 0},
		proposalWasmExec(actors.keeper, actors.oracle, setPriceMsg))
	permission := buildProposalWasmTx(t, &deps, proposalWasmExec(
		actors.admin, actors.oracle,
		`{"update_permissions":{"address":"x","permission":"y"}}`))
	candidates := [][]byte{
		keeperFill, expiry, priceToPerp, setPrices, sponsored, permission,
	}
	names := map[string]string{
		string(keeperFill): "keeperFill", string(expiry): "expiry",
		string(priceToPerp): "priceToPerp", string(setPrices): "setPrices",
		string(sponsored): "sponsored", string(permission): "permission",
		string(evmTx): "evmTx",
	}
	named := func(txs ...[]byte) []string {
		out := make([]string, len(txs))
		for i, txBytes := range txs {
			out[i] = names[string(txBytes)]
		}
		return out
	}
	prepare := func() []string {
		handler := app.NewEVMPrepareProposalHandler(
			deps.App, new(acceptingPrepareVerifier))
		ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
			Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
		})
		return named(handler(ctx, abci.RequestPrepareProposal{
			MaxTxBytes: 1_000_000,
			Txs:        candidates,
		}).Txs...)
	}

	proxy := instantiateXOracleAdapter(t, &deps,
		xoracle.XOracleAdapterProxyModeOf(actors.oracle))
	setXOraclePlugin(t, &deps, proxy.String())
	require.Equal(t, named(
		expiry, setPrices, permission, // oracle lane
		keeperFill, priceToPerp, sponsored, // zero-gas lane
		evmTx,
	), prepare(), "resolved Sai oracle")

	fixture := instantiateXOracleAdapter(t, &deps,
		xoracle.XOracleAdapterFixtureMode())
	setXOraclePlugin(t, &deps, fixture.String())
	require.Equal(t, named(
		keeperFill, priceToPerp, setPrices, // oracle lane: price writes by key
		expiry, sponsored, permission, // zero-gas lane
		evmTx,
	), prepare(), "fallback")
}
