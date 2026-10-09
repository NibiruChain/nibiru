package upgrades_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	sdkclienttx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/tx"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/prefix"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const (
	erisTestContract = "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m"
	erisTestAttacker = "nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8"
	erisTestOther    = "nibi1vs2z5q3rlxmtrvuxkscdcysn5led0z3qptfkcw"
	erisTestPayout   = "51100662747260"
)

type erisSnapshot struct {
	Height   int64  `json:"height"`
	Time     string `json:"time"`
	Contract struct {
		Info json.RawMessage `json:"contract_info"`
	} `json:"contract"`
	CW3 struct {
		Info json.RawMessage `json:"contract_info"`
	} `json:"cw3"`
	Models []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"models"`
	Balances map[string]struct {
		Amount string `json:"amount"`
	} `json:"balances"`
	History []json.RawMessage `json:"history"`
}

func setupErisReplay(t *testing.T) (*app.NibiruApp, sdk.Context) {
	t.Helper()
	bz, err := os.ReadFile("testdata/eris-v2.21/snapshot.json")
	require.NoError(t, err)
	var snapshot erisSnapshot
	require.NoError(t, json.Unmarshal(bz, &snapshot))
	compressed, err := os.ReadFile("testdata/eris-v2.21/hub.wasm.gz")
	require.NoError(t, err)
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	bytecode, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	checksum := sha256.Sum256(bytecode)
	require.Equal(t, "e1c2be31ae008a015efa16b51f74ca1a014f4b1d0952da12ba3f60aaffa66321", hex.EncodeToString(checksum[:]))
	a, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithChainID("cataclysm-1").WithBlockHeight(snapshot.Height)
	capturedTime, err := time.Parse(time.RFC3339Nano, snapshot.Time)
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(capturedTime)
	var info, cw3Info wasmtypes.ContractInfo
	require.NoError(t, a.AppCodec().UnmarshalJSON(snapshot.Contract.Info, &info))
	require.NoError(t, a.AppCodec().UnmarshalJSON(snapshot.CW3.Info, &cw3Info))
	models := make([]wasmtypes.Model, 0, len(snapshot.Models))
	for _, model := range snapshot.Models {
		key, err := hex.DecodeString(model.Key)
		require.NoError(t, err)
		value, err := base64.StdEncoding.DecodeString(model.Value)
		require.NoError(t, err)
		models = append(models, wasmtypes.Model{Key: key, Value: value})
	}
	history := make([]wasmtypes.ContractCodeHistoryEntry, len(snapshot.History))
	for i, entry := range snapshot.History {
		require.NoError(t, a.AppCodec().UnmarshalJSON(entry, &history[i]))
	}
	cw3Code, err := os.ReadFile("testdata/cw3_flex_multisig.wasm")
	require.NoError(t, err)
	cw3Hash := sha256.Sum256(cw3Code)
	gen := wasmtypes.GenesisState{
		Params: wasmtypes.DefaultParams(),
		Codes: []wasmtypes.Code{
			{CodeID: info.CodeID, CodeInfo: wasmtypes.CodeInfo{CodeHash: checksum[:], Creator: info.Creator, InstantiateConfig: wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody}}, CodeBytes: bytecode},
			{CodeID: cw3Info.CodeID, CodeInfo: wasmtypes.CodeInfo{CodeHash: cw3Hash[:], Creator: cw3Info.Creator, InstantiateConfig: wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody}}, CodeBytes: cw3Code},
		},
		Contracts: []wasmtypes.Contract{
			{ContractAddress: erisTestContract, ContractInfo: info, ContractState: models, ContractCodeHistory: history},
			// CW3 receives a Bank send, so no recipient contract execution or CW3 state is needed.
			{ContractAddress: upgrades.IncidentRecoveryCW3_v2_18, ContractInfo: cw3Info, ContractCodeHistory: []wasmtypes.ContractCodeHistoryEntry{{Operation: wasmtypes.ContractCodeHistoryOperationTypeInit, CodeID: cw3Info.CodeID, Updated: cw3Info.Created, Msg: []byte(`{}`)}}},
		},
		Sequences: []wasmtypes.Sequence{{IDKey: wasmtypes.KeySequenceCodeID, Value: 133}, {IDKey: wasmtypes.KeySequenceInstanceID, Value: 1}},
	}
	_, err = wasmkeeper.InitGenesis(ctx, &a.WasmKeeper, gen)
	require.NoError(t, err)
	for address, balance := range snapshot.Balances {
		value, ok := sdkmath.NewIntFromString(balance.Amount)
		require.True(t, ok)
		if value.IsPositive() {
			require.NoError(t, testapp.FundAccount(a.BankKeeper, ctx, sdk.MustAccAddressFromBech32(address), sdk.NewCoins(sdk.NewCoin("unibi", value))))
		}
	}
	// Exercise the actual sudo v1-to-v2 migration and deployment seeding with
	// the Eris claim, rather than running recovery only against migrated state.
	root := a.SudoKeeper.Sudoers.GetOr(ctx, sudo.Sudoers{}).Root
	legacy := protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), root)
	legacy = protowire.AppendString(protowire.AppendTag(legacy, 2, protowire.BytesType), erisTestOther)
	collections.Map[uint64, sudo.Sudoers](a.SudoKeeper.Sudoers).GetStore(ctx).Set(make([]byte, 8), legacy)
	versions := a.ModuleManager.GetVersionMap()
	versions[sudo.ModuleName] = 1
	a.UpgradeKeeper.SetModuleVersionMap(ctx, versions)
	return a, ctx
}

func runErisUpgrade(t *testing.T, a *app.NibiruApp, ctx sdk.Context) sdk.Event {
	t.Helper()
	handler := upgrades.Upgrade2_21_0.Handler.Handler(a.ModuleManager, a.Configurator(), &a.PublicKeepers)
	versions, err := handler(ctx, upgradetypes.Plan{Name: "v2.21.0"}, a.UpgradeKeeper.GetModuleVersionMap(ctx))
	require.NoError(t, err)
	a.UpgradeKeeper.SetModuleVersionMap(ctx, versions)
	require.Equal(t, uint64(2), versions[sudo.ModuleName])
	require.NoError(t, a.SudoKeeper.CheckPermissions(sdk.MustAccAddressFromBech32("nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l"), ctx, sudo.RoleWasmDeployer))
	events := ctx.EventManager().Events()
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == "eris_recovery" {
			return events[i]
		}
	}
	t.Fatal("missing recovery event")
	return sdk.Event{}
}

func erisEventAttribute(t *testing.T, event sdk.Event, name string) string {
	t.Helper()
	attr, ok := event.GetAttribute(name)
	require.True(t, ok, name)
	return attr.Value
}

func queryErisReplay(t *testing.T, a *app.NibiruApp, ctx sdk.Context, query string) []byte {
	t.Helper()
	bz, err := a.WasmKeeper.QuerySmart(ctx, sdk.MustAccAddressFromBech32(erisTestContract), []byte(query))
	require.NoError(t, err)
	return bz
}

func erisReplayState(a *app.NibiruApp, ctx sdk.Context) []wasmtypes.Model {
	var models []wasmtypes.Model
	a.WasmKeeper.IterateContractState(ctx, sdk.MustAccAddressFromBech32(erisTestContract), func(key, value []byte) bool {
		models = append(models, wasmtypes.Model{Key: bytes.Clone(key), Value: bytes.Clone(value)})
		return false
	})
	return models
}

func TestUpgrade221ErisMainnetReplay(t *testing.T) {
	a, ctx := setupErisReplay(t)
	contract := sdk.MustAccAddressFromBech32(erisTestContract)
	recipient := sdk.MustAccAddressFromBech32(upgrades.IncidentRecoveryCW3_v2_18)
	attacker := sdk.MustAccAddressFromBech32(erisTestAttacker)
	msg := &wasmtypes.MsgExecuteContract{Sender: erisTestAttacker, Contract: erisTestContract, Msg: []byte(`{"withdraw_unbonded":{"receiver":"` + recipient.String() + `"}}`)}
	builder, err := sdkclienttx.Factory{}.WithChainID("cataclysm-1").WithTxConfig(a.GetTxConfig()).BuildUnsignedTx(msg)
	require.NoError(t, err)
	_, err = (ante.AnteDecIncidentQuarantine{}).AnteHandle(ctx, builder.GetTx(), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		t.Fatal("quarantine bypassed")
		return ctx, nil
	})
	require.ErrorContains(t, err, "quarantined")
	beforeState := erisReplayState(a, ctx)
	beforeContract := a.BankKeeper.GetBalance(ctx, contract, "unibi")
	beforeCW3 := a.BankKeeper.GetBalance(ctx, recipient, "unibi")
	beforeAttacker := a.BankKeeper.GetAllBalances(ctx, attacker)
	beforeSupply := a.BankKeeper.GetSupply(ctx, "unibi")
	otherBefore := queryErisReplay(t, a, ctx, `{"unbond_requests_by_user":{"user":"`+erisTestOther+`"}}`)
	event := runErisUpgrade(t, a, ctx)
	require.Equal(t, "success", erisEventAttribute(t, event, "status"))
	require.Equal(t, erisTestPayout, erisEventAttribute(t, event, "amount"))
	t.Logf("deployed Eris recovery gas: %s", erisEventAttribute(t, event, "gas_used"))
	payout, _ := sdkmath.NewIntFromString(erisTestPayout)
	require.True(t, a.BankKeeper.GetBalance(ctx, contract, "unibi").Amount.Equal(beforeContract.Amount.Sub(payout)))
	require.True(t, a.BankKeeper.GetBalance(ctx, recipient, "unibi").Amount.Equal(beforeCW3.Amount.Add(payout)))
	require.Equal(t, beforeAttacker, a.BankKeeper.GetAllBalances(ctx, attacker))
	require.Equal(t, beforeSupply, a.BankKeeper.GetSupply(ctx, "unibi"))
	require.JSONEq(t, `[]`, string(queryErisReplay(t, a, ctx, `{"unbond_requests_by_user":{"user":"`+erisTestAttacker+`"}}`)))
	require.JSONEq(t, string(otherBefore), string(queryErisReplay(t, a, ctx, `{"unbond_requests_by_user":{"user":"`+erisTestOther+`"}}`)))
	require.JSONEq(t, `{"id":172,"reconciled":true,"total_shares":"40795903","utoken_unclaimed":"64545507","est_unbond_end_time":1788875609}`, string(queryErisReplay(t, a, ctx, `{"previous_batch":172}`)))
	// Only the batch record and the attacker's primary record and user index may change.
	beforeModels := map[string][]byte{}
	for _, model := range beforeState {
		beforeModels[string(model.Key)] = model.Value
	}
	changed := map[string]bool{}
	for _, model := range erisReplayState(a, ctx) {
		if !bytes.Equal(beforeModels[string(model.Key)], model.Value) {
			changed[string(model.Key)] = true
		}
		delete(beforeModels, string(model.Key))
	}
	for key := range beforeModels {
		changed[key] = true
	}
	require.Len(t, changed, 3)
	for key := range changed {
		require.True(t, bytes.Contains([]byte(key), []byte(erisTestAttacker)) || bytes.HasPrefix([]byte(key), append([]byte{0, 16}, []byte("previous_batches")...)))
	}
	// Re-running uses the contract's missing request as a no-op, with no further writes.
	afterState := erisReplayState(a, ctx)
	event = runErisUpgrade(t, a, ctx)
	require.Equal(t, "skipped", erisEventAttribute(t, event, "status"))
	require.Equal(t, afterState, erisReplayState(a, ctx))
	require.True(t, a.BankKeeper.GetBalance(ctx, recipient, "unibi").Amount.Equal(beforeCW3.Amount.Add(payout)))
}

func TestUpgrade221ErisOtherUserClaimsFirst(t *testing.T) {
	a, ctx := setupErisReplay(t)
	_, err := wasmkeeper.NewDefaultPermissionKeeper(a.WasmKeeper).Execute(ctx, sdk.MustAccAddressFromBech32(erisTestContract), sdk.MustAccAddressFromBech32(erisTestOther), []byte(`{"withdraw_unbonded":{}}`), nil)
	require.NoError(t, err)
	event := runErisUpgrade(t, a, ctx)
	require.Equal(t, "success", erisEventAttribute(t, event, "status"))
	require.Equal(t, "51100662747261", erisEventAttribute(t, event, "amount"))
	require.NotContains(t, string(queryErisReplay(t, a, ctx, `{"previous_batches":{"start_after":171,"limit":1}}`)), `"id":172`)
}

func TestUpgrade221ErisRecoversAfterSeedFailure(t *testing.T) {
	a, ctx := setupErisReplay(t)
	// Use the migrated schema with an invalid root to force the custom grant
	// step to fail. Recovery must still execute against the real Eris fixture.
	a.SudoKeeper.Sudoers.Set(ctx, sudo.Sudoers{Root: "invalid-root"})
	before := a.SudoKeeper.Sudoers.GetOr(ctx, sudo.Sudoers{})
	handler := upgrades.Upgrade2_21_0.Handler.Handler(a.ModuleManager, a.Configurator(), &a.PublicKeepers)
	versions, err := handler(ctx, upgradetypes.Plan{Name: "v2.21.0"}, a.ModuleManager.GetVersionMap())
	require.NoError(t, err)
	require.Equal(t, uint64(2), versions[sudo.ModuleName])
	require.Equal(t, before, a.SudoKeeper.Sudoers.GetOr(ctx, sudo.Sudoers{}))
	var recovery, failure sdk.Event
	for _, event := range ctx.EventManager().Events() {
		switch event.Type {
		case "eris_recovery":
			recovery = event
		case "upgrade_failure":
			failure = event
		}
	}
	require.Equal(t, "success", erisEventAttribute(t, recovery, "status"))
	require.Equal(t, erisTestPayout, erisEventAttribute(t, recovery, "amount"))
	require.Equal(t, "v2.21.0", erisEventAttribute(t, failure, "upgrade"))
	require.Contains(t, erisEventAttribute(t, failure, "error"), "seed Wasm deployers")
}

func mutateErisBatch(t *testing.T, a *app.NibiruApp, ctx sdk.Context, field string, value any) {
	t.Helper()
	key := append([]byte{0, 16}, []byte("previous_batches")...)
	key = binary.BigEndian.AppendUint64(key, 172)
	store := prefix.NewStore(ctx.KVStore(a.GetKey(wasmtypes.StoreKey)), wasmtypes.GetContractStorePrefix(sdk.MustAccAddressFromBech32(erisTestContract)))
	var batch map[string]any
	require.NoError(t, json.Unmarshal(store.Get(key), &batch))
	batch[field] = value
	bz, err := json.Marshal(batch)
	require.NoError(t, err)
	store.Set(key, bz)
}

func TestUpgrade221ErisFailuresRollbackAndContinue(t *testing.T) {
	for _, scenario := range []string{"unreconciled", "immature", "insufficient funds", "Bank dispatch failure", "unexpected code", "extra request", "out of gas"} {
		t.Run(scenario, func(t *testing.T) {
			a, ctx := setupErisReplay(t)
			contract := sdk.MustAccAddressFromBech32(erisTestContract)
			recipient := sdk.MustAccAddressFromBech32(upgrades.IncidentRecoveryCW3_v2_18)
			store := prefix.NewStore(ctx.KVStore(a.GetKey(wasmtypes.StoreKey)), wasmtypes.GetContractStorePrefix(contract))
			switch scenario {
			case "unreconciled":
				mutateErisBatch(t, a, ctx, "reconciled", false)
			case "immature":
				ctx = ctx.WithBlockTime(time.Unix(1788875609, 0))
			case "insufficient funds":
				require.NoError(t, a.BankKeeper.SendCoins(ctx, contract, recipient, sdk.NewCoins(a.BankKeeper.GetBalance(ctx, contract, "unibi"))))
			case "Bank dispatch failure":
				a.BankKeeper.SetSendEnabled(ctx, "unibi", false)
			case "unexpected code":
				info := a.WasmKeeper.GetContractInfo(ctx, contract)
				info.CodeID = 2
				bz, err := a.AppCodec().Marshal(info)
				require.NoError(t, err)
				ctx.KVStore(a.GetKey(wasmtypes.StoreKey)).Set(wasmtypes.GetContractAddressKey(contract), bz)
			case "extra request":
				// Copy the attacker's primary record and user index under batch 171. The
				// real bytecode reads both indexed claims, so the helper must reject them.
				oldID := binary.BigEndian.AppendUint64(nil, 172)
				newID := binary.BigEndian.AppendUint64(nil, 171)
				for _, model := range erisReplayState(a, ctx) {
					if bytes.Contains(model.Key, []byte(erisTestAttacker)) && bytes.Contains(model.Key, oldID) && bytes.Contains(model.Key, []byte("unbond_requests")) {
						key := bytes.ReplaceAll(model.Key, oldID, newID)
						value := bytes.ReplaceAll(model.Value, []byte(`"id":172`), []byte(`"id":171`))
						store.Set(key, value)
					}
				}
			case "out of gas":
				validators := make([]string, 200000)
				for i := range validators {
					validators[i] = "nibivaloper1xesqr8vjvy34jhu027zd70ypl0nnev5ecrzmn9"
				}
				bz, err := json.Marshal(validators)
				require.NoError(t, err)
				store.Set([]byte("validators"), bz)
			}
			before := erisReplayState(a, ctx)
			beforeContract := a.BankKeeper.GetAllBalances(ctx, contract)
			beforeCW3 := a.BankKeeper.GetAllBalances(ctx, recipient)
			event := runErisUpgrade(t, a, ctx)
			require.Equal(t, "failed", erisEventAttribute(t, event, "status"))
			require.Equal(t, "0", erisEventAttribute(t, event, "amount"))
			require.NotEmpty(t, erisEventAttribute(t, event, "error"))
			if scenario == "Bank dispatch failure" {
				require.Contains(t, erisEventAttribute(t, event, "error"), "dispatch")
			}
			if scenario == "extra request" {
				require.Contains(t, erisEventAttribute(t, event, "error"), "expected only Eris batch 172 request")
			}
			if scenario == "out of gas" {
				require.Contains(t, erisEventAttribute(t, event, "error"), "gas")
			}
			require.Equal(t, before, erisReplayState(a, ctx))
			require.Equal(t, beforeContract, a.BankKeeper.GetAllBalances(ctx, contract))
			require.Equal(t, beforeCW3, a.BankKeeper.GetAllBalances(ctx, recipient))
			// Discard the cached contract success events along with its state writes.
			for _, e := range ctx.EventManager().Events() {
				require.NotEqual(t, "wasm-erishub/unbonded_withdrawn", e.Type)
			}
			var failure sdk.Event
			for _, e := range ctx.EventManager().Events() {
				if e.Type == "upgrade_failure" {
					failure = e
				}
			}
			require.Equal(t, "v2.21.0", erisEventAttribute(t, failure, "upgrade"))
			require.Contains(t, erisEventAttribute(t, failure, "error"), erisEventAttribute(t, event, "error"))
		})
	}
}
