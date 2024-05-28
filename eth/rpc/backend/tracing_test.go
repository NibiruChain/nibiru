package backend

import (
	"fmt"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmlog "github.com/cometbft/cometbft/libs/log"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/eth/indexer"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/x/evm"
)

func (s *BackendSuite) TestTraceTransaction() {
	msgEthereumTx, _ := s.buildEthereumTx()
	msgEthereumTx2, _ := s.buildEthereumTx()

	txHash := msgEthereumTx.AsTransaction().Hash()
	txHash2 := msgEthereumTx2.AsTransaction().Hash()

	priv, _ := ethsecp256k1.GenerateKey()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())

	queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	RegisterParamsWithoutHeader(queryClient, 1)

	armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
	_ = s.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")

	ethSigner := gethcore.LatestSigner(s.backend.ChainConfig())

	txEncoder := s.backend.clientCtx.TxConfig.TxEncoder()

	msgEthereumTx.From = from.String()
	_ = msgEthereumTx.Sign(ethSigner, s.signer)

	tx, _ := msgEthereumTx.BuildTx(s.backend.clientCtx.TxConfig.NewTxBuilder(), evm.DefaultEVMDenom)
	txBz, _ := txEncoder(tx)

	msgEthereumTx2.From = from.String()
	_ = msgEthereumTx2.Sign(ethSigner, s.signer)

	tx2, _ := msgEthereumTx.BuildTx(s.backend.clientCtx.TxConfig.NewTxBuilder(), evm.DefaultEVMDenom)
	txBz2, _ := txEncoder(tx2)

	testCases := []struct {
		name          string
		registerMock  func()
		block         *types.Block
		responseBlock []*abci.ResponseDeliverTx
		expResult     interface{}
		expPass       bool
	}{
		{
			"fail - tx not found",
			func() {},
			&types.Block{Header: types.Header{Height: 1}, Data: types.Data{Txs: []types.Tx{}}},
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
			nil,
			false,
		},
		{
			"fail - block not found",
			func() {
				// var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
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
			map[string]interface{}{"test": "hello"},
			false,
		},
		{
			"pass - transaction found in a block with multiple transactions",
			func() {
				var (
					queryClient       = s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
					client            = s.backend.clientCtx.Client.(*mocks.Client)
					height      int64 = 1
				)
				_, err := RegisterBlockMultipleTxs(client, height, []types.Tx{txBz, txBz2})
				s.Require().NoError(err)
				RegisterTraceTransactionWithPredecessors(queryClient, msgEthereumTx, []*evm.MsgEthereumTx{msgEthereumTx})
				RegisterConsensusParams(client, height)
			},
			&types.Block{Header: types.Header{Height: 1, ChainID: ChainID}, Data: types.Data{Txs: []types.Tx{txBz, txBz2}}},
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
				{
					Code: 0,
					Events: []abci.Event{
						{Type: evm.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
							{Key: "ethereumTxHash", Value: txHash2.Hex()},
							{Key: "txIndex", Value: "1"},
							{Key: "amount", Value: "1000"},
							{Key: "txGasUsed", Value: "21000"},
							{Key: "txHash", Value: ""},
							{Key: "recipient", Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
						}},
					},
				},
			},
			map[string]interface{}{"test": "hello"},
			true,
		},
		{
			"pass - transaction found",
			func() {
				var (
					queryClient       = s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
					client            = s.backend.clientCtx.Client.(*mocks.Client)
					height      int64 = 1
				)
				_, err := RegisterBlock(client, height, txBz)
				s.Require().NoError(err)
				RegisterTraceTransaction(queryClient, msgEthereumTx)
				RegisterConsensusParams(client, height)
			},
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
			map[string]interface{}{"test": "hello"},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			db := dbm.NewMemDB()
			s.backend.indexer = indexer.NewKVIndexer(db, tmlog.NewNopLogger(), s.backend.clientCtx)

			err := s.backend.indexer.IndexBlock(tc.block, tc.responseBlock)
			s.Require().NoError(err)
			txResult, err := s.backend.TraceTransaction(txHash, nil)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expResult, txResult)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestTraceBlock() {
	msgEthTx, bz := s.buildEthereumTx()
	emptyBlock := types.MakeBlock(1, []types.Tx{}, nil, nil)
	emptyBlock.ChainID = ChainID
	filledBlock := types.MakeBlock(1, []types.Tx{bz}, nil, nil)
	filledBlock.ChainID = ChainID
	resBlockEmpty := tmrpctypes.ResultBlock{Block: emptyBlock, BlockID: emptyBlock.LastBlockID}
	resBlockFilled := tmrpctypes.ResultBlock{Block: filledBlock, BlockID: filledBlock.LastBlockID}

	testCases := []struct {
		name            string
		registerMock    func()
		expTraceResults []*evm.TxTraceResult
		resBlock        *tmrpctypes.ResultBlock
		config          *evm.TraceConfig
		expPass         bool
	}{
		{
			"pass - no transaction returning empty array",
			func() {},
			[]*evm.TxTraceResult{},
			&resBlockEmpty,
			&evm.TraceConfig{},
			true,
		},
		{
			"fail - cannot unmarshal data",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterTraceBlock(queryClient, []*evm.MsgEthereumTx{msgEthTx})
				RegisterConsensusParams(client, 1)
			},
			[]*evm.TxTraceResult{},
			&resBlockFilled,
			&evm.TraceConfig{},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			traceResults, err := s.backend.TraceBlock(1, tc.config, tc.resBlock)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expTraceResults, traceResults)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
