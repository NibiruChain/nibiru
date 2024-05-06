package backend

import (
	"encoding/json"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/x/evm"
)

func (s *BackendSuite) TestGetLogs() {
	_, bz := s.buildEthereumTx()
	block := tmtypes.MakeBlock(1, []tmtypes.Tx{bz}, nil, nil)
	logs := make([]*evm.Log, 0, 1)
	var log evm.Log
	err := json.Unmarshal([]byte("{\"test\": \"hello\"}"), &log) // TODO refactor this to unmarshall to a log struct successfully
	s.Require().NoError(err)

	logs = append(logs, &log)

	testCases := []struct {
		name         string
		registerMock func(hash common.Hash)
		blockHash    common.Hash
		expLogs      [][]*gethcore.Log
		expPass      bool
	}{
		{
			"fail - no block with that hash",
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashNotFound(client, hash, bz)
			},
			common.Hash{},
			nil,
			false,
		},
		{
			"fail - error fetching block by hash",
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, bz)
			},
			common.Hash{},
			nil,
			false,
		},
		{
			"fail - error getting block results",
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			common.Hash{},
			nil,
			false,
		},
		{
			"success - getting logs with block hash",
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				s.Require().NoError(err)
				_, err = RegisterBlockResultsWithEventLog(client, rpc.BlockNumber(1).Int64())
				s.Require().NoError(err)
			},
			common.BytesToHash(block.Hash()),
			[][]*gethcore.Log{evm.LogsToEthereum(logs)},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			tc.registerMock(tc.blockHash)
			logs, err := s.backend.GetLogs(tc.blockHash)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expLogs, logs)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestBloomStatus() {
	testCases := []struct {
		name         string
		registerMock func()
		expResult    uint64
		expPass      bool
	}{
		{
			"pass - returns the BloomBitsBlocks and the number of processed sections maintained",
			func() {},
			4096,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			tc.registerMock()
			bloom, _ := s.backend.BloomStatus()

			if tc.expPass {
				s.Require().Equal(tc.expResult, bloom)
			}
		})
	}
}
