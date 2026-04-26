package rpcapi_test

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func (s *BackendSuite) TestSetTxDefaults() {
	testCases := []struct {
		name       string
		jsonTxArgs evm.JsonTxArgs
		wantErr    string
	}{
		{
			name: "happy: minimal args set",
			jsonTxArgs: evm.JsonTxArgs{
				From:  &s.evmSenderEthAddr,
				To:    &recipient,
				Value: (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
			},
			wantErr: "",
		},
		{
			name: "happy: gas price set",
			jsonTxArgs: evm.JsonTxArgs{
				From:     &s.evmSenderEthAddr,
				To:       &recipient,
				GasPrice: (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
				Value:    (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
			},
			wantErr: "",
		},
		{
			name: "sad: no to (contract creation) and no data",
			jsonTxArgs: evm.JsonTxArgs{
				From: &s.evmSenderEthAddr,
			},
			wantErr: "contract creation without any data provided",
		},
		{
			name: "sad: transfer without from specified generates new empty account",
			jsonTxArgs: evm.JsonTxArgs{
				To:    &recipient,
				Value: (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
			},
			wantErr: "insufficient balance",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			jsonTxArgs, err := s.backend.SetTxDefaults(tc.jsonTxArgs)

			if tc.wantErr != "" {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(jsonTxArgs.Nonce)
			s.Require().NotNil(jsonTxArgs.Gas)
			s.Require().Greater(*jsonTxArgs.Nonce, hexutil.Uint64(0))
			s.Require().Greater(*jsonTxArgs.Gas, hexutil.Uint64(0))
			s.Require().Equal(jsonTxArgs.ChainID.ToInt().Int64(), appconst.ETH_CHAIN_ID_DEFAULT)
		})
	}
}

func (s *BackendSuite) TestDoCall() {
	jsonTxArgs := evm.JsonTxArgs{
		From:  &s.evmSenderEthAddr,
		To:    &recipient,
		Value: (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
	}

	txResponse, err := s.backend.DoCall(jsonTxArgs, rpc.EthPendingBlockNumber)
	s.Require().NoError(err)
	s.Require().NotNil(txResponse)
	s.Require().GreaterOrEqual(txResponse.GasUsed, gethparams.TxGas)
	txRespJsonBz, _ := json.Marshal(txResponse)
	s.T().Logf("txResponse from Backend.DoCall on penging block number: %s\n", txRespJsonBz)

	s.T().Log("typed eth_call should propagate the true insufficient-balance error")
	blockNumber := rpc.EthPendingBlockNumber
	callArgs := evm.JsonTxArgs{
		To:    &recipient,
		Value: (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
	}
	_, err = s.cli.EvmRpc.Eth.Call(
		callArgs,
		rpc.BlockNumberOrHash{
			BlockNumber: &blockNumber,
		},
		nil,
	)
	s.Require().ErrorContains(err, "insufficient balance for transfer")
}

func (s *BackendSuite) TestGasPrice() {
	gasPrice, err := s.cli.EvmRpc.Eth.GasPrice()
	s.Require().NoError(err)
	s.Require().NotNil(gasPrice)
	s.Require().Greater(gasPrice.ToInt().Int64(), int64(0))
}
