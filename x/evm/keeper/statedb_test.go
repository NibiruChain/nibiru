// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

// TestStateDBBalance tests the behavior of the StateDB with regards to account
// balances, ensuring correct conversion between native tokens (unibi) and EVM
// tokens (wei), as well as proper balance updates during transfers.
func (s *Suite) TestStateDBBalance() {
	deps := evmtest.NewTestDeps()
	{
		db := deps.StateDB()
		s.Equal("0", db.GetBalance(deps.Sender.EthAddr).String())

		s.T().Log("fund account in unibi. See expected wei amount.")
		err := testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			sdk.NewCoins(sdk.NewInt64Coin(evm.DefaultEVMDenom, 42)),
		)
		s.NoError(err)
		s.Equal(
			"42"+strings.Repeat("0", 12),
			db.GetBalance(deps.Sender.EthAddr).String(),
		)
		s.Equal(
			"42",
			deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, evm.DefaultEVMDenom).Amount.String(),
		)
	}

	s.T().Log("Send via EVM transfer. See expected wei amounts.")
	to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	{
		err := evmtest.TransferWei(&deps, to, evm.NativeToWei(big.NewInt(12)))
		s.Require().NoError(err)
		db := deps.StateDB()
		s.Equal(
			"30"+strings.Repeat("0", 12),
			db.GetBalance(deps.Sender.EthAddr).String(),
		)
		s.Equal(
			"12"+strings.Repeat("0", 12),
			db.GetBalance(to).String(),
		)

		s.T().Log("Send via EVM transfer with too little wei. Should error")
		err = evmtest.TransferWei(&deps, to, big.NewInt(12))
		s.Require().ErrorContains(err, "wei amount is too small")
	}

	s.T().Log("Send via bank transfer from account to account. See expected wei amounts.")
	{
		deps := evmtest.NewTestDeps()
		toNibiAddr := eth.EthAddrToNibiruAddr(to)
		err := testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			sdk.NewCoins(sdk.NewInt64Coin(evm.DefaultEVMDenom, 8)),
		)
		s.NoError(err)
		err = deps.App.BankKeeper.SendCoins(
			deps.Ctx, deps.Sender.NibiruAddr,
			toNibiAddr,
			sdk.NewCoins(sdk.NewInt64Coin(evm.DefaultEVMDenom, 3)),
		)
		s.NoError(err)

		db := deps.StateDB()
		s.Equal(
			"3"+strings.Repeat("0", 12),
			db.GetBalance(to).String(),
		)
		s.Equal(
			"5"+strings.Repeat("0", 12),
			db.GetBalance(deps.Sender.EthAddr).String(),
		)
	}
}
