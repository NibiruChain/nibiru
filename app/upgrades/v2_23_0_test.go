package upgrades_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	consensustypes "github.com/NibiruChain/nibiru/v2/x/consensus/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
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
