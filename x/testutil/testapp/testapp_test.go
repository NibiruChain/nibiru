package testapp_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

type TestappSuite struct {
	suite.Suite

	genOracle sdk.AccAddress
	pairs     common.AssetPairs
}

func (s *TestappSuite) SetupSuite() {
	app.SetPrefixes(app.AccountAddressPrefix)
	s.genOracle = sdk.MustAccAddressFromBech32(testapp.GenOracleAddress)
	s.pairs = pricefeedtypes.DefaultPairs
}

func (s *TestappSuite) TestGenesisPricefeed_A() {
	genPf := testapp.GenesisPricefeed()
	s.Assert().EqualValues(pricefeedtypes.NewParams(s.pairs), genPf.Params)
	s.Assert().EqualValues(pricefeedtypes.NewParams(s.pairs), genPf.Params)
	s.Assert().EqualValues(s.pairs[0].String(), genPf.PostedPrices[0].PairID)
	s.Assert().EqualValues(s.pairs[1].String(), genPf.PostedPrices[1].PairID)
	s.Assert().EqualValues([]sdk.AccAddress{s.genOracle}, genPf.GenesisOracles)
}

func (s *TestappSuite) TestNewTestGenesisState() {
	encodingConfig := app.MakeTestEncodingConfig()
	codec := encodingConfig.Marshaler

	defaultGenState := app.NewDefaultGenesisState(codec)
	testGenState := testapp.NewTestGenesisStateFromDefault()

	var testGenPfState pricefeedtypes.GenesisState
	testGenPfStateJSON := testGenState[pricefeedtypes.ModuleName]
	codec.MustUnmarshalJSON(testGenPfStateJSON, &testGenPfState)
	bzTest := codec.MustMarshalJSON(&testGenPfState)

	var defaultGenPfState pricefeedtypes.GenesisState
	defaultGenPfStateJSON := defaultGenState[pricefeedtypes.ModuleName]
	codec.MustUnmarshalJSON(defaultGenPfStateJSON, &defaultGenPfState)
	bzDefault := codec.MustMarshalJSON(&defaultGenPfState)

	s.Assert().NotEqualValues(bzTest, bzDefault)
	s.Assert().NotEqualValues(testGenPfState, defaultGenPfState)

	s.Assert().EqualValues(pricefeedtypes.NewParams(s.pairs), testGenPfState.Params)
	s.Assert().EqualValues(pricefeedtypes.NewParams(s.pairs), testGenPfState.Params)
	s.Assert().EqualValues(s.pairs[0].String(), testGenPfState.PostedPrices[0].PairID)
	s.Assert().EqualValues(s.pairs[1].String(), testGenPfState.PostedPrices[1].PairID)
	s.Assert().EqualValues([]sdk.AccAddress{s.genOracle}, testGenPfState.GenesisOracles)
}

func (s *TestappSuite) TestGenesisPricefeed_PostedPrices() {
	s.T().Log("no prices posted for default genesis")
	nibiruApp := testapp.NewTestApp(true)
	ctx := nibiruApp.NewContext(false, tmproto.Header{})
	currentPrices := nibiruApp.PricefeedKeeper.GetCurrentPrices(ctx)
	s.Assert().Len(currentPrices, 0)

	s.T().Log("prices posted for testing genesis")
	nibiruApp = testapp.NewTestAppWithGenesis(testapp.NewTestGenesisStateFromDefault())
	ctx = nibiruApp.NewContext(false, tmproto.Header{})
	oracles := []sdk.AccAddress{s.genOracle}
	oracleMap := nibiruApp.PricefeedKeeper.GetOraclesForPairs(ctx, s.pairs)
	for _, pair := range s.pairs {
		s.Assert().EqualValues(oracles, oracleMap[pair])
	}
	currentPrices = nibiruApp.PricefeedKeeper.GetCurrentPrices(ctx)
	s.Assert().Len(currentPrices, 2)
	s.Assert().Equal(common.PairGovStable.String(), currentPrices[0].PairID)
	s.Assert().Equal(common.PairCollStable.String(), currentPrices[1].PairID)
}

func TestTestappSuite(t *testing.T) {
	suite.Run(t, new(TestappSuite))
}
