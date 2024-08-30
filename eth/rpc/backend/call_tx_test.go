package backend

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmtest "github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *BackendSuite) TestResend() {
	txNonce := (hexutil.Uint64)(1)
	baseFee := math.NewInt(1)
	gasPrice := new(hexutil.Big)
	toAddr := evmtest.NewEthPrivAcc().EthAddr
	chainID := (*hexutil.Big)(s.backend.chainID)
	callArgs := evm.JsonTxArgs{
		From:                 nil,
		To:                   &toAddr,
		Gas:                  nil,
		GasPrice:             nil,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		Value:                gasPrice,
		Nonce:                &txNonce,
		Input:                nil,
		Data:                 nil,
		AccessList:           nil,
		ChainID:              chainID,
	}

	testCases := []struct {
		name         string
		registerMock func()
		args         evm.JsonTxArgs
		gasPrice     *hexutil.Big
		gasLimit     *hexutil.Uint64
		expHash      common.Hash
		expPass      bool
	}{
		{
			"fail - Missing transaction nonce",
			func() {},
			evm.JsonTxArgs{
				Nonce: nil,
			},
			nil,
			nil,
			common.Hash{},
			false,
		},
		{
			"pass - Can't set Tx defaults BaseFee disabled",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFeeDisabled(queryClient)
			},
			evm.JsonTxArgs{
				Nonce:   &txNonce,
				ChainID: callArgs.ChainID,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"pass - Can't set Tx defaults",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
			},
			evm.JsonTxArgs{
				Nonce: &txNonce,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"pass - MaxFeePerGas is nil",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFeeDisabled(queryClient)
			},
			evm.JsonTxArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: nil,
				GasPrice:             nil,
				MaxFeePerGas:         nil,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"fail - GasPrice and (MaxFeePerGas or MaxPriorityPerGas specified)",
			func() {},
			evm.JsonTxArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: nil,
				GasPrice:             gasPrice,
				MaxFeePerGas:         gasPrice,
			},
			nil,
			nil,
			common.Hash{},
			false,
		},
		{
			"fail - Block error",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				RegisterBlockError(client, 1)
			},
			evm.JsonTxArgs{
				Nonce: &txNonce,
			},
			nil,
			nil,
			common.Hash{},
			false,
		},
		{
			"pass - MaxFeePerGas is nil",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
			},
			evm.JsonTxArgs{
				Nonce:                &txNonce,
				GasPrice:             nil,
				MaxPriorityFeePerGas: gasPrice,
				MaxFeePerGas:         gasPrice,
				ChainID:              callArgs.ChainID,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"pass - Chain Id is nil",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
			},
			evm.JsonTxArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: gasPrice,
				ChainID:              nil,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"fail - Pending transactions error",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterEstimateGas(queryClient, callArgs)
				RegisterParams(queryClient, &header, 1)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterUnconfirmedTxsError(client, nil)
			},
			evm.JsonTxArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice,
			nil,
			common.Hash{},
			false,
		},
		{
			"fail - Not Ethereum txs",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterEstimateGas(queryClient, callArgs)
				RegisterParams(queryClient, &header, 1)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterUnconfirmedTxsEmpty(client, nil)
			},
			evm.JsonTxArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice,
			nil,
			common.Hash{},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			hash, err := s.backend.Resend(tc.args, tc.gasPrice, tc.gasLimit)

			if tc.expPass {
				s.Require().Equal(tc.expHash, hash)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestSendRawTransaction() {
	ethTx, bz := s.buildEthereumTx()

	// Sign the ethTx
	queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	RegisterParamsWithoutHeader(queryClient, 1)
	ethSigner := gethcore.LatestSigner(s.backend.ChainConfig())
	err := ethTx.Sign(ethSigner, s.signer)
	s.Require().NoError(err)

	rlpEncodedBz, _ := rlp.EncodeToBytes(ethTx.AsTransaction())
	cosmosTx, _ := ethTx.BuildTx(s.backend.clientCtx.TxConfig.NewTxBuilder(), evm.DefaultEVMDenom)
	txBytes, _ := s.backend.clientCtx.TxConfig.TxEncoder()(cosmosTx)

	testCases := []struct {
		name         string
		registerMock func()
		rawTx        []byte
		expHash      common.Hash
		expPass      bool
	}{
		{
			"sad - empty bytes",
			func() {},
			[]byte{},
			common.Hash{},
			false,
		},
		{
			"sad - no RLP encoded bytes",
			func() {},
			bz,
			common.Hash{},
			false,
		},
		{
			"sad - unprotected transactions",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				s.backend.allowUnprotectedTxs = false
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			rlpEncodedBz,
			common.Hash{},
			false,
		},
		{
			"sad - failed to get evm params",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				s.backend.allowUnprotectedTxs = true
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			rlpEncodedBz,
			common.Hash{},
			false,
		},
		{
			"sad - failed to broadcast transaction",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				s.backend.allowUnprotectedTxs = true
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterBroadcastTxError(client, txBytes)
			},
			rlpEncodedBz,
			common.HexToHash(ethTx.Hash),
			false,
		},
		{
			"pass - Gets the correct transaction hash of the eth transaction",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				s.backend.allowUnprotectedTxs = true
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterBroadcastTx(client, txBytes)
			},
			rlpEncodedBz,
			common.HexToHash(ethTx.Hash),
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			hash, err := s.backend.SendRawTransaction(tc.rawTx)

			if tc.expPass {
				s.Require().Equal(tc.expHash, hash)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestDoCall() {
	_, bz := s.buildEthereumTx()
	gasPrice := (*hexutil.Big)(big.NewInt(1))
	toAddr := evmtest.NewEthPrivAcc().EthAddr
	chainID := (*hexutil.Big)(s.backend.chainID)
	callArgs := evm.JsonTxArgs{
		From:                 nil,
		To:                   &toAddr,
		Gas:                  nil,
		GasPrice:             nil,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		Value:                gasPrice,
		Input:                nil,
		Data:                 nil,
		AccessList:           nil,
		ChainID:              chainID,
	}
	argsBz, err := json.Marshal(callArgs)
	s.Require().NoError(err)

	testCases := []struct {
		name         string
		registerMock func()
		blockNum     rpc.BlockNumber
		callArgs     evm.JsonTxArgs
		expEthTx     *evm.MsgEthereumTxResponse
		expPass      bool
	}{
		{
			"fail - Invalid request",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				s.Require().NoError(err)
				RegisterEthCallError(queryClient, &evm.EthCallRequest{Args: argsBz, ChainId: s.backend.chainID.Int64()})
			},
			rpc.BlockNumber(1),
			callArgs,
			&evm.MsgEthereumTxResponse{},
			false,
		},
		{
			"pass - Returned transaction response",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				s.Require().NoError(err)
				RegisterEthCall(queryClient, &evm.EthCallRequest{Args: argsBz, ChainId: s.backend.chainID.Int64()})
			},
			rpc.BlockNumber(1),
			callArgs,
			&evm.MsgEthereumTxResponse{},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			msgEthTx, err := s.backend.DoCall(tc.callArgs, tc.blockNum)

			if tc.expPass {
				s.Require().Equal(tc.expEthTx, msgEthTx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGasPrice() {
	defaultGasPrice := (*hexutil.Big)(big.NewInt(1))

	testCases := []struct {
		name         string
		registerMock func()
		expGas       *hexutil.Big
		expPass      bool
	}{
		{
			"pass - get the default gas price",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			defaultGasPrice,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			gasPrice, err := s.backend.GasPrice()
			if tc.expPass {
				s.Require().Equal(tc.expGas, gasPrice)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
