package cli_test

import (
	"bytes"
	"context"
	"encoding/json"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmbytes "github.com/NibiruChain/nibiru/v2/bft/libs/bytes"
	cmtrpcclient "github.com/NibiruChain/nibiru/v2/bft/rpc/client"
	rpcclientmock "github.com/NibiruChain/nibiru/v2/bft/rpc/client/mock"
	coretypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/cli"
	sdkclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	svrcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/cmd"
)

type balanceQueryRPC struct {
	rpcclientmock.Client
	request evm.QueryBalanceRequest
	path    string
}

func (m *balanceQueryRPC) ABCIQueryWithOptions(_ context.Context, path string, data tmbytes.HexBytes, _ cmtrpcclient.ABCIQueryOptions) (*coretypes.ResultABCIQuery, error) {
	m.path = path
	if err := m.request.Unmarshal(data); err != nil {
		return nil, err
	}
	response := &evm.QueryBalanceResponse{BalanceWei: "42", Bank: &evm.BalanceBank{
		CoinDenom: "example", BalanceBase: "13680317795", BalanceHuman: "13680.317795", Decimals: 6, Symbol: "USDC",
	}}
	value, err := response.Marshal()
	return &coretypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: value, Height: 1}}, err
}

func (s *Suite) TestCmdQueryBalanceTokenRequest() {
	cases := []struct{ input, want string }{
		{"USDC", "0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190"},
		{"uSdC", "0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190"},
		{"USDC.e", "0x0829F361A05D993d5CEb035cA6DF3446b060970b"},
		{"usdc.E", "0x0829F361A05D993d5CEb035cA6DF3446b060970b"},
		{evm.MainnetSaiUSDC, evm.MainnetSaiUSDC},
		{evm.MainnetLegacyUSDC, evm.MainnetLegacyUSDC},
		{"erc20/" + evm.MainnetSaiUSDC, "erc20/" + evm.MainnetSaiUSDC},
		{"erc20/" + evm.MainnetLegacyUSDC, "erc20/" + evm.MainnetLegacyUSDC},
		{"unibi", "unibi"},
		{"WNIBI", "WNIBI"},
		{"other-denom", "other-denom"},
	}
	for _, tc := range cases {
		s.Run(tc.input, func() {
			rpc := &balanceQueryRPC{}
			clientCtx := (TestCase{}).NewCtx(s).WithClient(rpc)
			cmd := cli.CmdQueryBalance()
			cmd.SetContext(svrcmd.CreateExecuteContext(context.Background()))
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{dummyEthAddr, tc.input})
			s.Require().NoError(sdkclient.SetCmdClientContextHandler(clientCtx, cmd))
			s.Require().NoError(cmd.Execute())
			s.Equal("/eth.evm.v1.Query/Balance", rpc.path)
			s.Equal(tc.want, rpc.request.Token)
			s.Equal(dummyEthAddr, rpc.request.Address)
			var response cli.QueryEvmBalanceResp
			s.Require().NoError(json.Unmarshal(output.Bytes(), &response))
			s.Equal("42", response.BalanceWei)
			s.Require().NotNil(response.BankBalanceBase)
			s.Equal("13680317795", *response.BankBalanceBase)
			s.Equal("13680.317795", *response.BankBalanceHuman)
		})
	}
}
