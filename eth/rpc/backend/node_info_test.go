package backend_test

import (
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth"
)

func (s *BackendSuite) TestAccounts() {
	accounts, err := s.backend.Accounts()
	s.Require().NoError(err)
	s.Require().Greater(len(accounts), 0)
	s.Require().Contains(accounts, gethcommon.BytesToAddress(s.node.ValAddress.Bytes()))
}

func (s *BackendSuite) TestSyncing() {
	syncing, err := s.backend.Syncing()
	s.Require().NoError(err)
	s.Require().False(syncing.(bool))
}

func (s *BackendSuite) TestRPCGasCap() {
	s.Require().Equal(config.DefaultConfig().JSONRPC.GasCap, s.backend.RPCGasCap())
}

func (s *BackendSuite) TestRPCEVMTimeout() {
	s.Require().Equal(config.DefaultConfig().JSONRPC.EVMTimeout, s.backend.RPCEVMTimeout())
}

func (s *BackendSuite) TestRPCFilterCap() {
	s.Require().Equal(config.DefaultConfig().JSONRPC.FilterCap, s.backend.RPCFilterCap())
}

func (s *BackendSuite) TestRPCLogsCap() {
	s.Require().Equal(config.DefaultConfig().JSONRPC.LogsCap, s.backend.RPCLogsCap())
}

func (s *BackendSuite) TestRPCBlockRangeCap() {
	s.Require().Equal(config.DefaultConfig().JSONRPC.BlockRangeCap, s.backend.RPCBlockRangeCap())
}

func (s *BackendSuite) TestRPCMinGasPrice() {
	s.Require().Equal(int64(eth.DefaultGasPrice), s.backend.RPCMinGasPrice())
}
