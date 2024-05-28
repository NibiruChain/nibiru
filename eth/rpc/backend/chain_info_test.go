package backend

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	"google.golang.org/grpc/metadata"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/abci/types"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/x/evm"
	evmtest "github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *BackendSuite) TestBaseFee() {
	baseFee := sdkmath.NewInt(1)

	testCases := []struct {
		name         string
		blockRes     *tmrpctypes.ResultBlockResults
		registerMock func()
		expBaseFee   *big.Int
		expPass      bool
	}{
		// TODO: test(eth): Test base fee query after it's enabled.
		// {
		// 	"fail - grpc BaseFee error",
		// 	&tmrpctypes.ResultBlockResults{Height: 1},
		// 	func() {
		// 		queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
		// 		RegisterBaseFeeError(queryClient)
		// 	},
		// 	nil,
		// 	false,
		// },
		{
			name: "pass - grpc BaseFee error - with non feemarket block event",
			blockRes: &tmrpctypes.ResultBlockResults{
				Height: 1,
				BeginBlockEvents: []types.Event{
					{
						Type: evm.EventTypeBlockBloom,
					},
				},
			},
			registerMock: func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeDisabled(queryClient)
			},
			expBaseFee: nil,
			expPass:    true,
		},
		{
			name:     "pass - base fee or london fork not enabled",
			blockRes: &tmrpctypes.ResultBlockResults{Height: 1},
			registerMock: func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeDisabled(queryClient)
			},
			expBaseFee: nil,
			expPass:    true,
		},
		{
			"pass",
			&tmrpctypes.ResultBlockResults{Height: 1},
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			baseFee.BigInt(),
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			baseFee, err := s.backend.BaseFee(tc.blockRes)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expBaseFee, baseFee)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestChainId() {
	expChainIDNumber, err := eth.ParseEthChainID(eth.EIP155ChainID_Testnet)
	s.Require().NoError(err)
	expChainID := (*hexutil.Big)(expChainIDNumber)
	testCases := []struct {
		name         string
		registerMock func()
		expChainID   *hexutil.Big
		expPass      bool
	}{
		{
			"pass - block is at or past the EIP-155 replay-protection fork block, return chainID from config ",
			func() {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsInvalidHeight(queryClient, &header, int64(1))
			},
			expChainID,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			chainID, err := s.backend.ChainID()
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expChainID, chainID)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestGetCoinbase() {
	validatorAcc := sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes())
	testCases := []struct {
		name         string
		registerMock func()
		accAddr      sdk.AccAddress
		expPass      bool
	}{
		{
			"fail - Can't retrieve status from node",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			validatorAcc,
			false,
		},
		{
			"fail - Can't query validator account",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccountError(queryClient)
			},
			validatorAcc,
			false,
		},
		{
			"pass - Gets coinbase account",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, validatorAcc)
			},
			validatorAcc,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			accAddr, err := s.backend.GetCoinbase()

			if tc.expPass {
				s.Require().Equal(tc.accAddr, accAddr)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestSuggestGasTipCap() {
	testCases := []struct {
		name         string
		registerMock func()
		baseFee      *big.Int
		expGasTipCap *big.Int
		expPass      bool
	}{
		{
			"pass - London hardfork not enabled or feemarket not enabled ",
			func() {},
			nil,
			big.NewInt(0),
			true,
		},
		{
			"pass - Gets the suggest gas tip cap ",
			func() {},
			nil,
			big.NewInt(0),
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			maxDelta, err := s.backend.SuggestGasTipCap(tc.baseFee)

			if tc.expPass {
				s.Require().Equal(tc.expGasTipCap, maxDelta)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestFeeHistory() {
	testCases := []struct {
		name           string
		registerMock   func(validator sdk.AccAddress)
		userBlockCount ethrpc.DecimalOrHex
		latestBlock    ethrpc.BlockNumber
		expFeeHistory  *rpc.FeeHistoryResult
		validator      sdk.AccAddress
		expPass        bool
	}{
		{
			"fail - can't get params ",
			func(validator sdk.AccAddress) {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				s.backend.cfg.JSONRPC.FeeHistoryCap = 0
				RegisterParamsError(queryClient, &header, ethrpc.BlockNumber(1).Int64())
			},
			1,
			-1,
			nil,
			nil,
			false,
		},
		{
			"fail - user block count higher than max block count ",
			func(validator sdk.AccAddress) {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				s.backend.cfg.JSONRPC.FeeHistoryCap = 0
				RegisterParams(queryClient, &header, ethrpc.BlockNumber(1).Int64())
			},
			1,
			-1,
			nil,
			nil,
			false,
		},
		{
			"fail - Tendermint block fetching error ",
			func(validator sdk.AccAddress) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				s.backend.cfg.JSONRPC.FeeHistoryCap = 2
				RegisterBlockError(client, ethrpc.BlockNumber(1).Int64())
			},
			1,
			1,
			nil,
			nil,
			false,
		},
		{
			"fail - Eth block fetching error",
			func(validator sdk.AccAddress) {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				s.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, ethrpc.BlockNumber(1).Int64(), nil)
				s.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			1,
			1,
			nil,
			nil,
			true,
		},
		{
			name: "pass - Valid FeeHistoryResults object",
			registerMock: func(validator sdk.AccAddress) {
				baseFee := sdkmath.NewInt(1)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				s.backend.cfg.JSONRPC.FeeHistoryCap = 2
				blockHeight := int64(1)
				_, err := RegisterBlock(client, blockHeight, nil)
				s.Require().NoError(err)

				_, err = RegisterBlockResults(client, blockHeight)
				s.Require().NoError(err)

				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
				RegisterConsensusParams(client, blockHeight)

				header := new(metadata.MD)
				RegisterParams(queryClient, header, blockHeight)
				RegisterParamsWithoutHeader(queryClient, blockHeight)
			},
			userBlockCount: 1,
			latestBlock:    1,
			expFeeHistory: &rpc.FeeHistoryResult{
				OldestBlock:  (*hexutil.Big)(big.NewInt(1)),
				BaseFee:      []*hexutil.Big{(*hexutil.Big)(big.NewInt(1)), (*hexutil.Big)(big.NewInt(1))},
				GasUsedRatio: []float64{0},
				Reward:       [][]*hexutil.Big{{(*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0))}},
			},
			validator: sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes()),
			expPass:   true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock(tc.validator)

			feeHistory, err := s.backend.FeeHistory(tc.userBlockCount, tc.latestBlock, []float64{25, 50, 75, 100})
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(feeHistory, tc.expFeeHistory)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
