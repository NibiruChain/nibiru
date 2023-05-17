package cw_struct_test

import (
	"bytes"
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
	var fileJson = getFileJson(t)

	testCaseMap := []string{
		"open_position",
		"close_position",
		"add_margin",
		"remove_margin",
		"donate_to_insurance_fund",
		"peg_shift",
		"depth_shift",
		"oracle_params",
		"set_market_enabled",
		"insurance_fund_withdraw",
	}

	for _, name := range testCaseMap {
		t.Run(name, func(t *testing.T) {
			var bindingMsg cw_struct.BindingMsg
			err := json.Unmarshal(fileJson[name], &bindingMsg)
			assert.NoErrorf(t, err, "name: %v", name)

			jsonBz, err := json.Marshal(bindingMsg)
			assert.NoErrorf(t, err, "jsonBz: %s", jsonBz)

			// Json files are not compacted, so we need to compact them before comparing
			compactJsonBz, err := compactJsonData(jsonBz)
			require.NoError(t, err)

			fileBytes, err := fileJson[name].MarshalJSON()
			require.NoError(t, err)
			compactFileBytes, err := compactJsonData(fileBytes)
			require.NoError(t, err)

			require.Equal(t, compactFileBytes.Bytes(), compactJsonBz.Bytes())
		})
	}
}

func compactJsonData(data []byte) (*bytes.Buffer, error) {
	compactData := new(bytes.Buffer)
	err := json.Compact(compactData, data)
	return compactData, err
}
