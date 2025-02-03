// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestERC20Calls() {
	deps := evmtest.NewTestDeps()
	bankDenom := "ibc/btc"
	funtoken := evmtest.CreateFunTokenForBankCoin(deps, bankDenom, &s.Suite)
	erc20 := funtoken.Erc20Addr.Address

	s.Run("Mint tokens - Fail from non-owner", func() {
		evmObj, _ := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Mint(
			erc20, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS,
			big.NewInt(69_420), deps.Ctx, evmObj,
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	})

	s.Run("successfully mint 69420 tokens", func() {
		evmObj, stateDB := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Mint(
			erc20,                  /*erc20Addr*/
			evm.EVM_MODULE_ADDRESS, /*sender*/
			evm.EVM_MODULE_ADDRESS, /*recipient*/
			big.NewInt(69_420),     /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)
		s.Require().NoError(stateDB.Commit())

		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), "expect 69420 tokens")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(0), "expect zero tokens")
	})

	s.Run("Transfer - Not enough funds", func() {
		evmObj, _ := deps.NewEVM()
		_, _, err := deps.EvmKeeper.ERC20().Transfer(
			erc20, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS,
			big.NewInt(9_420), deps.Ctx, evmObj,
		)
		s.ErrorContains(err, "ERC20: transfer amount exceeds balance")
		// balances unchanged
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(0), "expect zero balance")
	})

	s.Run("Transfer - Success (sanity check)", func() {
		evmObj, stateDB := deps.NewEVM()
		sentAmt, _, err := deps.EvmKeeper.ERC20().Transfer(
			erc20,                  /*erc20Addr*/
			evm.EVM_MODULE_ADDRESS, /*sender*/
			deps.Sender.EthAddr,    /*recipient*/
			big.NewInt(9_420),      /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)
		s.Require().NoError(stateDB.Commit())
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(9_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(60_000), "expect nonzero balance")
		s.Require().EqualValues(big.NewInt(9_420), sentAmt)
	})

	s.Run("Burn tokens - Allowed as non-owner", func() {
		evmObj, stateDB := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Burn(
			erc20,               /*erc20Addr*/
			deps.Sender.EthAddr, /*sender*/
			big.NewInt(6_000),   /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)
		s.Require().NoError(stateDB.Commit())
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(3_420), "expect 3420 tokens")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(60_000), "expect 60000 tokens")
	})

	s.Run("Burn tokens - Allowed as owner", func() {
		evmObj, stateDB := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Burn(
			erc20,                  /*erc20Addr*/
			evm.EVM_MODULE_ADDRESS, /*sender*/
			big.NewInt(6_000),      /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)
		s.Require().NoError(stateDB.Commit())
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(3_420), "expect 3420 tokens")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(54_000), "expect 54000 tokens")
	})
}

func (s *FunTokenFromErc20Suite) TestEmptyTransferResp() {
	deps := evmtest.NewTestDeps()

	// assert that the ERC20 contract is not deployed
	expectedERC20Addr := crypto.CreateAddress(deps.Sender.EthAddr, deps.NewStateDB().GetNonce(deps.Sender.EthAddr))
	s.T().Log("Deploy tether")

	name := "tether"
	symbol := "usdt"
	totalSupply := big.NewInt(1_000_000_000)
	decimal := big.NewInt(6)

	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestTransferEmptyResp,
		totalSupply, name, symbol, decimal,
	)
	s.Require().NoError(err)
	s.Require().Equal(expectedERC20Addr, deployResp.ContractAddr)
	s.T().Log("transfer with empty response success")

	evmObj, stateDB := deps.NewEVM()
	sentAmt, _, err := deps.EvmKeeper.ERC20().Transfer(
		deployResp.ContractAddr, /*erc20Addr*/
		deps.Sender.EthAddr,
		evm.EVM_MODULE_ADDRESS,
		big.NewInt(60_000), /*amount*/
		deps.Ctx,
		evmObj,
	)
	s.Require().NoError(err)
	s.Require().NoError(stateDB.Commit())
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	evmtest.AssertERC20BalanceEqualWithDescription(
		s.T(), deps, evmObj, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(60_000), "expect nonzero balance")
	s.Require().EqualValues(big.NewInt(60_000), sentAmt)
}
