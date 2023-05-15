package cw_struct_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

type TestSuiteBindingJsonTypes struct {
	suite.Suite
	fileJson map[string]json.RawMessage
}

func TestSuiteBindingJsonTypes_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteBindingJsonTypes))
}

func (s *TestSuiteBindingJsonTypes) SetupSuite() {
	app.SetPrefixes(app.AccountAddressPrefix)
	file, err := os.Open("query_resp.json")
	s.NoError(err)
	defer file.Close()

	var fileJson map[string]json.RawMessage
	err = json.NewDecoder(file).Decode(&fileJson)
	s.NoError(err)
	s.fileJson = fileJson
}

func (s *TestSuiteBindingJsonTypes) TestQueries() {
	testCaseMap := map[string]any{
		"all_markets":      new(cw_struct.AllMarketsResponse),
		"reserves":         new(cw_struct.ReservesResponse),
		"base_price":       new(cw_struct.BasePriceResponse),
		"position":         new(cw_struct.PositionResponse),
		"positions":        new(cw_struct.PositionsResponse),
		"module_params":    new(cw_struct.PerpParamsResponse),
		"premium_fraction": new(cw_struct.PremiumFractionResponse),
		"metrics":          new(cw_struct.MetricsResponse),
		"module_accounts":  new(cw_struct.ModuleAccountsResponse),
	}

	for name, cwRespPtr := range testCaseMap {
		s.T().Run(name, func(t *testing.T) {
			err := json.Unmarshal(s.fileJson[name], cwRespPtr)
			s.Assert().NoErrorf(err, "name: %v", name)
			jsonBz, err := json.Marshal(cwRespPtr)
			s.NoErrorf(err, "jsonBz: %s", jsonBz)
		})
	}
}

func (s *TestSuiteBindingJsonTypes) TestToAppMarket() {
	var lastCwMarket cw_struct.Market
	for _, appMarket := range genesis.START_MARKETS {
		dummyBlockHeight := int64(1)
		cwMarket := cw_struct.NewMarket(
			appMarket,
			"index price",
			appMarket.GetMarkPrice().String(),
			dummyBlockHeight,
		)

		// Test the ToAppMarket fn
		gotAppMarket, err := cwMarket.ToAppMarket()
		s.Assert().NoError(err)
		s.Assert().EqualValues(appMarket, gotAppMarket)

		lastCwMarket = cwMarket
	}

	// Test failure case
	sadCwMarket := lastCwMarket
	sadCwMarket.Pair = "ftt:ust:xxx-yyy!!!"
	_, err := sadCwMarket.ToAppMarket()
	s.Error(err)
}

func getFileJson(t *testing.T) (fileJson map[string]json.RawMessage) {
	file, err := os.Open("execute_msg.json")
	require.NoError(t, err)
	defer file.Close()

	err = json.NewDecoder(file).Decode(&fileJson)
	require.NoError(t, err)
	return fileJson
}

func (s *TestSuiteBindingJsonTypes) TestExecuteMsgs() {
	t := s.T()
	var fileJson map[string]json.RawMessage = getFileJson(t)

	testCaseMap := map[string]any{
		"open_position":            new(cw_struct.OpenPosition),
		"close_position":           new(cw_struct.ClosePosition),
		"add_margin":               new(cw_struct.AddMargin),
		"remove_margin":            new(cw_struct.RemoveMargin),
		"multi_liquidate":          new(cw_struct.MultiLiquidate),
		"donate_to_insurance_fund": new(cw_struct.DonateToInsuranceFund),
		"peg_shift":                new(cw_struct.PegShift),
		"depth_shift":              new(cw_struct.DepthShift),
		"insurance_fund_withdraw":  new(cw_struct.InsuranceFundWithdraw),
	}

	for name, cwExecuteMsgPtr := range testCaseMap {
		t.Run(name, func(t *testing.T) {
			err := json.Unmarshal(fileJson[name], cwExecuteMsgPtr)
			assert.NoErrorf(t, err, "name: %v", name)
			jsonBz, err := json.Marshal(cwExecuteMsgPtr)
			assert.NoErrorf(t, err, "jsonBz: %s", jsonBz)
		})
	}
}
