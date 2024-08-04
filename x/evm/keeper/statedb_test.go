// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app/appconst"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

// TestStateDBBalance tests the behavior of the StateDB with regards to account
// balances, ensuring correct conversion between native tokens (unibi) and EVM
// tokens (wei), as well as proper balance updates during transfers.
func (s *Suite) TestStateDBBalance() {
	deps := evmtest.NewTestDeps()
	db := deps.StateDB()
	s.Equal("0", db.GetBalance(deps.Sender.EthAddr).String())

	s.T().Log("fund account in unibi. See expected wei amount.")
	err := testapp.FundAccount(
		deps.Chain.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewInt64Coin(appconst.BondDenom, 42)),
	)
	s.NoError(err)
	s.Equal(
		"42"+strings.Repeat("0", 12),
		db.GetBalance(deps.Sender.EthAddr).String(),
	)

	s.T().Log("Send 12 unibi. See expected wei amounts.")
	to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	err = evmtest.TransferWei(&deps, to, evm.NativeToWei(big.NewInt(12)))
	s.Require().NoError(err)
	db = deps.StateDB()

	s.Equal(
		"30"+strings.Repeat("0", 12),
		db.GetBalance(deps.Sender.EthAddr).String(),
	)
	s.Equal(
		"12"+strings.Repeat("0", 12),
		db.GetBalance(to).String(),
	)

	s.T().Log("Send 12 wei. Should error")
	err = evmtest.TransferWei(&deps, to, big.NewInt(12))
	s.Require().ErrorContains(err, "wei amount is too small")
}
