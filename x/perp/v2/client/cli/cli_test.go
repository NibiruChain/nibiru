package cli_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/collections"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/perp/v2/client/cli"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg        testutilcli.Config
	network    *testutilcli.Network
	users      []sdk.AccAddress
	liquidator sdk.AccAddress
}

func (s *IntegrationTestSuite) SetupSuite() {
	/* 	Make test skip if -short is not used:
	All tests: `go test ./...`
	Unit tests only: `go test ./... -short`
	Integration tests only: `go test ./... -run Integration`
	https://stackoverflow.com/a/41407042/13305627 */
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)

	// setup market
	encodingConfig := app.MakeEncodingConfig()
	genState := genesis.NewTestGenesisState(encodingConfig)
	genState = genesis.AddPerpV2Genesis(genState)
	genState = genesis.AddOracleGenesis(genState)

	oracleGenesis := oracletypes.DefaultGenesisState()
	oracleGenesis.Params.Whitelist = []asset.Pair{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD),
	}
	oracleGenesis.Params.VotePeriod = 1_000
	oracleGenesis.ExchangeRates = []oracletypes.ExchangeRateTuple{
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.NewDec(20_000)},
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: sdk.NewDec(2_000)},
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.NUSD), ExchangeRate: sdk.NewDec(6_000)},
		{Pair: asset.Registry.Pair(denoms.OSMO, denoms.NUSD), ExchangeRate: sdk.NewDec(6_000)},
	}
	genState[oracletypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(oracleGenesis)

	s.cfg = testutilcli.BuildNetworkConfig(genState)
	s.cfg.NumValidators = 1
	s.cfg.Mnemonics = []string{"satisfy december text daring wheat vanish save viable holiday rural vessel shuffle dice skate promote fade badge federal sail during lend fever balance give"}
	network, err := testutilcli.New(
		s.T(),
		s.T().TempDir(),
		s.cfg,
	)
	s.Require().NoError(err)
	s.network = network

	s.NoError(s.network.WaitForNextBlock())

	val := s.network.Validators[0]

	for i := 0; i < 8; i++ {
		newUser := testutilcli.NewAccount(s.network, fmt.Sprintf("user%d", i))
		s.users = append(s.users, newUser)
		s.Require().NoError(
			testutilcli.FillWalletFromValidator(newUser,
				sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 10e6),
					sdk.NewInt64Coin(denoms.USDC, 1e3*common.TO_MICRO),
					sdk.NewInt64Coin(denoms.NUSD, 5e3*common.TO_MICRO),
				),
				val,
				denoms.NIBI,
			),
		)
		s.NoError(s.network.WaitForNextBlock())
	}

	s.liquidator = sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m")
	s.NoError(
		testutilcli.FillWalletFromValidator(s.liquidator,
			sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1000)),
			val,
			denoms.NIBI,
		),
	)
	s.NoError(s.network.WaitForNextBlock())
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestMultiLiquidate() {
	s.T().Log("opening positions")
	_, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[2], []string{
		"buy",
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String(),
		"15",      // Leverage
		"9000000", // Quote asset amount
		"0",       // Base asset limit
	})
	s.Require().NoError(err)

	_, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[3], []string{
		"buy",
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD).String(),
		"15",      // Leverage
		"9000000", // Quote asset amount
		"0",       // Base asset limit
	})
	s.Require().NoError(err)

	s.T().Log("opening counter positions")
	_, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[4], []string{
		"sell",
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String(),
		"15",       // Leverage
		"90000000", // Quote asset amount
		"0",
	})
	s.Require().NoError(err)

	_, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[5], []string{
		"sell",
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD).String(),
		"15",       // Leverage
		"90000000", // Quote asset amount
		"0",
	})
	s.Require().NoError(err)

	s.T().Log("wait 10 blocks")
	height, err := s.network.LatestHeight()
	s.Require().NoError(err)
	_, err = s.network.WaitForHeight(height + 10)
	s.Require().NoError(err)

	s.T().Log("liquidating all users...")
	_, err = testutilcli.ExecTx(s.network, cli.MultiLiquidateCmd(), s.liquidator, []string{
		fmt.Sprintf("%s:%s:%s", denoms.ATOM, denoms.NUSD, s.users[2].String()),
		fmt.Sprintf("%s:%s:%s", denoms.OSMO, denoms.NUSD, s.users[3].String()),
	})
	s.Require().NoError(err)
	err = s.network.WaitForNextBlock()
	s.Require().NoError(err)

	s.T().Log("check trader position")
	_, err = testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.ATOM, denoms.NUSD), s.users[2])
	s.Require().Error(err)

	_, err = testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.OSMO, denoms.NUSD), s.users[3])
	s.Require().Error(err)

	s.T().Log("closing positions")

	_, err = testutilcli.ExecTx(s.network, cli.ClosePositionCmd(), s.users[4], []string{
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String(),
	})
	s.Require().NoError(err)

	_, err = testutilcli.ExecTx(s.network, cli.ClosePositionCmd(), s.users[5], []string{
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD).String(),
	})
	s.Require().NoError(err)
}

// user[0] opens a long position
func (s *IntegrationTestSuite) TestOpenPositionsAndCloseCmd() {
	val := s.network.Validators[0]
	user := s.users[0]

	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	exchangeRate, err := testutilcli.QueryOracleExchangeRate(
		val.ClientCtx, pair,
	)
	s.T().Logf("0. current exchange rate is: %+v", exchangeRate)
	s.NoError(err)

	s.T().Log("A. check market balances")
	ammMarketDuo, err := testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.EqualValues(sdk.NewDec(10e6), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.NewDec(10e6), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("A. check trader has no existing positions")
	_, err = testutilcli.QueryPositionV2(
		val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user,
	)
	s.Error(err)

	s.T().Log("B. open position")
	txResp, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "2000000", // 2*10^6 uNUSD
		/* baseAssetLimit */ "1"},
	)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("B. check market balance after open position")
	ammMarketDuo, err = testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.T().Logf("ammMarketDuo: %s", ammMarketDuo.String())
	s.EqualValues(sdk.MustNewDecFromStr("9999666.677777407419752675"), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.MustNewDecFromStr("10000333.333333333333333333"), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("B. check trader position")
	queryResp, err := testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("333.322222592580247325"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2*common.TO_MICRO), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(2*common.TO_MICRO), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("1999999.999999999999998000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000002000"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatio)

	s.T().Log("C. open position with 2x leverage and zero baseAmtLimit")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "2",
		/* quoteAmt */ "2000000", // 2*10^6 uNUSD
		/* baseAmtLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("C. check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("999.900009999000099990"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(4*common.TO_MICRO), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(6*common.TO_MICRO), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("6000000.000000000000000000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000000"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666666666666666667"), queryResp.MarginRatio)

	s.T().Log("D. Open a reverse position smaller than the existing position")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"sell",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "1000000", // 100 uNUSD
		/* baseAssetLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("D. Check market after opening reverse position")

	ammMarketDuo, err = testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.T().Logf("ammMarketDuo: %s", ammMarketDuo.String())
	s.EqualValues(sdk.MustNewDecFromStr("9999166.736105324556286976"), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.MustNewDecFromStr("10000833.333333333333333333"), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("D. Check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("833.263894675443713024"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(4*common.TO_MICRO), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(5_000_000), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("4999999.999999999999998000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000002000"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.800000000000000000"), queryResp.MarginRatio)

	s.T().Log("E. Open a reverse position larger than the existing position")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"sell",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "8000000", // 8*10^6 uNUSD
		/* baseAssetLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("E. Check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("-500.025001250062503125"), queryResp.Position.Size_)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000002000"), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000002000"), queryResp.Position.Margin)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000000000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000002000"), queryResp.UnrealizedPnl)
	// there is a random delta due to twap margin ratio calculation and random block times in the in-process network
	s.InDelta(1, queryResp.MarginRatio.MustFloat64(), 0.008)

	s.T().Log("F. Close position")
	txResp, err = testutilcli.ExecTx(s.network, cli.ClosePositionCmd(), user, []string{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("F. check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.Error(err)
	s.T().Logf("query response: %+v", queryResp)

	status, ok := status.FromError(err)
	s.True(ok)
	s.EqualValues(codes.InvalidArgument, status.Code())
}

func (s *IntegrationTestSuite) TestPositionEmptyAndClose() {
	val := s.network.Validators[0]
	user := s.users[0]

	// verify trader has no position (empty)
	_, err := testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD), user)
	s.Error(err, "no position found")

	// close position should produce error
	res, err := testutilcli.ExecTx(s.network, cli.ClosePositionCmd(), user, []string{
		asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
	})
	s.NoError(s.network.WaitForNextBlock())
	resp, err := testutilcli.QueryTx(s.network.Validators[0].ClientCtx, res.TxHash)
	s.Require().NoError(err)
	s.Contains(resp.RawLog, collections.ErrNotFound.Error())
}

// user[0] opens a position and removes margin to trigger bad debt
func (s *IntegrationTestSuite) TestRemoveMargin() {
	// Open a position with first user
	s.T().Log("opening a position with user 0")
	_, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[0], []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		"10",      // Leverage
		"1000000", // Quote asset amount
		"0",
	})
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	// Remove margin to trigger bad debt on user 0
	s.T().Log("removing margin on user 0....")
	res, err := testutilcli.ExecTx(s.network, cli.RemoveMarginCmd(), s.users[0], []string{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		fmt.Sprintf("%s%s", "10000000", denoms.NUSD),
	})
	s.NoError(s.network.WaitForNextBlock())
	resp, err := testutilcli.QueryTx(s.network.Validators[0].ClientCtx, res.TxHash)
	s.Require().NoError(err)
	s.Contains(resp.RawLog, types.ErrBadDebt.Error())

	s.T().Log("removing margin on user 0....")
	_, err = testutilcli.ExecTx(s.network, cli.RemoveMarginCmd(), s.users[0], []string{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		fmt.Sprintf("%s%s", "1", denoms.NUSD),
	})
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())
}

// user[1] opens a position and adds margin
func (s *IntegrationTestSuite) TestX_AddMargin() {
	// Open a new position
	s.T().Log("opening a position with user 1....")
	txResp, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[1], []string{
		"buy",
		asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
		"10",      // Leverage
		"1000000", // Quote asset amount
		"0.0000001",
	})
	s.Require().NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	testCases := []struct {
		name           string
		args           []string
		expectedCode   uint32
		expectedMargin sdk.Dec
	}{
		{
			name: "PASS: add margin to correct position",
			args: []string{
				asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", denoms.NUSD),
			},
			expectedCode:   0,
			expectedMargin: sdk.NewDec(1_010_000),
		},
		{
			name: "FAIL: position not found",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", denoms.NUSD),
			},
			expectedCode: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			s.T().Log("adding margin on user 3....")
			txResp, err = testutilcli.ExecTx(s.network, cli.AddMarginCmd(), s.users[1], tc.args, testutilcli.WithTxCanFail(true))
			s.Require().NoError(err)
			s.Require().NoError(s.network.WaitForNextBlock())

			resp, err := testutilcli.QueryTx(s.network.Validators[0].ClientCtx, txResp.TxHash)
			s.Require().NoError(err)
			s.Require().EqualValues(tc.expectedCode, resp.Code)

			if tc.expectedCode == 0 {
				// query trader position
				queryResp, err := testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD), s.users[1])
				s.NoError(err)
				s.EqualValues(tc.expectedMargin, queryResp.Position.Margin)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestDonateToEcosystemFund() {
	s.T().Logf("donate to ecosystem fund")
	out, err := testutilcli.ExecTx(s.network, cli.DonateToEcosystemFundCmd(), sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"), []string{"100unusd"})
	s.NoError(err)
	s.Require().EqualValues(abcitypes.CodeTypeOK, out.Code)

	s.NoError(s.network.WaitForNextBlock())

	resp := new(sdk.Coin)
	s.NoError(
		testutilcli.ExecQuery(
			s.network.Validators[0].ClientCtx,
			bankcli.GetBalancesCmd(),
			[]string{"nibi1trh2mamq64u4g042zfeevvjk4cukrthvppfnc7", "--denom", "unusd"}, // nibi1trh2mamq64u4g042zfeevvjk4cukrthvppfnc7 is the perp_ef module account address
			resp,
		),
	)
	s.Require().EqualValues(sdk.NewInt64Coin("unusd", 100), *resp)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
