package cw_struct_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

type TestSuiteJsonMarshalQuery struct {
	suite.Suite
	fileJson map[string]json.RawMessage
}

func TestSuiteJsonMarshalQuery_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteJsonMarshalQuery))
}

func (s *TestSuiteJsonMarshalQuery) SetupSuite() {
	app.SetPrefixes("nibi")
	file, err := os.Open("queries.json")
	s.NoError(err)
	defer file.Close()

	var fileJson map[string]json.RawMessage
	err = json.NewDecoder(file).Decode(&fileJson)
	s.NoError(err)
	s.fileJson = fileJson
}

func (s *TestSuiteJsonMarshalQuery) TestQueries() {
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

func (s *TestSuiteJsonMarshalQuery) TestToAppMarket() {
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
	sadCwMarket.Pair = "fxs:ust:xxx-yyy!!!"
	_, err := sadCwMarket.ToAppMarket()
	s.Error(err)
}
