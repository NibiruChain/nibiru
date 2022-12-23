package testapp_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

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
	s.twapLookbackWindow = oracletypes.DefaultTwapLookbackWindow
}

func (s *TestappSuite) TestNewTestGenesisState() {
	encodingConfig := app.MakeTestEncodingConfig()
	codec := encodingConfig.Marshaler

	defaultGenState := app.NewDefaultGenesisState(codec)
	testGenState := simapp.NewTestGenesisStateFromDefault()

	var testGenOracleState oracletypes.GenesisState
	testGenOracleStateJSON := testGenState[oracletypes.ModuleName]
	codec.MustUnmarshalJSON(testGenOracleStateJSON, &testGenOracleState)
	bzTest := codec.MustMarshalJSON(&testGenOracleState)

	var defaultGenOracleState oracletypes.GenesisState
	defaultGenOracleStateJSON := defaultGenState[oracletypes.ModuleName]
	codec.MustUnmarshalJSON(defaultGenOracleStateJSON, &defaultGenOracleState)
	bzDefault := codec.MustMarshalJSON(&defaultGenOracleState)

	s.Assert().EqualValues(bzTest, bzDefault)
	s.Assert().EqualValues(testGenOracleState, defaultGenOracleState)
}

func TestTestappSuite(t *testing.T) {
	suite.Run(t, new(TestappSuite))
}
