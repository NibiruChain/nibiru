// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmmodule"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type Suite struct {
	suite.Suite
}

// TestKeeperSuite: Runs all the tests in the suite.
func TestKeeperSuite(t *testing.T) {
	s := new(Suite)
	suite.Run(t, s)
}

// TestExportInitGenesis
// - creates evm state with erc20 contract, sends tokens to user A and B
// - creates fungible token from unibi coin and sends to user C
// - exports / imports genesis
// - verifies that contracts are in place and user balances match
// - verifies that fungible token is in place and the balance is correct
func (s *Suite) TestExportInitGenesis() {
	deps := evmtest.NewTestDeps()
	erc20Contract := embeds.SmartContract_TestERC20
	fromUser := deps.Sender.EthAddr
	toUserA := gethcommon.HexToAddress("0xAE8A5F44A9b55Ae6D2c9C228253E8fAfb837d2F2")
	toUserB := gethcommon.HexToAddress("0xf893292542F2578F1004e62fd723901ddE5EC5Cf")
	toUserC := gethcommon.HexToAddress("0xe90f75496E744b92B52535bB05a29123D0D94D49")
	amountToSendA := big.NewInt(1550)
	amountToSendB := big.NewInt(333)
	amountToSendC := big.NewInt(228)

	// Create ERC-20 contract
	deployResp, err := evmtest.DeployContract(&deps, erc20Contract)
	s.Require().NoError(err)
	erc20Addr := deployResp.ContractAddr
	totalSupply, err := deps.EvmKeeper.ERC20().LoadERC20BigInt(
		deps.Ctx, erc20Contract.ABI, erc20Addr, "totalSupply",
	)
	s.Require().NoError(err)

	// Transfer ERC-20 tokens to user A
	_, err = deps.EvmKeeper.ERC20().Transfer(erc20Addr, fromUser, toUserA, amountToSendA, deps.Ctx)
	s.Require().NoError(err)

	// Transfer ERC-20 tokens to user B
	_, err = deps.EvmKeeper.ERC20().Transfer(erc20Addr, fromUser, toUserB, amountToSendB, deps.Ctx)
	s.Require().NoError(err)

	// Create fungible token from bank coin
	funToken := evmtest.CreateFunTokenForBankCoin(&deps, "unibi", &s.Suite)
	s.Require().NoError(err)
	funTokenAddr := funToken.Erc20Addr.Addr()

	// Fund sender's wallet
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin("unibi", totalSupply.Int64()))
	err = deps.App.BankKeeper.MintCoins(deps.Ctx, evm.ModuleName, spendableCoins)
	s.Require().NoError(err)
	err = deps.App.BankKeeper.SendCoinsFromModuleToAccount(
		deps.Ctx, evm.ModuleName, deps.Sender.NibiruAddr, spendableCoins,
	)
	s.Require().NoError(err)

	// Send fungible token coins from bank to evm
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		deps.Ctx,
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.Coin{Denom: "unibi", Amount: math.NewInt(amountToSendC.Int64())},
			ToEthAddr: eth.MustNewHexAddrFromStr(toUserC.String()),
		},
	)
	s.Require().NoError(err)

	// Export genesis
	evmGenesisState := evmmodule.ExportGenesis(deps.Ctx, &deps.EvmKeeper, deps.App.AccountKeeper)
	authGenesisState := deps.App.AccountKeeper.ExportGenesis(deps.Ctx)

	// Init genesis from the exported state
	deps = evmtest.NewTestDeps()
	deps.App.AccountKeeper.InitGenesis(deps.Ctx, *authGenesisState)
	evmmodule.InitGenesis(deps.Ctx, &deps.EvmKeeper, deps.App.AccountKeeper, *evmGenesisState)

	// Verify erc20 balances for users A, B and sender
	balance, err := deps.EvmKeeper.ERC20().BalanceOf(erc20Addr, toUserA, deps.Ctx)
	s.Require().NoError(err)
	s.Require().Equal(amountToSendA, balance)

	balance, err = deps.EvmKeeper.ERC20().BalanceOf(erc20Addr, toUserB, deps.Ctx)
	s.Require().NoError(err)
	s.Require().Equal(amountToSendB, balance)

	balance, err = deps.EvmKeeper.ERC20().BalanceOf(erc20Addr, fromUser, deps.Ctx)
	s.Require().NoError(err)
	s.Require().Equal(
		new(big.Int).Sub(totalSupply, big.NewInt(amountToSendA.Int64()+amountToSendB.Int64())),
		balance,
	)

	// Check that fungible token mapping is in place
	iter := deps.EvmKeeper.FunTokens.Indexes.BankDenom.ExactMatch(deps.Ctx, "unibi")
	funTokens := deps.EvmKeeper.FunTokens.Collect(deps.Ctx, iter)
	s.Require().Len(funTokens, 1)
	s.Require().Equal(funTokenAddr.String(), funTokens[0].Erc20Addr.String())
	s.Require().Equal("unibi", funTokens[0].BankDenom)
	s.Require().True(funTokens[0].IsMadeFromCoin)

	// Check that fungible token balance of user C is correct
	balance, err = deps.EvmKeeper.ERC20().BalanceOf(funTokenAddr, toUserC, deps.Ctx)
	s.Require().NoError(err)
	s.Require().Equal(amountToSendC, balance)
}
