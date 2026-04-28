package rpcapi_test

import (
	cmtlog "github.com/cometbft/cometbft/libs/log"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
)

func (s *BackendSuite) TestEthAPIConstructedFromBackend() {
	api := rpcapi.NewImplEthAPI(cmtlog.TestingLogger(), s.backend)

	chainID, err := api.ChainId()
	s.Require().NoError(err)
	s.Require().Equal(appconst.ETH_CHAIN_ID_DEFAULT, chainID.ToInt().Int64())

	blockNumber, err := api.BlockNumber()
	s.Require().NoError(err)
	s.Require().Greater(uint64(blockNumber), uint64(0))

	block, err := api.GetBlockByNumber(*s.SuccessfulTxTransfer().BlockNumberRpc, false)
	s.Require().NoError(err)
	s.Require().NotNil(block)

	latestBlock := rpc.EthLatestBlockNumber
	latestBlockOrHash := rpc.BlockNumberOrHash{BlockNumber: &latestBlock}
	balance, err := api.GetBalance(s.evmSenderEthAddr, latestBlockOrHash)
	s.Require().NoError(err)
	s.Require().NotNil(balance)
}
