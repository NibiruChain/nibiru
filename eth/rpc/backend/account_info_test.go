package backend

import (
	"fmt"
	"math/big"

	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmtest "github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *BackendSuite) TestGetCode() {
	blockNr := rpc.NewBlockNumber(big.NewInt(1))
	contractCode := []byte(
		"0xef616c92f3cfc9e92dc270d6acff9cea213cecc7020a76ee4395af09bdceb4837a1ebdb5735e11e7d3adb6104e0c3ac55180b4ddf5e54d022cc5e8837f6a4f971b",
	)

	testCases := []struct {
		name          string
		addr          common.Address
		blockNrOrHash rpc.BlockNumberOrHash
		registerMock  func(common.Address)
		expPass       bool
		expCode       hexutil.Bytes
	}{
		{
			"fail - BlockHash and BlockNumber are both nil ",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{},
			func(addr common.Address) {},
			false,
			nil,
		},
		{
			"fail - query client errors on getting Code",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(addr common.Address) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterCodeError(queryClient, addr)
			},
			false,
			nil,
		},
		{
			"pass",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(addr common.Address) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterCode(queryClient, addr, contractCode)
			},
			true,
			contractCode,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset
			tc.registerMock(tc.addr)

			code, err := s.backend.GetCode(tc.addr, tc.blockNrOrHash)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expCode, code)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetProof() {
	blockNrInvalid := rpc.NewBlockNumber(big.NewInt(1))
	blockNr := rpc.NewBlockNumber(big.NewInt(4))
	address1 := evmtest.NewEthPrivAcc().EthAddr

	testCases := []struct {
		name          string
		addr          common.Address
		storageKeys   []string
		blockNrOrHash rpc.BlockNumberOrHash
		registerMock  func(rpc.BlockNumber, common.Address)
		expPass       bool
		expAccRes     *rpc.AccountResult
	}{
		{
			"fail - BlockNumeber = 1 (invalidBlockNumber)",
			address1,
			[]string{},
			rpc.BlockNumberOrHash{BlockNumber: &blockNrInvalid},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterAccount(queryClient, addr, blockNrInvalid.Int64())
			},
			false,
			&rpc.AccountResult{},
		},
		{
			"fail - Block doesn't exist",
			address1,
			[]string{},
			rpc.BlockNumberOrHash{BlockNumber: &blockNrInvalid},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, bn.Int64())
			},
			false,
			&rpc.AccountResult{},
		},
		{
			"pass",
			address1,
			[]string{"0x0"},
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(bn rpc.BlockNumber, addr common.Address) {
				s.backend.ctx = rpc.NewContextWithHeight(bn.Int64())

				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterAccount(queryClient, addr, bn.Int64())

				// Use the IAVL height if a valid tendermint height is passed in.
				iavlHeight := bn.Int64()
				RegisterABCIQueryWithOptions(
					client,
					bn.Int64(),
					"store/evm/key",
					evm.StateKey(address1, common.HexToHash("0x0").Bytes()),
					tmrpcclient.ABCIQueryOptions{Height: iavlHeight, Prove: true},
				)
				RegisterABCIQueryWithOptions(
					client,
					bn.Int64(),
					"store/acc/key",
					authtypes.AddressStoreKey(sdk.AccAddress(address1.Bytes())),
					tmrpcclient.ABCIQueryOptions{Height: iavlHeight, Prove: true},
				)
			},
			true,
			&rpc.AccountResult{
				Address:      address1,
				AccountProof: []string{""},
				Balance:      (*hexutil.Big)(big.NewInt(0)),
				CodeHash:     common.HexToHash(""),
				Nonce:        0x0,
				StorageHash:  common.Hash{},
				StorageProof: []rpc.StorageResult{
					{
						Key:   "0x0",
						Value: (*hexutil.Big)(big.NewInt(2)),
						Proof: []string{""},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()
			tc.registerMock(*tc.blockNrOrHash.BlockNumber, tc.addr)

			accRes, err := s.backend.GetProof(tc.addr, tc.storageKeys, tc.blockNrOrHash)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expAccRes, accRes)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetStorageAt() {
	blockNr := rpc.NewBlockNumber(big.NewInt(1))

	testCases := []struct {
		name          string
		addr          common.Address
		key           string
		blockNrOrHash rpc.BlockNumberOrHash
		registerMock  func(common.Address, string, string)
		expPass       bool
		expStorage    hexutil.Bytes
	}{
		{
			"fail - BlockHash and BlockNumber are both nil",
			evmtest.NewEthPrivAcc().EthAddr,
			"0x0",
			rpc.BlockNumberOrHash{},
			func(addr common.Address, key string, storage string) {},
			false,
			nil,
		},
		{
			"fail - query client errors on getting Storage",
			evmtest.NewEthPrivAcc().EthAddr,
			"0x0",
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(addr common.Address, key string, storage string) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStorageAtError(queryClient, addr, key)
			},
			false,
			nil,
		},
		{
			"pass",
			evmtest.NewEthPrivAcc().EthAddr,
			"0x0",
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(addr common.Address, key string, storage string) {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStorageAt(queryClient, addr, key, storage)
			},
			true,
			hexutil.Bytes{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()
			tc.registerMock(tc.addr, tc.key, tc.expStorage.String())

			storage, err := s.backend.GetStorageAt(tc.addr, tc.key, tc.blockNrOrHash)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expStorage, storage)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetEvmGasBalance() {
	blockNr := rpc.NewBlockNumber(big.NewInt(1))

	testCases := []struct {
		name          string
		addr          common.Address
		blockNrOrHash rpc.BlockNumberOrHash
		registerMock  func(rpc.BlockNumber, common.Address)
		expPass       bool
		expBalance    *hexutil.Big
	}{
		{
			"fail - BlockHash and BlockNumber are both nil",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{},
			func(bn rpc.BlockNumber, addr common.Address) {
			},
			false,
			nil,
		},
		{
			"fail - tendermint client failed to get block",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, bn.Int64())
			},
			false,
			nil,
		},
		{
			"fail - query client failed to get balance",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalanceError(queryClient, addr, bn.Int64())
			},
			false,
			nil,
		},
		{
			"fail - invalid balance",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalanceInvalid(queryClient, addr, bn.Int64())
			},
			false,
			nil,
		},
		{
			"fail - pruned node state",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalanceNegative(queryClient, addr, bn.Int64())
			},
			false,
			nil,
		},
		{
			"pass",
			evmtest.NewEthPrivAcc().EthAddr,
			rpc.BlockNumberOrHash{BlockNumber: &blockNr},
			func(bn rpc.BlockNumber, addr common.Address) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				s.Require().NoError(err)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalance(queryClient, addr, bn.Int64())
			},
			true,
			(*hexutil.Big)(big.NewInt(1)),
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()

			// avoid nil pointer reference
			if tc.blockNrOrHash.BlockNumber != nil {
				tc.registerMock(*tc.blockNrOrHash.BlockNumber, tc.addr)
			}

			balance, err := s.backend.GetBalance(tc.addr, tc.blockNrOrHash)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expBalance, balance)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetTransactionCount() {
	testCases := []struct {
		name         string
		accExists    bool
		blockNum     rpc.BlockNumber
		registerMock func(common.Address, rpc.BlockNumber)
		expPass      bool
		expTxCount   hexutil.Uint64
	}{
		{
			"pass - account doesn't exist",
			false,
			rpc.NewBlockNumber(big.NewInt(1)),
			func(addr common.Address, bn rpc.BlockNumber) {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
			},
			true,
			hexutil.Uint64(0),
		},
		{
			"fail - block height is in the future",
			false,
			rpc.NewBlockNumber(big.NewInt(10000)),
			func(addr common.Address, bn rpc.BlockNumber) {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
			},
			false,
			hexutil.Uint64(0),
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()

			addr := evmtest.NewEthPrivAcc().EthAddr
			if tc.accExists {
				addr = common.BytesToAddress(s.acc.Bytes())
			}

			tc.registerMock(addr, tc.blockNum)

			txCount, err := s.backend.GetTransactionCount(addr, tc.blockNum)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expTxCount, *txCount)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
