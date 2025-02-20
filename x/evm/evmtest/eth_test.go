// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
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
	s.T().Log("Test NewEthTxMsgs")
	for _, ethTxMsg := range evmtest.NewEthTxMsgs(3) {
		s.NoError(ethTxMsg.ValidateBasic())
	}

	s.T().Log("Test NewEthTxMsgAsCmt")
	_, _, _ = evmtest.NewEthTxMsgAsCmt(s.T())
}
