package upgrades_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/rootmulti"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	consensuskeeper "github.com/NibiruChain/nibiru/v2/x/consensus/keeper"
	consensustypes "github.com/NibiruChain/nibiru/v2/x/consensus/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

type usdcUpgradeFixture struct {
	deps                 evmtest.TestDeps
	cfg                  upgrades.USDCMigrationConfigV223
	bankUser             sdk.AccAddress
	eoa, unrelated, zero gethcommon.Address
}

func setupUSDCUpgrade(t *testing.T, chainID string) usdcUpgradeFixture {
	t.Helper()
	return setupUSDCUpgradeFromDeps(t, chainID, evmtest.NewTestDeps())
}

func setupUSDCUpgradeFromDeps(t *testing.T, chainID string, deps evmtest.TestDeps) usdcUpgradeFixture {
	t.Helper()
	deps.SetCtx(deps.Ctx().WithChainID(chainID).WithBlockHeight(20).WithGasMeter(sdk.NewInfiniteGasMeter()))
	creator := sdk.AccAddress(bytes.Repeat([]byte{17}, 20))
	deps.App.SudoKeeper.Sudoers.Set(deps.Ctx(), sudo.Sudoers{Root: creator.String()})
	registry, checksum := deployHolderSet(t, deps, creator, []string{}, []byte("usdc-holders-v1"))
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 100))))
	deploy := func() gethcommon.Address {
		r, err := evmtest.DeployContract(&deps, embeds.SmartContract_ERC20MinterWithMetadataUpdates, "USDC fixture", "USDC", uint8(6))
		require.NoError(t, err)
		return r.ContractAddr
	}
	token, safe, saiEvm, unrelated := deploy(), deploy(), deploy(), deploy()
	cfg := upgrades.USDCMigrationConfigV223{Registry: registry, RegistryChecksum: checksum, Token: token, BankDenom: "erc20/" + token.Hex(), Safe: safe, SaiEvm: saiEvm, ReplacedBankHolders: []sdk.AccAddress{registry}}
	require.NoError(t, deps.EvmKeeper.FunTokens.SafeInsert(deps.Ctx(), token, cfg.BankDenom, false))
	eoa := gethcommon.HexToAddress("0x0000000000000000000000000000000000000200")
	bankUser := eth.EthAddrToNibiruAddr(gethcommon.HexToAddress("0x0000000000000000000000000000000000000100"))
	zero := gethcommon.HexToAddress("0x0000000000000000000000000000000000000300")
	mint := func(to gethcommon.Address, amount int64) {
		evmObj, sdb := deps.NewEVM()
		resp, err := deps.EvmKeeper.ERC20().Mint(token, deps.Sender.EthAddr, to, big.NewInt(amount), deps.Ctx(), evmObj)
		require.NoError(t, err)
		require.False(t, resp.Failed())
		sdb.Commit()
	}
	mint(deps.Sender.EthAddr, 70)
	mint(eoa, 7)
	mint(saiEvm, 11)
	mint(unrelated, 13)
	mint(safe, 5)
	for _, holder := range []struct {
		address sdk.AccAddress
		amount  int64
	}{{bankUser, 30}, {registry, 40}} {
		_, err := deps.EvmKeeper.ConvertEvmToCoin(sdk.WrapSDKContext(deps.Ctx()), &evm.MsgConvertEvmToCoin{Sender: deps.Sender.NibiruAddr.String(), Erc20Addr: eth.EIP55Addr{Address: token}, Amount: sdkmath.NewInt(holder.amount), ToAddr: holder.address.String()})
		require.NoError(t, err)
	}
	// Include protected addresses and an unrelated contract deliberately. The
	// handler must classify runtime code instead of trusting the input set.
	addresses := []string{eoa.Hex(), unrelated.Hex(), safe.Hex(), evm.EVM_MODULE_ADDRESS.Hex(), zero.Hex()}
	msg, err := json.Marshal(map[string]any{"update_addresses": map[string]any{"add": addresses, "remove": []string{}}})
	require.NoError(t, err)
	_, err = wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper).Execute(deps.Ctx(), registry, creator, msg, sdk.Coins{})
	require.NoError(t, err)
	deps.SetCtx(deps.Ctx().WithEventManager(sdk.NewEventManager()))
	return usdcUpgradeFixture{deps: deps, cfg: cfg, bankUser: bankUser, eoa: eoa, unrelated: unrelated, zero: zero}
}

func usdcUpgrade(f usdcUpgradeFixture) upgrades.Upgrade {
	return upgrades.Upgrade{UpgradeName: "v2.23.0", Handler: upgrades.Handler_v2_23{USDC: &f.cfg}}
}

func usdcBalance(t *testing.T, f *usdcUpgradeFixture, address gethcommon.Address) *big.Int {
	t.Helper()
	k := f.deps.EvmKeeper
	ctx := f.deps.Ctx()
	sdb := k.NewSDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	zero := new(big.Int)
	obj := k.NewEVM(sdb.Ctx(), core.Message{From: evm.EVM_READONLY_ADDR, To: &f.cfg.Token, Value: zero, GasPrice: zero, GasFeeCap: zero, GasTipCap: zero, BlobGasFeeCap: zero, GasLimit: evm.Erc20GasLimitQuery}, k.GetEVMConfig(ctx), nil, sdb)
	bal, err := k.ERC20().BalanceOf(f.cfg.Token, address, sdb.Ctx(), obj)
	require.NoError(t, err)
	return bal
}

func TestUpgrade2_23_0USDCCollectionHappyPath(t *testing.T) {
	for _, chainID := range []string{"localnet", "nibiru-testnet-3", "cataclysm-1"} {
		t.Run(chainID, func(t *testing.T) {
			var f usdcUpgradeFixture
			if chainID == "cataclysm-1" {
				deps := evmtest.NewTestDepsPreXConsensus()
				params := preUpgradeParams()
				deps.App.ConsensusKeeper.Set(deps.Ctx(), &params)
				versions := deps.App.UpgradeKeeper.GetModuleVersionMap(deps.Ctx())
				delete(versions, consensustypes.ModuleName)
				versions["upgrade"] = 2
				deps.App.UpgradeKeeper.SetModuleVersionMap(deps.Ctx(), versions)
				_, found := deps.App.ConsensusKeeper.ContractAddr(deps.Ctx())
				require.False(t, found, "mainnet starts before v2.22 enabling contracts")
				require.False(t, deps.App.UpgradeKeeper.ContractActive(deps.Ctx()))
				f = setupUSDCUpgradeFromDeps(t, chainID, deps)
			} else {
				f = setupUSDCUpgrade(t, chainID)
				_, found := f.deps.App.ConsensusKeeper.ContractAddr(f.deps.Ctx())
				require.True(t, found, "testnet/localnet starts with v2.22 enabling contracts")
				require.True(t, f.deps.App.UpgradeKeeper.ContractActive(f.deps.Ctx()))
			}
			eoaNonce := f.deps.EvmKeeper.GetAccNonce(f.deps.Ctx(), f.eoa)
			saiNonce := f.deps.EvmKeeper.GetAccNonce(f.deps.Ctx(), f.cfg.SaiEvm)
			readonlyNonce := f.deps.EvmKeeper.GetAccNonce(f.deps.Ctx(), evm.EVM_READONLY_ADDR)
			require.NoError(t, f.deps.RunUpgrade(usdcUpgrade(f)))
			_, found := f.deps.App.ConsensusKeeper.ContractAddr(f.deps.Ctx())
			require.True(t, found)
			require.True(t, f.deps.App.UpgradeKeeper.ContractActive(f.deps.Ctx()))
			require.Equal(t, uint64(1), f.deps.App.UpgradeKeeper.GetModuleVersionMap(f.deps.Ctx())[consensustypes.ModuleName])
			require.Equal(t, uint64(3), f.deps.App.UpgradeKeeper.GetModuleVersionMap(f.deps.Ctx())["upgrade"])
			require.Zero(t, f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), f.cfg.BankDenom).Amount.Sign())
			require.Zero(t, usdcBalance(t, &f, evm.EVM_MODULE_ADDRESS).Sign())
			require.Equal(t, "93", usdcBalance(t, &f, f.cfg.Safe).String()) // original5 + bank70 + EOA7 + SaiEvm11
			require.Zero(t, usdcBalance(t, &f, f.eoa).Sign())
			require.Zero(t, usdcBalance(t, &f, f.cfg.SaiEvm).Sign())
			require.Equal(t, "13", usdcBalance(t, &f, f.unrelated).String())
			require.Equal(t, eoaNonce, f.deps.EvmKeeper.GetAccNonce(f.deps.Ctx(), f.eoa))
			require.Equal(t, saiNonce, f.deps.EvmKeeper.GetAccNonce(f.deps.Ctx(), f.cfg.SaiEvm))
			require.Equal(t, readonlyNonce, f.deps.EvmKeeper.GetAccNonce(f.deps.Ctx(), evm.EVM_READONLY_ADDR))
			report, err := upgrades.ReconstructUSDCMigration(f.deps.Ctx().EventManager().ABCIEvents())
			require.NoError(t, err)
			require.Equal(t, "88", report.Collected)
			require.Equal(t, "37", report.Repayment)
			require.Equal(t, "51", report.CustodyRecovery)
			f.deps.SetCtx(f.deps.Ctx().WithEventManager(sdk.NewEventManager()))
			require.NoError(t, f.deps.RunUpgrade(usdcUpgrade(f)))
			report, err = upgrades.ReconstructUSDCMigration(f.deps.Ctx().EventManager().ABCIEvents())
			require.NoError(t, err)
			require.Equal(t, "0", report.Collected)
			require.Equal(t, "93", usdcBalance(t, &f, f.cfg.Safe).String())
		})
	}
}

func TestUpgrade2_23_0USDCPreflight(t *testing.T) {
	for _, mode := range []string{"missing_registry", "wrong_checksum", "wrong_mapping", "missing_safe"} {
		t.Run(mode, func(t *testing.T) {
			f := setupUSDCUpgrade(t, "cataclysm-1")
			switch mode {
			case "missing_registry":
				f.cfg.Registry = sdk.AccAddress(bytes.Repeat([]byte{99}, 32))
			case "wrong_checksum":
				f.cfg.RegistryChecksum = bytes.Repeat([]byte{55}, 32)
			case "wrong_mapping":
				f.cfg.BankDenom = "erc20/wrong"
			case "missing_safe":
				f.cfg.Safe = f.zero
			}
			require.NoError(t, f.deps.RunUpgrade(usdcUpgrade(f)))
			require.Equal(t, "70", f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), "erc20/"+f.cfg.Token.Hex()).Amount.String())
			require.Equal(t, "7", usdcBalance(t, &f, f.eoa).String())
			report, err := upgrades.ReconstructUSDCMigration(f.deps.Ctx().EventManager().ABCIEvents())
			require.NoError(t, err)
			require.Equal(t, "0", report.Collected)
		})
	}
}

// Add a delegate proxy that rejects transfers from one source. The token's
// balances remain in the original storage, and all other callers execute the
// real ERC20 bytecode. This exercises failures after Bank send/burn happened.
func rejectUSDCSource(t *testing.T, f *usdcUpgradeFixture, source gethcommon.Address, balanceQuery bool) {
	t.Helper()
	ctx := f.deps.Ctx()
	k := f.deps.EvmKeeper
	sdb := k.NewSDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	implementation := gethcommon.HexToAddress("0x000000000000000000000000000000000000f001")
	sdb.SetCode(implementation, sdb.GetCode(f.cfg.Token))
	// The filter compares either CALLER or the first ABI argument. Only the
	// selected source fails; all other calls delegate to real ERC20 runtime.
	load, prefixLength := "33", 27
	if balanceQuery {
		load, prefixLength = "600435", 29
	}
	prefix := load + "73" + hex.EncodeToString(source.Bytes()) + fmt.Sprintf("1461%04x57", prefixLength+45)
	proxy := "363d3d373d3d3d363d73" + hex.EncodeToString(implementation.Bytes()) + fmt.Sprintf("5af43d82803e903d9160%02x57fd5bf3", prefixLength+43)
	code, err := hex.DecodeString(prefix + proxy + "5b60006000fd")
	require.NoError(t, err)
	sdb.SetCode(f.cfg.Token, code)
	sdb.Commit()
}

func TestUpgrade2_23_0USDCRollbackAndContinuation(t *testing.T) {
	for _, mode := range []string{"bank", "erc20", "balance_query"} {
		t.Run(mode, func(t *testing.T) {
			f := setupUSDCUpgrade(t, "cataclysm-1")
			source := f.eoa
			if mode == "bank" {
				source = evm.EVM_MODULE_ADDRESS
			}
			rejectUSDCSource(t, &f, source, mode == "balance_query")
			require.NoError(t, f.deps.RunUpgrade(usdcUpgrade(f)))
			report, err := upgrades.ReconstructUSDCMigration(f.deps.Ctx().EventManager().ABCIEvents())
			require.NoError(t, err)
			if mode == "bank" {
				require.Equal(t, "70", f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), f.cfg.BankDenom).Amount.String())
				require.Equal(t, "30", f.deps.App.BankKeeper.GetBalance(f.deps.Ctx(), f.bankUser, f.cfg.BankDenom).Amount.String())
				require.Equal(t, "40", f.deps.App.BankKeeper.GetBalance(f.deps.Ctx(), f.cfg.Registry, f.cfg.BankDenom).Amount.String())
				require.Equal(t, "70", usdcBalance(t, &f, evm.EVM_MODULE_ADDRESS).String())
				require.Equal(t, "18", report.Collected)
			} else {
				require.Zero(t, f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), f.cfg.BankDenom).Amount.Sign())
				if mode != "balance_query" {
					require.Equal(t, "7", usdcBalance(t, &f, f.eoa).String())
				}
				require.Equal(t, "81", report.Collected)
				var failed bool
				for _, account := range report.Accounts {
					if account.Source == f.eoa.Hex() {
						failed = account.Status == "failed"
					}
				}
				require.True(t, failed, "failed source must remain visible while later holders continue")
			}
			require.Equal(t, "13", usdcBalance(t, &f, f.unrelated).String())
		})
	}
}

func TestUpgrade2_23_0USDCBeginBlockEvents(t *testing.T) {
	f := setupUSDCUpgrade(t, "cataclysm-1")
	upgrade := usdcUpgrade(f)
	f.deps.App.UpgradeKeeper.SetUpgradeHandler(upgrade.UpgradeName, upgrade.Handler.Handler(f.deps.App.ModuleManager, f.deps.App.Configurator(), &f.deps.App.PublicKeepers))
	// Schedule for the next block because ScheduleUpgrade rejects current/past heights.
	height := f.deps.Ctx().BlockHeight() + 1
	require.NoError(t, f.deps.App.UpgradeKeeper.ScheduleUpgrade(f.deps.Ctx(), upgradetypes.Plan{Name: upgrade.UpgradeName, Height: height}))
	f.deps.SetCtx(f.deps.Ctx().WithBlockHeight(height).WithEventManager(sdk.NewEventManager()))
	response := f.deps.App.BeginBlocker(f.deps.Ctx(), abci.RequestBeginBlock{Header: f.deps.Ctx().BlockHeader()})
	report, err := upgrades.ReconstructUSDCMigration(response.Events)
	require.NoError(t, err)
	require.Equal(t, "88", report.Collected)
	require.Equal(t, fmt.Sprint(height), fmt.Sprint(report.Height))
}

// Unbacked direct donations remain escrow rather than becoming an entitlement.
func TestUpgrade2_23_0USDCExcessEscrow(t *testing.T) {
	f := setupUSDCUpgrade(t, "cataclysm-1")
	obj, sdb := f.deps.NewEVM()
	_, err := f.deps.EvmKeeper.ERC20().Mint(f.cfg.Token, f.deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(9), sdb.Ctx(), obj)
	require.NoError(t, err)
	sdb.Commit()
	require.NoError(t, f.deps.RunUpgrade(usdcUpgrade(f)))
	report, err := upgrades.ReconstructUSDCMigration(f.deps.Ctx().EventManager().ABCIEvents())
	require.NoError(t, err)
	require.Equal(t, "88", report.Collected)
	require.Equal(t, "9", usdcBalance(t, &f, evm.EVM_MODULE_ADDRESS).String())
	require.Zero(t, f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), f.cfg.BankDenom).Amount.Sign())
	found := false
	for _, event := range f.deps.Ctx().EventManager().Events() {
		if event.Type != "usdc_migration" {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key == "escrow_excess" {
				require.Equal(t, "9", attribute.Value)
				found = true
			}
		}
	}
	require.True(t, found)
}

// Exercise both pagination paths and collection at the observed EOA-set scale.
func TestUpgrade2_23_0USDCFullHolderPagination(t *testing.T) {
	started := time.Now()
	f := setupUSDCUpgrade(t, "cataclysm-1")
	const bankHolders, erc20Holders = 211, 947
	for i := 0; i < bankHolders; i++ {
		address := eth.EthAddrToNibiruAddr(gethcommon.BigToAddress(big.NewInt(int64(0x10000 + i))))
		require.NoError(t, testapp.FundAccount(f.deps.App.BankKeeper, f.deps.Ctx(), address, sdk.NewCoins(sdk.NewInt64Coin(f.cfg.BankDenom, 1))))
	}
	obj, sdb := f.deps.NewEVM()
	mint := func(to gethcommon.Address, amount int64) {
		_, err := f.deps.EvmKeeper.ERC20().Mint(f.cfg.Token, f.deps.Sender.EthAddr, to, big.NewInt(amount), sdb.Ctx(), obj)
		require.NoError(t, err)
	}
	mint(evm.EVM_MODULE_ADDRESS, bankHolders)
	addresses := make([]string, 0, erc20Holders)
	for i := 0; i < erc20Holders; i++ {
		address := gethcommon.BigToAddress(big.NewInt(int64(0x20000 + i)))
		mint(address, 1)
		addresses = append(addresses, address.Hex())
	}
	sdb.Commit()
	creator := sdk.AccAddress(bytes.Repeat([]byte{17}, 20))
	for offset := 0; offset < len(addresses); offset += 500 {
		end := offset + 500
		if end > len(addresses) {
			end = len(addresses)
		}
		msg, err := json.Marshal(map[string]any{"update_addresses": map[string]any{"add": addresses[offset:end], "remove": []string{}}})
		require.NoError(t, err)
		_, err = wasmkeeper.NewDefaultPermissionKeeper(f.deps.App.WasmKeeper).Execute(f.deps.Ctx(), f.cfg.Registry, creator, msg, sdk.Coins{})
		require.NoError(t, err)
	}
	f.deps.SetCtx(f.deps.Ctx().WithEventManager(sdk.NewEventManager()))
	require.NoError(t, f.deps.RunUpgrade(usdcUpgrade(f)))
	report, err := upgrades.ReconstructUSDCMigration(f.deps.Ctx().EventManager().ABCIEvents())
	require.NoError(t, err)
	require.Equal(t, "1246", report.Collected)
	require.Equal(t, "1195", report.Repayment)
	require.Equal(t, "51", report.CustodyRecovery)
	require.Equal(t, "1251", usdcBalance(t, &f, f.cfg.Safe).String())
	require.Zero(t, f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), f.cfg.BankDenom).Amount.Sign())
	require.Zero(t, usdcBalance(t, &f, evm.EVM_MODULE_ADDRESS).Sign())
	counts := make(map[string]int)
	for _, account := range report.Accounts {
		if account.Status == "collected" {
			counts[account.Representation]++
		}
	}
	require.Equal(t, bankHolders+2, counts["bank"])
	require.Equal(t, erc20Holders+2, counts["erc20"])
	t.Logf("collected %d bank holders and %d ERC20 holders in %s", counts["bank"], counts["erc20"], time.Since(started))
}

func TestUpgrade2_23_0RequiredActivationFailureStopsCollection(t *testing.T) {
	deps := evmtest.NewTestDepsPreXConsensus()
	versions := deps.App.UpgradeKeeper.GetModuleVersionMap(deps.Ctx())
	delete(versions, consensustypes.ModuleName)
	versions["upgrade"] = 2
	deps.App.UpgradeKeeper.SetModuleVersionMap(deps.Ctx(), versions)
	f := setupUSDCUpgradeFromDeps(t, "cataclysm-1", deps)
	// An unbound required migration must return its error, never be treated as
	// an optional holder failure or allow the custom collection to proceed.
	f.deps.App.UpgradeKeeper.BindWasm(nil, nil, nil)
	err := f.deps.RunUpgrade(usdcUpgrade(f))
	require.ErrorContains(t, err, "activate x-upgrade: upgrade contract dependencies not bound")
	require.False(t, f.deps.App.UpgradeKeeper.ContractActive(f.deps.Ctx()))
	require.Equal(t, uint64(2), f.deps.App.UpgradeKeeper.GetModuleVersionMap(f.deps.Ctx())["upgrade"])
	require.Equal(t, "70", f.deps.App.BankKeeper.GetSupply(f.deps.Ctx(), f.cfg.BankDenom).Amount.String())
	require.Equal(t, "7", usdcBalance(t, &f, f.eoa).String())
	for _, event := range f.deps.Ctx().EventManager().Events() {
		require.NotEqual(t, "usdc_migration", event.Type)
	}
}

// preUpgradeParams stands in for the x/consensus params of a live chain
// before the enabling migration, with a Version that the upgrade must keep.
func preUpgradeParams() cmtproto.ConsensusParams {
	return cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{MaxBytes: 22_020_096, MaxGas: 300_000_000},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 100_000,
			MaxAgeDuration:  48 * time.Hour,
			MaxBytes:        1_048_576,
		},
		Validator: &cmtproto.ValidatorParams{PubKeyTypes: []string{"ed25519"}},
		Version:   &cmtproto.VersionParams{App: 0},
	}
}

// runConsensusUpgrade2_23 runs the v2.23 handler like x/upgrade does: with the stored
// version map, recording the result only on success. It returns the events
// the handler emitted.
func runConsensusUpgrade2_23(
	t *testing.T, nibiru *app.NibiruApp, ctx sdk.Context,
) sdk.Events {
	t.Helper()
	handler := upgrades.Upgrade2_23_0.Handler.Handler(
		nibiru.ModuleManager, nibiru.Configurator(), &nibiru.PublicKeepers)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	plan := upgradetypes.Plan{
		Name: upgrades.Upgrade2_23_0.UpgradeName, Height: ctx.BlockHeight(),
	}
	versions, err := handler(ctx, plan,
		nibiru.UpgradeKeeper.GetModuleVersionMap(ctx))
	require.NoError(t, err)
	nibiru.UpgradeKeeper.SetModuleVersionMap(ctx, versions)
	return ctx.EventManager().Events()
}

// preUpgradeChain returns a chain without x-consensus whose x/consensus store
// holds params, and whose version map predates the x/consensus module, so the
// handler sees exactly what a live chain at the upgrade height has.
func preUpgradeChain(
	t *testing.T, chainID string, params *cmtproto.ConsensusParams,
) (*app.NibiruApp, sdk.Context) {
	t.Helper()
	nibiru, _ := testapp.NewNibiruTestAppPreXConsensus()
	ctx := testapp.NewContext(nibiru).WithChainID(chainID)
	_, found := nibiru.ConsensusKeeper.ContractAddr(ctx)
	require.False(t, found)
	nibiru.ConsensusKeeper.Set(ctx, params)
	versions := nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)
	delete(versions, consensustypes.ModuleName)
	nibiru.UpgradeKeeper.SetModuleVersionMap(ctx, module.VersionMap(versions))
	return nibiru, ctx
}

// TestUpgrade2_23_0CreatesXConsensus checks the upgrade on mainnet and
// testnet chain IDs: x-consensus is created at the predicted address, as its
// own Wasm admin and a wasm_deployer member, with exactly the stored params and the commit wait unset,
// and the stored params, Version included, are unchanged by the next refresh. A second run is a
// no-op. The mainnet chain ID exercises the Wasm deployer guard, which the
// governance permission keeper passes.
func TestUpgrade2_23_0CreatesXConsensus(t *testing.T) {
	for _, chainID := range []string{
		appconst.SDK_CHAIN_ID_MAINNET, "nibiru-testnet-2", "localnet",
	} {
		t.Run(chainID, func(t *testing.T) {
			pre := preUpgradeParams()
			nibiru, ctx := preUpgradeChain(t, chainID, &pre)

			events := runConsensusUpgrade2_23(t, nibiru, ctx)
			for _, ev := range events {
				require.NotEqual(t, "upgrade_failure", ev.Type, ev.Attributes)
			}
			versions := nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)
			require.Equal(t, uint64(1), versions[consensustypes.ModuleName])

			addr, found := nibiru.ConsensusKeeper.ContractAddr(ctx)
			require.True(t, found)
			info := nibiru.WasmKeeper.GetContractInfo(ctx, addr)
			require.Equal(t, consensuskeeper.CreatorAddr().String(), info.Creator)
			require.Equal(t, addr.String(), info.Admin)
			checksum := nibiru.WasmKeeper.GetCodeInfo(ctx, info.CodeID).CodeHash
			require.Equal(t, consensuskeeper.PredictContractAddr(checksum), addr)
			// The contract is not the root, so passing this check means it
			// is a wasm_deployer member.
			require.NoError(t, nibiru.SudoKeeper.CheckPermissions(
				addr, ctx, sudo.RoleWasmDeployer))

			req, err := json.Marshal(consensustypes.QueryParams())
			require.NoError(t, err)
			resp, err := nibiru.WasmKeeper.QuerySmart(ctx, addr, req)
			require.NoError(t, err)
			var got consensustypes.ConsensusParams
			require.NoError(t, json.Unmarshal(resp, &got))
			want, err := consensustypes.ConsensusParamsFromProto(pre)
			require.NoError(t, err)
			require.Equal(t, want, got)
			require.Nil(t, got.MinBlockCommitDurationMs)
			_, ok := nibiru.ConsensusKeeper.CommitWait()
			require.False(t, ok)

			// The first refresh after the upgrade writes nothing new.
			require.NoError(t, nibiru.ConsensusKeeper.Refresh(ctx))
			require.Equal(t, pre, *nibiru.GetConsensusParams(ctx))

			// Running the handler again creates nothing.
			runConsensusUpgrade2_23(t, nibiru, ctx)
			var count int
			nibiru.WasmKeeper.IterateContractsByCreator(ctx, consensuskeeper.CreatorAddr(),
				func(sdk.AccAddress) bool { count++; return false })
			require.Equal(t, 1, count)
		})
	}
}

// TestUpgrade2_23_0FailureContinues checks the failure path: with stored
// params that cannot seed the contract (no validator params), the handler
// reports an upgrade_failure event, keeps the migrations, creates no
// contract, and does not halt.
func TestUpgrade2_23_0FailureContinues(t *testing.T) {
	broken := preUpgradeParams()
	broken.Validator = nil
	nibiru, ctx := preUpgradeChain(t, "localnet", &broken)
	events := runConsensusUpgrade2_23(t, nibiru, ctx)

	var failed bool
	for _, ev := range events {
		if ev.Type == "upgrade_failure" {
			failed = true
		}
	}
	require.True(t, failed, "expected an upgrade_failure event")
	_, found := nibiru.ConsensusKeeper.ContractAddr(ctx)
	require.False(t, found)
	versions := nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.Equal(t, uint64(1), versions[consensustypes.ModuleName])
}

// TestUpgrade2_23_0KeepsConsensusStore checks that moving the x/consensus
// store from the Cosmos SDK module to the Nibiru module loses nothing at the
// upgrade: params written before the upgrade read back identical after the
// upgrade, the following EndBlock and Commit, and the committed store list
// still includes the "consensus" store. The app mounts that store itself now;
// a missing mount would drop it from the commit info and split the app hash.
func TestUpgrade2_23_0KeepsConsensusStore(t *testing.T) {
	pre := preUpgradeParams()
	nibiru, ctx := preUpgradeChain(t, "localnet", &pre)
	require.True(t, nibiru.ConsensusKeeper.Has(ctx))

	runConsensusUpgrade2_23(t, nibiru, ctx)
	require.Equal(t, pre, *nibiru.GetConsensusParams(ctx))

	res := nibiru.EndBlock(abci.RequestEndBlock{Height: ctx.BlockHeight()})
	require.Equal(t, pre, *res.ConsensusParamUpdates)
	commitID := nibiru.Commit()
	require.NotEmpty(t, commitID.Data)

	// The committed state, read through baseapp's param store and the gRPC
	// query, holds the same params.
	committed := nibiru.NewContext(true, cmtproto.Header{
		Height: nibiru.LastBlockHeight(), Time: ctx.BlockTime(),
	})
	require.Equal(t, pre, *nibiru.GetConsensusParams(committed))
	resp, err := nibiru.ConsensusKeeper.Params(
		sdk.WrapSDKContext(committed), &consensustypes.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, pre, *resp.Params)

	cms, ok := nibiru.CommitMultiStore().(*rootmulti.Store)
	require.True(t, ok)
	info, err := cms.GetCommitInfo(nibiru.LastBlockHeight())
	require.NoError(t, err)
	var names []string
	for _, store := range info.StoreInfos {
		names = append(names, store.Name)
	}
	require.Contains(t, names, consensustypes.StoreKey)
	require.Equal(t, 1, countOf(names, consensustypes.StoreKey))
	t.Logf("committed stores: %v", names)
}

// countOf returns how often want occurs in names.
func countOf(names []string, want string) int {
	var n int
	for _, name := range names {
		if name == want {
			n++
		}
	}
	return n
}

const holderSetFixtureSHA256 = "0a18618d6d1dcf9ef0cf4a92252abf879b96a22066b1e7e57e2efd1c958204c8"

func holderSetBytecode(t *testing.T) ([]byte, []byte) {
	t.Helper()
	compressed, err := os.ReadFile("testdata/v2.23/usdc_holder_set.wasm.gz")
	require.NoError(t, err)
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	bytecode, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	checksum := sha256.Sum256(bytecode)
	require.Equal(t, holderSetFixtureSHA256, hex.EncodeToString(checksum[:]))
	return bytecode, checksum[:]
}

func holderSetAddress(n int) string { return fmt.Sprintf("0x%040x", n) }

func deployHolderSet(t *testing.T, deps evmtest.TestDeps, creator sdk.AccAddress, initial []string, salt []byte) (sdk.AccAddress, []byte) {
	t.Helper()
	bytecode, checksum := holderSetBytecode(t)
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	codeID, gotChecksum, err := keeper.Create(deps.Ctx(), creator, bytecode, &wasmtypes.AllowEverybody)
	require.NoError(t, err)
	require.Equal(t, checksum, gotChecksum)
	msg, err := json.Marshal(map[string]any{"admin": creator.String(), "addresses": initial})
	require.NoError(t, err)
	contract, _, err := keeper.Instantiate2(deps.Ctx(), codeID, creator, nil, msg,
		"temporary v2.23 holder set", sdk.Coins{}, salt, false)
	require.NoError(t, err)
	require.Equal(t, wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, nil), contract)
	return contract, checksum
}

// TestUSDCHolderSetKeeper executes the gzipped fixture through the real Wasm
// keeper. It models predeployment, batched SQL imports, and a last-minute
// operator update before the migration reads every page. No balances are stored.
func TestUSDCHolderSetKeeper(t *testing.T) {
	for _, chainID := range []string{"localnet", "nibiru-testnet-3", "cataclysm-1"} {
		t.Run(chainID, func(t *testing.T) {
			testUSDCHolderSetKeeper(t, chainID)
		})
	}
}

func testUSDCHolderSetKeeper(t *testing.T, chainID string) {
	t.Helper()
	deps := evmtest.NewTestDeps()
	deps.SetCtx(deps.Ctx().WithChainID(chainID).WithGasMeter(sdk.NewInfiniteGasMeter()))
	creator := sdk.AccAddress(bytes.Repeat([]byte{17}, 20))
	deps.App.SudoKeeper.Sudoers.Set(deps.Ctx(), sudo.Sudoers{Root: creator.String()})
	contract, checksum := deployHolderSet(t, deps, creator, []string{}, []byte("usdc-holders-v1"))
	empty, err := upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, checksum)
	require.NoError(t, err)
	require.Empty(t, empty)
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	update := func(sender sdk.AccAddress, add, remove []string) error {
		if add == nil {
			add = []string{}
		}
		if remove == nil {
			remove = []string{}
		}
		msg, err := json.Marshal(map[string]any{"update_addresses": map[string]any{"add": add, "remove": remove}})
		require.NoError(t, err)
		// Ordinary message execution commits only successful contract writes.
		ctx, commit := deps.Ctx().CacheContext()
		_, err = keeper.Execute(ctx, contract, sender, msg, sdk.Coins{})
		if err == nil {
			commit()
		}
		return err
	}
	for start := 1; start <= 1007; start += 500 {
		var add []string
		for n := start; n < start+500 && n <= 1007; n++ {
			add = append(add, strings.ToUpper(holderSetAddress(n)))
		}
		require.NoError(t, update(creator, add, nil))
	}
	require.ErrorContains(t, update(sdk.AccAddress(bytes.Repeat([]byte{18}, 20)), []string{holderSetAddress(2000)}, nil), "unauthorized")
	require.Error(t, update(creator, []string{holderSetAddress(2000), "invalid"}, []string{holderSetAddress(1)}))
	// The final operator push changes the candidate set after initial loading.
	require.NoError(t, update(creator,
		[]string{holderSetAddress(1008), holderSetAddress(1008)}, []string{holderSetAddress(1)}))
	got, err := upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, checksum)
	require.NoError(t, err)
	var want []gethcommon.Address
	for n := 2; n <= 1008; n++ {
		want = append(want, gethcommon.HexToAddress(holderSetAddress(n)))
	}
	require.Equal(t, want, got)
	// Neither a wrong artifact nor a missing predeployment can yield candidates.
	wrongChecksum := bytes.Repeat([]byte{42}, 32)
	got, err = upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, wrongChecksum)
	require.ErrorContains(t, err, "unexpected bytecode")
	require.Nil(t, got)
	missing := sdk.AccAddress(bytes.Repeat([]byte{99}, 32))
	before := deps.App.WasmKeeper.PeekAutoIncrementID(deps.Ctx(), wasmtypes.KeySequenceCodeID)
	got, err = upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, missing, checksum)
	require.ErrorContains(t, err, "not deployed")
	require.Nil(t, got)
	require.Equal(t, before, deps.App.WasmKeeper.PeekAutoIncrementID(deps.Ctx(), wasmtypes.KeySequenceCodeID))
}

// TestUSDCHolderSetInstantiate2Parity proves address parity with the same real
// artifact/creator/salt and different chains, initial state, and code IDs.
func TestUSDCHolderSetInstantiate2Parity(t *testing.T) {
	var shared sdk.AccAddress
	creator := sdk.AccAddress(bytes.Repeat([]byte{17}, 20))
	for index, chainID := range []string{"localnet", "nibiru-testnet-3", "cataclysm-1"} {
		t.Run(chainID, func(t *testing.T) {
			deps := evmtest.NewTestDeps()
			deps.SetCtx(deps.Ctx().WithChainID(chainID).WithGasMeter(sdk.NewInfiniteGasMeter()))
			// Provision deployment authority explicitly, including on mainnet.
			deps.App.SudoKeeper.Sudoers.Set(deps.Ctx(), sudo.Sudoers{Root: creator.String()})
			bytecode, _ := holderSetBytecode(t)
			keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
			for n := 0; n < index; n++ {
				_, _, err := keeper.Create(deps.Ctx(), creator, bytecode, &wasmtypes.AllowEverybody)
				require.NoError(t, err)
			}
			initial := []string{holderSetAddress(index + 1)}
			contract, checksum := deployHolderSet(t, deps, creator, initial, []byte("usdc-holders-v1"))
			if shared == nil {
				shared = contract
			}
			require.Equal(t, shared, contract)
			got, err := upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, checksum)
			require.NoError(t, err)
			require.Equal(t, []gethcommon.Address{gethcommon.HexToAddress(initial[0])}, got)
		})
	}
}

// A fixture rebuild must also update the production profile. Tests using their
// own fixture profile otherwise could pass while the shipped handler looked at
// an address derived from obsolete bytecode.
func TestUpgrade2_23_0RegistryProfileMatchesFixture(t *testing.T) {
	_, checksum := holderSetBytecode(t)
	profile := upgrades.DefaultUSDCMigrationConfigV223()
	require.Equal(t, checksum, profile.RegistryChecksum)
	creator := sdk.MustAccAddressFromBech32(upgrades.USDCHolderSetCreatorV223)
	require.Equal(t, wasmkeeper.BuildContractAddressPredictable(checksum, creator,
		[]byte(upgrades.USDCHolderSetSaltV223), nil), profile.Registry)
}

func reportEvent(kind string, attrs ...string) abci.Event {
	attributes := []sdk.Attribute{sdk.NewAttribute("upgrade", "v2.23.0")}
	for i := 0; i < len(attrs); i += 2 {
		attributes = append(attributes, sdk.NewAttribute(attrs[i], attrs[i+1]))
	}
	return abci.Event(sdk.NewEvent(kind, attributes...))
}

func reportTyped(t *testing.T, event interface {
	ProtoMessage()
	Reset()
	String() string
},
) abci.Event {
	t.Helper()
	got, err := sdk.TypedEventToEvent(event)
	require.NoError(t, err)
	return abci.Event(got)
}

func reportFixture(t *testing.T) []abci.Event {
	t.Helper()
	token, safe := gethcommon.HexToAddress("0x100"), gethcommon.HexToAddress("0x200")
	bankUser, err := sdk.Bech32ifyAddressBytes("nibi", bytes.Repeat([]byte{3}, 20))
	require.NoError(t, err)
	bankCore, err := sdk.Bech32ifyAddressBytes("nibi", bytes.Repeat([]byte{4}, 32))
	require.NoError(t, err)
	eoa, sai := gethcommon.HexToAddress("0x300"), gethcommon.HexToAddress("0x400")
	denom := "erc20/" + token.Hex()
	frame := func(status string) abci.Event {
		return reportEvent("usdc_migration", "status", status, "height", "49130384", "registry", "registry", "token", token.Hex(), "denom", denom, "safe", safe.Hex())
	}
	conversion := func(sender string, amount int64) abci.Event {
		return reportTyped(t, &evm.EventConvertCoinToEvm{
			Sender: sender, Erc20ContractAddress: token.Hex(), ToEthAddr: safe.Hex(),
			BankCoin: sdk.NewInt64Coin(denom, amount),
			EvmLogs:  []evm.LogLite{evmtest.LogLiteEventErc20Transfer(token, evm.EVM_MODULE_ADDRESS, safe, big.NewInt(amount))},
		})
	}
	transfer := func(sender gethcommon.Address, amount int64) abci.Event {
		lite := evmtest.LogLiteEventErc20Transfer(token, sender, safe, big.NewInt(amount))
		// No signed tx exists for upgrade calls. Different movements can share
		// zero tx hashes and log indices, so those are not deduplication keys.
		return reportTyped(t, &evm.EventTxLog{Logs: []evm.Log{{Address: lite.Address, Topics: lite.Topics, Data: lite.Data}}})
	}
	account := func(representation, source, amount, disposition, status string) abci.Event {
		return reportEvent("usdc_migration_account", "representation", representation, "source", source,
			"destination", safe.Hex(), "amount", amount, "disposition", disposition, "status", status)
	}
	return []abci.Event{
		transfer(eoa, 999), // Other movement in the same block, outside the frame.
		frame("started"),
		conversion(bankUser, 100), account("bank", bankUser, "100", "repayment", "collected"),
		conversion(bankCore, 200), account("bank", bankCore, "200", "custody_recovery", "collected"),
		transfer(eoa, 300), account("erc20", eoa.Hex(), "300", "repayment", "collected"),
		transfer(sai, 400), account("erc20", sai.Hex(), "400", "custody_recovery", "collected"),
		account("erc20", gethcommon.HexToAddress("0x500").Hex(), "55", "repayment", "skipped"),
		account("erc20", gethcommon.HexToAddress("0x600").Hex(), "22", "repayment", "failed"),
		account("erc20", gethcommon.HexToAddress("0x700").Hex(), "0", "repayment", "zero"),
		frame("completed"),
		transfer(eoa, 888),
	}
}

func TestUSDCMigrationReportFromBlockEvents(t *testing.T) {
	// JSON round-trip the block events. The reader receives neither keepers,
	// the original balances, nor a separate migration ledger/config object.
	bz, err := json.Marshal(reportFixture(t))
	require.NoError(t, err)
	var events []abci.Event
	require.NoError(t, json.Unmarshal(bz, &events))
	report, err := upgrades.ReconstructUSDCMigration(events)
	require.NoError(t, err)
	require.Equal(t, int64(49130384), report.Height)
	require.Equal(t, "completed", report.Status)
	require.Equal(t, "1000", report.Collected)
	require.Equal(t, "400", report.Repayment)
	require.Equal(t, "600", report.CustodyRecovery)
	require.Len(t, report.Accounts, 7)
	wantCore, err := sdk.Bech32ifyAddressBytes("nibi", bytes.Repeat([]byte{4}, 32))
	require.NoError(t, err)
	require.Equal(t, wantCore, report.Accounts[1].Source)
	readable, err := json.MarshalIndent(report, "", "  ")
	require.NoError(t, err)
	t.Log(string(readable))
}

func TestUSDCMigrationReportRejectsIncompleteOrDuplicateAccounting(t *testing.T) {
	for name, change := range map[string]func([]abci.Event) []abci.Event{
		"no closing frame": func(events []abci.Event) []abci.Event { return events[:13] },
		"collected record without movement": func(events []abci.Event) []abci.Event {
			return append(events[:2:2], events[3:]...)
		},
		"movement without account": func(events []abci.Event) []abci.Event {
			return append(events[:3:3], events[4:]...)
		},
		"duplicate collected record": func(events []abci.Event) []abci.Event {
			return append(events[:4:4], append([]abci.Event{events[3]}, events[4:]...)...)
		},
		"duplicate backing movement": func(events []abci.Event) []abci.Event {
			return append(events[:3:3], append([]abci.Event{events[2]}, events[3:]...)...)
		},
		"account after closing frame": func(events []abci.Event) []abci.Event {
			return append(events, events[3])
		},
	} {
		t.Run(name, func(t *testing.T) {
			report, err := upgrades.ReconstructUSDCMigration(change(reportFixture(t)))
			require.Error(t, err)
			require.Nil(t, report)
		})
	}
}

func TestUSDCMigrationReportSkippedOrFailedPreflight(t *testing.T) {
	for _, status := range []string{"skipped", "failed"} {
		t.Run(status, func(t *testing.T) {
			event := reportEvent("usdc_migration", "status", status, "height", "5", "reason", "preflight")
			report, err := upgrades.ReconstructUSDCMigration([]abci.Event{event})
			require.NoError(t, err)
			require.Equal(t, status, report.Status)
			require.Equal(t, "0", report.Collected)
			require.Empty(t, report.Accounts)
		})
	}
}
