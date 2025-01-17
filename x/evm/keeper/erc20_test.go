// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestERC20Calls() {
	deps := evmtest.NewTestDeps()
	bankDenom := "ibc/btc"
	funtoken := evmtest.CreateFunTokenForBankCoin(deps, bankDenom, &s.Suite)
	contract := funtoken.Erc20Addr.Address

	s.Run("Mint tokens - Fail from non-owner", func() {
		_, err := deps.EvmKeeper.ERC20().Mint(
			contract, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS,
			big.NewInt(69_420), deps.Ctx, deps.NewEVM(),
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	})

	s.Run("Mint tokens - Success", func() {
		evmObj := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Mint(
			contract,               /*erc20Addr*/
			evm.EVM_MODULE_ADDRESS, /*sender*/
			evm.EVM_MODULE_ADDRESS, /*recipient*/
			big.NewInt(69_420),     /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)

		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, deps.Sender.EthAddr, big.NewInt(0), "expect zero balance")
	})

	s.Run("Transfer - Not enough funds", func() {
		evmObj := deps.NewEVM()
		_, _, err := deps.EvmKeeper.ERC20().Transfer(
			contract, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS,
			big.NewInt(9_420), deps.Ctx, evmObj,
		)
		s.ErrorContains(err, "ERC20: transfer amount exceeds balance")
		// balances unchanged
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, deps.Sender.EthAddr, big.NewInt(0), "expect zero balance")
	})

	s.Run("Transfer - Success (sanity check)", func() {
		evmObj := deps.NewEVM()
		sentAmt, _, err := deps.EvmKeeper.ERC20().Transfer(
			contract,               /*erc20Addr*/
			evm.EVM_MODULE_ADDRESS, /*sender*/
			deps.Sender.EthAddr,    /*recipient*/
			big.NewInt(9_420),      /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)

		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, contract, deps.Sender.EthAddr, big.NewInt(9_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(60_000), "expect nonzero balance")
		s.Require().EqualValues(big.NewInt(9_420), sentAmt)
	})

	s.Run("Burn tokens - Allowed as non-owner", func() {
		evmObj := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Burn(
			contract,            /*erc20Addr*/
			deps.Sender.EthAddr, /*sender*/
			big.NewInt(6_000),   /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)

		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, deps.Sender.EthAddr, big.NewInt(3_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(60_000), "expect nonzero balance")
	})

	s.Run("Burn tokens - Allowed as owner", func() {
		evmObj := deps.NewEVM()
		_, err := deps.EvmKeeper.ERC20().Burn(
			contract,               /*erc20Addr*/
			evm.EVM_MODULE_ADDRESS, /*sender*/
			big.NewInt(6_000),      /*amount*/
			deps.Ctx,
			evmObj,
		)
		s.Require().NoError(err)

		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, deps.Sender.EthAddr, big.NewInt(3_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(54_000), "expect nonzero balance")
	})
}
