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
)

func TestSuitePerpExt_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuitePerpExt))
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

type TestSuitePerpExt struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress
	queryPlugin      binding.QueryPlugin

	contractPerp sdk.AccAddress
	fields       ExampleFields
	ratesMap     map[asset.Pair]sdk.Dec
}

func (s *TestSuitePerpExt) SetupPerpGenesis() app.GenesisState {
	genesisState := genesis.NewTestGenesisState()
	genesisState = genesis.AddPerpGenesis(genesisState)
	genesisState = genesis.AddOracleGenesis(genesisState)
	return genesisState
}

func (s *TestSuitePerpExt) SetupSuite() {
	s.fields = GetHappyFields()
	sender := testutil.AccAddress()
	s.contractDeployer = sender

	genesisState := s.SetupPerpGenesis()
	nibiru := testapp.NewNibiruTestApp(genesisState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})
	coins := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000)),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(420)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx

	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
	s.queryPlugin = *binding.NewQueryPlugin(
		&nibiru.PerpKeeper,
		&nibiru.PerpAmmKeeper,
	)
	s.OnSetupEnd()
}

func (s *TestSuitePerpExt) OnSetupEnd() {
	s.ratesMap = SetExchangeRates(s.Suite, s.nibiru, s.ctx)
}

// ————————————————————————————————————————————————————————————————————————————
// # Tests
//
// - TestPremiumFraction
// - TestAllMarkets
// - TestMetricsHappy
// - TestMetricsSad
// - TestMetricsSad
// - TestMetricsSad
// ————————————————————————————————————————————————————————————————————————————

func (s *TestSuitePerpExt) TestPremiumFraction() {
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

func (s *TestSuitePerpExt) TestAllMarkets() {
	type CwMarketMap map[asset.Pair]cw_struct.Market

	marketMap := make(CwMarketMap)
	for pair, appMarket := range genesis.START_MARKETS {
		rate := s.ratesMap[pair]
		marketMap[pair] = cw_struct.NewMarket(
			appMarket,
			rate.String(),
			appMarket.GetMarkPrice().String(),
			s.ctx.BlockHeight(),
		)
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

func (s *TestSuitePerpExt) TestMetrics_Happy() {
	for pair := range genesis.START_MARKETS {
		cwReq := &cw_struct.MetricsRequest{Pair: pair.String()}
		cwResp, err := s.queryPlugin.Perp.Metrics(s.ctx, cwReq)
		s.NoErrorf(err, "cwResp: %s", cwResp)
	}
}

func (s *TestSuitePerpExt) TestMetrics_Sad() {
	cwReq := &cw_struct.MetricsRequest{Pair: "fxs:ust"}
	cwResp, err := s.queryPlugin.Perp.Metrics(s.ctx, cwReq)
	s.Errorf(err, "cwResp: %s", cwResp)
}
