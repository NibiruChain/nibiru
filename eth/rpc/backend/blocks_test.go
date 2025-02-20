package backend

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"

	"github.com/cometbft/cometbft/abci/types"
	cmtrpc "github.com/cometbft/cometbft/rpc/core/types"
	cmt "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/x/evm"
	evmtest "github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *BackendSuite) TestBlockNumber() {
	testCases := []struct {
		name         string
		registerMock func()
		wantBlockNum hexutil.Uint64
		wantPass     bool
	}{
		{
			name: "fail - invalid block header height",
			registerMock: func() {
				var header metadata.MD
				height := int64(1)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsInvalidHeight(queryClient, &header, height)
			},
			wantBlockNum: 0x0,
			wantPass:     false,
		},
		{
			name: "fail - invalid block header",
			registerMock: func() {
				var header metadata.MD
				height := int64(1)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsInvalidHeader(queryClient, &header, height)
			},
			wantBlockNum: 0x0,
			wantPass:     false,
		},
		{
			name: "pass - app state header height 1",
			registerMock: func() {
				var header metadata.MD
				height := int64(1)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, height)
			},
			wantBlockNum: 0x1,
			wantPass:     true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			blockNumber, err := s.backend.BlockNumber()

			if tc.wantPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.wantBlockNum, blockNumber)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetBlockByNumber() {
	var (
		blockRes *cmtrpc.ResultBlockResults
		resBlock *cmtrpc.ResultBlock
	)
	msgEthereumTx, bz := s.buildEthereumTx()

	testCases := []struct {
		name         string
		blockNumber  rpc.BlockNumber
		fullTx       bool
		baseFee      *big.Int
		validator    sdk.AccAddress
		ethTx        *evm.MsgEthereumTx
		ethTxBz      []byte
		registerMock func(rpc.BlockNumber, math.Int, sdk.AccAddress, []byte)
		wantNoop     bool
		wantPass     bool
	}{
		{
			name:        "pass - tendermint block not found",
			blockNumber: rpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     math.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			ethTx:       nil,
			ethTxBz:     nil,
			registerMock: func(blockNum rpc.BlockNumber, _ math.Int, _ sdk.AccAddress, _ []byte) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			wantNoop: true,
			wantPass: true,
		},
		{
			name:        "pass - block not found (e.g. request block height that is greater than current one)",
			blockNumber: rpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     math.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			ethTx:       nil,
			ethTxBz:     nil,
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockNotFound(client, height)
			},
			wantNoop: true,
			wantPass: true,
		},
		{
			name:        "pass - block results error",
			blockNumber: rpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     math.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			ethTx:       nil,
			ethTxBz:     nil,
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				RegisterBlockResultsError(client, blockNum.Int64())
			},
			wantNoop: true,
			wantPass: true,
		},
		{
			name:        "pass - without tx",
			blockNumber: rpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     math.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			ethTx:       nil,
			ethTxBz:     nil,
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				blockRes, _ = RegisterBlockResults(client, blockNum.Int64())
				RegisterConsensusParams(client, height)

				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			wantNoop: false,
			wantPass: true,
		},
		{
			name:        "pass - with tx",
			blockNumber: rpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     math.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			ethTx:       msgEthereumTx,
			ethTxBz:     bz,
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				blockRes, _ = RegisterBlockResults(client, blockNum.Int64())
				RegisterConsensusParams(client, height)

				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			wantNoop: false,
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(tc.blockNumber, math.NewIntFromBigInt(tc.baseFee), tc.validator, tc.ethTxBz)

			block, err := s.backend.GetBlockByNumber(tc.blockNumber, tc.fullTx)

			if tc.wantPass {
				if tc.wantNoop {
					s.Require().Nil(block)
				} else {
					expBlock := s.buildFormattedBlock(
						blockRes,
						resBlock,
						tc.fullTx,
						tc.ethTx,
						tc.validator,
						tc.baseFee,
					)
					s.Require().Equal(expBlock, block)
				}
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetBlockByHash() {
	var (
		blockRes *cmtrpc.ResultBlockResults
		resBlock *cmtrpc.ResultBlock
	)
	msgEthereumTx, bz := s.buildEthereumTx()

	block := cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		fullTx       bool
		baseFee      *big.Int
		validator    sdk.AccAddress
		tx           *evm.MsgEthereumTx
		txBz         []byte
		registerMock func(
			common.Hash, math.Int, sdk.AccAddress, []byte)
		wantNoop bool
		wantPass bool
	}{
		{
			name:      "fail - tendermint failed to get block",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, txBz)
			},
			wantNoop: false,
			wantPass: false,
		},
		{
			name:      "noop - tendermint blockres not found",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashNotFound(client, hash, txBz)
			},
			wantNoop: true,
			wantPass: true,
		},
		{
			name:      "noop - tendermint failed to fetch block result",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				RegisterBlockResultsError(client, height)
			},
			wantNoop: true,
			wantPass: true,
		},
		{
			name:      "pass - without tx",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				blockRes, _ = RegisterBlockResults(client, height)
				RegisterConsensusParams(client, height)

				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			wantNoop: false,
			wantPass: true,
		},
		{
			name:      "pass - with tx",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			tx:        msgEthereumTx,
			txBz:      bz,
			registerMock: func(hash common.Hash, baseFee math.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				blockRes, _ = RegisterBlockResults(client, height)
				RegisterConsensusParams(client, height)

				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			wantNoop: false,
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(tc.hash, math.NewIntFromBigInt(tc.baseFee), tc.validator, tc.txBz)

			block, err := s.backend.GetBlockByHash(tc.hash, tc.fullTx)

			if tc.wantPass {
				if tc.wantNoop {
					s.Require().Nil(block)
				} else {
					expBlock := s.buildFormattedBlock(
						blockRes,
						resBlock,
						tc.fullTx,
						tc.tx,
						tc.validator,
						tc.baseFee,
					)
					s.Require().Equal(expBlock, block)
				}
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetBlockTransactionCountByHash() {
	_, bz := s.buildEthereumTx()
	block := cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil)
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		registerMock func(common.Hash)
		wantCount    hexutil.Uint
		wantPass     bool
	}{
		{
			name: "fail - block not found",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, nil)
			},
			wantCount: hexutil.Uint(0),
			wantPass:  false,
		},
		{
			name: "fail - tendermint client failed to get block result",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, nil)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			wantCount: hexutil.Uint(0),
			wantPass:  false,
		},
		{
			name: "pass - block without tx",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				s.Require().NoError(err)
			},
			wantCount: hexutil.Uint(0),
			wantPass:  true,
		},
		{
			name: "pass - block with tx",
			hash: common.BytesToHash(block.Hash()),
			registerMock: func(hash common.Hash) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				s.Require().NoError(err)
			},
			wantCount: hexutil.Uint(1),
			wantPass:  true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			tc.registerMock(tc.hash)
			count := s.backend.GetBlockTransactionCountByHash(tc.hash)
			if tc.wantPass {
				s.Require().Equal(tc.wantCount, *count)
			} else {
				s.Require().Nil(count)
			}
		})
	}
}

func (s *BackendSuite) TestGetBlockTransactionCountByNumber() {
	_, bz := s.buildEthereumTx()
	block := cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil)
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		blockNum     rpc.BlockNumber
		registerMock func(rpc.BlockNumber)
		wantCount    hexutil.Uint
		wantPass     bool
	}{
		{
			name:     "fail - block not found",
			blockNum: rpc.BlockNumber(emptyBlock.Height),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			wantCount: hexutil.Uint(0),
			wantPass:  false,
		},
		{
			name:     "fail - tendermint client failed to get block result",
			blockNum: rpc.BlockNumber(emptyBlock.Height),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			wantCount: hexutil.Uint(0),
			wantPass:  false,
		},
		{
			name:     "pass - block without tx",
			blockNum: rpc.BlockNumber(emptyBlock.Height),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				s.Require().NoError(err)
			},
			wantCount: hexutil.Uint(0),
			wantPass:  true,
		},
		{
			name:     "pass - block with tx",
			blockNum: rpc.BlockNumber(block.Height),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, bz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				s.Require().NoError(err)
			},
			wantCount: hexutil.Uint(1),
			wantPass:  true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			tc.registerMock(tc.blockNum)
			count := s.backend.GetBlockTransactionCountByNumber(tc.blockNum)
			if tc.wantPass {
				s.Require().Equal(tc.wantCount, *count)
			} else {
				s.Require().Nil(count)
			}
		})
	}
}

func (s *BackendSuite) TestTendermintBlockByNumber() {
	var expResultBlock *cmtrpc.ResultBlock

	testCases := []struct {
		name           string
		blockNumber    rpc.BlockNumber
		registerMock   func(rpc.BlockNumber)
		wantBlockFound bool
		wantPass       bool
	}{
		{
			name:        "fail - client error",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			wantBlockFound: false,
			wantPass:       false,
		},
		{
			name:        "noop - block not found",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockNotFound(client, height)
				s.Require().NoError(err)
			},
			wantBlockFound: false,
			wantPass:       true,
		},
		{
			name:        "fail - blockNum < 0 with app state height error",
			blockNumber: rpc.BlockNumber(-1),
			registerMock: func(_ rpc.BlockNumber) {
				var header metadata.MD
				appHeight := int64(1)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsError(queryClient, &header, appHeight)
			},
			wantBlockFound: false,
			wantPass:       false,
		},
		{
			name:        "pass - blockNum < 0 with app state height >= 1",
			blockNumber: rpc.BlockNumber(-1),
			registerMock: func(blockNum rpc.BlockNumber) {
				var header metadata.MD
				appHeight := int64(1)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, appHeight)

				tmHeight := appHeight
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, tmHeight, nil)
			},
			wantBlockFound: true,
			wantPass:       true,
		},
		{
			name:        "pass - blockNum = 0 (defaults to blockNum = 1 due to a difference between tendermint heights and geth heights)",
			blockNumber: rpc.BlockNumber(0),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
			},
			wantBlockFound: true,
			wantPass:       true,
		},
		{
			name:        "pass - blockNum = 1",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
			},
			wantBlockFound: true,
			wantPass:       true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			tc.registerMock(tc.blockNumber)
			resultBlock, err := s.backend.TendermintBlockByNumber(tc.blockNumber)

			if tc.wantPass {
				s.Require().NoError(err)

				if !tc.wantBlockFound {
					s.Require().Nil(resultBlock)
				} else {
					s.Require().Equal(expResultBlock, resultBlock)
					s.Require().Equal(expResultBlock.Block.Header.Height, resultBlock.Block.Header.Height)
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestTendermintBlockResultByNumber() {
	var expBlockRes *cmtrpc.ResultBlockResults

	testCases := []struct {
		name         string
		blockNumber  int64
		registerMock func(int64)
		wantPass     bool
	}{
		{
			name:        "fail",
			blockNumber: 1,
			registerMock: func(blockNum int64) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockResultsError(client, blockNum)
			},
			wantPass: false,
		},
		{
			name:        "pass",
			blockNumber: 1,
			registerMock: func(blockNum int64) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, blockNum)
				s.Require().NoError(err)
				expBlockRes = &cmtrpc.ResultBlockResults{
					Height:     blockNum,
					TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
				}
			},
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(tc.blockNumber)

			blockRes, err := s.backend.TendermintBlockResultByNumber(&tc.blockNumber) //#nosec G601 -- fine for tests

			if tc.wantPass {
				s.Require().NoError(err)
				s.Require().Equal(expBlockRes, blockRes)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestBlockNumberFromTendermint() {
	var resBlock *cmtrpc.ResultBlock

	_, bz := s.buildEthereumTx()
	block := cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil)
	blockNum := rpc.NewBlockNumber(big.NewInt(block.Height))
	blockHash := common.BytesToHash(block.Hash())

	testCases := []struct {
		name         string
		blockNum     *rpc.BlockNumber
		hash         *common.Hash
		registerMock func(*common.Hash)
		wantPass     bool
	}{
		{
			"error - without blockHash or blockNum",
			nil,
			nil,
			func(hash *common.Hash) {},
			false,
		},
		{
			"error - with blockHash, tendermint client failed to get block",
			nil,
			&blockHash,
			func(hash *common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, *hash, bz)
			},
			false,
		},
		{
			"pass - with blockHash",
			nil,
			&blockHash,
			func(hash *common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, *hash, bz)
			},
			true,
		},
		{
			"pass - without blockHash & with blockNumber",
			&blockNum,
			nil,
			func(hash *common.Hash) {},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			blockNrOrHash := rpc.BlockNumberOrHash{
				BlockNumber: tc.blockNum,
				BlockHash:   tc.hash,
			}

			tc.registerMock(tc.hash)
			blockNum, err := s.backend.BlockNumberFromTendermint(blockNrOrHash)

			if tc.wantPass {
				s.Require().NoError(err)
				if tc.hash == nil {
					s.Require().Equal(*tc.blockNum, blockNum)
				} else {
					expHeight := rpc.NewBlockNumber(big.NewInt(resBlock.Block.Height))
					s.Require().Equal(expHeight, blockNum)
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestBlockNumberFromTendermintByHash() {
	var resBlock *cmtrpc.ResultBlock

	_, bz := s.buildEthereumTx()
	block := cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil)
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		registerMock func(common.Hash)
		wantPass     bool
	}{
		{
			"fail - tendermint client failed to get block",
			common.BytesToHash(block.Hash()),
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, bz)
			},
			false,
		},
		{
			"pass - block without tx",
			common.BytesToHash(emptyBlock.Hash()),
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, bz)
			},
			true,
		},
		{
			"pass - block with tx",
			common.BytesToHash(block.Hash()),
			func(hash common.Hash) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, bz)
			},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			tc.registerMock(tc.hash)
			blockNum, err := s.backend.BlockNumberFromTendermintByHash(tc.hash)
			if tc.wantPass {
				expHeight := big.NewInt(resBlock.Block.Height)
				s.Require().NoError(err)
				s.Require().Equal(expHeight, blockNum)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestBlockBloom() {
	testCases := []struct {
		name           string
		blockRes       *cmtrpc.ResultBlockResults
		wantBlockBloom gethcore.Bloom
		wantPass       bool
	}{
		{
			"fail - empty block result",
			&cmtrpc.ResultBlockResults{},
			gethcore.Bloom{},
			false,
		},
		{
			"fail - non block bloom event type",
			&cmtrpc.ResultBlockResults{
				EndBlockEvents: []types.Event{{Type: evm.EventTypeEthereumTx}},
			},
			gethcore.Bloom{},
			false,
		},
		{
			"fail - nonblock bloom attribute key",
			&cmtrpc.ResultBlockResults{
				EndBlockEvents: []types.Event{
					{
						Type: evm.EventTypeBlockBloom,
						Attributes: []types.EventAttribute{
							{Key: evm.AttributeKeyEthereumTxHash},
						},
					},
				},
			},
			gethcore.Bloom{},
			false,
		},
		{
			"pass - block bloom attribute key",
			&cmtrpc.ResultBlockResults{
				EndBlockEvents: []types.Event{
					{
						Type: evm.EventTypeBlockBloom,
						Attributes: []types.EventAttribute{
							{Key: evm.AttributeKeyEthereumBloom},
						},
					},
				},
			},
			gethcore.Bloom{},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			blockBloom, err := s.backend.BlockBloom(tc.blockRes)

			if tc.wantPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.wantBlockBloom, blockBloom)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetEthBlockFromTendermint() {
	msgEthereumTx, bz := s.buildEthereumTx()
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		baseFee      *big.Int
		validator    sdk.AccAddress
		height       int64
		resBlock     *cmtrpc.ResultBlock
		blockRes     *cmtrpc.ResultBlockResults
		fullTx       bool
		registerMock func(math.Int, sdk.AccAddress, int64)
		wantTxs      bool
		wantPass     bool
	}{
		{
			name:      "pass - block without tx",
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(common.Address{}.Bytes()),
			height:    int64(1),
			resBlock:  &cmtrpc.ResultBlock{Block: emptyBlock},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			fullTx: false,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			wantTxs:  false,
			wantPass: true,
		},
		{
			name:      "pass - block with tx - with BaseFee error",
			baseFee:   nil,
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			fullTx: true,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
				RegisterValidatorAccount(queryClient, validator)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			wantTxs:  true,
			wantPass: true,
		},
		{
			name:      "pass - block with tx - with ValidatorAccount error",
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(common.Address{}.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			fullTx: true,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccountError(queryClient)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			wantTxs:  true,
			wantPass: true,
		},
		{
			name:      "pass - block with tx - with ConsensusParams error - BlockMaxGas defaults to max uint32",
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			fullTx: true,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParamsError(client, height)
			},
			wantTxs:  true,
			wantPass: true,
		},
		{
			name:      "pass - block with tx - with ShouldIgnoreGasUsed - empty txs",
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height: 1,
				TxsResults: []*types.ResponseDeliverTx{
					{
						Code:    11,
						GasUsed: 0,
						Log:     "no block gas left to run tx: out of gas",
					},
				},
			},
			fullTx: true,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			wantTxs:  false,
			wantPass: true,
		},
		{
			name:      "pass - block with tx - non fullTx",
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			fullTx: false,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			wantTxs:  true,
			wantPass: true,
		},
		{
			name:      "pass - block with tx",
			baseFee:   math.NewInt(1).BigInt(),
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			fullTx: true,
			registerMock: func(baseFee math.Int, validator sdk.AccAddress, height int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			wantTxs:  true,
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(math.NewIntFromBigInt(tc.baseFee), tc.validator, tc.height)

			block, err := s.backend.RPCBlockFromTendermintBlock(tc.resBlock, tc.blockRes, tc.fullTx)

			var expBlock map[string]interface{}
			header := tc.resBlock.Block.Header
			gasLimit := int64(^uint32(0)) // for `MaxGas = -1` (DefaultConsensusParams)
			gasUsed := new(big.Int).SetUint64(uint64(tc.blockRes.TxsResults[0].GasUsed))

			root := common.Hash{}.Bytes()
			receipt := gethcore.NewReceipt(root, false, gasUsed.Uint64())
			bloom := gethcore.CreateBloom(gethcore.Receipts{receipt})

			ethRPCTxs := []interface{}{}

			if tc.wantTxs {
				if tc.fullTx {
					rpcTx, err := rpc.NewRPCTxFromEthTx(
						msgEthereumTx.AsTransaction(),
						common.BytesToHash(header.Hash()),
						uint64(header.Height),
						uint64(0),
						tc.baseFee,
						s.backend.chainID,
					)
					s.Require().NoError(err)
					ethRPCTxs = []interface{}{rpcTx}
				} else {
					ethRPCTxs = []interface{}{common.HexToHash(msgEthereumTx.Hash)}
				}
			}

			expBlock = rpc.FormatBlock(
				header,
				tc.resBlock.Block.Size(),
				gasLimit,
				gasUsed,
				ethRPCTxs,
				bloom,
				common.BytesToAddress(tc.validator.Bytes()),
				tc.baseFee,
			)

			if tc.wantPass {
				s.Require().Equal(expBlock, block)
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestEthMsgsFromTendermintBlock() {
	msgEthereumTx, bz := s.buildEthereumTx()

	testCases := []struct {
		name     string
		resBlock *cmtrpc.ResultBlock
		blockRes *cmtrpc.ResultBlockResults
		wantMsgs []*evm.MsgEthereumTx
	}{
		{
			"tx in not included in block - unsuccessful tx without ExceedBlockGasLimit error",
			&cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			&cmtrpc.ResultBlockResults{
				TxsResults: []*types.ResponseDeliverTx{
					{
						Code: 1,
					},
				},
			},
			[]*evm.MsgEthereumTx(nil),
		},
		{
			"tx included in block - unsuccessful tx with ExceedBlockGasLimit error",
			&cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			&cmtrpc.ResultBlockResults{
				TxsResults: []*types.ResponseDeliverTx{
					{
						Code: 1,
						Log:  rpc.ErrExceedBlockGasLimit,
					},
				},
			},
			[]*evm.MsgEthereumTx{msgEthereumTx},
		},
		{
			"pass",
			&cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			&cmtrpc.ResultBlockResults{
				TxsResults: []*types.ResponseDeliverTx{
					{
						Code: 0,
						Log:  rpc.ErrExceedBlockGasLimit,
					},
				},
			},
			[]*evm.MsgEthereumTx{msgEthereumTx},
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			msgs := s.backend.EthMsgsFromTendermintBlock(tc.resBlock, tc.blockRes)
			s.Require().Equal(tc.wantMsgs, msgs)
		})
	}
}

func (s *BackendSuite) TestHeaderByNumber() {
	var expResultBlock *cmtrpc.ResultBlock

	_, bz := s.buildEthereumTx()

	testCases := []struct {
		name         string
		blockNumber  rpc.BlockNumber
		baseFee      *big.Int
		registerMock func(rpc.BlockNumber, math.Int)
		wantPass     bool
	}{
		{
			name:        "fail - tendermint client failed to get block",
			blockNumber: rpc.BlockNumber(1),
			baseFee:     math.NewInt(1).BigInt(),
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			wantPass: false,
		},
		{
			name:        "fail - block not found for height",
			blockNumber: rpc.BlockNumber(1),
			baseFee:     math.NewInt(1).BigInt(),
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockNotFound(client, height)
				s.Require().NoError(err)
			},
			wantPass: false,
		},
		{
			name:        "fail - block not found for height",
			blockNumber: rpc.BlockNumber(1),
			baseFee:     math.NewInt(1).BigInt(),
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			wantPass: false,
		},
		{
			name:        "pass - without Base Fee, failed to fetch from prunned block",
			blockNumber: rpc.BlockNumber(1),
			baseFee:     nil,
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
				_, err := RegisterBlockResults(client, height)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			wantPass: true,
		},
		{
			name:        "pass - blockNum = 1, without tx",
			blockNumber: rpc.BlockNumber(1),
			baseFee:     math.NewInt(1).BigInt(),
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
				_, err := RegisterBlockResults(client, height)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			wantPass: true,
		},
		{
			name:        "pass - blockNum = 1, with tx",
			blockNumber: rpc.BlockNumber(1),
			baseFee:     math.NewInt(1).BigInt(),
			registerMock: func(blockNum rpc.BlockNumber, baseFee math.Int) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, bz)
				_, err := RegisterBlockResults(client, height)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			tc.registerMock(tc.blockNumber, math.NewIntFromBigInt(tc.baseFee))
			header, err := s.backend.HeaderByNumber(tc.blockNumber)

			if tc.wantPass {
				expHeader := rpc.EthHeaderFromTendermint(expResultBlock.Block.Header, gethcore.Bloom{}, tc.baseFee)
				s.Require().NoError(err)
				s.Require().Equal(expHeader, header)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestHeaderByHash() {
	var expResultBlock *cmtrpc.ResultBlock

	_, bz := s.buildEthereumTx()
	block := cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil)
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		baseFee      *big.Int
		registerMock func(common.Hash, math.Int)
		wantPass     bool
	}{
		{
			name:    "fail - tendermint client failed to get block",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: math.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee math.Int) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, bz)
			},
			wantPass: false,
		},
		{
			name:    "fail - block not found for height",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: math.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee math.Int) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashNotFound(client, hash, bz)
			},
			wantPass: false,
		},
		{
			name:    "fail - block not found for height",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: math.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee math.Int) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			wantPass: false,
		},
		{
			name:    "pass - without Base Fee, failed to fetch from prunned block",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: nil,
			registerMock: func(hash common.Hash, baseFee math.Int) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlockByHash(client, hash, bz)
				_, err := RegisterBlockResults(client, height)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			wantPass: true,
		},
		{
			name:    "pass - blockNum = 1, without tx",
			hash:    common.BytesToHash(emptyBlock.Hash()),
			baseFee: math.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee math.Int) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlockByHash(client, hash, nil)
				_, err := RegisterBlockResults(client, height)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			wantPass: true,
		},
		{
			name:    "pass - with tx",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: math.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee math.Int) {
				height := int64(1)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlockByHash(client, hash, bz)
				_, err := RegisterBlockResults(client, height)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			wantPass: true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries

			tc.registerMock(tc.hash, math.NewIntFromBigInt(tc.baseFee))
			header, err := s.backend.HeaderByHash(tc.hash)

			if tc.wantPass {
				expHeader := rpc.EthHeaderFromTendermint(expResultBlock.Block.Header, gethcore.Bloom{}, tc.baseFee)
				s.Require().NoError(err)
				s.Require().Equal(expHeader, header)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestEthBlockByNumber() {
	msgEthereumTx, bz := s.buildEthereumTx()
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		blockNumber  rpc.BlockNumber
		registerMock func(rpc.BlockNumber)
		expEthBlock  *gethcore.Block
		wantPass     bool
	}{
		{
			name:        "fail - tendermint client failed to get block",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			expEthBlock: nil,
			wantPass:    false,
		},
		{
			name:        "fail - block result not found for height",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, blockNum.Int64())
			},
			expEthBlock: nil,
			wantPass:    false,
		},
		{
			name:        "pass - block without tx",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, blockNum.Int64())
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				baseFee := math.NewInt(1)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: gethcore.NewBlock(
				rpc.EthHeaderFromTendermint(
					emptyBlock.Header,
					gethcore.Bloom{},
					math.NewInt(1).BigInt(),
				),
				[]*gethcore.Transaction{},
				nil,
				nil,
				nil,
			),
			wantPass: true,
		},
		{
			name:        "pass - block with tx",
			blockNumber: rpc.BlockNumber(1),
			registerMock: func(blockNum rpc.BlockNumber) {
				height := blockNum.Int64()
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, bz)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, blockNum.Int64())
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				baseFee := math.NewInt(1)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: gethcore.NewBlock(
				rpc.EthHeaderFromTendermint(
					emptyBlock.Header,
					gethcore.Bloom{},
					math.NewInt(1).BigInt(),
				),
				[]*gethcore.Transaction{msgEthereumTx.AsTransaction()},
				nil,
				nil,
				trie.NewStackTrie(nil),
			),
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(tc.blockNumber)

			ethBlock, err := s.backend.EthBlockByNumber(tc.blockNumber)

			if tc.wantPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expEthBlock.Header(), ethBlock.Header())
				s.Require().Equal(tc.expEthBlock.Uncles(), ethBlock.Uncles())
				s.Require().Equal(tc.expEthBlock.ReceiptHash(), ethBlock.ReceiptHash())
				for i, tx := range tc.expEthBlock.Transactions() {
					s.Require().Equal(tx.Data(), ethBlock.Transactions()[i].Data())
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestEthBlockFromTendermintBlock() {
	msgEthereumTx, bz := s.buildEthereumTx()
	emptyBlock := cmt.MakeBlock(1, []cmt.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		baseFee      *big.Int
		resBlock     *cmtrpc.ResultBlock
		blockRes     *cmtrpc.ResultBlockResults
		registerMock func(math.Int, int64)
		expEthBlock  *gethcore.Block
		wantPass     bool
	}{
		{
			name:    "pass - block without tx",
			baseFee: math.NewInt(1).BigInt(),
			resBlock: &cmtrpc.ResultBlock{
				Block: emptyBlock,
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
			},
			registerMock: func(baseFee math.Int, blockNum int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: gethcore.NewBlock(
				rpc.EthHeaderFromTendermint(
					emptyBlock.Header,
					gethcore.Bloom{},
					math.NewInt(1).BigInt(),
				),
				[]*gethcore.Transaction{},
				nil,
				nil,
				nil,
			),
			wantPass: true,
		},
		{
			name:    "pass - block with tx",
			baseFee: math.NewInt(1).BigInt(),
			resBlock: &cmtrpc.ResultBlock{
				Block: cmt.MakeBlock(1, []cmt.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpc.ResultBlockResults{
				Height:     1,
				TxsResults: []*types.ResponseDeliverTx{{Code: 0, GasUsed: 0}},
				EndBlockEvents: []types.Event{
					{
						Type: evm.EventTypeBlockBloom,
						Attributes: []types.EventAttribute{
							{Key: evm.AttributeKeyEthereumBloom},
						},
					},
				},
			},
			registerMock: func(baseFee math.Int, blockNum int64) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: gethcore.NewBlock(
				rpc.EthHeaderFromTendermint(
					emptyBlock.Header,
					gethcore.Bloom{},
					math.NewInt(1).BigInt(),
				),
				[]*gethcore.Transaction{msgEthereumTx.AsTransaction()},
				nil,
				nil,
				trie.NewStackTrie(nil),
			),
			wantPass: true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(math.NewIntFromBigInt(tc.baseFee), tc.blockRes.Height)

			ethBlock, err := s.backend.EthBlockFromTendermintBlock(tc.resBlock, tc.blockRes)

			if tc.wantPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expEthBlock.Header(), ethBlock.Header())
				s.Require().Equal(tc.expEthBlock.Uncles(), ethBlock.Uncles())
				s.Require().Equal(tc.expEthBlock.ReceiptHash(), ethBlock.ReceiptHash())
				for i, tx := range tc.expEthBlock.Transactions() {
					s.Require().Equal(tx.Data(), ethBlock.Transactions()[i].Data())
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}
