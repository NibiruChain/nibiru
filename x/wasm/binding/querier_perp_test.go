package binding_test

import (
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/wasm/binding"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
	"github.com/NibiruChain/nibiru/x/wasm/binding/wasmbin"

	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestSuitePerpQuerier_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuitePerpQuerier))
}

func SetExchangeRates(
	testSuite suite.Suite,
	nibiru *app.NibiruApp,
	ctx sdk.Context,
) (exchangeRateMap map[asset.Pair]sdk.Dec) {
	s := testSuite
	exchangeRateTuples := []oracletypes.ExchangeRateTuple{
		{
			Pair:         asset.Registry.Pair(denoms.ETH, denoms.NUSD),
			ExchangeRate: sdk.NewDec(1_000)},
		{
			Pair:         asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			ExchangeRate: sdk.NewDec(10)},
	}

	for _, exchangeRateTuple := range exchangeRateTuples {
		pair := exchangeRateTuple.Pair
		exchangeRate := exchangeRateTuple.ExchangeRate
		nibiru.OracleKeeper.SetPrice(ctx, pair, exchangeRate)

		rate, err := nibiru.OracleKeeper.ExchangeRates.Get(ctx, pair)
		s.Assert().NoError(err)
		s.Assert().EqualValues(exchangeRate, rate)
	}

	return oracletypes.ExchangeRateTuples(exchangeRateTuples).ToMap()
}

// ————————————————————————————————————————————————————————————————————————————
// # Test Setup
// ————————————————————————————————————————————————————————————————————————————

type TestSuitePerpQuerier struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress
	queryPlugin      binding.QueryPlugin

	contractPerp sdk.AccAddress
	fields       ExampleFields
	ratesMap     map[asset.Pair]sdk.Dec
}

func SetupPerpGenesis() app.GenesisState {
	genesisState := genesis.NewTestGenesisState()
	genesisState = genesis.AddPerpGenesis(genesisState)
	genesisState = genesis.AddOracleGenesis(genesisState)
	return genesisState
}

func (s *TestSuitePerpQuerier) SetupSuite() {
	s.fields = GetHappyFields()
	sender := testutil.AccAddress()
	s.contractDeployer = sender

	genesisState := SetupPerpGenesis()
	nibiru := testapp.NewNibiruTestApp(genesisState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})
	coins := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(10_000_000)),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(1_420_000)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx

	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
	s.queryPlugin = binding.NewQueryPlugin(
		nibiru.PerpKeeper,
		nibiru.PerpAmmKeeper,
	)
	s.OnSetupEnd()
}

func (s *TestSuitePerpQuerier) OnSetupEnd() {
	s.ratesMap = SetExchangeRates(s.Suite, s.nibiru, s.ctx)
}

// ————————————————————————————————————————————————————————————————————————————
// # Tests
//
// - TestPremiumFraction
// - TestAllMarkets
// - TestMetrics
// - TestModuleAccounts
// - TestModuleParams
// - TestPosition
// ————————————————————————————————————————————————————————————————————————————

func (s *TestSuitePerpQuerier) TestPremiumFraction() {
	testCases := map[string]struct {
		cwReq     *cw_struct.PremiumFractionRequest
		cwResp    *cw_struct.PremiumFractionResponse
		expectErr bool
	}{
		"invalid pair": {
			cwReq:     &cw_struct.PremiumFractionRequest{Pair: "nonsense"},
			expectErr: true,
		},
		"happy": {
			cwReq: &cw_struct.PremiumFractionRequest{Pair: s.fields.Pair},
			cwResp: &cw_struct.PremiumFractionResponse{
				Pair:             s.fields.Pair,
				CPF:              sdk.MustNewDecFromStr("0.5"),
				EstimatedNextCPF: sdk.MustNewDecFromStr("0.5"),
			},
			expectErr: false,
		},
	}

	for name, testCase := range testCases {
		s.T().Run(name, func(t *testing.T) {
			cwResp, err := s.queryPlugin.Perp.PremiumFraction(
				s.ctx, testCase.cwReq,
			)

			if testCase.expectErr {
				s.Error(err)
				return
			}

			s.NoErrorf(err, "cwResp: %s", cwResp)
			s.Assert().EqualValues(cwResp.Pair, cwResp.Pair)
			s.Assert().EqualValues(cwResp.CPF.String(), cwResp.CPF.String())
			s.Assert().EqualValues(cwResp.EstimatedNextCPF.String(), cwResp.EstimatedNextCPF.String())
		})
	}
}

func (s *TestSuitePerpQuerier) TestAllMarkets() {
	type CwMarketMap map[asset.Pair]cw_struct.Market

	marketMap := make(CwMarketMap)
	for pair, appMarket := range genesis.START_MARKETS {
		rate := s.ratesMap[pair]
		cwMarket := cw_struct.NewMarket(
			appMarket,
			rate.String(),
			appMarket.GetMarkPrice().String(),
			s.ctx.BlockHeight(),
		)
		marketMap[pair] = cwMarket

		// Test the ToAppMarket fn
		gotAppMarket, err := cwMarket.ToAppMarket()
		s.Assert().NoError(err)
		s.Assert().EqualValues(appMarket, gotAppMarket)
	}

	testCases := map[string]struct {
		cwReq     *cw_struct.AllMarketsRequest
		marketMap CwMarketMap
		expectErr bool
	}{
		"happy": {
			cwReq:     &cw_struct.AllMarketsRequest{},
			marketMap: marketMap,
			expectErr: false,
		},
	}

	for name, testCase := range testCases {
		s.T().Run(name, func(t *testing.T) {
			cwResp, err := s.queryPlugin.Perp.AllMarkets(s.ctx)

			if testCase.expectErr {
				s.Error(err)
				return
			}

			s.NoErrorf(err, "cwResp: %s", cwResp)
			for pair, cwMarketWant := range testCase.marketMap {
				cwMarketOut := cwResp.MarketMap[pair.String()]

				jsonWant, err := json.Marshal(cwMarketWant)
				s.Assert().NoError(err)
				jsonGot, err := json.Marshal(cwMarketOut)
				s.Assert().NoError(err)

				s.Assert().EqualValuesf(
					cwMarketWant, cwMarketOut,
					"\nwant: %s\ngot: %s", jsonWant, jsonGot,
				)
			}
		})
	}
}

func (s *TestSuitePerpQuerier) TestMetrics() {
	// happy case
	for pair := range genesis.START_MARKETS {
		cwReq := &cw_struct.MetricsRequest{Pair: pair.String()}
		cwResp, err := s.queryPlugin.Perp.Metrics(s.ctx, cwReq)
		s.NoErrorf(err, "cwResp: %s", cwResp)
	}

	// sad case
	cwReq := &cw_struct.MetricsRequest{Pair: "ftt:ust"}
	cwResp, err := s.queryPlugin.Perp.Metrics(s.ctx, cwReq)
	s.Errorf(err, "cwResp: %s", cwResp)
}

func (s *TestSuitePerpQuerier) TestModuleAccounts() {
	cwReq := &cw_struct.ModuleAccountsRequest{}
	cwResp, err := s.queryPlugin.Perp.ModuleAccounts(s.ctx, cwReq)
	s.NoErrorf(err, "\ncwResp: %s", cwResp)
}

func (s *TestSuitePerpQuerier) TestModuleParams() {
	cwReq := &cw_struct.PerpParamsRequest{}
	cwResp, err := s.queryPlugin.Perp.ModuleParams(s.ctx, cwReq)
	s.NoErrorf(err, "\ncwResp: %s", cwResp)

	jsonBz, err := json.Marshal(cwResp)
	s.NoErrorf(err, "jsonBz: %s", jsonBz)

	freshCwResp := new(cw_struct.PerpParamsResponse)
	err = json.Unmarshal(jsonBz, freshCwResp)
	s.NoErrorf(err, "freshCwResp: %s", freshCwResp)
}

func (s *TestSuitePerpQuerier) TestPosition() {
	trader := s.contractDeployer
	pair := genesis.PerpAmmGenesis().Markets[0].Pair
	margin := sdk.NewInt(1_000_000)
	leverage := sdk.NewDec(5)
	baseAmtLimit := sdk.ZeroDec()

	s.T().Log("Request should error since the trader hasn't yet opened a position")
	cwReq := &cw_struct.PositionRequest{
		Trader: trader.String(),
		Pair:   pair.String(),
	}
	cwResp, err := s.queryPlugin.Perp.Position(s.ctx, cwReq)
	s.Errorf(err, "\ncwResp: %s", cwResp)

	s.T().Log("Open a position")
	resp, err := s.nibiru.PerpKeeper.OpenPosition(
		s.ctx, pair, perpammtypes.Direction_LONG,
		trader, margin, leverage, baseAmtLimit,
	)
	s.NoError(err)

	s.T().Log("Successfully query position")
	cwResp, err = s.queryPlugin.Perp.Position(s.ctx, cwReq)
	s.NoErrorf(err, "\ncwResp: %s", cwResp)

	// Verify that the response marshals to JSON
	jsonBz, err := json.Marshal(cwResp)
	s.NoErrorf(err, "jsonBz: %s", jsonBz)
	// and unmarshals from JSON
	freshCwResp := new(cw_struct.PositionResponse)
	err = json.Unmarshal(jsonBz, freshCwResp)
	s.NoErrorf(err, "freshCwResp: %s", freshCwResp)
	s.Assert().EqualValues(resp.ExchangedNotionalValue, leverage.MulInt(margin))
	s.Assert().EqualValues(cwResp.Position.OpenNotional, leverage.MulInt(margin))
	s.Assert().EqualValues(cwResp.Position.Margin, margin.ToDec())

	s.T().Log("fail due to invalid pair")
	cwReq = &cw_struct.PositionRequest{
		Trader: trader.String(),
		Pair:   "ftt:ust:xyz",
	}
	cwResp, err = s.queryPlugin.Perp.Position(s.ctx, cwReq)
	s.Errorf(err, "\ncwResp: %s", cwResp)

	s.T().Log("test multiple positions query")
	positionResponses := []cw_struct.PositionResponse{*freshCwResp}
	s.DoPositionsTest(trader, positionResponses)
}

func (s *TestSuitePerpQuerier) DoPositionsTest(
	trader sdk.AccAddress, responses []cw_struct.PositionResponse,
) {
	s.T().Log("test multiple positions query")
	cwReq := &cw_struct.PositionsRequest{
		Trader: trader.String(),
	}
	cwResp, err := s.queryPlugin.Perp.Positions(s.ctx, cwReq)
	s.NoErrorf(err, "\ncwResp: %s", cwResp)

	for _, resp := range responses {
		pair := resp.Position.Pair
		pos := cwResp.Positions[pair]
		s.Assert().EqualValues(resp.Position, pos)
	}
}
