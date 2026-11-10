package wasmext_test

import (
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
)

// Changing these aliases changes contract-observable results. Contract and
// external queries must use one definition, activated in a coordinated release.
func (s *Suite) TestStargateUSDCResolution() {
	deps := evmtest.NewTestDeps()
	eoaAddr := sdk.AccAddress(gethcommon.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
	wasmAddr := sdk.MustAccAddressFromBech32("nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m")
	for _, token := range []struct {
		address, symbol string
		amount          int64
	}{
		{evm.MainnetSaiUSDC, "USDC", 13_680_317795},
		{evm.MainnetLegacyUSDC, "USDC.e", 2_000001},
		// A literal denom must not shadow the explicitly supported USDC.e alias.
		{"", "USDC.e", 3_000002},
	} {
		denom := "erc20/" + token.address
		if token.address == "" {
			denom = token.symbol
		}
		deps.App.BankKeeper.SetDenomMetaData(deps.Ctx(), banktypes.Metadata{
			DenomUnits: []*banktypes.DenomUnit{{Denom: denom, Exponent: 0}, {Denom: token.symbol, Exponent: 6}},
			Base:       denom, Display: token.symbol, Name: token.symbol, Symbol: token.symbol,
		})
		if token.address != "" {
			s.Require().NoError(deps.EvmKeeper.FunTokens.SafeInsert(deps.Ctx(), gethcommon.HexToAddress(token.address), denom, false))
		}
		for _, addr := range []sdk.AccAddress{eoaAddr, wasmAddr} {
			s.Require().NoError(testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), addr, sdk.NewCoins(sdk.NewInt64Coin(denom, token.amount))))
		}
	}
	query := wasmkeeper.AcceptListStargateQuerier(wasmext.WasmAcceptedStargateQueries(), deps.App.GRPCQueryRouter(), deps.App.AppCodec())
	for _, addr := range []sdk.AccAddress{eoaAddr, wasmAddr} {
		for _, tc := range []struct{ token, denom, amount string }{
			{"USDC", "erc20/" + evm.MainnetSaiUSDC, "13680317795"},
			{"uSdC", "erc20/" + evm.MainnetSaiUSDC, "13680317795"},
			{"USDC.e", "erc20/" + evm.MainnetLegacyUSDC, "2000001"},
			{evm.MainnetSaiUSDC, "erc20/" + evm.MainnetSaiUSDC, "13680317795"},
			{evm.MainnetLegacyUSDC, "erc20/" + evm.MainnetLegacyUSDC, "2000001"},
			{"erc20/" + evm.MainnetSaiUSDC, "erc20/" + evm.MainnetSaiUSDC, "13680317795"},
			{"erc20/" + evm.MainnetLegacyUSDC, "erc20/" + evm.MainnetLegacyUSDC, "2000001"},
		} {
			s.Run(addr.String()+"/"+tc.token, func() {
				ctx := deps.Ctx()
				req := &evm.QueryBalanceRequest{Address: addr.String(), Token: tc.token}
				data, err := req.Marshal()
				s.Require().NoError(err)
				raw, err := query(ctx, &wvm.StargateQuery{Path: "/eth.evm.v1.Query/Balance", Data: data})
				s.Require().NoError(err)
				var resp evm.QueryBalanceResponse
				s.Require().NoError(deps.App.AppCodec().UnmarshalJSON(raw, &resp))
				s.Require().NotNil(resp.Bank)
				s.Equal(tc.denom, resp.Bank.CoinDenom)
				s.Equal(tc.amount, resp.Bank.BalanceBase)
				external, err := deps.EvmKeeper.Balance(sdk.WrapSDKContext(deps.Ctx()), req)
				s.Require().NoError(err)
				s.Equal(external, &resp, "contract and external queries must agree")
			})
		}
	}
}
