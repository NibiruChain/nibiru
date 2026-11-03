package app_test

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	upgradekeeper "github.com/NibiruChain/nibiru/v2/x/upgrade/keeper"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
)

// This uses the real Wasm VM. A permitted but enormous delay makes checked
// height addition fail, proving the contract -> host receipt path without a
// mock error or a test-only failure switch in production contract code.
func TestUpgradeSelectionFailureIsTerminalUntilGovernanceReplacement(t *testing.T) {
	chain, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithBlockHeight(100).WithBlockTime(time.Unix(1_800_000_000, 0).UTC())
	k := chain.UpgradeKeeper
	require.True(t, k.ContractActive(ctx))
	addr, _ := k.ContractAddr(ctx)
	state, err := chain.SudoKeeper.Sudoers.Get(ctx)
	require.NoError(t, err)
	root, err := sdk.AccAddressFromBech32(state.Root)
	require.NoError(t, err)
	wasm := wasmkeeper.NewDefaultPermissionKeeper(chain.WasmKeeper)
	setDelay := func(delay uint64) {
		t.Helper()
		msg, err := json.Marshal(map[string]any{"update_config": map[string]string{
			"activation_delay_blocks": strconv.FormatUint(delay, 10),
		}})
		require.NoError(t, err)
		_, err = wasm.Execute(ctx, addr, root, msg, sdk.NewCoins())
		require.NoError(t, err)
	}
	schedule := func(name string, target time.Time) {
		t.Helper()
		msg := &upgradetypes.MsgSoftwareUpgrade{Authority: authtypes.NewModuleAddress("gov").String(), Plan: upgradetypes.Plan{
			Name: name, Time: target,
		}}
		_, err := chain.MsgServiceRouter().Handler(msg)(ctx, msg)
		require.NoError(t, err)
	}

	setDelay(math.MaxInt64)
	targetTime := ctx.BlockTime().Add(time.Second)
	schedule("overflow-selection", targetTime)
	require.NoError(t, k.TrySelectExecutionHeight(ctx))
	require.Nil(t, k.GetSelectionFailure(ctx), "nothing is attempted before the target")
	ctx = ctx.WithBlockTime(targetTime).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.TrySelectExecutionHeight(ctx))
	failure := k.GetSelectionFailure(ctx)
	require.NotNil(t, failure)
	require.Equal(t, upgradekeeper.SelectionContractError, failure.Category)
	require.Equal(t, "overflow-selection", failure.Name)
	_, found := k.GetUpgradePlan(ctx)
	require.False(t, found, "failed time selection must not expose an executable plan")
	require.Len(t, ctx.EventManager().Events(), 1)
	require.Equal(t, "upgrade_selection_failed", ctx.EventManager().Events()[0].Type)

	// Wasm restores the pending request while upgrade genesis restores the
	// exceptional host receipt. Together they must keep the failed ID terminal.
	imported, importedCtx := restoreUpgradeTestGenesis(t, chain, ctx)
	require.Equal(t, failure, imported.UpgradeKeeper.GetSelectionFailure(importedCtx))
	require.Equal(t, upgradeContractRecords(t, chain, ctx), upgradeContractRecords(t, imported, importedCtx))
	importedCtx = importedCtx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, imported.UpgradeKeeper.TrySelectExecutionHeight(importedCtx))
	require.Empty(t, importedCtx.EventManager().Events())
	_, found = imported.UpgradeKeeper.GetUpgradePlan(importedCtx)
	require.False(t, found)

	// Repairing policy does not silently restart the failed request.
	setDelay(2)
	ctx = ctx.WithBlockHeight(101).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.TrySelectExecutionHeight(ctx))
	require.Empty(t, ctx.EventManager().Events())
	require.Equal(t, failure, k.GetSelectionFailure(ctx))
	_, found = k.GetUpgradePlan(ctx)
	require.False(t, found)

	// A newly authorized request clears suppression and gets a distinct ID.
	nextTime := ctx.BlockTime().Add(time.Second)
	schedule("replacement-selection", nextTime)
	require.Nil(t, k.GetSelectionFailure(ctx))
	ctx = ctx.WithBlockHeight(102).WithBlockTime(nextTime)
	require.NoError(t, k.TrySelectExecutionHeight(ctx))
	plan, found := k.GetUpgradePlan(ctx)
	require.True(t, found)
	require.Equal(t, "replacement-selection", plan.Name)
	require.Equal(t, int64(104), plan.Height)
	require.True(t, plan.Time.IsZero())

	// Root rotation is still checked live, independently of scheduling authority.
	chain.SudoKeeper.Sudoers.Set(ctx, sudo.Sudoers{Root: sdk.AccAddress(make([]byte, 20)).String()})
	_, err = wasm.Execute(ctx, addr, root, []byte(`{"update_config":{"activation_delay_blocks":"2"}}`), sdk.NewCoins())
	require.Error(t, err)
}

// CurrentPlan must remain absent while a timestamp is unresolved, but that
// absence must not allow a binary carrying the future handler to run early.
func TestUpgradeTimeTargetRejectsEarlyBinary(t *testing.T) {
	for _, reached := range []bool{false, true} {
		t.Run(strconv.FormatBool(reached), func(t *testing.T) {
			chain, ctx := testapp.NewNibiruTestAppAndContext()
			ctx = ctx.WithBlockHeight(100).WithBlockTime(time.Unix(1_800_000_000, 0).UTC())
			k := chain.UpgradeKeeper
			targetTime := ctx.BlockTime().Add(time.Hour)
			require.NoError(t, k.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: "future-binary", Time: targetTime}))
			k.SetUpgradeHandler("future-binary", func(_ sdk.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
				t.Fatal("premature binary must not execute its handler")
				return versions, nil
			})
			if reached {
				ctx = ctx.WithBlockTime(targetTime)
			}
			_, found := k.GetUpgradePlan(ctx)
			require.False(t, found)
			err := k.TrySelectExecutionHeight(ctx)
			require.ErrorContains(t, err, "BINARY UPDATED BEFORE TRIGGER!")
			require.PanicsWithError(t, err.Error(), func() {
				chain.BeginBlocker(ctx, abci.RequestBeginBlock{Header: ctx.BlockHeader()})
			})
			require.Nil(t, k.GetSelectionFailure(ctx), "an early binary is not a contract failure")
			_, found = k.GetUpgradePlan(ctx)
			require.False(t, found, "the early binary must not materialize the plan")
		})
	}
}
