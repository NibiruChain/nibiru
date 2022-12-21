package testapp_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

type TestappSuite struct {
	suite.Suite

	genOracle          sdk.AccAddress
	pairs              common.AssetPairs
	twapLookbackWindow time.Duration
}

func (s *TestappSuite) SetupSuite() {
	app.SetPrefixes(app.AccountAddressPrefix)
	s.genOracle = sdk.MustAccAddressFromBech32(simapp.GenOracleAddress)
	s.pairs = oracletypes.DefaultPairs
	s.twapLookbackWindow = oracletypes.DefaultLookbackWindow
}

// TestPricefeedGenesis verifies that the expected pricefeed state for integration tests
func (s *TestappSuite) TestPricefeedGenesis() {
	genPf := simapp.PricefeedGenesis()
	s.Assert().EqualValues(oracletypes.NewParams(s.pairs, s.twapLookbackWindow), genPf.Params)
	s.Assert().EqualValues(oracletypes.NewParams(s.pairs, s.twapLookbackWindow), genPf.Params)
	s.Assert().EqualValues(s.pairs[0].String(), genPf.PostedPrices[0].PairID)
	s.Assert().EqualValues(s.pairs[1].String(), genPf.PostedPrices[1].PairID)
	expectedGenesisOracles := []string{s.genOracle.String()}
	for _, oracleStr := range expectedGenesisOracles {
		s.Assert().Contains(genPf.GenesisOracles, oracleStr)
	}
}

func (s *TestappSuite) TestNewTestGenesisState() {
	encodingConfig := app.MakeTestEncodingConfig()
	codec := encodingConfig.Marshaler

	defaultGenState := app.NewDefaultGenesisState(codec)
	testGenState := simapp.NewTestGenesisStateFromDefault()

	var testGenPfState oracletypes.GenesisState
	testGenPfStateJSON := testGenState[oracletypes.ModuleName]
	codec.MustUnmarshalJSON(testGenPfStateJSON, &testGenPfState)
	bzTest := codec.MustMarshalJSON(&testGenPfState)

	var defaultGenPfState oracletypes.GenesisState
	defaultGenPfStateJSON := defaultGenState[oracletypes.ModuleName]
	codec.MustUnmarshalJSON(defaultGenPfStateJSON, &defaultGenPfState)
	bzDefault := codec.MustMarshalJSON(&defaultGenPfState)

	s.Assert().NotEqualValues(bzTest, bzDefault)
	s.Assert().NotEqualValues(testGenPfState, defaultGenPfState)

	s.Assert().EqualValues(oracletypes.NewParams(s.pairs, s.twapLookbackWindow), testGenPfState.Params)
	s.Assert().EqualValues(oracletypes.NewParams(s.pairs, s.twapLookbackWindow), testGenPfState.Params)
	s.Assert().EqualValues(s.pairs[0].String(), testGenPfState.PostedPrices[0].PairID)
	s.Assert().EqualValues(s.pairs[1].String(), testGenPfState.PostedPrices[1].PairID)
	expectedGenesisOracles := []string{s.genOracle.String()}
	for _, oracleStr := range expectedGenesisOracles {
		s.Assert().Contains(testGenPfState.GenesisOracles, oracleStr)
	}
}

func (s *TestappSuite) TestPricefeedGenesis_PostedPrices() {
	s.T().Log("no prices posted for default genesis")
	nibiruApp := simapp.NewTestNibiruApp(true)
	ctx := nibiruApp.NewContext(false, tmproto.Header{})
	currentPrices := nibiruApp.PricefeedKeeper.GetCurrentPrices(ctx)
	s.Assert().Len(currentPrices, 0)

	s.T().Log("prices posted for testing genesis")
	nibiruApp = simapp.NewTestNibiruAppWithGenesis(simapp.NewTestGenesisStateFromDefault())
	ctx = nibiruApp.NewContext(false, tmproto.Header{})
	oracles := []sdk.AccAddress{s.genOracle}
	oracleMap := nibiruApp.PricefeedKeeper.GetOraclesForPairs(ctx, s.pairs)
	for _, pair := range s.pairs {
		s.Assert().EqualValues(oracles, oracleMap[pair])
	}
	currentPrices = nibiruApp.PricefeedKeeper.GetCurrentPrices(ctx)
	s.Assert().Len(currentPrices, 2)
	s.Assert().Equal(common.Pair_NIBI_NUSD.String(), currentPrices[0].PairID)
	s.Assert().Equal(common.Pair_USDC_NUSD.String(), currentPrices[1].PairID)
}

func TestTestappSuite(t *testing.T) {
	suite.Run(t, new(TestappSuite))
}
