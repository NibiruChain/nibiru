// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

type SuiteEVMTest struct {
	suite.Suite
}

func TestSuiteEVMTest(t *testing.T) {
	suite.Run(t, new(SuiteEVMTest))
}

func (s *SuiteEVMTest) TestSampleFns() {
	s.T().Log("Test NewEthTxMsg")
	ethTxMsg := evmtest.NewEthTxMsg()
	err := ethTxMsg.ValidateBasic()
	s.NoError(err)

	s.T().Log("Test NewEthTxMsgs")
	for _, ethTxMsg := range evmtest.NewEthTxMsgs(3) {
		s.NoError(ethTxMsg.ValidateBasic())
	}

	s.T().Log("Test NewEthTxMsgs")
	_, _, _ = evmtest.NewEthTxMsgAsCmt(s.T())
}
