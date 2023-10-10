package bindings_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
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
		"all_markets":      new(bindings.AllMarketsResponse),
		"reserves":         new(bindings.ReservesResponse),
		"base_price":       new(bindings.BasePriceResponse),
		"position":         new(bindings.PositionResponse),
		"positions":        new(bindings.PositionsResponse),
		"module_params":    new(bindings.PerpParamsResponse),
		"premium_fraction": new(bindings.PremiumFractionResponse),
		"metrics":          new(bindings.MetricsResponse),
		"module_accounts":  new(bindings.ModuleAccountsResponse),
		"oracle_prices":    new(bindings.OraclePricesResponse),
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
	var lastCwMarket bindings.Market
	for _, ammMarket := range genesis.START_MARKETS {
		dummyBlockHeight := int64(1)
		cwMarket := bindings.NewMarket(
			ammMarket.Market,
			ammMarket.Amm,
			"index price",
			ammMarket.Amm.InstMarkPrice().String(),
			dummyBlockHeight,
		)

		// Test the ToAppMarket fn
		gotAppMarket, err := cwMarket.ToAppMarket()
		s.Assert().NoError(err)
		s.Assert().EqualValues(ammMarket.Market, gotAppMarket)

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
	fileJson := getFileJson(t)

	testCaseMap := []string{
		"market_order",
		"close_position",
		"add_margin",
		"remove_margin",
		"donate_to_insurance_fund",
		"peg_shift",
		"depth_shift",
		"edit_oracle_params",
		"set_market_enabled",
		"insurance_fund_withdraw",
		"create_market",
		"no_op",
	}

	for _, name := range testCaseMap {
		t.Run(name, func(t *testing.T) {
			var bindingMsg bindings.NibiruMsg
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

			var reconsitutedBindingMsg bindings.NibiruMsg
			err = json.Unmarshal(compactFileBytes.Bytes(), &reconsitutedBindingMsg)
			require.NoError(t, err)

			compactFileStr := compactFileBytes.String()
			compactJsonStr := compactJsonBz.String()
			require.EqualValuesf(
				t, bindingMsg, reconsitutedBindingMsg,
				"compactFileStr: %s\ncompactJsonStr: ", compactFileStr, compactJsonStr,
			)
		})
	}
}

func compactJsonData(data []byte) (*bytes.Buffer, error) {
	compactData := new(bytes.Buffer)
	err := json.Compact(compactData, data)
	return compactData, err
}
