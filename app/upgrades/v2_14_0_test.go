package upgrades_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/denoms"
	nutiltestutil "github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
)

const (
	artifactDir             = "testdata"
	artifactCW3FlexMultisig = "cw3_flex_multisig.wasm"
	artifactCW4Group        = "cw4_group.wasm"

	chainIDMainnet = "cataclysm-1"

	addrLegacyMultisig = "nibi1xfdlvvmx03nyav4840z25m33qrwzy67p4eadn7"
	addrKevinNanoS     = "nibi1w3s6gdtt09ekhmwzfszc6qtxh298fstu895gn8"

	addrTreasuryAddSigner = "nibi1cj6edencz3tetxuwkxcdrlfwpg8cr02ef3uvz2"
	addrGimeno            = "nibi1wwhdx03msygelmm8tm5z6nzh4dklwkettwd5vj"
	addrJC                = "nibi1aj6vgnj5hh0ehe5memz0f38z2lla2z5gdj5vst"
)

// TestUpgrade2_14_0_HappyPath stores and instantiates real CW4 group and CW3
// flex multisig Wasm artifacts, then runs the full upgrade handler. The test
// assumes artifacts matching the deployed contract code are present under the
// v2.14-specific artifact directory so the test exercises bytecode exactly as
// supplied by mainnet.
func TestUpgrade2_14_0_HappyPath(t *testing.T) {
	deps := evmtest.NewTestDeps()
	deps.SetCtx(deps.Ctx().WithChainID(chainIDMainnet))
	ctx := deps.Ctx()

	legacyMultisig := mustAccAddress(addrLegacyMultisig)
	kevinNanoS := mustAccAddress(addrKevinNanoS)
	newSigner := mustAccAddress(addrTreasuryAddSigner)

	cw4CodeID := storeWasmArtifact(t, deps, artifactCW4Group)
	cw3CodeID := storeWasmArtifact(t, deps, artifactCW3FlexMultisig)

	treasuryCW4 := instantiateCW4Group(
		t,
		deps,
		cw4CodeID,
		legacyMultisig,
		legacyMultisig,
		treasuryMembersBeforeUpgrade(),
		"treasury cw4 group",
	)
	treasuryCW3 := instantiateCW3FlexMultisig(
		t,
		deps,
		cw3CodeID,
		legacyMultisig,
		legacyMultisig,
		treasuryCW4,
		3,
		86_400,
		"treasury cw3",
	)

	hotWalletCW4 := instantiateCW4Group(
		t,
		deps,
		cw4CodeID,
		kevinNanoS,
		kevinNanoS,
		hotWalletMembers(),
		"hot wallet cw4 group",
	)
	hotWalletCW3 := instantiateCW3FlexMultisig(
		t,
		deps,
		cw3CodeID,
		kevinNanoS,
		kevinNanoS,
		hotWalletCW4,
		2,
		31_536_000,
		"hot wallet cw3",
	)
	executeCW4UpdateAdmin(t, deps, hotWalletCW4, kevinNanoS, hotWalletCW3)

	hotWalletBalance := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdkmath.NewInt(34_508_752_254)))
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, ctx, hotWalletCW3, hotWalletBalance))
	require.True(t, deps.App.BankKeeper.GetAllBalances(ctx, treasuryCW3).IsZero())

	prevAddrCfg := upgrades.AddrCfg_v2_14
	upgrades.AddrCfg_v2_14 = testUpgrade2_14_0AddressConfig(treasuryCW3, treasuryCW4, hotWalletCW3, hotWalletCW4)
	defer func() {
		upgrades.AddrCfg_v2_14 = prevAddrCfg
	}()

	eventsBeforeUpgrade := deps.Ctx().EventManager().Events()
	require.NoError(t, runUpgradeForTest(deps, upgrades.Upgrade2_14_0))
	eventsInUpgrade := nutiltestutil.FilterNewEvents(eventsBeforeUpgrade, deps.Ctx().EventManager().Events())
	eventsJSON, err := json.MarshalIndent(eventsInUpgrade, "", "  ")
	require.NoError(t, err)
	t.Logf("v2.14 upgrade events:\n%s", eventsJSON)
	assertUpgrade2_14_0Events(
		t,
		eventsInUpgrade,
		treasuryCW3,
		treasuryCW4,
		hotWalletCW3,
		hotWalletCW4,
		legacyMultisig,
		hotWalletBalance,
	)

	treasuryMembers := queryCW4Members(t, deps, treasuryCW4)
	require.NotContains(t, treasuryMembers, addrGimeno)
	require.NotContains(t, treasuryMembers, addrJC)
	require.NotContains(t, treasuryMembers, addrKevinNanoS)
	require.Equal(t, uint64(1), treasuryMembers[newSigner.String()])
	require.Len(t, treasuryMembers, 7)

	require.Equal(t, treasuryCW3.String(), queryCW4Admin(t, deps, treasuryCW4))
	require.Equal(t, treasuryCW3.String(), contractAdmin(t, deps, treasuryCW4))
	require.Equal(t, treasuryCW3.String(), contractAdmin(t, deps, treasuryCW3))

	require.Equal(t, hotWalletCW3.String(), queryCW4Admin(t, deps, hotWalletCW4))
	require.Equal(t, treasuryCW3.String(), contractAdmin(t, deps, hotWalletCW3))
	require.Equal(t, treasuryCW3.String(), contractAdmin(t, deps, hotWalletCW4))
	require.True(t, deps.App.BankKeeper.GetAllBalances(ctx, hotWalletCW3).IsZero())
	require.Equal(t, hotWalletBalance, deps.App.BankKeeper.GetAllBalances(ctx, treasuryCW3))
}

// assertUpgrade2_14_0Events checks the observable event footprint of the happy
// path upgrade: CW4 executes, wasm admin transfers, and the Hot Wallet bank
// sweep.
func assertUpgrade2_14_0Events(
	t *testing.T,
	events sdk.Events,
	treasuryCW3 sdk.AccAddress,
	treasuryCW4 sdk.AccAddress,
	hotWalletCW3 sdk.AccAddress,
	hotWalletCW4 sdk.AccAddress,
	legacyMultisig sdk.AccAddress,
	hotWalletBalance sdk.Coins,
) {
	t.Helper()

	// CW4 membership update from the legacy multisig.
	requireEventWithAttrs(t, events, "wasm", map[string]string{
		"_contract_address": treasuryCW4.String(),
		"action":            "update_members",
		"added":             "1",
		"removed":           "3",
		"sender":            legacyMultisig.String(),
	})

	// CW4 admin handoff from the legacy multisig to Treasury CW3.
	requireEventWithAttrs(t, events, "wasm", map[string]string{
		"_contract_address": treasuryCW4.String(),
		"action":            "update_admin",
		"admin":             treasuryCW3.String(),
		"sender":            legacyMultisig.String(),
	})

	// Wasm module admin updates for Treasury and Hot Wallet contracts.
	for _, contract := range []sdk.AccAddress{
		treasuryCW4,
		treasuryCW3,
		hotWalletCW3,
		hotWalletCW4,
	} {
		requireEventWithAttrs(t, events, "update_contract_admin", map[string]string{
			"_contract_address": contract.String(),
			"new_admin_address": treasuryCW3.String(),
		})
	}

	// Bank send events emitted by sweeping the Hot Wallet CW3 balance.
	requireEventWithAttrs(t, events, "coin_spent", map[string]string{
		"spender": hotWalletCW3.String(),
		"amount":  hotWalletBalance.String(),
	})
	requireEventWithAttrs(t, events, "coin_received", map[string]string{
		"receiver": treasuryCW3.String(),
		"amount":   hotWalletBalance.String(),
	})
	requireEventWithAttrs(t, events, "transfer", map[string]string{
		"sender":    hotWalletCW3.String(),
		"recipient": treasuryCW3.String(),
		"amount":    hotWalletBalance.String(),
	})
	requireEventWithAttrs(t, events, "wei_change", map[string]string{
		"wei_change_reason": "bank.SendCoins",
		"wei_change_addrs":  hotWalletCW3.String(),
	})
	requireEventWithAttrs(t, events, "wei_change", map[string]string{
		"wei_change_addrs": treasuryCW3.String(),
	})
}

// requireEventWithAttrs finds an event of the given type containing all wanted
// attribute values. Attribute matching uses substring semantics through
// nutiltestutil.EventHasAttributeValue, which is useful for compound attributes
// such as wei_change_addrs.
func requireEventWithAttrs(
	t *testing.T,
	events sdk.Events,
	eventType string,
	wantAttrs map[string]string,
) {
	t.Helper()

	for _, event := range nutiltestutil.FindEventsOfType(events, eventType) {
		matches := true
		for key, want := range wantAttrs {
			if err := nutiltestutil.EventHasAttributeValue(event, key, want); err != nil {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}

	eventsJSON, err := json.MarshalIndent(events, "", "  ")
	require.NoError(t, err)
	t.Fatalf("event %q with attributes %+v not found in events:\n%s", eventType, wantAttrs, eventsJSON)
}

// storeWasmArtifact uploads one downloaded mainnet Wasm artifact into the test
// app and skips cleanly when the artifact is not present on disk.
func storeWasmArtifact(t *testing.T, deps evmtest.TestDeps, artifactName string) uint64 {
	t.Helper()

	artifactPath := filepath.Join(artifactDir, artifactName)
	wasmBytecode, err := os.ReadFile(artifactPath)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("missing %s; build/copy deployed-compatible artifact to %s", artifactName, artifactPath)
	}
	require.NoError(t, err)

	codeID, _, err := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper).Create(
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		wasmBytecode,
		&wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody},
	)
	require.NoError(t, err)
	return codeID
}

// instantiateCW4Group creates a real cw4-group contract with the supplied admin
// and members, matching the contract state the upgrade handler mutates.
func instantiateCW4Group(
	t *testing.T,
	deps evmtest.TestDeps,
	codeID uint64,
	creator sdk.AccAddress,
	admin sdk.AccAddress,
	members []cw4Member,
	label string,
) sdk.AccAddress {
	t.Helper()

	initMsg := struct {
		Admin   *string     `json:"admin"`
		Members []cw4Member `json:"members"`
	}{
		Admin:   ptr(admin.String()),
		Members: members,
	}

	contractAddr, _, err := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper).Instantiate(
		deps.Ctx(),
		codeID,
		creator,
		admin,
		mustJSON(t, initMsg),
		label,
		sdk.Coins{},
	)
	require.NoError(t, err)
	return contractAddr
}

// instantiateCW3FlexMultisig creates a real CW3 flex multisig wired to the
// provided CW4 group contract.
func instantiateCW3FlexMultisig(
	t *testing.T,
	deps evmtest.TestDeps,
	codeID uint64,
	creator sdk.AccAddress,
	admin sdk.AccAddress,
	groupAddr sdk.AccAddress,
	threshold uint64,
	votingPeriodSeconds uint64,
	label string,
) sdk.AccAddress {
	t.Helper()

	initMsg := map[string]any{
		"group_addr": groupAddr.String(),
		"threshold": map[string]any{
			"absolute_count": map[string]any{"weight": threshold},
		},
		"max_voting_period": map[string]any{"time": votingPeriodSeconds},
		"executor":          nil,
		"proposal_deposit":  nil,
	}

	contractAddr, _, err := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper).Instantiate(
		deps.Ctx(),
		codeID,
		creator,
		admin,
		mustJSON(t, initMsg),
		label,
		sdk.Coins{},
	)
	require.NoError(t, err)
	return contractAddr
}

// executeCW4UpdateAdmin prepares Hot Wallet state by making its CW4 group admin
// the Hot Wallet CW3 contract before the upgrade runs.
func executeCW4UpdateAdmin(
	t *testing.T,
	deps evmtest.TestDeps,
	cw4Group sdk.AccAddress,
	currentAdmin sdk.AccAddress,
	newAdmin sdk.AccAddress,
) {
	t.Helper()

	execMsg := map[string]any{
		"update_admin": map[string]any{"admin": newAdmin.String()},
	}
	_, err := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper).Execute(
		deps.Ctx(),
		cw4Group,
		currentAdmin,
		mustJSON(t, execMsg),
		sdk.Coins{},
	)
	require.NoError(t, err)
}

// runUpgradeForTest invokes a registered upgrade handler with the same app
// wiring used by other upgrade tests.
func runUpgradeForTest(deps evmtest.TestDeps, upgrade upgrades.Upgrade) error {
	upgradeHandler := upgrade.CreateUpgradeHandler(
		deps.App.ModuleManager,
		module.NewConfigurator(
			deps.App.AppCodec(),
			deps.App.MsgServiceRouter(),
			deps.App.GRPCQueryRouter(),
		),
		&deps.App.PublicKeepers,
		deps.App.GetIBCKeeper().ClientKeeper,
	)

	plan := upgradetypes.Plan{
		Name:                upgrade.UpgradeName,
		Time:                time.Time{},
		Height:              deps.Ctx().BlockHeight(),
		Info:                "Testing Upgrade " + upgrade.UpgradeName,
		UpgradedClientState: (*codectypes.Any)(nil),
	}
	if err := plan.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid upgrade.Plan: %w", err)
	}

	_, err := upgradeHandler(
		deps.Ctx(),
		plan,
		deps.App.UpgradeKeeper.GetModuleVersionMap(deps.Ctx()),
	)
	return err
}

// testUpgrade2_14_0AddressConfig keeps production signers/admins but swaps the
// deployed contract addresses for freshly instantiated test contracts.
func testUpgrade2_14_0AddressConfig(
	treasuryCW3 sdk.AccAddress,
	treasuryCW4 sdk.AccAddress,
	hotWalletCW3 sdk.AccAddress,
	hotWalletCW4 sdk.AccAddress,
) upgrades.Upgrade2_14_AddrCfg {
	cfg := upgrades.AddrCfg_v2_14
	cfg.TreasuryRemoveSigners = append([]sdk.AccAddress(nil), cfg.TreasuryRemoveSigners...)
	cfg.TreasuryCW3 = treasuryCW3
	cfg.TreasuryCW4Group = treasuryCW4
	cfg.HotWalletCW3 = hotWalletCW3
	cfg.HotWalletCW4Group = hotWalletCW4
	return cfg
}

// treasuryMembersBeforeUpgrade returns the Treasury CW4 member set used to
// model mainnet before the v2.14 cleanup.
func treasuryMembersBeforeUpgrade() []cw4Member {
	return []cw4Member{
		{Addr: "nibi1372pyz4cctz4ns434gdt82qc46a0eh48jqprr7", Weight: 1},
		{Addr: "nibi1aj6vgnj5hh0ehe5memz0f38z2lla2z5gdj5vst", Weight: 1},
		{Addr: "nibi1ljhfmddrxt3axx2y0f5dvt0mxhxkkvs3pewdmq", Weight: 1},
		{Addr: "nibi1ny2qataqqgwplzwj4u553h0nkmyqth4ty7y00g", Weight: 1},
		{Addr: "nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l", Weight: 1},
		{Addr: "nibi1ss0s7fmw8n8t093mqt5was5c76k9amu0d7a5u0", Weight: 1},
		{Addr: "nibi1w3s6gdtt09ekhmwzfszc6qtxh298fstu895gn8", Weight: 1},
		{Addr: "nibi1wwhdx03msygelmm8tm5z6nzh4dklwkettwd5vj", Weight: 1},
		{Addr: "nibi1ykvzjt49tx202jd2xwtr9s7tkmf0g9jxvrj506", Weight: 1},
	}
}

// hotWalletMembers returns a representative dormant Hot Wallet CW4 member set.
// The upgrade intentionally leaves this membership unchanged.
func hotWalletMembers() []cw4Member {
	return []cw4Member{
		{Addr: "nibi1aj6vgnj5hh0ehe5memz0f38z2lla2z5gdj5vst", Weight: 1},
		{Addr: "nibi1ljhfmddrxt3axx2y0f5dvt0mxhxkkvs3pewdmq", Weight: 1},
		{Addr: "nibi1ny2qataqqgwplzwj4u553h0nkmyqth4ty7y00g", Weight: 1},
		{Addr: "nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l", Weight: 1},
		{Addr: "nibi1w3s6gdtt09ekhmwzfszc6qtxh298fstu895gn8", Weight: 1},
		{Addr: "nibi1wwhdx03msygelmm8tm5z6nzh4dklwkettwd5vj", Weight: 1},
	}
}

// queryCW4Members reads the live cw4-group member list after the upgrade and
// returns it as an address-to-weight map for assertions.
func queryCW4Members(t *testing.T, deps evmtest.TestDeps, cw4Group sdk.AccAddress) map[string]uint64 {
	t.Helper()

	respBz, err := deps.App.WasmKeeper.QuerySmart(
		deps.Ctx(),
		cw4Group,
		[]byte(`{"list_members":{}}`),
	)
	require.NoError(t, err)

	var resp cw4ListMembersResponse
	require.NoError(t, json.Unmarshal(respBz, &resp))

	members := map[string]uint64{}
	for _, member := range resp.Members {
		members[member.Addr] = member.Weight
	}
	return members
}

// queryCW4Admin reads the cw4-group contract-state admin.
func queryCW4Admin(t *testing.T, deps evmtest.TestDeps, cw4Group sdk.AccAddress) string {
	t.Helper()

	respBz, err := deps.App.WasmKeeper.QuerySmart(
		deps.Ctx(),
		cw4Group,
		[]byte(`{"admin":{}}`),
	)
	require.NoError(t, err)

	var resp cw4AdminResponse
	require.NoError(t, json.Unmarshal(respBz, &resp))
	return resp.Admin
}

// contractAdmin reads wasm module metadata admin for a contract.
func contractAdmin(t *testing.T, deps evmtest.TestDeps, contract sdk.AccAddress) string {
	t.Helper()

	info := deps.App.WasmKeeper.GetContractInfo(deps.Ctx(), contract)
	require.NotNil(t, info)
	return info.Admin
}

// mustJSON marshals test messages and fails the test on impossible local JSON
// construction errors.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	bz, err := json.Marshal(v)
	require.NoError(t, err)
	return bz
}

// mustAccAddress converts a known-good bech32 account literal for test setup.
func mustAccAddress(addr string) sdk.AccAddress {
	return sdk.MustAccAddressFromBech32(addr)
}

// ptr returns a pointer to a literal value for JSON message construction.
func ptr[T any](v T) *T {
	return &v
}

type cw4Member struct {
	Addr   string `json:"addr"`
	Weight uint64 `json:"weight"`
}

type cw4ListMembersResponse struct {
	Members []cw4Member `json:"members"`
}

type cw4AdminResponse struct {
	Admin string `json:"admin"`
}
