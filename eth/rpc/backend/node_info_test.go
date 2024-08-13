package backend

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend/mocks"
)

func (s *BackendSuite) TestRPCMinGasPrice() {
	testCases := []struct {
		name           string
		registerMock   func()
		expMinGasPrice int64
		expPass        bool
	}{
		{
			"pass - default gas price",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			eth.DefaultGasPrice,
			true,
		},
		{
			"pass - min gas price is 0",
			func() {
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeader(queryClient, 1)
			},
			eth.DefaultGasPrice,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			minPrice := s.backend.RPCMinGasPrice()
			if tc.expPass {
				s.Require().Equal(tc.expMinGasPrice, minPrice)
			} else {
				s.Require().NotEqual(tc.expMinGasPrice, minPrice)
			}
		})
	}
}

func (s *BackendSuite) TestAccounts() {
	testCases := []struct {
		name         string
		registerMock func()
		expAddr      []common.Address
		expPass      bool
	}{
		{
			"pass - returns empty address",
			func() {},
			[]common.Address{},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			output, err := s.backend.Accounts()

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expAddr, output)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestSyncing() {
	testCases := []struct {
		name         string
		registerMock func()
		expResponse  interface{}
		expPass      bool
	}{
		{
			"fail - Can't get status",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			false,
			false,
		},
		{
			"pass - Node not catching up",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatus(client)
			},
			false,
			true,
		},
		{
			"pass - Node is catching up",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatus(client)
				status, _ := client.Status(s.backend.ctx)
				status.SyncInfo.CatchingUp = true
			},
			map[string]interface{}{
				"startingBlock": hexutil.Uint64(0),
				"currentBlock":  hexutil.Uint64(0),
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			output, err := s.backend.Syncing()

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expResponse, output)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
