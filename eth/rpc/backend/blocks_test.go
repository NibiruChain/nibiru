package backend_test

import (
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
)

func (s *BackendSuite) TestBlockNumber() {
	blockHeight, err := s.backend.BlockNumber()
	s.Require().NoError(err)
	blockHeightU64, err := hexutil.DecodeUint64(blockHeight.String())
	s.NoError(err)
	s.Greater(blockHeightU64, uint64(1))

	latestHeight, _ := s.network.LatestHeight()
	resp, err := s.backend.BlockNumber()
	s.Require().NoError(err, resp)
	s.Require().Equal(uint64(latestHeight), uint64(blockHeight))
}

func (s *BackendSuite) TestGetBlockByNumberr() {
	block, err := s.backend.GetBlockByNumber(transferTxBlockNumber, true)
	s.Require().NoError(err)
	s.Require().NotNil(block)
	s.Require().Greater(len(block["transactions"].([]interface{})), 0)
	s.Require().NotNil(block["size"])
	s.Require().NotNil(block["nonce"])
	s.Require().Equal(int64(block["number"].(hexutil.Uint64)), transferTxBlockNumber.Int64())
}

func (s *BackendSuite) TestGetBlockByHash() {
	blockMap, err := s.backend.GetBlockByHash(transferTxBlockHash, true)
	s.Require().NoError(err)
	AssertBlockContents(s, blockMap)
}

func (s *BackendSuite) TestBlockNumberFromTendermint() {
	testCases := []struct {
		name            string
		blockNrOrHash   rpc.BlockNumberOrHash
		wantBlockNumber rpc.BlockNumber
		wantErr         string
	}{
		{
			name: "happy: block number specified",
			blockNrOrHash: rpc.BlockNumberOrHash{
				BlockNumber: &transferTxBlockNumber,
			},
			wantBlockNumber: transferTxBlockNumber,
			wantErr:         "",
		},
		{
			name: "happy: block hash specified",
			blockNrOrHash: rpc.BlockNumberOrHash{
				BlockHash: &transferTxBlockHash,
			},
			wantBlockNumber: transferTxBlockNumber,
			wantErr:         "",
		},
		{
			name:            "sad: neither block number nor hash specified",
			blockNrOrHash:   rpc.BlockNumberOrHash{},
			wantBlockNumber: 0,
			wantErr:         "BlockHash and BlockNumber cannot be both nil",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			blockNumber, err := s.backend.BlockNumberFromTendermint(tc.blockNrOrHash)

			if tc.wantErr != "" {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(tc.wantBlockNumber, blockNumber)
		})
	}
}

func (s *BackendSuite) TestEthBlockByNumber() {
	block, err := s.backend.EthBlockByNumber(transferTxBlockNumber)
	s.Require().NoError(err)
	s.Require().NotNil(block)
	s.Require().Equal(transferTxBlockNumber.Int64(), block.Number().Int64())
	s.Require().Greater(block.Transactions().Len(), 0)
	s.Require().NotNil(block.ParentHash())
	s.Require().NotNil(block.UncleHash())
}

func (s *BackendSuite) TestGetBlockTransactionCountByHash() {
	txCount := s.backend.GetBlockTransactionCountByHash(transferTxBlockHash)
	s.Require().Greater((uint64)(*txCount), uint64(0))
}

func (s *BackendSuite) TestGetBlockTransactionCountByNumber() {
	txCount := s.backend.GetBlockTransactionCountByNumber(transferTxBlockNumber)
	s.Require().Greater((uint64)(*txCount), uint64(0))
}

func AssertBlockContents(s *BackendSuite, blockMap map[string]interface{}) {
	s.Require().NotNil(blockMap)
	s.Require().Greater(len(blockMap["transactions"].([]interface{})), 0)
	s.Require().NotNil(blockMap["size"])
	s.Require().NotNil(blockMap["nonce"])
	s.Require().Equal(int64(blockMap["number"].(hexutil.Uint64)), transferTxBlockNumber.Int64())
}
