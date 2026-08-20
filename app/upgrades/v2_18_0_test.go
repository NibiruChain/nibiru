package upgrades_test

import (
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

func TestV218IncidentRecoveryProductionConfig(t *testing.T) {
	require.Equal(t, "cataclysm-1", upgrades.IncidentRecoveryChainID_v2_18)
	require.Equal(t, "nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v", upgrades.IncidentRecoveryCW3_v2_18)
	require.Equal(t, "0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2", upgrades.IncidentRecoverySafe_v2_18)
	require.Equal(t, []string{
		"0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B",
		"0x3E0D2aCd91F72FeD140d2b9FF5C7005D60b87972",
		"0x761ff39a47ab53ba07f66971015d656c37391517",
	}, upgrades.IncidentRecoverySources_v2_18)
	require.Equal(t, []string{
		"0x0829F361A05D993d5CEb035cA6DF3446b060970b",
		"0xcdA5b77E2E2268D9E09c874c1b9A4c3F07b37555",
		"0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b",
		"0xf4e097E36d2064E2bDCA96e60439f3A369522003",
		"0x84f682626302EA7BCA2A7c338b84863292131319",
		"0x43F2376D5D03553aE72F4A8093bbe9de4336EB08",
	}, upgrades.IncidentRecoveryERC20s_v2_18)

	require.Equal(t, "nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8", eth.EthAddrToNibiruAddr(gethcommon.HexToAddress(upgrades.IncidentRecoverySources_v2_18[0])).String())
	require.Equal(t, "nibi18cxj4nv37uh769qd9w0lt3cqt4sts7tjnchmmw", eth.EthAddrToNibiruAddr(gethcommon.HexToAddress(upgrades.IncidentRecoverySources_v2_18[1])).String())
}

func TestV218IncidentRecovery(t *testing.T) {
	t.Run("recovers Bank coins and all configured ERC20 balances", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		deps.SetCtx(deps.Ctx().WithChainID(appconst.SDK_CHAIN_ID_MAINNET))

		eoaSource := gethcommon.HexToAddress("0x1000000000000000000000000000000000000001")
		contractSource := gethcommon.HexToAddress("0x2000000000000000000000000000000000000002")
		unrelated := gethcommon.HexToAddress("0x3000000000000000000000000000000000000003")
		safe := gethcommon.HexToAddress("0x4000000000000000000000000000000000000004")
		cw3 := testutil.NewAccAddress()

		contractSDB := deps.NewStateDB()
		contractSDB.SetCode(contractSource, []byte{0x00})
		deps.Commit()
		require.True(t, deps.EvmKeeper.GetAccount(deps.Ctx(), contractSource).IsContract())

		tokens := deployRecoveryTokensV218(t, &deps, []gethcommon.Address{eoaSource, contractSource, unrelated}, 6)
		for idx, token := range tokens {
			requireERC20BalanceV218(t, &deps, token, eoaSource, big.NewInt(int64(100+idx)))
			requireERC20BalanceV218(t, &deps, token, contractSource, big.NewInt(int64(200+idx)))
		}
		bankEOA := sdk.NewCoins(
			sdk.NewInt64Coin(appconst.DENOM_UNIBI, 11),
			sdk.NewInt64Coin("factory/recovery/one", 12),
		)
		bankContract := sdk.NewCoins(
			sdk.NewInt64Coin(appconst.DENOM_UNIBI, 21),
			sdk.NewInt64Coin("ibc/recovery-two", 22),
		)
		bankUnrelated := sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 31))
		require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), eth.EthAddrToNibiruAddr(eoaSource), bankEOA))
		require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), eth.EthAddrToNibiruAddr(contractSource), bankContract))
		require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), eth.EthAddrToNibiruAddr(unrelated), bankUnrelated))
		eoaNonceBefore := deps.EvmKeeper.GetAccNonce(deps.Ctx(), eoaSource)
		contractNonceBefore := deps.EvmKeeper.GetAccNonce(deps.Ctx(), contractSource)

		cfg := upgrades.RecoveryConfigV218{
			ChainID: appconst.SDK_CHAIN_ID_MAINNET,
			Sources: []gethcommon.Address{eoaSource, contractSource},
			ERC20s:  tokens,
			BankTo:  cw3,
			ERC20To: safe,
		}
		runRecoveryUpgradeV218(t, &deps, cfg, false)

		require.Empty(t, deps.App.BankKeeper.GetAllBalances(deps.Ctx(), eth.EthAddrToNibiruAddr(eoaSource)))
		require.Empty(t, deps.App.BankKeeper.GetAllBalances(deps.Ctx(), eth.EthAddrToNibiruAddr(contractSource)))
		require.True(t, bankEOA.Add(bankContract...).IsEqual(deps.App.BankKeeper.GetAllBalances(deps.Ctx(), cw3)))
		require.True(t, bankUnrelated.IsEqual(deps.App.BankKeeper.GetAllBalances(deps.Ctx(), eth.EthAddrToNibiruAddr(unrelated))))
		require.Equal(t, eoaNonceBefore, deps.EvmKeeper.GetAccNonce(deps.Ctx(), eoaSource))
		require.Equal(t, contractNonceBefore, deps.EvmKeeper.GetAccNonce(deps.Ctx(), contractSource))

		for idx, token := range tokens {
			amountEOA := big.NewInt(int64(100 + idx))
			amountContract := big.NewInt(int64(200 + idx))
			amountUnrelated := big.NewInt(int64(300 + idx))
			totalSupply := new(big.Int).Add(amountEOA, amountContract)
			totalSupply.Add(totalSupply, amountUnrelated)
			requireERC20BalanceV218(t, &deps, token, eoaSource, new(big.Int))
			requireERC20BalanceV218(t, &deps, token, contractSource, new(big.Int))
			requireERC20BalanceV218(t, &deps, token, safe, new(big.Int).Add(amountEOA, amountContract))
			requireERC20BalanceV218(t, &deps, token, unrelated, amountUnrelated)
			requireERC20SupplyV218(t, &deps, token, totalSupply)
		}

		var recoveryEvents int
		for _, event := range deps.Ctx().EventManager().Events() {
			if event.Type == "incident_fund_recovery" {
				recoveryEvents++
			}
		}
		require.Equal(t, 14, recoveryEvents, "12 ERC20 recoveries and 2 Bank recoveries")
	})

	t.Run("rolls back every change when one token transfer runs out of gas", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		deps.SetCtx(deps.Ctx().WithChainID(appconst.SDK_CHAIN_ID_MAINNET))

		source := deps.Sender.EthAddr
		safe := gethcommon.HexToAddress("0x6000000000000000000000000000000000000006")
		cw3 := testutil.NewAccAddress()
		validToken := deployRecoveryTokensV218(t, &deps, []gethcommon.Address{source}, 1)[0]
		requireERC20BalanceV218(t, &deps, validToken, source, big.NewInt(100))
		failingToken, err := evmtest.DeployContract(
			&deps,
			embeds.SmartContract_TestERC20MaliciousTransfer,
			"Gas Intensive Token",
			"GAS",
			uint8(18),
		)
		require.NoError(t, err)
		failingBalance := new(big.Int).Mul(big.NewInt(1_000_000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		requireERC20BalanceV218(t, &deps, failingToken.ContractAddr, source, failingBalance)
		bankFunds := sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 41))
		require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), eth.EthAddrToNibiruAddr(source), bankFunds))
		bankBefore := deps.App.BankKeeper.GetAllBalances(deps.Ctx(), eth.EthAddrToNibiruAddr(source))

		cfg := upgrades.RecoveryConfigV218{
			ChainID: appconst.SDK_CHAIN_ID_MAINNET,
			Sources: []gethcommon.Address{source},
			ERC20s:  []gethcommon.Address{validToken, failingToken.ContractAddr},
			BankTo:  cw3,
			ERC20To: safe,
		}
		runRecoveryUpgradeV218(t, &deps, cfg, true)

		requireERC20BalanceV218(t, &deps, validToken, source, big.NewInt(100))
		requireERC20BalanceV218(t, &deps, validToken, safe, new(big.Int))
		requireERC20BalanceV218(t, &deps, failingToken.ContractAddr, source, failingBalance)
		require.True(t, bankBefore.IsEqual(deps.App.BankKeeper.GetAllBalances(deps.Ctx(), eth.EthAddrToNibiruAddr(source))))
		require.Empty(t, deps.App.BankKeeper.GetAllBalances(deps.Ctx(), cw3))
	})

	t.Run("runs migrations but skips recovery outside mainnet", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		deps.SetCtx(deps.Ctx().WithChainID("nibiru-testnet-2"))

		source := gethcommon.HexToAddress("0x8000000000000000000000000000000000000008")
		safe := gethcommon.HexToAddress("0x9000000000000000000000000000000000000009")
		cw3 := testutil.NewAccAddress()
		token := deployRecoveryTokensV218(t, &deps, []gethcommon.Address{source}, 1)[0]
		bankFunds := sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 51))
		require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), eth.EthAddrToNibiruAddr(source), bankFunds))

		cfg := upgrades.RecoveryConfigV218{
			ChainID: appconst.SDK_CHAIN_ID_MAINNET,
			Sources: []gethcommon.Address{source},
			ERC20s:  []gethcommon.Address{token},
			BankTo:  cw3,
			ERC20To: safe,
		}
		// A successful real handler invocation proves RunMigrations completed.
		// The chain ID mismatch must bypass only the forced recovery portion.
		runRecoveryUpgradeV218(t, &deps, cfg, false)

		requireERC20BalanceV218(t, &deps, token, source, big.NewInt(100))
		requireERC20BalanceV218(t, &deps, token, safe, new(big.Int))
		require.True(t, bankFunds.IsEqual(deps.App.BankKeeper.GetAllBalances(deps.Ctx(), eth.EthAddrToNibiruAddr(source))))
		require.Empty(t, deps.App.BankKeeper.GetAllBalances(deps.Ctx(), cw3))
	})
}

func deployRecoveryTokensV218(
	t *testing.T,
	deps *evmtest.TestDeps,
	recipients []gethcommon.Address,
	tokenCount int,
) []gethcommon.Address {
	t.Helper()
	require.NoError(t, testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 1)),
	))
	tokens := make([]gethcommon.Address, tokenCount)
	for idx := range tokens {
		deployed, err := evmtest.DeployContract(
			deps,
			embeds.SmartContract_ERC20MinterWithMetadataUpdates,
			"Recovery Test Token",
			"RTT",
			uint8(18),
		)
		require.NoError(t, err)
		tokens[idx] = deployed.ContractAddr

		evmObj, sdb := deps.NewEVM()
		for recipientIdx, recipient := range recipients {
			amount := big.NewInt(int64((recipientIdx+1)*100 + idx))
			resp, err := deps.EvmKeeper.ERC20().Mint(
				tokens[idx], deps.Sender.EthAddr, recipient, amount, deps.Ctx(), evmObj,
			)
			require.NoError(t, err)
			require.False(t, resp.Failed())
		}
		sdb.Commit()
	}
	return tokens
}

func runRecoveryUpgradeV218(
	t *testing.T,
	deps *evmtest.TestDeps,
	cfg upgrades.RecoveryConfigV218,
	wantError bool,
) {
	t.Helper()
	upgrade := upgrades.Upgrade{
		UpgradeName: "v2.18.0-test",
		Handler: upgrades.Handler_v2_18{
			Recovery: &cfg,
		},
	}
	err := deps.RunUpgrade(upgrade)
	if wantError {
		require.ErrorContains(t, err, "transfer VM error")
		return
	}
	require.NoError(t, err)
}

func requireERC20BalanceV218(
	t *testing.T,
	deps *evmtest.TestDeps,
	token gethcommon.Address,
	account gethcommon.Address,
	want *big.Int,
) {
	t.Helper()
	evmObj, _ := deps.NewEVM()
	got, err := deps.EvmKeeper.ERC20().BalanceOf(token, account, deps.Ctx(), evmObj)
	require.NoError(t, err)
	require.Zero(t, got.Cmp(want), "token %s account %s: got %s want %s", token, account, got, want)
}

func requireERC20SupplyV218(
	t *testing.T,
	deps *evmtest.TestDeps,
	token gethcommon.Address,
	want *big.Int,
) {
	t.Helper()
	evmObj, _ := deps.NewEVM()
	got, err := deps.EvmKeeper.ERC20().TotalSupply(token, deps.Ctx(), evmObj)
	require.NoError(t, err)
	require.Zero(t, got.Cmp(want), "token %s: got supply %s want %s", token, got, want)
}
