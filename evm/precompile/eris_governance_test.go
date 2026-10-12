package precompile_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const erisEscrow = "nibi1us4rh4a9rexvde8l3m8f8nlz6wcf9qg57zk2w06kqkm4vtp96vkst9kuwd"
const erisGauge = "nibi1qh59hdelxfwah7g7e8k0lxu4upqatq5w4jdw60yv9shhlhm86ckq4txl92"
const erisDenom = "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI"
const erisValidator = "nibivaloper1cz36jg407s0z4f2vd7u6yc44v480dtpljrhdf6"

var erisToken = common.HexToAddress("0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b")

// setupErisGovernance imports an unmodified mainnet Wasm snapshot. The new test
// account has no lock or votes in that snapshot. A standard ERC20Minter deployed
// with six decimals supplies the receipt mapping; copying its initialized code
// and storage to the wrapper's fixed address avoids changing production constants.
func setupErisGovernance(t *testing.T) (evmtest.TestDeps, common.Address) {
	t.Helper()
	raw, err := os.ReadFile("testdata/eris-governance/snapshot.json")
	require.NoError(t, err)
	var snapshot struct {
		Height    int64  `json:"height"`
		Time      string `json:"time"`
		Contracts []struct {
			Name    string          `json:"name"`
			Address string          `json:"address"`
			Info    json.RawMessage `json:"info"`
			Hash    string          `json:"hash"`
			Models  []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"models"`
			History []json.RawMessage `json:"history"`
		} `json:"contracts"`
	}
	require.NoError(t, json.Unmarshal(raw, &snapshot))
	deps := evmtest.NewTestDeps()
	captured, err := time.Parse(time.RFC3339Nano, snapshot.Time)
	require.NoError(t, err)
	deps.SetCtx(deps.Ctx().WithChainID("cataclysm-1").WithBlockHeight(snapshot.Height).WithBlockTime(captured))
	gen := wasmtypes.GenesisState{Params: wasmtypes.DefaultParams()}
	for _, c := range snapshot.Contracts {
		compressed, err := os.ReadFile("testdata/eris-governance/" + c.Name + ".wasm.gz")
		require.NoError(t, err)
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		require.NoError(t, err)
		code, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		checksum := sha256.Sum256(code)
		require.Equal(t, c.Hash, hex.EncodeToString(checksum[:]))
		var info wasmtypes.ContractInfo
		require.NoError(t, deps.App.AppCodec().UnmarshalJSON(c.Info, &info))
		models := make([]wasmtypes.Model, 0, len(c.Models))
		for _, m := range c.Models {
			key, err := hex.DecodeString(m.Key)
			require.NoError(t, err)
			value, err := base64.StdEncoding.DecodeString(m.Value)
			require.NoError(t, err)
			models = append(models, wasmtypes.Model{Key: key, Value: value})
		}
		history := make([]wasmtypes.ContractCodeHistoryEntry, len(c.History))
		for i, h := range c.History {
			require.NoError(t, deps.App.AppCodec().UnmarshalJSON(h, &history[i]))
		}
		gen.Codes = append(gen.Codes, wasmtypes.Code{CodeID: info.CodeID, CodeInfo: wasmtypes.CodeInfo{CodeHash: checksum[:], Creator: info.Creator, InstantiateConfig: wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody}}, CodeBytes: code})
		gen.Contracts = append(gen.Contracts, wasmtypes.Contract{ContractAddress: c.Address, ContractInfo: info, ContractState: models, ContractCodeHistory: history})
	}
	gen.Sequences = []wasmtypes.Sequence{{IDKey: wasmtypes.KeySequenceCodeID, Value: 200}, {IDKey: wasmtypes.KeySequenceInstanceID, Value: 100}}
	_, err = wasmkeeper.InitGenesis(deps.Ctx(), &deps.App.WasmKeeper, gen)
	require.NoError(t, err)
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, sdk.NewCoins(sdk.NewInt64Coin("unibi", 100_000_000))))
	tokenContract := embeds.SmartContract_ERC20Minter
	tokenContract.MustLoad()
	token, err := evmtest.DeployContract(&deps, tokenContract, "stNIBI", "stNIBI", uint8(6))
	require.NoError(t, err)
	deps.Commit()
	sdb := deps.NewStateDB()
	code := sdb.GetCode(token.ContractAddr)
	storage := sdb.GetStorageForOneContract(token.ContractAddr)
	sdb.SetCode(erisToken, code)
	for k, v := range storage {
		sdb.SetState(erisToken, k, v)
	}
	deps.Commit()
	// The Bank-origin mapping's ERC20 mint authority belongs to the EVM module.
	input, err := tokenContract.ABI.Pack("transferOwnership", evm.EVM_MODULE_ADDRESS)
	require.NoError(t, err)
	obj, _ := deps.NewEVM()
	resp, err := deps.EvmKeeper.CallContract(obj, deps.Sender.EthAddr, &erisToken, input, 1_000_000, evm.COMMIT_ETH_TX, nil)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)
	deps.Commit()
	require.NoError(t, deps.EvmKeeper.FunTokens.SafeInsert(deps.Ctx(), erisToken, erisDenom, true))
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, sdk.NewCoins(sdk.NewInt64Coin(erisDenom, 100_000_000))))
	implementation, err := evmtest.DeployContract(&deps, embeds.SmartContract_ErisEvm)
	require.NoError(t, err)
	proxy, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestDelegateProxy, implementation.ContractAddr)
	require.NoError(t, err)
	deps.Commit()
	return deps, proxy.ContractAddr
}
func erisExecute(t *testing.T, deps *evmtest.TestDeps, proxy common.Address, method string, args ...any) *evm.MsgEthereumTxResponse {
	t.Helper()
	input, err := embeds.SmartContract_ErisEvm.ABI.Pack(method, args...)
	require.NoError(t, err)
	obj, _ := deps.NewEVM()
	resp, err := deps.EvmKeeper.CallContract(obj, deps.Sender.EthAddr, &proxy, input, 10_000_000, evm.COMMIT_ETH_TX, nil)
	if err != nil {
		require.Contains(t, err.Error(), "VMError")
		resp = &evm.MsgEthereumTxResponse{VmError: err.Error()}
	}
	deps.Commit()
	return resp
}
func erisQuery(t *testing.T, deps *evmtest.TestDeps, contract, msg string) []byte {
	t.Helper()
	raw, err := deps.App.WasmKeeper.QuerySmart(deps.Ctx(), sdk.MustAccAddressFromBech32(contract), []byte(msg))
	require.NoError(t, err)
	return raw
}
func erisLock(t *testing.T, deps *evmtest.TestDeps) map[string]any {
	t.Helper()
	var lock map[string]any
	require.NoError(t, json.Unmarshal(erisQuery(t, deps, erisEscrow, `{"lock_info":{"user":"`+deps.Sender.NibiruAddr.String()+`"}}`), &lock))
	return lock
}

// TestErisGovernanceReplay uses the canonical implementation behind a proxy and
// actual Nibiru Wasm/FunToken precompiles. No precompile responses are mocked.
func TestErisGovernanceReplay(t *testing.T) {
	deps, proxy := setupErisGovernance(t)
	version := erisExecute(t, &deps, proxy, "governanceVersion")
	require.Empty(t, version.VmError)
	require.Equal(t, "1", new(big.Int).SetBytes(version.Ret).String())
	require.Empty(t, erisExecute(t, &deps, proxy, "createLock", big.NewInt(20_000_000), uint64(13*604800)).VmError)
	lock := erisLock(t, &deps)
	require.Equal(t, "20000000", lock["amount"])
	// The position belongs to the transaction sender, not the proxy or implementation.
	require.Empty(t, erisExecute(t, &deps, proxy, "vote", []string{erisValidator}, []uint16{10000}).VmError)
	user := erisQuery(t, &deps, erisGauge, `{"user_info":{"user":"`+deps.Sender.NibiruAddr.String()+`"}}`)
	require.Contains(t, string(user), erisValidator)
	require.Empty(t, erisExecute(t, &deps, proxy, "increaseLockAmount", big.NewInt(5_000_000)).VmError)
	require.Equal(t, "25000000", erisLock(t, &deps)["amount"])
	before := erisLock(t, &deps)
	require.Empty(t, erisExecute(t, &deps, proxy, "extendLockTime", uint64(4*604800)).VmError)
	after := erisLock(t, &deps)
	user = erisQuery(t, &deps, erisGauge, `{"user_info":{"user":"`+deps.Sender.NibiruAddr.String()+`"}}`)
	require.Equal(t, before["end"].(float64)+4, after["end"])
	// Shape-valid but non-allowlisted address fails in the live gauge after the
	// escrow extension and its callbacks. Both state changes must roll back.
	invalid := "nibivaloper1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"
	require.NotEmpty(t, erisExecute(t, &deps, proxy, "extendLockAndVote", uint64(604800), []string{invalid}, []uint16{10000}).VmError)
	require.Equal(t, after, erisLock(t, &deps))
	require.JSONEq(t, string(user), string(erisQuery(t, &deps, erisGauge, `{"user_info":{"user":"`+deps.Sender.NibiruAddr.String()+`"}}`)))
	require.Empty(t, erisExecute(t, &deps, proxy, "extendLockAndVote", uint64(604800), []string{erisValidator}, []uint16{10000}).VmError)
	require.Equal(t, after["end"].(float64)+1, erisLock(t, &deps)["end"])
	bankBefore := deps.App.BankKeeper.GetBalance(deps.Ctx(), deps.Sender.NibiruAddr, erisDenom).Amount.String()
	require.NotEmpty(t, erisExecute(t, &deps, proxy, "withdrawEscrow").VmError)
	require.Equal(t, bankBefore, deps.App.BankKeeper.GetBalance(deps.Ctx(), deps.Sender.NibiruAddr, erisDenom).Amount.String())
	deps.SetCtx(deps.Ctx().WithBlockTime(deps.Ctx().BlockTime().Add(19 * 7 * 24 * time.Hour)).WithBlockHeight(deps.Ctx().BlockHeight() + 100))
	require.Empty(t, erisExecute(t, &deps, proxy, "withdrawEscrow").VmError)
	require.Equal(t, "0", erisLock(t, &deps)["amount"])
	require.Equal(t, bankBefore, deps.App.BankKeeper.GetBalance(deps.Ctx(), deps.Sender.NibiruAddr, erisDenom).Amount.String(), "existing Bank receipts must remain")
	obj, _ := deps.NewEVM()
	bal, err := deps.EvmKeeper.ERC20().BalanceOf(erisToken, deps.Sender.EthAddr, deps.Ctx(), obj)
	require.NoError(t, err)
	require.Equal(t, "25000000", bal.String())
	deps.Commit()
	// Deposit with 5 Bank receipts and 25 ERC20 receipts converts only the 15
	// ERC20 shortfall. No ERC20 approval is required by FunToken's delegatecall.
	require.NoError(t, deps.App.BankKeeper.SendCoins(deps.Ctx(), deps.Sender.NibiruAddr, sdk.MustAccAddressFromBech32(erisEscrow), sdk.NewCoins(sdk.NewInt64Coin(erisDenom, 70_000_000))))
	require.Empty(t, erisExecute(t, &deps, proxy, "createLock", big.NewInt(20_000_000), uint64(3*604800)).VmError)
	require.Equal(t, "20000000", erisLock(t, &deps)["amount"])
	obj, _ = deps.NewEVM()
	bal, err = deps.EvmKeeper.ERC20().BalanceOf(erisToken, deps.Sender.EthAddr, deps.Ctx(), obj)
	require.NoError(t, err)
	require.Equal(t, "10000000", bal.String())
	deps.Commit()
	// A duplicate create fails after converting ERC20 receipts. The failed Wasm
	// execution must roll back that conversion and preserve the existing lock.
	require.NotEmpty(t, erisExecute(t, &deps, proxy, "createLock", big.NewInt(5_000_000), uint64(3*604800)).VmError)
	require.Equal(t, "20000000", erisLock(t, &deps)["amount"])
	require.Equal(t, "0", deps.App.BankKeeper.GetBalance(deps.Ctx(), deps.Sender.NibiruAddr, erisDenom).Amount.String())
	obj, _ = deps.NewEVM()
	bal, err = deps.EvmKeeper.ERC20().BalanceOf(erisToken, deps.Sender.EthAddr, deps.Ctx(), obj)
	require.NoError(t, err)
	require.Equal(t, "10000000", bal.String())
	deps.Commit()
	deps.SetCtx(deps.Ctx().WithBlockTime(deps.Ctx().BlockTime().Add(4 * 7 * 24 * time.Hour)).WithBlockHeight(deps.Ctx().BlockHeight() + 100))
	current := uint64((deps.Ctx().BlockTime().Unix() - 1667174400) / 604800)
	require.Empty(t, erisExecute(t, &deps, proxy, "extendLockTime", uint64(13*604800)).VmError)
	require.Equal(t, float64(current+13), erisLock(t, &deps)["end"], "expired relock starts at the current period")

}

func TestErisVoteRejectsMalformedPayloads(t *testing.T) {
	deps, proxy := setupErisGovernance(t)
	for _, tc := range []struct {
		name       string
		validators []string
		weights    []uint16
		want       string
	}{
		{"empty", nil, nil, "Invalid vote arrays"},
		{"mismatch", []string{erisValidator}, nil, "Invalid vote arrays"},
		{"duplicate", []string{erisValidator, erisValidator}, []uint16{5000, 5000}, "Duplicate validator"},
		{"zero", []string{erisValidator}, []uint16{0}, "Zero vote weight"},
		{"wrong total", []string{erisValidator}, []uint16{9999}, "Vote must total 10000"},
		{"JSON injection", []string{erisValidator[:49] + `"`}, []uint16{10000}, "Invalid validator character"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Contains(t, erisExecute(t, &deps, proxy, "vote", tc.validators, tc.weights).VmError, tc.want)
		})
	}
}
