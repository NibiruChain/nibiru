package cli_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
	"github.com/NibiruChain/nibiru/v2/x/txfees/cli"
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

var (
	_ suite.SetupAllSuite    = (*TestSuite)(nil)
	_ suite.TearDownAllSuite = (*TestSuite)(nil)
)

type TestSuite struct {
	suite.Suite

	cfg     testnetwork.Config
	network *testnetwork.Network
	val     *testnetwork.Validator
}

func TestIntegrationTestSuite(t *testing.T) {
	testutil.RetrySuiteRunIfDbClosed(t, func() {
		suite.Run(t, new(TestSuite))
	}, 2)
}

// TestTokenFactory: Runs the test suite with a deterministic order.
func (s *TestSuite) TestTokenFactory() {

}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	encodingConfig := app.MakeEncodingConfig()
	genState := genesis.NewTestGenesisState(encodingConfig.Codec)
	cfg := testnetwork.BuildNetworkConfig(genState)
	cfg.NumValidators = 1
	network, err := testnetwork.New(s.T(), s.T().TempDir(), cfg)
	s.NoError(err)

	s.cfg = cfg
	s.network = network
	s.val = network.Validators[0]
	s.NoError(s.network.WaitForNextBlock())
}

func (s *TestSuite) TestQueryFeeTokens() {
	feeTokensResp := new(types.QueryFeeTokensResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.GetQueryCmd(), []string{"fee-tokens"}, feeTokensResp,
		),
	)
	s.Equal(feeTokensResp.FeeTokens, types.DefaultGenesis().Feetokens)
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
