package keeper_test

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// TestSaiPulseArtifacts runs real release bytecode through the application's
// ABCI handlers. Artifact paths are explicit so tests never silently download
// code or commit generated Wasm. Run with just test-sai-pulse <artifact-dir>.
func TestSaiPulseArtifacts(t *testing.T) {
	dir := os.Getenv("SAI_PULSE_ARTIFACTS")
	if dir == "" {
		t.Skip("set SAI_PULSE_ARTIFACTS to a verified Sai Perps artifact directory")
	}
	chain, ctx := testapp.NewNibiruTestAppAndContext()
	owner := sdk.MustAccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
	store := func(name string) uint64 {
		code, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		t.Logf("artifact %s sha256=%x", name, sha256.Sum256(code))
		msg := wasmtypes.MsgStoreCodeFixture(func(m *wasmtypes.MsgStoreCode) {
			m.Sender, m.WASMByteCode = owner.String(), code
		})
		result, err := chain.MsgServiceRouter().Handler(msg)(ctx, msg)
		require.NoError(t, err)
		var resp wasmtypes.MsgStoreCodeResponse
		require.NoError(t, chain.AppCodec().Unmarshal(result.Data, &resp))
		return resp.CodeID
	}
	minterCode := store("vault_token_minter.wasm")
	vaultCode := store("vault.wasm")
	pulseCode := store("sai_pulse.wasm")

	// The real minter creates a tokenfactory denom at instantiation. Give its
	// predicted address tf_oper permission only in this fresh in-memory chain.
	minterAddr := keeper.BuildContractAddressClassic(minterCode, 1)
	roles := chain.SudoKeeper.Sudoers.GetOr(ctx, sudo.Sudoers{})
	roles.Roles = append(roles.Roles, sudo.RoleMembers{
		Role: sudo.RoleTFOper, Members: []string{minterAddr.String()},
	})
	chain.SudoKeeper.Sudoers.Set(ctx, roles)
	minter := instantiatePulseArtifact(t, chain, ctx, owner, minterCode, map[string]any{
		"denom": "pulse_shares", "owner": owner.String(),
	})
	require.Equal(t, minterAddr, minter)

	var vaults []string
	for range 2 {
		vault := instantiatePulseArtifact(t, chain, ctx, owner, vaultCode, map[string]any{
			"name": "Pulse integration vault", "symbol": "SLP", "collateral_denom": "unibi",
			"owner": owner.String(), "manager": owner.String(), "admin": owner.String(),
			"open_trades_pnl_feed": owner.String(), "perp_contract": owner.String(), "oracle": owner.String(),
			"vault_token_minter_contract": minter.String(), "min_lock_duration": 1,
			"max_acc_open_pnl_delta": "0.05", "max_daily_acc_pnl_delta": "0.10",
			"withdraw_lock_thresholds_percentage":  []string{"0.05", "0.10"},
			"max_supply_increase_daily_percentage": "0.05", "losses_burn_percentage": "0.01",
			"max_discount_percentage": "0.02", "max_discount_threshold_percentage": "1.20",
		})
		vaults = append(vaults, vault.String())
	}
	pulse := instantiatePulseArtifact(t, chain, ctx, owner, pulseCode, map[string]any{
		"owner": owner.String(), "slp_addrs": vaults, "interval": 1000,
	})
	register := func(address string) {
		_, err := chain.SudoKeeper.EditWasmBlockHooksContract(ctx, &sudo.MsgEditSudoers{
			Sender: owner.String(), Action: string(sudo.EditWasmBlockHooksContract), Contracts: []string{address},
		})
		require.NoError(t, err)
	}
	register(pulse.String())
	const eventType = "wasm-sai/pulse/vault_snapshot"
	end := func(height int64) []abci.Event {
		ctx = ctx.WithBlockHeight(height).WithEventManager(sdk.NewEventManager())
		gasBefore := ctx.GasMeter().GasConsumed()
		started := time.Now()
		// No DeliverTx occurs here. These are block lifecycle events.
		events := runWasmBlockHook(t, chain, ctx, "end")
		t.Logf("height=%d gas=%d elapsed=%s", height, ctx.GasMeter().GasConsumed()-gasBefore, time.Since(started))
		return events
	}
	assertSnapshots := func(events []abci.Event) {
		snapshots := testutil.FindAbciEventsOfType(events, eventType)
		require.Len(t, snapshots, len(vaults))
		for i, event := range snapshots {
			attrs := make(map[string]string)
			for _, attr := range event.Attributes {
				attrs[attr.Key] = attr.Value
			}
			require.Equal(t, pulse.String(), attrs["_contract_address"])
			require.Equal(t, vaults[i], attrs["slp_addr"])
			var data map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(attrs["data"]), &data))
			require.Len(t, data, 3)
			for field, query := range map[string]string{
				"vault_snapshot":   `{"vault_snapshot":{}}`,
				"available_assets": `{"available_assets":{}}`,
				"revenue_info":     `{"get_revenue_info":{}}`,
			} {
				want, err := chain.WasmKeeper.QuerySmart(ctx, sdk.MustAccAddressFromBech32(vaults[i]), []byte(query))
				require.NoError(t, err)
				require.JSONEq(t, string(want), string(data[field]), field)
			}
			t.Logf("vault=%s event_data_bytes=%d", vaults[i], len(attrs["data"]))
		}
	}

	// Instantiation starts paused even on a due height.
	require.Empty(t, testutil.FindAbciEventsOfType(end(1000), eventType))
	executePulseArtifact(t, chain, ctx, owner, pulse, map[string]any{"resume": map[string]any{}})
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.Empty(t, testutil.FindAbciEventsOfType(runWasmBlockHook(t, chain, ctx, "begin"), eventType))
	require.Empty(t, testutil.FindAbciEventsOfType(end(1999), eventType))
	assertSnapshots(end(2000))
	require.Empty(t, testutil.FindAbciEventsOfType(end(2001), eventType))
	assertSnapshots(end(3000))

	// A configured contract that does not implement vault queries fails before
	// emitting a snapshot; independent valid vault calls still succeed.
	executePulseArtifact(t, chain, ctx, owner, pulse, map[string]any{"update_config": map[string]any{
		"slp_addrs": append([]string{pulse.String()}, vaults...),
	}})
	events := end(4000)
	assertSnapshots(events)
	assertWasmBlockHookSummary(t, events, "end_block", 3, []keeper.WasmBlockHookDispatchFailure{
		{Idx: 0, ContractAddr: pulse.String()},
	})
	require.Empty(t, testutil.FindAbciEventsOfType(end(4001), eventType))

	executePulseArtifact(t, chain, ctx, owner, pulse, map[string]any{"pause": map[string]any{}})
	require.Empty(t, testutil.FindAbciEventsOfType(end(5000), eventType))
	register("")
	executePulseArtifact(t, chain, ctx, owner, pulse, map[string]any{"resume": map[string]any{}})
	events = end(6000)
	require.Empty(t, testutil.FindAbciEventsOfType(events, eventType))
	require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.EventTypeWasmBlockHookSummary))
}

func instantiatePulseArtifact(t *testing.T, chain *app.NibiruApp, ctx sdk.Context, owner sdk.AccAddress, codeID uint64, init any) sdk.AccAddress {
	t.Helper()
	msgJSON, err := json.Marshal(init)
	require.NoError(t, err)
	msg := &wasmtypes.MsgInstantiateContract{
		Sender: owner.String(), Admin: owner.String(), CodeID: codeID,
		Label: "Sai Pulse integration", Msg: msgJSON, Funds: sdk.Coins{},
	}
	result, err := chain.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)
	var resp wasmtypes.MsgInstantiateContractResponse
	require.NoError(t, chain.AppCodec().Unmarshal(result.Data, &resp))
	return sdk.MustAccAddressFromBech32(resp.Address)
}

func executePulseArtifact(t *testing.T, chain *app.NibiruApp, ctx sdk.Context, owner, contract sdk.AccAddress, execute any) {
	t.Helper()
	msgJSON, err := json.Marshal(execute)
	require.NoError(t, err)
	msg := &wasmtypes.MsgExecuteContract{
		Sender: owner.String(), Contract: contract.String(), Msg: msgJSON, Funds: sdk.Coins{},
	}
	_, err = chain.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)
}
