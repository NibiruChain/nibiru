package backend

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmlog "github.com/cometbft/cometbft/libs/log"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/indexer"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func (s *BackendSuite) TestGetTransactionByHash() {
	msgEthereumTx, _ := s.buildEthereumTx()
	txHash := msgEthereumTx.AsTransaction().Hash()

	txBz := s.signAndEncodeEthTx(msgEthereumTx)
	block := &types.Block{Header: types.Header{Height: 1, ChainID: "test"}, Data: types.Data{Txs: []types.Tx{txBz}}}
	responseDeliver := []*abci.ResponseDeliverTx{
		{
			Code: 0,
			Events: []abci.Event{
				{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
					{Key: "ethereumTxHash", Value: txHash.Hex()},
					{Key: "txIndex", Value: "0"},
					{Key: "amount", Value: "1000"},
					{Key: "txGasUsed", Value: "21000"},
					{Key: "txHash", Value: ""},
					{Key: "recipient", Value: ""},
				}},
			},
		},
	}

	rpcTransaction, _ := rpc.NewRPCTxFromEthTx(msgEthereumTx.AsTransaction(), common.Hash{}, 0, 0, big.NewInt(1), s.backend.chainID)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evm.MsgEthereumTx
		expRPCTx     *rpc.EthTxJsonRPC
		expPass      bool
	}{
		{
			"fail - Block error",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			msgEthereumTx,
			rpcTransaction,
			false,
		},
		{
			"fail - Block Result error",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, 1, txBz)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			msgEthereumTx,
			nil,
			true,
		},
		{
			"pass - Base fee error",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, txBz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFeeError(queryClient)
			},
			msgEthereumTx,
			rpcTransaction,
			true,
		},
		{
			"pass - Transaction found and returned",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, txBz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			msgEthereumTx,
			rpcTransaction,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			db := dbm.NewMemDB()
			s.backend.indexer = indexer.NewKVIndexer(db, tmlog.NewNopLogger(), s.backend.clientCtx)
			err := s.backend.indexer.IndexBlock(block, responseDeliver)
			s.Require().NoError(err)

			rpcTx, err := s.backend.GetTransactionByHash(common.HexToHash(tc.tx.Hash))

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionsByHashPending() {
	msgEthereumTx, bz := s.buildEthereumTx()
	rpcTransaction, _ := rpc.NewRPCTxFromEthTx(msgEthereumTx.AsTransaction(), common.Hash{}, 0, 0, big.NewInt(1), s.backend.chainID)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evm.MsgEthereumTx
		expRPCTx     *rpc.EthTxJsonRPC
		expPass      bool
	}{
		{
			"fail - Pending transactions returns error",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxsError(client, nil)
			},
			msgEthereumTx,
			nil,
			true,
		},
		{
			"fail - Tx not found return nil",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxs(client, nil, nil)
			},
			msgEthereumTx,
			nil,
			true,
		},
		{
			"pass - Tx found and returned",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxs(client, nil, types.Txs{bz})
			},
			msgEthereumTx,
			rpcTransaction,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := s.backend.getTransactionByHashPending(common.HexToHash(tc.tx.Hash))

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTxByEthHash() {
	msgEthereumTx, bz := s.buildEthereumTx()
	rpcTransaction, _ := rpc.NewRPCTxFromEthTx(msgEthereumTx.AsTransaction(), common.Hash{}, 0, 0, big.NewInt(1), s.backend.chainID)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evm.MsgEthereumTx
		expRPCTx     *rpc.EthTxJsonRPC
		expPass      bool
	}{
		{
			"fail - Indexer disabled can't find transaction",
			func() {
				s.backend.indexer = nil
				client := s.backend.clientCtx.Client.(*mocks.Client)
				query := fmt.Sprintf("%s.%s='%s'", evm.TypeMsgEthereumTx, evm.AttributeKeyEthereumTxHash, common.HexToHash(msgEthereumTx.Hash).Hex())
				RegisterTxSearch(client, query, bz)
			},
			msgEthereumTx,
			rpcTransaction,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := s.backend.GetTxByEthHash(common.HexToHash(tc.tx.Hash))

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionByBlockHashAndIndex() {
	_, bz := s.buildEthereumTx()

	testCases := []struct {
		name         string
		registerMock func()
		blockHash    common.Hash
		expRPCTx     *rpc.EthTxJsonRPC
		expPass      bool
	}{
		{
			"pass - block not found",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, common.Hash{}, bz)
			},
			common.Hash{},
			nil,
			true,
		},
		{
			"pass - Block results error",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, common.Hash{}, bz)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			common.Hash{},
			nil,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := s.backend.GetTransactionByBlockHashAndIndex(tc.blockHash, 1)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionByBlockAndIndex() {
	msgEthTx, bz := s.buildEthereumTx()

	defaultBlock := types.MakeBlock(1, []types.Tx{bz}, nil, nil)
	defaultResponseDeliverTx := []*abci.ResponseDeliverTx{
		{
			Code: 0,
			Events: []abci.Event{
				{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
					{Key: "ethereumTxHash", Value: common.HexToHash(msgEthTx.Hash).Hex()},
					{Key: "txIndex", Value: "0"},
					{Key: "amount", Value: "1000"},
					{Key: "txGasUsed", Value: "21000"},
					{Key: "txHash", Value: ""},
					{Key: "recipient", Value: ""},
				}},
			},
		},
	}

	txFromMsg, _ := rpc.NewRPCTxFromMsg(
		msgEthTx,
		common.BytesToHash(defaultBlock.Hash().Bytes()),
		1,
		0,
		big.NewInt(1),
		s.backend.chainID,
	)
	testCases := []struct {
		name         string
		registerMock func()
		block        *tmrpctypes.ResultBlock
		idx          hexutil.Uint
		expRPCTx     *rpc.EthTxJsonRPC
		expPass      bool
	}{
		{
			"pass - block txs index out of bound",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				s.Require().NoError(err)
			},
			&tmrpctypes.ResultBlock{Block: types.MakeBlock(1, []types.Tx{bz}, nil, nil)},
			1,
			nil,
			true,
		},
		{
			"pass - Can't fetch base fee",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFeeError(queryClient)
			},
			&tmrpctypes.ResultBlock{Block: defaultBlock},
			0,
			txFromMsg,
			true,
		},
		{
			"pass - Gets Tx by transaction index",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				db := dbm.NewMemDB()
				s.backend.indexer = indexer.NewKVIndexer(db, tmlog.NewNopLogger(), s.backend.clientCtx)
				txBz := s.signAndEncodeEthTx(msgEthTx)
				block := &types.Block{Header: types.Header{Height: 1, ChainID: "test"}, Data: types.Data{Txs: []types.Tx{txBz}}}
				err := s.backend.indexer.IndexBlock(block, defaultResponseDeliverTx)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			&tmrpctypes.ResultBlock{Block: defaultBlock},
			0,
			txFromMsg,
			true,
		},
		{
			"pass - returns the Ethereum format transaction by the Ethereum hash",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			&tmrpctypes.ResultBlock{Block: defaultBlock},
			0,
			txFromMsg,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := s.backend.GetTransactionByBlockAndIndex(tc.block, tc.idx)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionByBlockNumberAndIndex() {
	msgEthTx, bz := s.buildEthereumTx()
	defaultBlock := types.MakeBlock(1, []types.Tx{bz}, nil, nil)
	txFromMsg, _ := rpc.NewRPCTxFromMsg(
		msgEthTx,
		common.BytesToHash(defaultBlock.Hash().Bytes()),
		1,
		0,
		big.NewInt(1),
		s.backend.chainID,
	)
	testCases := []struct {
		name         string
		registerMock func()
		blockNum     rpc.BlockNumber
		idx          hexutil.Uint
		expRPCTx     *rpc.EthTxJsonRPC
		expPass      bool
	}{
		{
			"fail -  block not found return nil",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			0,
			0,
			nil,
			true,
		},
		{
			"pass - returns the transaction identified by block number and index",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			0,
			0,
			txFromMsg,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := s.backend.GetTransactionByBlockNumberAndIndex(tc.blockNum, tc.idx)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionByTxIndex() {
	_, bz := s.buildEthereumTx()

	testCases := []struct {
		name         string
		registerMock func()
		height       int64
		index        uint
		expTxResult  *eth.TxResult
		expPass      bool
	}{
		{
			"fail - Ethereum tx with query not found",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				s.backend.indexer = nil
				RegisterTxSearch(client, "tx.height=0 AND ethereum_tx.txIndex=0", bz)
			},
			0,
			0,
			&eth.TxResult{},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			txResults, err := s.backend.GetTxByTxIndex(tc.height, tc.index)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(txResults, tc.expTxResult)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestQueryTendermintTxIndexer() {
	testCases := []struct {
		name         string
		registerMock func()
		txGetter     func(*rpc.ParsedTxs) *rpc.ParsedTx
		query        string
		expTxResult  *eth.TxResult
		expPass      bool
	}{
		{
			"fail - Ethereum tx with query not found",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterTxSearchEmpty(client, "")
			},
			func(txs *rpc.ParsedTxs) *rpc.ParsedTx {
				return &rpc.ParsedTx{}
			},
			"",
			&eth.TxResult{},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			txResults, err := s.backend.queryTendermintTxIndexer(tc.query, tc.txGetter)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(txResults, tc.expTxResult)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionReceipt() {
	msgEthereumTx, _ := s.buildEthereumTx()
	txHash := msgEthereumTx.AsTransaction().Hash()

	txBz := s.signAndEncodeEthTx(msgEthereumTx)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evm.MsgEthereumTx
		block        *types.Block
		blockResult  []*abci.ResponseDeliverTx
		expTxReceipt map[string]interface{}
		expPass      bool
	}{
		{
			"fail - Receipts do not match",
			func() {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterParams(queryClient, &header, 1)
				RegisterParamsWithoutHeader(queryClient, 1)
				_, err := RegisterBlock(client, 1, txBz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
			},
			msgEthereumTx,
			&types.Block{Header: types.Header{Height: 1}, Data: types.Data{Txs: []types.Tx{txBz}}},
			[]*abci.ResponseDeliverTx{
				{
					Code: 0,
					Events: []abci.Event{
						{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
							{Key: "ethereumTxHash", Value: txHash.Hex()},
							{Key: "txIndex", Value: "0"},
							{Key: "amount", Value: "1000"},
							{Key: "txGasUsed", Value: "21000"},
							{Key: "txHash", Value: ""},
							{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
						}},
					},
				},
			},
			map[string]interface{}(nil),
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			tc.registerMock()

			db := dbm.NewMemDB()
			s.backend.indexer = indexer.NewKVIndexer(db, tmlog.NewNopLogger(), s.backend.clientCtx)
			err := s.backend.indexer.IndexBlock(tc.block, tc.blockResult)
			s.Require().NoError(err)

			txReceipt, err := s.backend.GetTransactionReceipt(common.HexToHash(tc.tx.Hash))
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(txReceipt, tc.expTxReceipt)
			} else {
				s.Require().NotEqual(txReceipt, tc.expTxReceipt)
			}
		})
	}
}

func (s *BackendSuite) TestGetGasUsed() {
	origin := s.backend.cfg.JSONRPC.FixRevertGasRefundHeight
	testCases := []struct {
		name                     string
		fixRevertGasRefundHeight int64
		txResult                 *eth.TxResult
		price                    *big.Int
		gas                      uint64
		exp                      uint64
	}{
		{
			"success txResult",
			1,
			&eth.TxResult{
				Height:  1,
				Failed:  false,
				GasUsed: 53026,
			},
			new(big.Int).SetUint64(0),
			0,
			53026,
		},
		{
			"fail txResult before cap",
			2,
			&eth.TxResult{
				Height:  1,
				Failed:  true,
				GasUsed: 53026,
			},
			new(big.Int).SetUint64(200000),
			5000000000000,
			1000000000000000000,
		},
		{
			"fail txResult after cap",
			2,
			&eth.TxResult{
				Height:  3,
				Failed:  true,
				GasUsed: 53026,
			},
			new(big.Int).SetUint64(200000),
			5000000000000,
			53026,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.backend.cfg.JSONRPC.FixRevertGasRefundHeight = tc.fixRevertGasRefundHeight
			s.Require().Equal(tc.exp, s.backend.GetGasUsed(tc.txResult, tc.price, tc.gas))
			s.backend.cfg.JSONRPC.FixRevertGasRefundHeight = origin
		})
	}
}
