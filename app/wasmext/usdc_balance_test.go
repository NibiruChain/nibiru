package wasmext_test

import (
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
)

// Contract-originated Stargate queries retain v2.23.0 responses and gas even
// when a corrected external query would select a different asset.
func (s *Suite) TestStargateUSDCCompatibility() {
	deps := evmtest.NewTestDeps()
	wasmAddr := sdk.MustAccAddressFromBech32("nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m")
	for _, token := range []struct {
		address, symbol string
		amount          int64
	}{
		{evm.MainnetSaiUSDC, "USDC", 13_680_317795},
		{evm.MainnetLegacyUSDC, "USDC.e", 2_000001},
		// USDC.e was a literal bank denom before the patch, not an alias.
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
		for _, addr := range []sdk.AccAddress{deps.Sender.NibiruAddr, wasmAddr} {
			s.Require().NoError(testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), addr, sdk.NewCoins(sdk.NewInt64Coin(denom, token.amount))))
		}
	}
	query := wasmext.WasmStargateQuerier(deps.App.GRPCQueryRouter(), deps.App.AppCodec())
	for _, addr := range []sdk.AccAddress{deps.Sender.NibiruAddr, wasmAddr} {
		// Gas values are pinned from the same fixture run against the unmodified
		// v2.23.0 Keeper.Balance implementation, not the patched resolver.
		for _, tc := range []struct {
			token, denom, amount string
			eoaGas, wasmGas      uint64
		}{
			{"USDC", "erc20/" + evm.MainnetLegacyUSDC, "2000001", 104840, 7995},
			{"uSdC", "erc20/" + evm.MainnetLegacyUSDC, "2000001", 104840, 7995},
			{"USDC.e", "USDC.e", "3000002", 5666, 5798},
			{evm.MainnetSaiUSDC, "erc20/" + evm.MainnetSaiUSDC, "13680317795", 104828, 7983},
			{evm.MainnetLegacyUSDC, "erc20/" + evm.MainnetLegacyUSDC, "2000001", 104840, 7995},
			{"erc20/" + evm.MainnetSaiUSDC, "erc20/" + evm.MainnetSaiUSDC, "13680317795", 104972, 8127},
			{"erc20/" + evm.MainnetLegacyUSDC, "erc20/" + evm.MainnetLegacyUSDC, "2000001", 104984, 8139},
		} {
			s.Run(addr.String()+"/"+tc.token, func() {
				ctx := deps.Ctx().WithGasMeter(storetypes.NewInfiniteGasMeter())
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
				wantGas := tc.eoaGas
				if len(addr) == 32 {
					wantGas = tc.wasmGas
				}
				s.Equal(wantGas, ctx.GasMeter().GasConsumed(), "contract query gas must match v2.23.0")
				// The outer context is unchanged; a subsequent external query uses Sai.
				if strings.EqualFold(tc.token, "USDC") {
					external, err := deps.EvmKeeper.Balance(sdk.WrapSDKContext(deps.Ctx()), req)
					s.Require().NoError(err)
					s.Require().NotNil(external.Bank)
					// Kept separate from contract assertions to expose marker leakage.
					s.Equal("erc20/"+evm.MainnetSaiUSDC, external.Bank.CoinDenom)
				}
			})
		}
	}
}
