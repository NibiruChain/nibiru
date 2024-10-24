// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestCallContractTx() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Deploy some ERC20")
	deployArgs := []any{"name", "SYMBOL", uint8(18)}
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20Minter,
		deployArgs...,
	)
	s.Require().NoError(err, deployResp)
	contractAddr := crypto.CreateAddress(deps.Sender.EthAddr, deployResp.Nonce)
	gotContractAddr := deployResp.ContractAddr
	s.Require().Equal(contractAddr, gotContractAddr)

	s.T().Log("expect zero balance")
	{
		wantBal := big.NewInt(0)
		evmtest.AssertERC20BalanceEqual(
			s.T(), deps, contractAddr, deps.Sender.EthAddr, wantBal,
		)
	}

	abi := deployResp.ContractData.ABI
	s.T().Log("mint some tokens")
	{
		amount := big.NewInt(69_420)
		to := deps.Sender.EthAddr
		callArgs := []any{to, amount}
		input, err := abi.Pack(
			"mint", callArgs...,
		)
		s.Require().NoError(err)
		_, resp, err := evmtest.CallContractTx(
			&deps,
			contractAddr,
			input,
			deps.Sender,
		)
		s.Require().NoError(err)
		s.Require().Empty(resp.VmError)
	}

	s.T().Log("expect nonzero balance")
	{
		wantBal := big.NewInt(69_420)
		evmtest.AssertERC20BalanceEqual(
			s.T(), deps, contractAddr, deps.Sender.EthAddr, wantBal,
		)
	}
}

func (s *Suite) TestTransferWei() {
	deps := evmtest.NewTestDeps()

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420))),
	))

	randomAcc := evmtest.NewEthPrivAcc()
	to := randomAcc.EthAddr
	err := evmtest.TransferWei(&deps, to, evm.NativeToWei(big.NewInt(420)))
	s.Require().NoError(err)

	evmtest.AssertBankBalanceEqual(
		s.T(), deps, evm.EVMBankDenom, deps.Sender.EthAddr, big.NewInt(69_000),
	)

	s.Run("DeployAndExecuteERC20Transfer", func() {
		evmtest.DeployAndExecuteERC20Transfer(&deps, s.T())
	})
}
