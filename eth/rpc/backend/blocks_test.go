package backend_test

import (
	"context"
	"math/big"

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
	wantFullTx := true
	resp, err := s.backend.GetBlockByNumber(
		rpc.NewBlockNumber(big.NewInt(latestHeight)),
		wantFullTx,
	)
	s.Require().NoError(err, resp)

	// TODO: test backend.GetBlockByHash
	// s.backend.GetBlockByHash()
	block, err := s.node.RPCClient.Block(
		context.Background(),
		&latestHeight,
	)
	s.NoError(err, block)
	blockResults, err := s.node.RPCClient.BlockResults(
		context.Background(),
		&latestHeight,
	)
	s.NoError(err, blockResults)
}
