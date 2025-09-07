// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule_test

import (
	"math/big"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/MakeNowJust/heredoc/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
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
	suite.Run(t, new(Suite))
}

// TestExportInitGenesis
// - creates evm state with erc20 contract, sends tokens to user A and B
// - creates fungible token mapping for the erc20 and send some to C
// - exports and imports genesis
// - verifies that contracts are in place and user balances match
// - verifies that fungible token is in place and the balance is correct
func (s *Suite) TestExportInitGenesis() {
	var (
		deps = evmtest.NewTestDeps()
		// Keep the sender as a variable since "deps" will be reset after we export the genesis
		depsSender    = deps.Sender
		erc20Contract = embeds.SmartContract_TestERC20
		toUserA       = gethcommon.HexToAddress("0xAE8A5F44A9b55Ae6D2c9C228253E8fAfb837d2F2")
		toUserB       = gethcommon.HexToAddress("0xf893292542F2578F1004e62fd723901ddE5EC5Cf")
		toUserC       = gethcommon.HexToAddress("0xe90f75496E744b92B52535bB05a29123D0D94D49")
		amountToSendA = big.NewInt(1500)
		amountToSendB = big.NewInt(350)
		amountToSendC = big.NewInt(200)

		// [evm.FunToken] mapping used throughout the test
		ft evm.FunToken

		totalSupply *big.Int
		erc20Addr   gethcommon.Address
	)

	// Create re-usable function to perform the assertion later afer exporting
	// the genesis.
	assertBalsOfAB := func(deps evmtest.TestDeps, evmObj *vm.EVM) {
		evmtest.FunTokenBalanceAssert{
			Account:      toUserA,
			FunToken:     ft,
			BalanceBank:  big.NewInt(0),
			BalanceERC20: amountToSendA,
		}.Assert(s.T(), deps, evmObj)
		evmtest.FunTokenBalanceAssert{
			Account:      toUserB,
			FunToken:     ft,
			BalanceBank:  big.NewInt(0),
			BalanceERC20: amountToSendB,
		}.Assert(s.T(), deps, evmObj)
	}

	assertBalsAfterConvert := func(deps evmtest.TestDeps) {
		s.T().Log("Assert balances for users A, B, C, the sender and the EVM")

		// NOTE: You need a fresh evmObj here
		evmObj, _ := deps.NewEVM()

		assertBalsOfAB(deps, evmObj)

		evmtest.FunTokenBalanceAssert{
			Account:      toUserC,
			FunToken:     ft,
			BalanceBank:  amountToSendC,
			BalanceERC20: big.NewInt(0),
			Description:  "C receives amountToSendC as Bank Coins",
		}.Assert(s.T(), deps, evmObj)
		evmtest.FunTokenBalanceAssert{
			Account:      evm.EVM_MODULE_ADDRESS,
			FunToken:     ft,
			BalanceBank:  big.NewInt(0),
			BalanceERC20: amountToSendC,
			Description:  "EVM holds ERC20 tokens to back the newly minted Bank Coins",
		}.Assert(s.T(), deps, evmObj)
		evmtest.FunTokenBalanceAssert{
			Account:     depsSender.EthAddr,
			FunToken:    ft,
			BalanceBank: big.NewInt(0),
			BalanceERC20: new(big.Int).Sub(
				totalSupply,
				big.NewInt(amountToSendA.Int64()+amountToSendB.Int64()+amountToSendC.Int64()),
			),
			Description: "Sender should lose amountToSendC",
		}.Assert(s.T(), deps, evmObj)
	}

	{
		// Create ERC-20 contract
		deployResp, err := evmtest.DeployContract(&deps, erc20Contract)
		s.Require().NoError(err)
		erc20Addr = deployResp.ContractAddr

		evmObj, sdb := deps.NewEVM()
		totalSupply, err = deps.EvmKeeper.ERC20().LoadERC20BigInt(
			deps.Ctx, evmObj, erc20Contract.ABI, erc20Addr, "totalSupply",
		)
		s.Require().NoError(err)
		s.Require().Equal("1000000"+strings.Repeat("0", 18), totalSupply.String())
		s.T().Log(heredoc.Docf(
			`Deployed ERC20 with total supply held by deps.Sender
totalSupply: %s
deps.Sender.EthAddr: %s
erc20Addr: %s
amountToSendA: %s
amountToSendB: %s
amountToSendC: %s`,
			totalSupply,
			deps.Sender.EthAddr.Hex(),
			erc20Addr,
			amountToSendA,
			amountToSendB,
			amountToSendC,
		))

		s.T().Log("Transfer ERC-20 tokens to [userA, userB]")
		_, _, err = deps.EvmKeeper.ERC20().Transfer(erc20Addr, deps.Sender.EthAddr, toUserA, amountToSendA, deps.Ctx, evmObj)
		s.Require().NoError(err)

		s.T().Log("Transfer ERC-20 tokens to user B")
		_, _, err = deps.EvmKeeper.ERC20().Transfer(erc20Addr, deps.Sender.EthAddr, toUserB, amountToSendB, deps.Ctx, evmObj)
		s.Require().NoError(err)
		s.NoError(
			sdb.Commit(),
		)
	}

	s.T().Logf("Create FunToken from the er20 %s", erc20Addr)
	{
		// There's a fee for "evm.MsgCreateFunToken". The sender account needs
		// needs funds for that.
		err := testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			sdk.NewCoins(
				sdk.NewCoin(evm.EVMBankDenom, deps.EvmKeeper.GetParams(deps.Ctx).CreateFuntokenFee),
			),
		)
		s.Require().NoError(err)

		createFuntokenResp, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromErc20: &eth.EIP55Addr{Address: erc20Addr},
				Sender:    deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().NoError(err)
		ft = createFuntokenResp.FuntokenMapping
	}

	s.T().Log("balance assertions after transfer to [userA, userB]")

	{
		evmObj, _ := deps.NewEVM()
		assertBalsOfAB(deps, evmObj)
		evmtest.FunTokenBalanceAssert{
			Account:      toUserC,
			FunToken:     ft,
			BalanceBank:  big.NewInt(0),
			BalanceERC20: big.NewInt(0),
			Description:  "C should not have any funds yet",
		}.Assert(s.T(), deps, evmObj)
		evmtest.FunTokenBalanceAssert{
			Account:     deps.Sender.EthAddr,
			FunToken:    ft,
			BalanceBank: big.NewInt(0),
			BalanceERC20: new(big.Int).Sub(
				totalSupply,
				big.NewInt(amountToSendA.Int64()+amountToSendB.Int64()),
			),
			Description: "Sender has total supply minus A and B",
		}.Assert(s.T(), deps, evmObj)
		evmtest.FunTokenBalanceAssert{
			Account:      evm.EVM_MODULE_ADDRESS,
			FunToken:     ft,
			BalanceBank:  big.NewInt(0),
			BalanceERC20: big.NewInt(0),
			Description:  "inactive so far",
		}.Assert(s.T(), deps, evmObj)
	}

	s.T().Log("Send fungible token coins from bank to evm")
	{
		_, err := deps.EvmKeeper.ConvertEvmToCoin(
			deps.Ctx,
			&evm.MsgConvertEvmToCoin{
				Sender:    deps.Sender.NibiruAddr.String(),
				Erc20Addr: ft.Erc20Addr,
				Amount:    sdkmath.NewIntFromBigInt(amountToSendC),
				ToAddr:    toUserC.Hex(),
			},
		)
		s.Require().NoError(err)
	}

	assertBalsAfterConvert(deps)

	s.T().Log("Export genesis")

	evmGenesisState := evmmodule.ExportGenesis(deps.Ctx, deps.EvmKeeper, deps.App.AccountKeeper)
	authGenesisState := deps.App.AccountKeeper.ExportGenesis(deps.Ctx)
	bankGensisState := deps.App.BankKeeper.ExportGenesis(deps.Ctx)

	s.T().Log("Init genesis from the exported state for modules [auth, bank, evm]")
	{
		deps = evmtest.NewTestDeps()
		deps.App.AccountKeeper.InitGenesis(deps.Ctx, *authGenesisState)
		deps.App.BankKeeper.InitGenesis(deps.Ctx, bankGensisState)
		evmmodule.InitGenesis(deps.Ctx, deps.EvmKeeper, deps.App.AccountKeeper, *evmGenesisState)

		s.T().Log("Assert balances for users A, B, C, the sender and the EVM")
		assertBalsAfterConvert(deps)

		s.T().Log("Check that fungible token mapping is in place")
		iter := deps.EvmKeeper.FunTokens.Indexes.BankDenom.ExactMatch(deps.Ctx, ft.BankDenom)
		funTokens := deps.EvmKeeper.FunTokens.Collect(deps.Ctx, iter)
		s.Require().Len(funTokens, 1)
		s.Equal(ft.Erc20Addr.Hex(), funTokens[0].Erc20Addr.String())
		s.Equal(ft.BankDenom, funTokens[0].BankDenom)
		s.False(funTokens[0].IsMadeFromCoin)
	}
}
