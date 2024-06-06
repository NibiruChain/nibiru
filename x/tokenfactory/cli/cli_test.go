package cli_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/tokenfactory/cli"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ suite.SetupAllSuite    = (*TestSuite)(nil)
	_ suite.TearDownAllSuite = (*TestSuite)(nil)
)

type TestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
	val     *testutilcli.Validator
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// TestTokenFactory: Runs the test suite with a deterministic order.
func (s *TestSuite) TestTokenFactory() {
	s.Run("CreateDenomTest", s.CreateDenomTest)
	s.Run("MintBurnTest", s.MintBurnTest)
	s.Run("ChangeAdminTest", s.ChangeAdminTest)
}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	testapp.EnsureNibiruPrefix()
	encodingConfig := app.MakeEncodingConfig()
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

func (s *TestSuite) CreateDenomTest() {
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
		types.TFDenom{Creator: creator.String(), Subdenom: "nusd"}.Denom().String(),
		types.TFDenom{Creator: creator.String(), Subdenom: "stnibi"}.Denom().String(),
		types.TFDenom{Creator: creator.String(), Subdenom: "stnusd"}.Denom().String(),
	}
	s.ElementsMatch(denoms, wantDenoms)
}

func (s *TestSuite) MintBurnTest() {
	creator := s.val.Address
	mint := func(coin string, mintTo string, wantErr bool) {
		mintToArg := fmt.Sprintf("--mint-to=%s", mintTo)
		_, err := s.network.ExecTxCmd(
			cli.NewTxCmd(), creator, []string{"mint", coin, mintToArg})
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())
	}

	burn := func(coin string, burnFrom string, wantErr bool) {
		burnFromArg := fmt.Sprintf("--burn-from=%s", burnFrom)
		_, err := s.network.ExecTxCmd(
			cli.NewTxCmd(), creator, []string{"burn", coin, burnFromArg})
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())
	}

	t := s.T()
	t.Log("mint successfully")
	denom := types.TFDenom{
		Creator:  creator.String(),
		Subdenom: "nusd",
	}
	coin := sdk.NewInt64Coin(denom.Denom().String(), 420)
	wantErr := false
	mint(coin.String(), creator.String(), wantErr) // happy

	t.Log("want error: unregistered denom")
	coin.Denom = "notadenom"
	wantErr = true
	mint(coin.String(), creator.String(), wantErr)
	burn(coin.String(), creator.String(), wantErr)

	t.Log("want error: invalid coin")
	mint("notacoin_231,,", creator.String(), wantErr)
	burn("notacoin_231,,", creator.String(), wantErr)

	t.Log(`want error: unable to parse "mint-to" or "burn-from"`)
	coin.Denom = denom.Denom().String()
	mint(coin.String(), "invalidAddr", wantErr)
	burn(coin.String(), "invalidAddr", wantErr)

	t.Log("burn successfully")
	coin.Denom = denom.Denom().String()
	wantErr = false
	burn(coin.String(), creator.String(), wantErr) // happy
}

func (s *TestSuite) ChangeAdminTest() {
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
			cli.NewQueryCmd(), []string{"denom-info", denom.Denom().String()}, infoResp,
		),
	)
	s.Equal(infoResp.Admin, admin.String())

	s.T().Log("Change to a new admin")
	_, err := s.network.ExecTxCmd(
		cli.NewTxCmd(),
		admin, []string{"change-admin", denom.Denom().String(), newAdmin.String()})
	s.Require().NoError(err)

	s.T().Log("Verify new admin is in state")
	infoResp = new(types.QueryDenomInfoResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"denom-info", denom.Denom().String()}, infoResp,
		),
	)
	s.Equal(infoResp.Admin, newAdmin.String())
}

func (s *TestSuite) TestQueryModuleParams() {
	paramResp := new(types.QueryParamsResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"params"}, paramResp,
		),
	)
	s.Equal(paramResp.Params, types.DefaultModuleParams())
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
