package cli_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

var START_VPOOLS = map[asset.Pair]vpooltypes.Vpool{
	asset.Registry.Pair(denoms.ETH, denoms.NUSD): {
		Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		BaseAssetReserve:  sdk.NewDec(10 * common.MICRO),
		QuoteAssetReserve: sdk.NewDec(60_000 * common.MICRO),
		SqrtDepth:         common.MustSqrtDec(sdk.NewDec(600_000 * common.MICRO * common.MICRO)),
		Config: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	},
	asset.Registry.Pair(denoms.NIBI, denoms.NUSD): {
		Pair:              asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		BaseAssetReserve:  sdk.NewDec(500_000),
		QuoteAssetReserve: sdk.NewDec(5 * common.MICRO),
		SqrtDepth:         common.MustSqrtDec(sdk.NewDec(5 * 500_000 * common.MICRO)),
		Config: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.04"),
			MaxLeverage:            sdk.MustNewDecFromStr("20"),
		},
	},
}

func (s *IntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	encodingConfig := app.MakeTestEncodingConfig()
	genesisState := genesis.NewTestGenesisState()
	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []vpooltypes.Vpool{
		START_VPOOLS[asset.Registry.Pair(denoms.ETH, denoms.NUSD)],
		START_VPOOLS[asset.Registry.Pair(denoms.NIBI, denoms.NUSD)],
	}

	oracleGenesis := oracletypes.DefaultGenesisState()
	oracleGenesis.ExchangeRates = []oracletypes.ExchangeRateTuple{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: sdk.NewDec(1_000)},
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: sdk.NewDec(10)},
	}
	oracleGenesis.Params.VotePeriod = 1_000

	genesisState[vpooltypes.ModuleName] = encodingConfig.Marshaler.
		MustMarshalJSON(vpoolGenesis)

	s.cfg = testutilcli.BuildNetworkConfig(genesisState)
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	s.Require().NoError(s.network.WaitForNextBlock())
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestCmdCreatePoolProposal() {
	val := s.network.Validators[0]

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	proposal := &vpooltypes.CreatePoolProposal{
		Title:             "Create ETH:USD pool",
		Description:       "Creates an ETH:USD pool",
		Pair:              "ETH:USD",
		QuoteAssetReserve: sdk.NewDec(1 * common.MICRO),
		BaseAssetReserve:  sdk.NewDec(1 * common.MICRO),
		Config: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.10"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.05"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.05"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(s.T(), string(val.ClientCtx.Codec.MustMarshalJSON(proposal)))
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &vpooltypes.CreatePoolProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	// ----------------------------------------------------------------------
	s.T().Log("Submit proposal and unmarshal tx response")
	// ----------------------------------------------------------------------
	args := []string{
		proposalFile.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
	}
	cmd := cli.CmdCreatePoolProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	testutilcli.PassGovProposal(s.Suite, s.network)

	// ----------------------------------------------------------------------
	s.T().Log("verify that the new proposed pool exists")
	// ----------------------------------------------------------------------
	s.Require().NoError(s.network.WaitForNextBlock())

	vpoolsQueryResp := &vpooltypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdGetVpools(), nil, vpoolsQueryResp))

	found := false
	for _, pool := range vpoolsQueryResp.Pools {
		if pool.Pair.Equal(proposal.Pair) {
			s.EqualValues(vpooltypes.Vpool{
				Pair:              proposal.Pair,
				BaseAssetReserve:  proposal.BaseAssetReserve,
				QuoteAssetReserve: proposal.QuoteAssetReserve,
				SqrtDepth:         common.MustSqrtDec(proposal.BaseAssetReserve.Mul(proposal.QuoteAssetReserve)),
				Config:            proposal.Config,
			}, pool)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}

func (s *IntegrationTestSuite) TestGetPrices() {
	val := s.network.Validators[0]

	s.T().Log("check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD))
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("10000000"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.MustNewDecFromStr("60000000000"), reserveAssets.QuoteAssetReserve)

	s.T().Log("check prices")
	priceInfo, err := testutilcli.QueryBaseAssetPrice(val.ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD), "add", "100")
	s.T().Logf("priceInfo: %+v", priceInfo)
	s.EqualValues(sdk.MustNewDecFromStr("599994.000059999400006000"), priceInfo.PriceInQuoteDenom)
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestCmdEditPoolConfigProposal() {
	val := s.network.Validators[0]

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	startVpool := START_VPOOLS[asset.Registry.Pair(denoms.ETH, denoms.NUSD)]
	proposal := &vpooltypes.EditPoolConfigProposal{
		Title:       "NIP-3: Edit config of the ueth:unusd vpool",
		Description: "enables higher max leverage on ueth:unusd",
		Pair:        startVpool.Pair,
		Config: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.03"),
			MaxLeverage:            sdk.MustNewDecFromStr("25"),
		},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(s.T(), string(val.ClientCtx.Codec.MustMarshalJSON(proposal)))
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &vpooltypes.EditPoolConfigProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	// ----------------------------------------------------------------------
	s.T().Log("Submit proposal and unmarshal tx response")
	// ----------------------------------------------------------------------
	args := []string{
		proposalFile.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
	}
	cmd := cli.CmdEditPoolConfigProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	testutilcli.PassGovProposal(s.Suite, s.network)

	// ----------------------------------------------------------------------
	s.T().Log("verify that the newly proposed vpool config has been set")
	// ----------------------------------------------------------------------
	s.Require().NoError(s.network.WaitForNextBlock())

	vpoolsQueryResp := &vpooltypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdGetVpools(), nil, vpoolsQueryResp))

	found := false
	for _, vpool := range vpoolsQueryResp.Pools {
		if vpool.Pair.Equal(proposal.Pair) {
			s.EqualValues(vpooltypes.Vpool{
				Pair:              proposal.Pair,
				BaseAssetReserve:  startVpool.BaseAssetReserve,
				QuoteAssetReserve: startVpool.QuoteAssetReserve,
				SqrtDepth:         startVpool.SqrtDepth,
				Config:            proposal.Config,
			}, vpool)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}

func (s *IntegrationTestSuite) TestCmdEditSwapInvariantsProposal() {
	val := s.network.Validators[0]

	// ----------------------------------------------------------------------
	s.T().Log("load example proposal json as bytes")
	// ----------------------------------------------------------------------
	startVpool := START_VPOOLS[asset.Registry.Pair(denoms.NIBI, denoms.NUSD)]
	proposal := &vpooltypes.EditSwapInvariantsProposal{
		Title:       "NIP-4: Change the swap invariant for NIBI.",
		Description: "increase swap invariant for many virtual pools",
		SwapInvariantMaps: []vpooltypes.EditSwapInvariantsProposal_SwapInvariantMultiple{
			{Pair: startVpool.Pair, Multiplier: sdk.NewDec(100)},
		},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(
		s.T(), string(val.ClientCtx.Codec.MustMarshalJSON(proposal)),
	)
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &vpooltypes.EditSwapInvariantsProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	vpoolsQueryResp := new(vpooltypes.QueryAllPoolsResponse)
	s.Require().NoError(testutilcli.ExecQuery(
		s.network.Validators[0].ClientCtx,
		cli.CmdGetVpools(), nil, vpoolsQueryResp))
	var vpoolBefore vpooltypes.Vpool
	for _, vpool := range vpoolsQueryResp.Pools {
		if vpool.Pair.Equal(proposal.SwapInvariantMaps[0].Pair) {
			vpoolBefore = vpool
			break
		}
	}

	// ----------------------------------------------------------------------
	s.T().Log("Submit proposal and unmarshal tx response")
	// ----------------------------------------------------------------------
	args := []string{
		proposalFile.Name(),
		fmt.Sprintf("--%s=1000unibi", govcli.FlagDeposit),
	}
	cmd := cli.CmdEditSwapInvariantsProposal()
	flags.AddTxFlagsToCmd(cmd)
	txResp, err := testutilcli.ExecTx(s.network, cmd, val.Address, args)
	s.Require().NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	testutilcli.PassGovProposal(s.Suite, s.network)

	// ----------------------------------------------------------------------
	s.T().Log("verify that the newly proposed liquidity depth changes go through")
	// ----------------------------------------------------------------------
	s.Require().NoError(s.network.WaitForNextBlock())

	vpoolsQueryResp = new(vpooltypes.QueryAllPoolsResponse)
	s.Require().NoError(testutilcli.ExecQuery(
		s.network.Validators[0].ClientCtx, cli.CmdGetVpools(), nil, vpoolsQueryResp,
	))

	found := false
	for _, vpool := range vpoolsQueryResp.Pools {
		proposalPair := proposal.SwapInvariantMaps[0].Pair

		if vpool.Pair.Equal(proposalPair) {
			// get multiplier applied to the reserves, which should be 10.
			multiplierToSqrtDepth := common.MustSqrtDec(proposal.SwapInvariantMaps[0].Multiplier)
			s.Assert().EqualValues(sdk.NewDec(10).String(), multiplierToSqrtDepth.String())

			// get vpool after proposal
			vpoolAfter := vpooltypes.Vpool{
				Pair:              proposalPair,
				BaseAssetReserve:  vpoolBefore.BaseAssetReserve.Mul(multiplierToSqrtDepth),
				QuoteAssetReserve: vpoolBefore.QuoteAssetReserve.Mul(multiplierToSqrtDepth),
				Config:            vpoolBefore.Config,
			}
			sqrtDepthAfter, err := vpoolAfter.ComputeSqrtDepth()
			s.Require().NoError(err)
			vpoolAfter.SqrtDepth = sqrtDepthAfter

			s.EqualValues(vpoolAfter, vpool)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}
