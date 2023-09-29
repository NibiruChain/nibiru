package cli_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/tokenfactory/cli"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

var _ suite.SetupAllSuite = (*IntegrationTestSuite)(nil)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
	val     *testutilcli.Validator
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestTokenFactory() {
	s.T().Run("CreateDenomTest", s.CreateDenomTest)
	s.T().Run("ChangeAdminTest", s.ChangeAdminTest)
}

func (s *IntegrationTestSuite) SetupSuite() {
	// Make test skip if -short is not used:
	// All tests: `go test ./...`
	// Unit tests only: `go test ./... -short`
	// Integration tests only: `go test ./... -run Integration`
	// See: https://stackoverflow.com/a/41407042/13305627
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	testapp.EnsureNibiruPrefix()
	encodingConfig := app.MakeEncodingConfigAndRegister()
	genState := genesis.NewTestGenesisState(encodingConfig)
	cfg := testutilcli.BuildNetworkConfig(genState)
	cfg.NumValidators = 1
	network, err := testutilcli.New(s.T(), s.T().TempDir(), cfg)
	s.NoError(err)

	s.cfg = cfg
	s.network = network
	s.val = network.Validators[0]
	s.NoError(s.network.WaitForNextBlock())
}

func (s *IntegrationTestSuite) CreateDenomTest(t *testing.T) {
	creator := s.val.Address
	createDenom := func(subdenom string, wantErr bool) {
		_, err := s.network.ExecTxCmd(
			cli.NewTxCmd(),
			creator, []string{"create-denom", subdenom})
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())
	}

	createDenom("nusd", false)
	createDenom("nusd", true) // Can't create the same denom twice.
	createDenom("stnibi", false)
	createDenom("stnusd", false)

	denomResp := new(types.QueryDenomsResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.CmdQueryDenoms(), []string{creator.String()}, denomResp,
		),
	)
	denoms := denomResp.Denoms
	wantDenoms := []string{
		types.TFDenom{Creator: creator.String(), Subdenom: "nusd"}.String(),
		types.TFDenom{Creator: creator.String(), Subdenom: "stnibi"}.String(),
		types.TFDenom{Creator: creator.String(), Subdenom: "stnusd"}.String(),
	}
	s.ElementsMatch(denoms, wantDenoms)
}

func (s *IntegrationTestSuite) ChangeAdminTest(t *testing.T) {
	creator := s.val.Address
	admin := creator
	newAdmin := testutil.AccAddress()
	denom := types.TFDenom{
		Creator:  creator.String(),
		Subdenom: "stnibi",
	}

	s.T().Log("Verify current admin is creator")
	infoResp := new(types.QueryDenomInfoResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"denom-info", denom.String()}, infoResp,
		),
	)
	s.Equal(infoResp.Admin, admin.String())

	s.T().Log("Change to a new admin")
	_, err := s.network.ExecTxCmd(
		cli.NewTxCmd(),
		admin, []string{"change-admin", denom.String(), newAdmin.String()})
	s.Require().NoError(err)

	s.T().Log("Verify new admin is in state")
	infoResp = new(types.QueryDenomInfoResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"denom-info", denom.String()}, infoResp,
		),
	)
	s.Equal(infoResp.Admin, newAdmin.String())
}

func (s *IntegrationTestSuite) TestQueryModuleParams() {
	paramResp := new(types.QueryParamsResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"params"}, paramResp,
		),
	)
	s.Equal(paramResp.Params, types.DefaultModuleParams())
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
