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
	"github.com/NibiruChain/nibiru/x/perp/amm/cli"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	genesisState := genesis.NewTestGenesisState()
	genesisState = genesis.AddPerpGenesis(genesisState)
	genesisState = genesis.AddOracleGenesis(genesisState)

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
	proposal := &perpammtypes.CreatePoolProposal{
		Title:             "Create ETH:USD pool",
		Description:       "Creates an ETH:USD pool",
		Pair:              "ETH:USD",
		QuoteAssetReserve: sdk.NewDec(1 * common.TO_MICRO),
		BaseAssetReserve:  sdk.NewDec(1 * common.TO_MICRO),
		Config: perpammtypes.MarketConfig{
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
	proposal = &perpammtypes.CreatePoolProposal{}
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

	marketsQueryResp := &perpammtypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdGetMarkets(), nil, marketsQueryResp))

	found := false
	for _, pool := range marketsQueryResp.Markets {
		if pool.Pair.Equal(proposal.Pair) {
			s.EqualValues(perpammtypes.Market{
				Pair:              proposal.Pair,
				BaseAssetReserve:  proposal.BaseAssetReserve,
				QuoteAssetReserve: proposal.QuoteAssetReserve,
				SqrtDepth:         common.MustSqrtDec(proposal.BaseAssetReserve.Mul(proposal.QuoteAssetReserve)),
				Config:            proposal.Config,
				Bias:              sdk.ZeroDec(),
				PegMultiplier:     sdk.OneDec(),
			}, pool)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}

func (s *IntegrationTestSuite) TestGetPrices() {
	val := s.network.Validators[0]

	s.T().Log("check market balances")
	reserveAssets, err := testutilcli.QueryMarketReserveAssets(val.ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD))
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
	startMarket := genesis.START_MARKETS[asset.Registry.Pair(denoms.ETH, denoms.NUSD)]
	proposal := &perpammtypes.EditPoolConfigProposal{
		Title:       "NIP-3: Edit config of the ueth:unusd market",
		Description: "enables higher max leverage on ueth:unusd",
		Pair:        startMarket.Pair,
		Config: perpammtypes.MarketConfig{
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
	proposal = &perpammtypes.EditPoolConfigProposal{}
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
	s.T().Log("verify that the newly proposed market config has been set")
	// ----------------------------------------------------------------------
	s.Require().NoError(s.network.WaitForNextBlock())

	marketsQueryResp := &perpammtypes.QueryAllPoolsResponse{}
	s.Require().NoError(testutilcli.ExecQuery(s.network.Validators[0].ClientCtx, cli.CmdGetMarkets(), nil, marketsQueryResp))

	found := false
	for _, market := range marketsQueryResp.Markets {
		if market.Pair.Equal(proposal.Pair) {
			s.EqualValues(perpammtypes.Market{
				Pair:              proposal.Pair,
				BaseAssetReserve:  startMarket.BaseAssetReserve,
				QuoteAssetReserve: startMarket.QuoteAssetReserve,
				SqrtDepth:         startMarket.SqrtDepth,
				Config:            proposal.Config,
				Bias:              sdk.ZeroDec(),
				PegMultiplier:     sdk.ZeroDec(),
			}, market)
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
	startMarket := genesis.START_MARKETS[asset.Registry.Pair(denoms.NIBI, denoms.NUSD)]
	proposal := &perpammtypes.EditSwapInvariantsProposal{
		Title:       "NIP-4: Change the swap invariant for NIBI.",
		Description: "increase swap invariant for many virtual pools",
		SwapInvariantMaps: []perpammtypes.EditSwapInvariantsProposal_SwapInvariantMultiple{
			{Pair: startMarket.Pair, Multiplier: sdk.NewDec(100)},
		},
	}
	proposalFile := sdktestutil.WriteToNewTempFile(
		s.T(), string(val.ClientCtx.Codec.MustMarshalJSON(proposal)),
	)
	contents, err := os.ReadFile(proposalFile.Name())
	s.Require().NoError(err)

	s.T().Log("Unmarshal json bytes into proposal object; check validity")
	proposal = &perpammtypes.EditSwapInvariantsProposal{}
	val.ClientCtx.Codec.MustUnmarshalJSON(contents, proposal)
	s.Require().NoError(proposal.ValidateBasic())

	marketsQueryResp := new(perpammtypes.QueryAllPoolsResponse)
	s.Require().NoError(testutilcli.ExecQuery(
		s.network.Validators[0].ClientCtx,
		cli.CmdGetMarkets(), nil, marketsQueryResp))
	var marketBefore perpammtypes.Market
	for _, market := range marketsQueryResp.Markets {
		if market.Pair.Equal(proposal.SwapInvariantMaps[0].Pair) {
			marketBefore = market
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

	marketsQueryResp = new(perpammtypes.QueryAllPoolsResponse)
	s.Require().NoError(testutilcli.ExecQuery(
		s.network.Validators[0].ClientCtx, cli.CmdGetMarkets(), nil, marketsQueryResp,
	))

	found := false
	for _, market := range marketsQueryResp.Markets {
		proposalPair := proposal.SwapInvariantMaps[0].Pair

		if market.Pair.Equal(proposalPair) {
			// get multiplier applied to the reserves, which should be 10.
			multiplierToSqrtDepth := common.MustSqrtDec(proposal.SwapInvariantMaps[0].Multiplier)
			s.Assert().EqualValues(sdk.NewDec(10).String(), multiplierToSqrtDepth.String())

			// get market after proposal
			marketAfter := perpammtypes.Market{
				Pair:              proposalPair,
				BaseAssetReserve:  marketBefore.BaseAssetReserve.Mul(multiplierToSqrtDepth),
				QuoteAssetReserve: marketBefore.QuoteAssetReserve.Mul(multiplierToSqrtDepth),
				Config:            marketBefore.Config,
				Bias:              sdk.ZeroDec(),
				PegMultiplier:     sdk.ZeroDec(),
			}
			sqrtDepthAfter, err := marketAfter.ComputeSqrtDepth()
			s.Require().NoError(err)
			marketAfter.SqrtDepth = sqrtDepthAfter

			s.EqualValues(marketAfter, market)
			found = true
		}
	}
	s.True(found, "pool does not exist")
}
