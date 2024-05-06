package backend

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
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

func (s *BackendSuite) TestSetGasPrice() {
	defaultGasPrice := (*hexutil.Big)(big.NewInt(1))
	testCases := []struct {
		name         string
		registerMock func()
		gasPrice     hexutil.Big
		expOutput    bool
	}{
		{
			"pass - cannot get server config",
			func() {
				s.backend.clientCtx.Viper = viper.New()
			},
			*defaultGasPrice,
			false,
		},
		{
			"pass - cannot find coin denom",
			func() {
				s.backend.clientCtx.Viper = viper.New()
				s.backend.clientCtx.Viper.Set("telemetry.global-labels", []interface{}{})
			},
			*defaultGasPrice,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()
			output := s.backend.SetGasPrice(tc.gasPrice)
			s.Require().Equal(tc.expOutput, output)
		})
	}
}

// TODO: Combine these 2 into one test since the code is identical
func (s *BackendSuite) TestListAccounts() {
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

			output, err := s.backend.ListAccounts()

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expAddr, output)
			} else {
				s.Require().Error(err)
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

func (s *BackendSuite) TestSetEtherbase() {
	testCases := []struct {
		name         string
		registerMock func()
		etherbase    common.Address
		expResult    bool
	}{
		{
			"pass - Failed to get coinbase address",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			common.Address{},
			false,
		},
		{
			"pass - the minimum fee is not set",
			func() {
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, s.acc)
			},
			common.Address{},
			false,
		},
		{
			"fail - error querying for account",
			func() {
				var header metadata.MD
				client := s.backend.clientCtx.Client.(*mocks.Client)
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, s.acc)
				RegisterParams(queryClient, &header, 1)
				c := sdk.NewDecCoin(eth.EthBaseDenom, math.NewIntFromBigInt(big.NewInt(1)))
				s.backend.cfg.SetMinGasPrices(sdk.DecCoins{c})
				delAddr, _ := s.backend.GetCoinbase()
				// account, _ := suite.backend.clientCtx.AccountRetriever.GetAccount(suite.backend.clientCtx, delAddr)
				delCommonAddr := common.BytesToAddress(delAddr.Bytes())
				request := &authtypes.QueryAccountRequest{Address: sdk.AccAddress(delCommonAddr.Bytes()).String()}
				requestMarshal, _ := request.Marshal()
				RegisterABCIQueryWithOptionsError(
					client,
					"/cosmos.auth.v1beta1.Query/Account",
					requestMarshal,
					tmrpcclient.ABCIQueryOptions{Height: int64(1), Prove: false},
				)
			},
			common.Address{},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			output := s.backend.SetEtherbase(tc.etherbase)

			s.Require().Equal(tc.expResult, output)
		})
	}
}

func (s *BackendSuite) TestImportRawKey() {
	priv, _ := ethsecp256k1.GenerateKey()
	privHex := common.Bytes2Hex(priv.Bytes())
	pubAddr := common.BytesToAddress(priv.PubKey().Address().Bytes())

	testCases := []struct {
		name         string
		registerMock func()
		privKey      string
		password     string
		expAddr      common.Address
		expPass      bool
	}{
		{
			"fail - not a valid private key",
			func() {},
			"",
			"",
			common.Address{},
			false,
		},
		{
			"pass - returning correct address",
			func() {},
			privHex,
			"",
			pubAddr,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			output, err := s.backend.ImportRawKey(tc.privKey, tc.password)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expAddr, output)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
