package precompile_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// Mainnet addresses and denoms of the Eris contracts in the replay snapshot.
// They must match testdata/eris-governance/capture.py and the constants in
// ErisEvm.sol, because the replay imports live contract state at these
// addresses instead of instantiating new contracts.
const (
	// erisEscrow is the voting escrow. Locking receipts here gives voting
	// power, and every lock change calls back into both gauges.
	erisEscrow = "nibi1us4rh4a9rexvde8l3m8f8nlz6wcf9qg57zk2w06kqkm4vtp96vkst9kuwd"
	// erisGauge is the validator allocation gauge that the vote methods
	// write to.
	erisGauge = "nibi1qh59hdelxfwah7g7e8k0lxu4upqatq5w4jdw60yv9shhlhm86ckq4txl92"
	// erisDenom is the hub's liquid staking receipt (ampNIBI) in Bank form.
	erisDenom = "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI"
	// erisValidator is on the hub's validator allowlist at the snapshot
	// height, so the gauge accepts votes for it.
	erisValidator = "nibivaloper1cz36jg407s0z4f2vd7u6yc44v480dtpljrhdf6"
	// erisSnapshotDir holds the captured mainnet state. See its README.md for
	// how it was captured and how to regenerate it.
	erisSnapshotDir = "testdata/eris-governance/"
	// erisWeek is the escrow's lock period length in seconds. Lock ends are
	// counted in whole periods since erisPeriodStart.
	erisWeek = 604800
	// erisPeriodStart is the Unix time of the escrow's period 0, used to
	// compute the current period after the test advances block time.
	erisPeriodStart = 1667174400
)

// erisToken is the ERC20 address that ErisEvm.sol hardcodes for the ampNIBI
// receipt. The test copies a locally deployed ERC20Minter to this address so
// the production constant doesn't need a test-only override.
var erisToken = common.HexToAddress("0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b")

// erisSnapshot is the decoded form of testdata/eris-governance/snapshot.json.gz:
// one mainnet block's code, contract info, and raw storage for the four Eris
// governance contracts.
type erisSnapshot struct {
	Height    int64                  `json:"height"`
	Time      string                 `json:"time"`
	Contracts []erisContractSnapshot `json:"contracts"`
}

// erisContractSnapshot is one captured contract. Info and History stay as raw
// JSON because they decode with the app's codec, which belongs to each test's
// fresh app. Code and State are filled in by decodeErisSnapshot.
type erisContractSnapshot struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Info    json.RawMessage   `json:"info"`
	Hash    string            `json:"hash"`
	History []json.RawMessage `json:"history"`
	Models  []struct {
		Key   string `json:"key"`   // hex
		Value string `json:"value"` // base64
	} `json:"models"`

	// Code is the uncompressed Wasm bytecode, checked against Hash.
	Code []byte `json:"-"`
	// State is Models decoded into raw storage keys and values.
	State []wasmtypes.Model `json:"-"`
}

// The snapshot is several hundred kilobytes of JSON and Wasm to decompress
// and decode. Every test needs a fresh chain, but the decoded fixture is
// read-only, so it's loaded once per test binary and shared.
var (
	erisSnapshotOnce sync.Once
	erisSnapshotData *erisSnapshot
	erisSnapshotErr  error
)

// loadErisSnapshot returns the decoded snapshot, decompressing and decoding
// it on the first call only. Callers must not modify the result.
func loadErisSnapshot(t *testing.T) *erisSnapshot {
	t.Helper()
	erisSnapshotOnce.Do(func() {
		erisSnapshotData, erisSnapshotErr = decodeErisSnapshot()
	})
	require.NoError(t, erisSnapshotErr)
	return erisSnapshotData
}

// decodeErisSnapshot reads the gzipped snapshot and each contract's gzipped
// Wasm, verifies every code hash against the snapshot, and decodes storage.
func decodeErisSnapshot() (*erisSnapshot, error) {
	raw, err := readGzipFile(erisSnapshotDir + "snapshot.json.gz")
	if err != nil {
		return nil, err
	}
	var snapshot erisSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	for i := range snapshot.Contracts {
		c := &snapshot.Contracts[i]
		c.Code, err = readGzipFile(erisSnapshotDir + c.Name + ".wasm.gz")
		if err != nil {
			return nil, err
		}
		checksum := sha256.Sum256(c.Code)
		if got := hex.EncodeToString(checksum[:]); got != c.Hash {
			return nil, fmt.Errorf("%s code hash %s, snapshot says %s", c.Name, got, c.Hash)
		}
		c.State = make([]wasmtypes.Model, 0, len(c.Models))
		for _, m := range c.Models {
			key, err := hex.DecodeString(m.Key)
			if err != nil {
				return nil, fmt.Errorf("%s storage key %q: %w", c.Name, m.Key, err)
			}
			value, err := base64.StdEncoding.DecodeString(m.Value)
			if err != nil {
				return nil, fmt.Errorf("%s storage value for key %q: %w", c.Name, m.Key, err)
			}
			c.State = append(c.State, wasmtypes.Model{Key: key, Value: value})
		}
	}
	return &snapshot, nil
}

// readGzipFile returns the uncompressed contents of a gzip file.
func readGzipFile(path string) ([]byte, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// setupErisGovernance returns a fresh chain with the four Eris contracts
// imported from the mainnet snapshot, and the address of a delegate proxy
// that points at the canonical ErisEvm implementation.
//
// The Wasm state is imported unmodified. The test account starts with no lock
// or votes, so every position the test sees is one it created through the
// proxy. The receipt ERC20 is a standard six-decimal ERC20Minter deployed
// locally, then copied with its storage to erisToken, and its ownership moves
// to the EVM module as it would for a Bank-origin FunToken mapping. Bank
// receipt backing and the account's funds are synthetic, so the replay covers
// the contracts and precompiles, not the rest of mainnet's module state.
func setupErisGovernance(t *testing.T) (evmtest.TestDeps, common.Address) {
	t.Helper()
	snapshot := loadErisSnapshot(t)
	deps := evmtest.NewTestDeps()
	captured, err := time.Parse(time.RFC3339Nano, snapshot.Time)
	require.NoError(t, err)
	deps.SetCtx(deps.Ctx().
		WithChainID("cataclysm-1").
		WithBlockHeight(snapshot.Height).
		WithBlockTime(captured))

	gen := wasmtypes.GenesisState{Params: wasmtypes.DefaultParams()}
	for _, c := range snapshot.Contracts {
		var info wasmtypes.ContractInfo
		require.NoError(t, deps.App.AppCodec().UnmarshalJSON(c.Info, &info))
		history := make([]wasmtypes.ContractCodeHistoryEntry, len(c.History))
		for i, h := range c.History {
			require.NoError(t, deps.App.AppCodec().UnmarshalJSON(h, &history[i]))
		}
		checksum := sha256.Sum256(c.Code)
		gen.Codes = append(gen.Codes, wasmtypes.Code{
			CodeID: info.CodeID,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash:          checksum[:],
				Creator:           info.Creator,
				InstantiateConfig: wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody},
			},
			CodeBytes: c.Code,
		})
		gen.Contracts = append(gen.Contracts, wasmtypes.Contract{
			ContractAddress:     c.Address,
			ContractInfo:        info,
			ContractState:       c.State,
			ContractCodeHistory: history,
		})
	}
	// Start the sequences above the captured code and instance IDs so nothing
	// created during the test collides with imported state.
	gen.Sequences = []wasmtypes.Sequence{
		{IDKey: wasmtypes.KeySequenceCodeID, Value: 200},
		{IDKey: wasmtypes.KeySequenceInstanceID, Value: 100},
	}
	_, err = wasmkeeper.InitGenesis(deps.Ctx(), &deps.App.WasmKeeper, gen)
	require.NoError(t, err)
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewInt64Coin("unibi", 100_000_000))))

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
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewInt64Coin(erisDenom, 100_000_000))))

	implementation, err := evmtest.DeployContract(&deps, embeds.SmartContract_ErisEvm)
	require.NoError(t, err)
	proxy, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestDelegateProxy, implementation.ContractAddr)
	require.NoError(t, err)
	deps.Commit()
	return deps, proxy.ContractAddr
}

// erisExecute calls an ErisEvm method through the proxy as the test sender and
// commits. A revert comes back as a response with VmError set instead of a Go
// error, so tests can assert on both outcomes the same way.
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

// erisQuery runs a smart query against one of the imported Wasm contracts.
func erisQuery(t *testing.T, deps *evmtest.TestDeps, contract, msg string) []byte {
	t.Helper()
	raw, err := deps.App.WasmKeeper.QuerySmart(deps.Ctx(), sdk.MustAccAddressFromBech32(contract), []byte(msg))
	require.NoError(t, err)
	return raw
}

// erisLock returns the test sender's escrow lock. Looking it up by the
// sender's Nibiru address also proves the proxy acted as the sender, not as
// itself or the implementation.
func erisLock(t *testing.T, deps *evmtest.TestDeps) map[string]any {
	t.Helper()
	var lock map[string]any
	msg := `{"lock_info":{"user":"` + deps.Sender.NibiruAddr.String() + `"}}`
	require.NoError(t, json.Unmarshal(erisQuery(t, deps, erisEscrow, msg), &lock))
	return lock
}

// erisUserVotes returns the test sender's allocation in the validator gauge.
func erisUserVotes(t *testing.T, deps *evmtest.TestDeps) []byte {
	t.Helper()
	return erisQuery(t, deps, erisGauge, `{"user_info":{"user":"`+deps.Sender.NibiruAddr.String()+`"}}`)
}

// erisERC20Balance returns the test sender's ERC20 receipt balance.
func erisERC20Balance(t *testing.T, deps *evmtest.TestDeps) string {
	t.Helper()
	obj, _ := deps.NewEVM()
	bal, err := deps.EvmKeeper.ERC20().BalanceOf(erisToken, deps.Sender.EthAddr, deps.Ctx(), obj)
	require.NoError(t, err)
	deps.Commit()
	return bal.String()
}

// erisBankBalance returns the test sender's Bank receipt balance.
func erisBankBalance(deps *evmtest.TestDeps) string {
	return deps.App.BankKeeper.GetBalance(deps.Ctx(), deps.Sender.NibiruAddr, erisDenom).Amount.String()
}

// TestErisGovernanceReplay drives the voting escrow and validator gauge
// through the canonical ErisEvm implementation behind a delegate proxy, using
// the real Wasm and FunToken precompiles against captured mainnet contracts.
// Nothing is mocked. It covers what a unit test can't: that the live escrow
// calls back into both live gauges, that a failure late in a transaction
// rolls back earlier Wasm and FunToken effects, and that receipts move
// between Bank and ERC20 form the way ErisEvm.sol documents.
func TestErisGovernanceReplay(t *testing.T) {
	deps, proxy := setupErisGovernance(t)

	// The web app reads this to decide whether to show EVM voting controls.
	version := erisExecute(t, &deps, proxy, "governanceVersion")
	require.Empty(t, version.VmError)
	require.Equal(t, "1", new(big.Int).SetBytes(version.Ret).String())

	// The position belongs to the transaction sender, not the proxy or
	// implementation.
	require.Empty(t, erisExecute(t, &deps, proxy, "createLock", big.NewInt(20_000_000), uint64(13*erisWeek)).VmError)
	require.Equal(t, "20000000", erisLock(t, &deps)["amount"])
	require.Empty(t, erisExecute(t, &deps, proxy, "vote", []string{erisValidator}, []uint16{10000}).VmError)
	require.Contains(t, string(erisUserVotes(t, &deps)), erisValidator)

	require.Empty(t, erisExecute(t, &deps, proxy, "increaseLockAmount", big.NewInt(5_000_000)).VmError)
	require.Equal(t, "25000000", erisLock(t, &deps)["amount"])
	before := erisLock(t, &deps)
	require.Empty(t, erisExecute(t, &deps, proxy, "extendLockTime", uint64(4*erisWeek)).VmError)
	after := erisLock(t, &deps)
	votes := erisUserVotes(t, &deps)
	require.Equal(t, before["end"].(float64)+4, after["end"])

	// A shape-valid but non-allowlisted validator fails in the live gauge after
	// the escrow extension and its gauge callbacks. Both must roll back.
	invalid := "nibivaloper1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"
	require.NotEmpty(t, erisExecute(t, &deps, proxy, "extendLockAndVote", uint64(erisWeek), []string{invalid}, []uint16{10000}).VmError)
	require.Equal(t, after, erisLock(t, &deps))
	require.JSONEq(t, string(votes), string(erisUserVotes(t, &deps)))
	require.Empty(t, erisExecute(t, &deps, proxy, "extendLockAndVote", uint64(erisWeek), []string{erisValidator}, []uint16{10000}).VmError)
	require.Equal(t, after["end"].(float64)+1, erisLock(t, &deps)["end"])

	// Withdrawing before the lock expires fails and moves nothing. After
	// expiry, only the newly withdrawn receipts come back, as ERC20, and the
	// existing Bank receipts stay in Bank form.
	bankBefore := erisBankBalance(&deps)
	require.NotEmpty(t, erisExecute(t, &deps, proxy, "withdrawEscrow").VmError)
	require.Equal(t, bankBefore, erisBankBalance(&deps))
	deps.SetCtx(deps.Ctx().
		WithBlockTime(deps.Ctx().BlockTime().Add(19 * 7 * 24 * time.Hour)).
		WithBlockHeight(deps.Ctx().BlockHeight() + 100))
	require.Empty(t, erisExecute(t, &deps, proxy, "withdrawEscrow").VmError)
	require.Equal(t, "0", erisLock(t, &deps)["amount"])
	require.Equal(t, bankBefore, erisBankBalance(&deps), "existing Bank receipts must remain")
	require.Equal(t, "25000000", erisERC20Balance(t, &deps))

	// With 5 Bank receipts and 25 ERC20 receipts, a 20 deposit uses the Bank
	// receipts first and converts only the 15 ERC20 shortfall. FunToken's
	// delegatecall needs no ERC20 approval.
	require.NoError(t, deps.App.BankKeeper.SendCoins(deps.Ctx(), deps.Sender.NibiruAddr,
		sdk.MustAccAddressFromBech32(erisEscrow), sdk.NewCoins(sdk.NewInt64Coin(erisDenom, 70_000_000))))
	require.Empty(t, erisExecute(t, &deps, proxy, "createLock", big.NewInt(20_000_000), uint64(3*erisWeek)).VmError)
	require.Equal(t, "20000000", erisLock(t, &deps)["amount"])
	require.Equal(t, "10000000", erisERC20Balance(t, &deps))

	// A duplicate create fails after converting ERC20 receipts. The failed
	// Wasm execution must roll back that conversion and keep the existing lock.
	require.NotEmpty(t, erisExecute(t, &deps, proxy, "createLock", big.NewInt(5_000_000), uint64(3*erisWeek)).VmError)
	require.Equal(t, "20000000", erisLock(t, &deps)["amount"])
	require.Equal(t, "0", erisBankBalance(&deps))
	require.Equal(t, "10000000", erisERC20Balance(t, &deps))

	// Relocking an expired position starts from the current period, not from
	// the old end.
	deps.SetCtx(deps.Ctx().
		WithBlockTime(deps.Ctx().BlockTime().Add(4 * 7 * 24 * time.Hour)).
		WithBlockHeight(deps.Ctx().BlockHeight() + 100))
	current := uint64((deps.Ctx().BlockTime().Unix() - erisPeriodStart) / erisWeek)
	require.Empty(t, erisExecute(t, &deps, proxy, "extendLockTime", uint64(13*erisWeek)).VmError)
	require.Equal(t, float64(current+13), erisLock(t, &deps)["end"], "expired relock starts at the current period")
}

// TestErisVoteRejectsMalformedPayloads checks the input validation ErisEvm.sol
// does before it builds the gauge's JSON message. Validator strings are
// interpolated into JSON, so a quote or other unexpected character must be
// rejected rather than passed through. Allocations must be complete (10000
// basis points), unique, and nonzero.
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
