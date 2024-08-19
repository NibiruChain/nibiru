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
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)
	contract := funtoken.Erc20Addr.Address

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		_, err := deps.EvmKeeper.ERC20().Mint(contract, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), deps.Ctx)
		s.ErrorContains(err, evm.ErrOwnable)
	}

	s.T().Log("Mint tokens - Success")
	{
		_, err := deps.EvmKeeper.ERC20().Mint(contract, evm.EVM_MODULE_ADDRESS, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), deps.Ctx)
		s.Require().NoError(err)

		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, deps.Sender.EthAddr, big.NewInt(0))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420))
	}

	s.T().Log("Transfer - Not enough funds")
	{
		_, err := deps.EvmKeeper.ERC20().Transfer(contract, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(9_420), deps.Ctx)
		s.ErrorContains(err, "ERC20: transfer amount exceeds balance")
		// balances unchanged
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, deps.Sender.EthAddr, big.NewInt(0))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420))
	}

	s.T().Log("Transfer - Success (sanity check)")
	{
		_, err := deps.EvmKeeper.ERC20().Transfer(contract, evm.EVM_MODULE_ADDRESS, deps.Sender.EthAddr, big.NewInt(9_420), deps.Ctx)
		s.Require().NoError(err)
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, deps.Sender.EthAddr, big.NewInt(9_420))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(60_000))
	}

	s.T().Log("Burn tokens - Allowed as non-owner")
	{
		_, err := deps.EvmKeeper.ERC20().Burn(contract, deps.Sender.EthAddr, big.NewInt(420), deps.Ctx)
		s.Require().NoError(err)

		_, err = deps.EvmKeeper.ERC20().Burn(contract, evm.EVM_MODULE_ADDRESS, big.NewInt(6_000), deps.Ctx)
		s.Require().NoError(err)

		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, deps.Sender.EthAddr, big.NewInt(9_000))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(54_000))
	}
}
