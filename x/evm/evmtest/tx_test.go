// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

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
	evmResp, err := evmtest.TxTransferWei{
		Deps:      &deps,
		To:        to,
		AmountWei: evm.NativeToWei(big.NewInt(420)),
	}.Run()
	s.Require().NoErrorf(err, "%#v", evmResp)
	s.False(evmResp.Failed(), "%#v", evmResp)

	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, evm.EVMBankDenom, deps.Sender.EthAddr, big.NewInt(69_000), "expect nonzero balance",
	)

	s.Run("DeployAndExecuteERC20Transfer", func() {
		evmtest.DeployAndExecuteERC20Transfer(&deps, s.T())
	})
}
