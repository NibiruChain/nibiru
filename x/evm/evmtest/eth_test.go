// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type Suite struct {
	suite.Suite
}

func TestSuiteEVM(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestSampleFns() {
	s.T().Log("Test NewEthTxMsg")
	ethTxMsg := evmtest.NewEthTxMsgs(1)[0]
	err := ethTxMsg.ValidateBasic()
	s.NoError(err)

	s.T().Log("Test NewEthTxMsgs")
	for _, ethTxMsg := range evmtest.NewEthTxMsgs(3) {
		s.NoError(ethTxMsg.ValidateBasic())
	}

	s.T().Log("Test NewEthTxMsgs")
	_, _, _ = evmtest.NewEthTxMsgAsCmt(s.T())
}

func (s *Suite) TestERC20Helpers() {
	deps := evmtest.NewTestDeps()
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, "token", &s.Suite)

	evmtest.AssertERC20BalanceEqual(
		s.T(), deps,
		funtoken.Erc20Addr.Address,
		deps.Sender.EthAddr,
		big.NewInt(0),
	)
}
