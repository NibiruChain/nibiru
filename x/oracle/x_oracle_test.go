package oracle_test

import (
	"encoding/json"
	"strings"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/oracle"
)

func TestXOracleAdapterFixtureModeSmartQueries(t *testing.T) {
	deps := evmtest.NewTestDeps()
	contractAddr := instantiateXOracleAdapterFixture(t, &deps)

	t.Run("legacy exchange rate", func(t *testing.T) {
		for _, tc := range []struct {
			pair       string
			symbol     string
			tokenIndex uint16
			price      string
			price18    string
		}{
			{
				pair:       "unibi:uusd",
				symbol:     "nibi",
				tokenIndex: 49,
				price:      "138",
				price18:    "138000000000000000000",
			},
			{
				pair:       "ubtc:uusd",
				symbol:     "btc",
				tokenIndex: 3,
				price:      "420",
				price18:    "420000000000000000000",
			},
		} {
			var resp oracle.XOracleAdapterLegacyExchangeRateResp
			queryXOracleAdapter(
				t,
				deps,
				contractAddr,
				oracle.XOracleAdapterQueryMsg{
					LegacyExchangeRate: &oracle.XOracleAdapterLegacyExchangeRateQuery{
						Pair: tc.pair,
					},
				},
				&resp,
			)

			require.Equal(t, tc.symbol, resp.Symbol)
			require.Equal(t, tc.tokenIndex, resp.TokenIndex)
			require.Equal(t, tc.price, resp.PriceDecimal)
			require.Equal(t, tc.price18, resp.Price18)
			require.EqualValues(t, 18, resp.Decimals)
			require.NotNil(t, resp.UpdateTimeSeconds)
			require.EqualValues(t, deps.Ctx().BlockTime().Unix(), *resp.UpdateTimeSeconds)
		}
	})

	t.Run("legacy exchange rates", func(t *testing.T) {
		var resp oracle.XOracleAdapterLegacyExchangeRatesResp
		queryXOracleAdapter(
			t,
			deps,
			contractAddr,
			oracle.XOracleAdapterQueryMsg{
				LegacyExchangeRates: &oracle.XOracleAdapterLegacyExchangeRatesQuery{},
			},
			&resp,
		)

		require.Len(t, resp.Rates, 5)
		gotBySymbol := map[string]oracle.XOracleAdapterLegacyExchangeRateResp{}
		for _, rate := range resp.Rates {
			gotBySymbol[rate.Symbol] = rate
		}
		for _, tc := range []struct {
			symbol  string
			price18 string
		}{
			{"atom", "138" + strings.Repeat("0", 18)},
			{"btc", "420000000000000000000"},
			{"eth", "69000000000000000000"},
			{"nibi", "138000000000000000000"},
			{"usdc", "69000000000000000000"},
		} {
			require.Equal(t, tc.price18, gotBySymbol[tc.symbol].Price18)
		}
	})

	t.Run("get price by token index", func(t *testing.T) {
		var resp oracle.XOracleAdapterPriceResp
		queryXOracleAdapter(
			t,
			deps,
			contractAddr,
			oracle.XOracleAdapterQueryMsg{
				GetPrice: &oracle.XOracleAdapterGetPriceQuery{Index: 49},
			},
			&resp,
		)

		require.Equal(t, "138", resp.Price)
		require.Nil(t, resp.LastOracleAddress)
		require.NotNil(t, resp.LastUpdateTime)
		require.EqualValues(t, deps.Ctx().BlockTime().Unix(), *resp.LastUpdateTime)
	})
}

func instantiateXOracleAdapterFixture(
	t *testing.T,
	deps *evmtest.TestDeps,
) sdk.AccAddress {
	t.Helper()

	wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	codeID, _, err := wasmPermissionedKeeper.Create(
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		oracle.XOracleAdapterWasm,
		&wasm.AccessConfig{Permission: wasm.AccessTypeEverybody},
	)
	require.NoError(t, err)

	instantiateMsg, err := json.Marshal(oracle.XOracleAdapterInstantiateMsg{
		Owner: deps.Sender.NibiruAddr.String(),
		Mode:  oracle.XOracleAdapterFixtureMode(),
		LegacyMappings: []oracle.XOracleAdapterLegacyMapping{
			{Pair: "uusdc:uusd", TokenIndex: 1},
			{Pair: "ubtc:uusd", TokenIndex: 3},
			{Pair: "ueth:uusd", TokenIndex: 4},
			{Pair: "uatom:uusd", TokenIndex: 5},
			{Pair: "unibi:uusd", TokenIndex: 49},
		},
	})
	require.NoError(t, err)

	contractAddr, _, err := wasmPermissionedKeeper.Instantiate(
		deps.Ctx(),
		codeID,
		deps.Sender.NibiruAddr,
		deps.Sender.NibiruAddr,
		instantiateMsg,
		"test x-oracle adapter",
		sdk.Coins{},
	)
	require.NoError(t, err)

	return contractAddr
}

func queryXOracleAdapter(
	t *testing.T,
	deps evmtest.TestDeps,
	contractAddr sdk.AccAddress,
	queryMsg oracle.XOracleAdapterQueryMsg,
	resp any,
) {
	t.Helper()

	queryBz, err := json.Marshal(queryMsg)
	require.NoError(t, err)

	respBz, err := deps.App.WasmKeeper.QuerySmart(
		deps.Ctx(),
		contractAddr,
		queryBz,
	)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(respBz, resp))
}
